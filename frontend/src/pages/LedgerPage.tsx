import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { ledgerApi, type LedgerAccount, type LedgerContext, type LedgerDraft, type LedgerEntry } from '../api/ledger';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import {
  accountKey, findDuplicates, indexDuplicates, memberForAccount, suggestAttribution,
  type DuplicateMatch,
} from '../utils/ledgerMatching.mjs';
import { localDate, manualSourceKey, parseSms, smsFingerprint, toPaise } from '../utils/smsParser.mjs';
import { BankSms, hasAndroidSms } from '../native/sms';
import './LedgerPage.css';

const money = (paise: number) => (paise / 100).toLocaleString('en-IN', { style: 'currency', currency: 'INR' });
const blank = (): LedgerDraft => ({
  source_key: '', source: 'manual', status: 'confirmed', kind: 'expense', amount_paise: 0,
  date: localDate(Date.now()), description: '', category: 'Uncategorized', bank: '', account: '', due_date: '',
  profile_id: '', linked_account_type: '', linked_account_id: '', sms_received_at: '',
});
const errorText = (error: unknown) => error instanceof Error ? error.message : 'Operation failed. Please retry.';
const emptyContext = (): LedgerContext => ({ members: [], accounts: [] });
const linkedKey = (entry: LedgerDraft) => entry.linked_account_id ? `${entry.linked_account_type}:${entry.linked_account_id}` : '';
const accountLabel = (account: LedgerAccount) =>
  `${account.name || 'Unnamed'} (${account.type === 'credit_card' ? 'card' : 'account'}${account.bank ? `, ${account.bank}` : ''}${account.last_four ? `, XX${account.last_four}` : ''}${account.status === 'closed' ? ', closed' : ''})`;

function duplicateSignature(draft: LedgerDraft, matches: DuplicateMatch[], editID: string) {
  return JSON.stringify([
    editID, draft.date, draft.kind, draft.amount_paise, draft.bank, draft.account, draft.profile_id,
    draft.linked_account_type, draft.linked_account_id, draft.sms_received_at,
    matches.map(match => [match.entry.id, match.entry.updated_at, match.reason]).sort(),
  ]);
}

function DuplicateWarnings({ matches, memberName, onShow }: {
  matches: DuplicateMatch[];
  memberName: (id: string) => string;
  onShow: (entry: LedgerEntry) => void;
}) {
  if (matches.length === 0) return null;
  return (
    <div className="ledger-duplicates">
      <strong>Possible duplicate{matches.length === 1 ? '' : 's'} ({matches.length})</strong>
      <p>Compare the records before confirming. Similar payments can be legitimate; nothing is rejected automatically.</p>
      <ul>
        {matches.map(({ entry, reason }) => (
          <li key={entry.id}>
            {reason === 'exact' ? 'Identical SMS already imported' : reason === 'within_minutes' ? 'SMS times are within 5 minutes' : 'Same day; receipt time unavailable'}
            {' — '}{entry.date}, {money(entry.amount_paise)}, {entry.bank || 'bank not specified'}
            {entry.account && ` XX${entry.account}`}, {memberName(entry.profile_id)}, {entry.status}
            {' — '}{entry.description}.{' '}
            <button type="button" className="ledger-link-button" onClick={() => onShow(entry)}>Show record</button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function LedgerPage() {
  const { user, viewAsUserId, isViewOnly } = useAuth();
  const owner = viewAsUserId || user?.id || '';
  const deviceKey = `finboard:ledger-device-member:${user?.id || ''}`;
  const [storedDeviceMember, setStoredDeviceMember] = useState(() => localStorage.getItem(deviceKey) || '');
  const [snapshot, setSnapshot] = useState<{ owner: string; entries: LedgerEntry[]; context: LedgerContext }>({
    owner: '', entries: [], context: emptyContext(),
  });
  const [loadError, setLoadError] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState<LedgerDraft>(blank);
  const [editID, setEditID] = useState('');
  const [amount, setAmount] = useState('');
  const [acknowledged, setAcknowledged] = useState('');
  const [month, setMonth] = useState(() => localDate(Date.now()).slice(0, 7));
  const [status, setStatus] = useState('all');
  const [memberFilter, setMemberFilter] = useState('all');
  const [accountFilter, setAccountFilter] = useState('all');
  const [sender, setSender] = useState('');
  const [body, setBody] = useState('');
  const [receivedAt, setReceivedAt] = useState(() => `${localDate(Date.now())}T12:00`);
  const [days, setDays] = useState(7);
  const [scanMode, setScanMode] = useState<'days' | 'date'>('date');
  const [scanFrom, setScanFrom] = useState(() => `${localDate(Date.now()).slice(0, 7)}-01`);
  const [consent, setConsent] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let active = true;
    Promise.all([ledgerApi.list(), ledgerApi.context()]).then(([ledger, context]) => {
      if (active) {
        setSnapshot({ owner, entries: ledger.data, context: context.data });
        setLoadError('');
      }
    }).catch(err => {
      if (active) setLoadError(apiErrorMessage(err, 'Could not load the ledger and family/account details.'));
    });
    return () => { active = false; };
  }, [owner, reloadKey]);

  const ready = snapshot.owner === owner && !loadError;
  const entries = useMemo(() => snapshot.owner === owner ? snapshot.entries : [], [snapshot.owner, snapshot.entries, owner]);
  const context = snapshot.owner === owner ? snapshot.context : emptyContext();
  const deviceMember = !isViewOnly && context.members.some(member => member.id === storedDeviceMember) ? storedDeviceMember : '';
  const memberName = (id: string) => !id ? 'Unassigned' : context.members.find(member => member.id === id)?.name || `Deleted member (${id.slice(-6)})`;
  const accountName = (entry: LedgerDraft) => {
    if (!entry.linked_account_id) return 'Not linked';
    const account = context.accounts.find(item => accountKey(item) === linkedKey(entry));
    return account ? accountLabel(account) : `Deleted account${entry.account ? ` (XX${entry.account})` : ''}`;
  };
  const matchesAccount = (entry: LedgerDraft) => accountFilter === 'all'
    || (accountFilter === 'unlinked' ? !entry.linked_account_id : linkedKey(entry) === accountFilter);
  const matchesMember = (entry: LedgerDraft) => memberFilter === 'all'
    || (memberFilter === 'unassigned' ? !entry.profile_id : entry.profile_id === memberFilter);
  const inPeriod = (entry: LedgerDraft) => !month || entry.date.startsWith(month);
  const filtered = entries.filter(entry => inPeriod(entry) && matchesAccount(entry) && matchesMember(entry) && (status === 'all' || entry.status === status));
  const total = (kind: LedgerDraft['kind']) => entries.filter(entry =>
    entry.status === 'confirmed' && entry.kind === kind && inPeriod(entry) && matchesAccount(entry) && matchesMember(entry)
  ).reduce((sum, entry) => sum + entry.amount_paise, 0);

  const memberTotals = new Map<string, { income: number; expense: number; refund: number }>();
  for (const member of context.members) memberTotals.set(member.id, { income: 0, expense: 0, refund: 0 });
  for (const entry of entries) {
    if (entry.status !== 'confirmed' || !inPeriod(entry) || !matchesAccount(entry)) continue;
    if (entry.kind !== 'income' && entry.kind !== 'expense' && entry.kind !== 'refund') continue;
    const current = memberTotals.get(entry.profile_id) || { income: 0, expense: 0, refund: 0 };
    current[entry.kind] += entry.amount_paise;
    memberTotals.set(entry.profile_id, current);
  }
  const missingMembers = [...new Set(entries.map(entry => entry.profile_id).filter(id => id && !context.members.some(member => member.id === id)))];
  const missingAccounts = new Map<string, LedgerEntry>();
  for (const entry of entries) {
    const key = linkedKey(entry);
    if (key && !context.accounts.some(account => accountKey(account) === key)) missingAccounts.set(key, entry);
  }
  const duplicatesByID = useMemo(() => indexDuplicates(entries), [entries]);
  const formDraft = useMemo(() => {
    try {
      return { ...form, amount_paise: toPaise(amount), id: editID || undefined };
    } catch {
      return null;
    }
  }, [form, amount, editID]);
  const formMatches = formDraft ? findDuplicates(formDraft, entries) : [];
  const formSignature = formDraft ? duplicateSignature(formDraft, formMatches, editID) : '';

  const reset = () => { setForm(blank()); setEditID(''); setAmount(''); setAcknowledged(''); };
  const refresh = () => setReloadKey(key => key + 1);
  const showEntry = (entry: LedgerEntry) => {
    setMonth(''); setStatus('all'); setMemberFilter('all'); setAccountFilter('all');
    window.setTimeout(() => document.getElementById(`ledger-entry-${entry.id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' }), 50);
  };

  const setDeviceMember = (id: string) => {
    try {
      if (id) localStorage.setItem(deviceKey, id);
      else localStorage.removeItem(deviceKey);
      setStoredDeviceMember(id);
      setError('');
    } catch {
      setError('Could not save this device setting. Check browser or app storage and try again.');
    }
  };

  const suggest = (draft: LedgerDraft): { draft: LedgerDraft; message: string; account: LedgerAccount | null } => {
    const result = suggestAttribution(draft, context, deviceMember);
    const linkedAccountType: LedgerDraft['linked_account_type'] = result.account?.type || '';
    return {
      draft: {
        ...draft, profile_id: result.profileId,
        linked_account_type: linkedAccountType, linked_account_id: result.account?.id || '',
      },
      message: result.message,
      account: result.account,
    };
  };

  const suggestAccountForManualEntry = (draft: LedgerDraft): { draft: LedgerDraft; message: string; account: LedgerAccount | null } => {
    const result = suggest(draft);
    if (!result.account) {
      return { draft: { ...draft, linked_account_type: '', linked_account_id: '' }, message: result.message, account: null };
    }
    const memberIsOwner = result.account.member_ids.includes(draft.profile_id);
    if (!result.draft.profile_id && draft.profile_id && (result.account.member_ids.length === 0 || memberIsOwner)) {
      result.draft.profile_id = draft.profile_id;
    }
    return result;
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (isViewOnly || !ready) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const draft = { ...form, amount_paise: toPaise(amount), source_key: form.source_key || await manualSourceKey() };
      setForm(draft);
      const latest = (await ledgerApi.list()).data;
      setSnapshot(current => current.owner === owner ? { ...current, entries: latest } : current);
      const matches = findDuplicates({ ...draft, id: editID || undefined }, latest);
      if (!editID && matches.some(match => match.reason === 'exact')) {
        setError('This SMS was already imported. Show and review the existing record instead.');
        return;
      }
      if (draft.status === 'confirmed' && matches.length && acknowledged !== duplicateSignature(draft, matches, editID)) {
        setError('Possible duplicates found. Compare them and confirm the review checkbox before saving.');
        return;
      }
      if (editID) await ledgerApi.update(editID, draft);
      else await ledgerApi.create(draft);
      reset(); setBody(''); setNotice('Saved. Only confirmed records affect totals.'); refresh();
    } catch (err) {
      setError(apiErrorMessage(err, errorText(err)));
    } finally { setBusy(false); }
  };

  const previewSms = async () => {
    setError(''); setNotice('');
    try {
      const message = { sender, body, receivedAt };
      const parsed = parseSms(message);
      if (!parsed.draft) { setNotice(parsed.reason || 'No supported transaction.'); return; }
      const result = suggest({ ...parsed.draft, source_key: await smsFingerprint(message) });
      setForm(result.draft); setEditID(''); setAcknowledged('');
      setAmount((parsed.draft.amount_paise / 100).toFixed(2));
      setNotice(`${parsed.warning || ''} ${result.message}`);
    } catch (err) { setError(errorText(err)); }
  };

  const [scanBounds] = useState(() => ({ today: localDate(Date.now()), min: localDate(Date.now() - 89 * 86_400_000) }));
  const [scanSummary, setScanSummary] = useState('');
  const { today, min: scanMinDate } = scanBounds;
  const scanFromMs = new Date(`${scanFrom}T00:00:00`).getTime();
  const scanValid = scanMode === 'date'
    ? /^\d{4}-\d{2}-\d{2}$/.test(scanFrom) && scanFrom >= scanMinDate && scanFrom <= today && Number.isFinite(scanFromMs)
    : Number.isInteger(days) && days >= 1 && days <= 90;

  const scan = async () => {
    if (!consent || isViewOnly || !ready || !scanValid) return;
    setBusy(true); setError(''); setNotice(''); setScanSummary('Scanning messages...');
    let queued = 0;
    let skipped = 0;
    let unlinked = 0;
    let unassigned = 0;
    try {
      const result = await BankSms.scan(scanMode === 'date' ? { days: 90, sinceMs: scanFromMs } : { days });
      for (const message of result.messages) {
        const parsed = parseSms(message);
        if (!parsed.draft) { skipped++; continue; }
        const attributed = suggest({ ...parsed.draft, source_key: await smsFingerprint(message) });
        const saved = (await ledgerApi.create(attributed.draft)).data;
        queued++;
        if (!saved.linked_account_id) unlinked++;
        if (!saved.profile_id) unassigned++;
      }
      setStatus('pending'); setMonth(''); setMemberFilter('all'); setAccountFilter('all');
      const summary = `Found ${result.messages.length} bank messages: ${queued} queued as pending for review, ${skipped} not recognized.${result.truncated ? ' Scan limit reached: choose a later start date.' : ''}`;
      setScanSummary(summary);
      setNotice(`${summary} Existing imports are kept, not duplicated. ${unlinked} could not be linked to one account; ${unassigned} have no suggested member. Review the account, member and possible duplicates on each pending record.`);
    } catch (err) {
      const message = `${queued} alerts processed before scan stopped. ${apiErrorMessage(err, errorText(err))} Retry is safe for identical SMS records.`;
      setError(message); setScanSummary(message);
    } finally { setBusy(false); refresh(); }
  };

  const edit = (entry: LedgerEntry) => {
    setForm(entry); setEditID(entry.id); setAmount((entry.amount_paise / 100).toFixed(2)); setAcknowledged('');
    setNotice('Check the account and member. Choose transfer for repayments; refunds are not necessarily income. Mark a duplicate rejected to exclude it.');
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const exportCSV = async () => {
    setBusy(true); setError('');
    try {
      const result = await ledgerApi.export();
      const url = URL.createObjectURL(result.data);
      const link = document.createElement('a');
      link.href = url; link.download = 'income-expenses.csv'; link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) { setError(apiErrorMessage(err, 'Could not export ledger.')); }
    finally { setBusy(false); }
  };

  return (
    <div className="ledger-page">
      <div className="page-header"><h1>Income &amp; Expenses</h1>
        {!hasAndroidSms && <button className="btn btn-outline" disabled={busy || !ready} onClick={exportCSV}>Export CSV</button>}
      </div>
      <p className="text-muted mb-4">Shared household ledger. Transfers and bill notices are excluded from spending. Records do not update investment cashflow, net worth or the Credit Cards module.</p>
      {loadError && <div role="alert" className="card text-danger">{loadError} <button className="btn btn-outline" onClick={refresh}>Retry</button></div>}
      {!ready && !loadError && <p>Loading ledger and family/account details...</p>}
      {error && <div role="alert" className="card text-danger">{error}</div>}
      {notice && <div role="status" className="card">{notice}</div>}

      {ready && <><div className="card-grid">
        <div className="stat-card"><div className="label">Confirmed income</div><div className="value positive">{money(total('income'))}</div></div>
        <div className="stat-card"><div className="label">Confirmed expenses less refunds</div><div className="value negative">{money(total('expense') - total('refund'))}</div></div>
        <div className="stat-card"><div className="label">Net income minus expenses</div><div className="value">{money(total('income') - total('expense') + total('refund'))}</div></div>
      </div>
      <p className="text-muted">Totals use the selected month, member and account. Only confirmed records count; the status filter affects the list only.</p></>}

      {!isViewOnly && <>
        <section className="card">
          <h2>This phone / device</h2>
          <div className="form-group">
            <label htmlFor="ledger-device-member">This device belongs to</label>
            <select id="ledger-device-member" value={deviceMember} disabled={!ready || busy} onChange={e => setDeviceMember(e.target.value)}>
              <option value="">No default member</option>
              {context.members.map(member => <option value={member.id} key={member.id}>{member.name}</option>)}
            </select>
          </div>
          <p className="text-muted">Saved only on this device for this login. New SMS imports use it when the account owner is unknown, or when it is a joint owner. A sole account owner takes precedence. Review every suggestion.</p>
          {ready && storedDeviceMember && !deviceMember && <p className="text-danger">The saved device member no longer exists. Choose a member again.</p>}
          {ready && context.members.length === 0 && <p>Add members on the <Link to="/family">Family</Link> page first.</p>}
        </section>

        <section className="card">
          <h2>{editID ? 'Review / edit transaction' : 'Add transaction'}</h2>
          <form onSubmit={save}>
            <fieldset disabled={busy || !ready}>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-kind">Type</label>
                  <select id="ledger-kind" value={form.kind} onChange={e => {
                    const kind = e.target.value;
                    if (kind === 'income' || kind === 'expense' || kind === 'refund' || kind === 'transfer' || kind === 'bill') setForm({ ...form, kind });
                  }}><option value="expense">Expense</option><option value="income">Income</option><option value="refund">Expense refund / reversal</option><option value="transfer">Transfer / card repayment</option><option value="bill">Bill notice (not an expense)</option></select>
                </div>
                <div className="form-group"><label htmlFor="ledger-amount">Amount (INR)</label><input id="ledger-amount" inputMode="decimal" required value={amount} onChange={e => setAmount(e.target.value)} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-date">Transaction date</label><input id="ledger-date" type="date" required value={form.date} onChange={e => setForm({ ...form, date: e.target.value })} /></div>
                <div className="form-group"><label htmlFor="ledger-status">Status</label>
                  <select id="ledger-status" value={form.status} disabled={!editID && form.source === 'sms'} onChange={e => {
                    const value = e.target.value;
                    if (value === 'pending' || value === 'confirmed' || value === 'rejected') setForm({ ...form, status: value });
                  }}><option value="pending">Pending review</option><option value="confirmed">Confirmed</option><option value="rejected">Rejected / duplicate</option></select>
                </div>
              </div>
              <div className="form-group"><label htmlFor="ledger-description">Description (do not paste SMS or OTPs here)</label><input id="ledger-description" maxLength={200} required value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-category">Category</label><input id="ledger-category" maxLength={60} value={form.category} onChange={e => setForm({ ...form, category: e.target.value })} /></div>
                <div className="form-group"><label htmlFor="ledger-bank">Bank / card issuer</label><input id="ledger-bank" maxLength={60} value={form.bank} onChange={e => {
                  const suggestion = suggestAccountForManualEntry({ ...form, bank: e.target.value });
                  setForm(suggestion.draft);
                  if (/^\d{4}$/.test(suggestion.draft.account)) setNotice(suggestion.message);
                }} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-account">Account / card last four only</label><input id="ledger-account" inputMode="numeric" pattern="[0-9]{4}" maxLength={4} value={form.account} onChange={e => {
                  const suggestion = suggestAccountForManualEntry({ ...form, account: e.target.value });
                  setForm(suggestion.draft);
                  if (/^\d{4}$/.test(suggestion.draft.account)) setNotice(suggestion.message);
                }} /></div>
                <div className="form-group"><label htmlFor="ledger-due">Due date (optional)</label><input id="ledger-due" type="date" value={form.due_date} onChange={e => setForm({ ...form, due_date: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-linked-account">Saved account / card</label>
                  <select id="ledger-linked-account" value={linkedKey(form)} onChange={e => {
                    const account = context.accounts.find(item => accountKey(item) === e.target.value);
                    if (!account) {
                      setForm({ ...form, linked_account_type: '', linked_account_id: '' });
                      return;
                    }
                    setForm({
                      ...form, linked_account_type: account.type, linked_account_id: account.id,
                      bank: account.bank || form.bank, account: account.last_four,
                      profile_id: memberForAccount(account, deviceMember),
                    });
                    setNotice(account.member_ids.length > 1 ? 'Joint account: check which member made this transaction.' : 'Account selected. Check the member before saving.');
                  }}>
                    <option value="">Not linked</option>
                    {form.linked_account_id && !context.accounts.some(account => accountKey(account) === linkedKey(form)) && <option value={linkedKey(form)}>{accountName(form)}</option>}
                    {context.accounts.map(account => <option value={accountKey(account)} key={accountKey(account)}>{accountLabel(account)}</option>)}
                  </select>
                </div>
                <div className="form-group"><label htmlFor="ledger-member">Transaction member</label>
                  <select id="ledger-member" value={form.profile_id} onChange={e => setForm({ ...form, profile_id: e.target.value })}>
                    <option value="">Unassigned</option>
                    {form.profile_id && !context.members.some(member => member.id === form.profile_id) && <option value={form.profile_id}>Deleted member</option>}
                    {context.members.map(member => <option value={member.id} key={member.id}>{member.name}</option>)}
                  </select>
                </div>
              </div>
              <div className="ledger-actions">
                <button className="btn btn-outline" type="button" onClick={() => { const result = suggest(form); setForm(result.draft); setNotice(result.message); }}>Suggest from bank / last four</button>
                <Link to="/family" className="btn btn-outline">Set account owners</Link>
              </div>
              <p className="text-muted">Link saved accounts/cards here and assign their owners on Family. Editing the bank or last four clears the link. Existing unassigned records are not changed automatically.</p>
              <DuplicateWarnings matches={formMatches} memberName={memberName} onShow={showEntry} />
              {form.status === 'confirmed' && formMatches.length > 0 && !formMatches.some(match => match.reason === 'exact') &&
                <label className="ledger-consent"><input type="checkbox" checked={acknowledged === formSignature} onChange={e => setAcknowledged(e.target.checked ? formSignature : '')} /> I compared these records and want to confirm this transaction.</label>}
              <div className="ledger-actions"><button className="btn btn-primary" type="submit">{busy ? 'Working...' : 'Save'}</button><button className="btn btn-outline" type="button" onClick={reset}>Clear / cancel edit</button></div>
            </fieldset>
          </form>
        </section>

        <section className="card">
          <h2>Bank / credit-card SMS import</h2>
          <p>Initial templates for HDFC, ICICI, Axis, SBI, OneCard and YES BANK. Parsing stays on this device. Only structured fields, SMS receipt time and a fingerprint are sent to your backend, not SMS bodies. Sender names are not proof of authenticity.</p>
          {hasAndroidSms && <div className="ledger-import">
            <label className="ledger-consent"><input type="checkbox" checked={consent} disabled={busy || !ready} onChange={e => setConsent(e.target.checked)} /> I allow this scan to read supported bank senders from my SMS inbox and save supported alerts as pending records to my account. OTPs and unrelated messages are excluded.</label>
            <div className="form-row">
              <div className="form-group"><label htmlFor="scan-mode">Scan messages</label>
                <select id="scan-mode" value={scanMode} onChange={e => setScanMode(e.target.value === 'days' ? 'days' : 'date')}>
                  <option value="date">From a date</option><option value="days">From the last few days</option>
                </select>
              </div>
              {scanMode === 'date'
                ? <div className="form-group"><label htmlFor="scan-from">From date (up to 90 days ago)</label><input id="scan-from" type="date" min={scanMinDate} max={today} value={scanFrom} onChange={e => setScanFrom(e.target.value)} /></div>
                : <div className="form-group"><label htmlFor="scan-days">Last days (1-90)</label><input id="scan-days" type="number" min={1} max={90} value={days} onChange={e => setDays(Number(e.target.value))} /></div>}
            </div>
            {!scanValid && <p className="text-danger">{scanMode === 'date' ? `Choose a date from ${scanMinDate} to today.` : 'Enter a number of days from 1 to 90.'}</p>}
            <div className="ledger-actions">
              <button className="btn btn-primary" disabled={busy || !ready || !consent || !scanValid} onClick={scan}>Scan and queue for review</button>
              <button className="btn btn-outline" disabled={busy} onClick={() => { BankSms.openSettings().catch(err => setError(errorText(err))); }}>Manage SMS permission</button>
            </div>
            <p className="text-muted">Scan is user-triggered, not a background listener. Rescan for new messages. Revoke permission in Android settings at any time.</p>
            {scanSummary && <p role="status" className="card"><strong>{scanSummary}</strong>{!busy && ' Pending records are listed below in the review inbox.'}</p>}
          </div>}
          <details><summary>Paste a message manually</summary>
            <p className="text-muted">Do not paste OTPs. A pasted message may duplicate a scanned/manual record: inspect existing records before confirming.</p>
            <div className="form-group"><label htmlFor="sms-sender">Sender (e.g. VM-HDFCBK)</label><input id="sms-sender" value={sender} onChange={e => setSender(e.target.value)} /></div>
            <div className="form-group"><label htmlFor="sms-date">SMS received date/time</label><input id="sms-date" type="datetime-local" value={receivedAt} onChange={e => setReceivedAt(e.target.value)} /></div>
            <div className="form-group"><label htmlFor="sms-body">Message (not stored)</label><textarea id="sms-body" maxLength={5000} rows={4} value={body} onChange={e => setBody(e.target.value)} /></div>
            <div className="ledger-actions"><button className="btn btn-outline" disabled={busy || !ready} onClick={previewSms}>Preview in transaction form</button><button className="btn btn-outline" onClick={() => setBody('')}>Clear message</button></div>
          </details>
        </section>
      </>}

      <section className="card">
        <h2>Transactions and review inbox</h2>
        <div className="form-row">
          <div className="form-group"><label htmlFor="ledger-month">Month (clear for all months)</label><input id="ledger-month" type="month" value={month} onChange={e => setMonth(e.target.value)} /></div>
          <div className="form-group"><label htmlFor="ledger-filter">Status</label><select id="ledger-filter" value={status} onChange={e => setStatus(e.target.value)}><option value="all">All</option><option value="pending">Pending review</option><option value="confirmed">Confirmed</option><option value="rejected">Rejected</option></select></div>
        </div>
        <div className="form-row">
          <div className="form-group"><label htmlFor="ledger-member-filter">Member</label><select id="ledger-member-filter" value={memberFilter} onChange={e => setMemberFilter(e.target.value)}>
            <option value="all">All members</option><option value="unassigned">Unassigned</option>
            {context.members.map(member => <option value={member.id} key={member.id}>{member.name}</option>)}
            {missingMembers.map(id => <option value={id} key={id}>Deleted member ({id.slice(-6)})</option>)}
          </select></div>
          <div className="form-group"><label htmlFor="ledger-account-filter">Account / card</label><select id="ledger-account-filter" value={accountFilter} onChange={e => setAccountFilter(e.target.value)}>
            <option value="all">All accounts</option><option value="unlinked">Not linked</option>
            {context.accounts.map(account => <option value={accountKey(account)} key={accountKey(account)}>{accountLabel(account)}</option>)}
            {[...missingAccounts].map(([key, entry]) => <option value={key} key={key}>{accountName(entry)}</option>)}
          </select></div>
        </div>
        {ready && <><h3>Confirmed totals by member</h3>
        <p className="text-muted">For the selected month and account, regardless of the member and status filters. Unassigned records remain separate.</p>
        {memberTotals.size === 0 ? <p>No confirmed income or expenses for this period.</p> :
          <ul className="ledger-member-totals">{[...memberTotals].map(([id, amounts]) => <li key={id || 'unassigned'}>
            <strong>{memberName(id)}</strong><span>Income {money(amounts.income)}</span><span>Expenses less refunds {money(amounts.expense - amounts.refund)}</span><span>Net {money(amounts.income - amounts.expense + amounts.refund)}</span>
          </li>)}</ul>}</>}
        {!ready ? null : filtered.length === 0 ? <p>No matching records. Pending imports are not included in totals.</p>
          : <div className="table-container"><table className="ledger-table"><thead><tr><th>Date</th><th>Description / member</th><th>Type / status</th><th>Amount</th><th>Account / due</th>{!isViewOnly && <th>Action</th>}</tr></thead>
            <tbody>{filtered.map(entry => <tr key={entry.id} id={`ledger-entry-${entry.id}`}>
              <td data-label="Date">{entry.date}</td>
              <td data-label="Transaction"><div>{entry.description}<br /><small>{entry.category} / {entry.source}</small><br /><strong>{entry.status === 'pending' ? 'Suggested member' : 'Member'}: {memberName(entry.profile_id)}</strong>
                <DuplicateWarnings matches={duplicatesByID[entry.id] || []} memberName={memberName} onShow={showEntry} /></div></td>
              <td data-label="Type / status"><div>{entry.kind}<br />{entry.status}</div></td>
              <td data-label="Amount">{money(entry.amount_paise)}</td>
              <td data-label="Account"><div>{accountName(entry)}<br />{entry.bank} {entry.account && `XX${entry.account}`}<br />{entry.due_date && `Due ${entry.due_date}`}</div></td>
              {!isViewOnly && <td data-label="Action"><button className="btn btn-outline btn-sm" disabled={busy} onClick={() => edit(entry)}>Review / edit</button></td>}
            </tr>)}</tbody></table></div>}
      </section>
    </div>
  );
}
