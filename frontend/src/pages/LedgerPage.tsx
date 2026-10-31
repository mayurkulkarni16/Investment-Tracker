import { useEffect, useState, type FormEvent } from 'react';
import { ledgerApi, type LedgerDraft, type LedgerEntry } from '../api/ledger';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { localDate, manualSourceKey, parseSms, smsFingerprint, toPaise } from '../utils/smsParser.mjs';
import { BankSms, hasAndroidSms } from '../native/sms';
import './LedgerPage.css';

const money = (paise: number) => (paise / 100).toLocaleString('en-IN', { style: 'currency', currency: 'INR' });
const blank = (): LedgerDraft => ({
  source_key: '', source: 'manual', status: 'confirmed', kind: 'expense', amount_paise: 0,
  date: localDate(Date.now()), description: '', category: 'Uncategorized', bank: '', account: '', due_date: '',
});
const errorText = (error: unknown) => error instanceof Error ? error.message : 'Operation failed. Please retry.';

export default function LedgerPage() {
  const { user, viewAsUserId, isViewOnly } = useAuth();
  const owner = viewAsUserId || user?.id || '';
  const [snapshot, setSnapshot] = useState<{ owner: string; entries: LedgerEntry[] }>({ owner: '', entries: [] });
  const [loadError, setLoadError] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState<LedgerDraft>(blank);
  const [editID, setEditID] = useState('');
  const [amount, setAmount] = useState('');
  const [month, setMonth] = useState(() => localDate(Date.now()).slice(0, 7));
  const [status, setStatus] = useState('all');
  const [sender, setSender] = useState('');
  const [body, setBody] = useState('');
  const [receivedAt, setReceivedAt] = useState(() => `${localDate(Date.now())}T12:00`);
  const [days, setDays] = useState(7);
  const [consent, setConsent] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let active = true;
    ledgerApi.list().then(response => {
      if (active) { setSnapshot({ owner, entries: response.data }); setLoadError(''); }
    }).catch(err => { if (active) setLoadError(apiErrorMessage(err, 'Could not load ledger.')); });
    return () => { active = false; };
  }, [owner, reloadKey]);

  const entries = snapshot.owner === owner ? snapshot.entries : [];
  const filtered = entries.filter(entry => (!month || entry.date.startsWith(month)) && (status === 'all' || entry.status === status));
  const total = (kind: LedgerDraft['kind']) => entries.filter(entry =>
    entry.status === 'confirmed' && entry.kind === kind && (!month || entry.date.startsWith(month))
  ).reduce((sum, entry) => sum + entry.amount_paise, 0);
  const reset = () => { setForm(blank()); setEditID(''); setAmount(''); };
  const refresh = () => setReloadKey(key => key + 1);

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (isViewOnly) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const draft = { ...form, amount_paise: toPaise(amount), source_key: form.source_key || await manualSourceKey() };
      setForm(draft);
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
      setForm({ ...parsed.draft, source_key: await smsFingerprint(message) });
      setEditID(''); setAmount((parsed.draft.amount_paise / 100).toFixed(2));
      setNotice(parsed.warning || '');
    } catch (err) { setError(errorText(err)); }
  };

  const scan = async () => {
    if (!consent || isViewOnly) return;
    setBusy(true); setError(''); setNotice('');
    let queued = 0;
    let skipped = 0;
    try {
      const result = await BankSms.scan({ days });
      for (const message of result.messages) {
        const parsed = parseSms(message);
        if (!parsed.draft) { skipped++; continue; }
        await ledgerApi.create({ ...parsed.draft, source_key: await smsFingerprint(message) });
        queued++;
      }
      setStatus('pending'); setMonth('');
      setNotice(`Processed ${queued} supported alerts (existing imports are kept, not duplicated); skipped ${skipped} unsupported alerts. ${result.truncated ? 'Scan limit reached: use a shorter window. ' : ''}Review each pending record before confirmation.`);
    } catch (err) {
      setError(`${queued} alerts processed before scan stopped. ${apiErrorMessage(err, errorText(err))} Retry is safe for identical SMS records.`);
    } finally { setBusy(false); refresh(); }
  };

  const edit = (entry: LedgerEntry) => {
    setForm(entry); setEditID(entry.id); setAmount((entry.amount_paise / 100).toFixed(2));
    setNotice('Check for existing manual/card records. Choose transfer for repayments; refunds are not necessarily income. Mark rejected to exclude a duplicate.');
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
        {!hasAndroidSms && <button className="btn btn-outline" disabled={busy} onClick={exportCSV}>Export CSV</button>}
      </div>
      <p className="text-muted mb-4">Personal ledger only. Transfers and bill notices are excluded from spending. These records do not yet update investment cashflow, net worth or the Credit Cards module.</p>
      {error && <div role="alert" className="card text-danger">{error}</div>}
      {notice && <div role="status" className="card">{notice}</div>}
      <div className="card-grid">
        <div className="stat-card"><div className="label">Confirmed income</div><div className="value positive">{money(total('income'))}</div></div>
        <div className="stat-card"><div className="label">Confirmed expenses less refunds</div><div className="value negative">{money(total('expense') - total('refund'))}</div></div>
        <div className="stat-card"><div className="label">Net income minus expenses</div><div className="value">{money(total('income') - total('expense') + total('refund'))}</div></div>
      </div>

      {!isViewOnly && <>
        <section className="card">
          <h2>{editID ? 'Review / edit transaction' : 'Add transaction'}</h2>
          <form onSubmit={save}>
            <fieldset disabled={busy}>
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
                <div className="form-group"><label htmlFor="ledger-bank">Bank / card issuer</label><input id="ledger-bank" maxLength={60} value={form.bank} onChange={e => setForm({ ...form, bank: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label htmlFor="ledger-account">Account / card last four only</label><input id="ledger-account" inputMode="numeric" pattern="[0-9]{4}" maxLength={4} value={form.account} onChange={e => setForm({ ...form, account: e.target.value })} /></div>
                <div className="form-group"><label htmlFor="ledger-due">Due date (optional)</label><input id="ledger-due" type="date" value={form.due_date} onChange={e => setForm({ ...form, due_date: e.target.value })} /></div>
              </div>
              <div className="ledger-actions"><button className="btn btn-primary" type="submit">{busy ? 'Working...' : 'Save'}</button><button className="btn btn-outline" type="button" onClick={reset}>Clear / cancel edit</button></div>
            </fieldset>
          </form>
        </section>
        <section className="card">
          <h2>Bank / credit-card SMS import</h2>
          <p>Initial templates for HDFC, ICICI, Axis, SBI, OneCard and YES BANK. Parsing stays on this device. Only structured fields and a fingerprint are sent to your backend, not SMS bodies. Sender names are not proof of authenticity.</p>
          {hasAndroidSms && <div className="ledger-import">
            <label className="ledger-consent"><input type="checkbox" checked={consent} disabled={busy} onChange={e => setConsent(e.target.checked)} /> I allow this scan to read supported bank senders from my SMS inbox and save supported alerts as pending records to my account. OTPs and unrelated messages are excluded.</label>
            <div className="form-group"><label htmlFor="scan-days">Scan last days (1-90)</label><input id="scan-days" type="number" min={1} max={90} value={days} onChange={e => setDays(Number(e.target.value))} /></div>
            <div className="ledger-actions">
              <button className="btn btn-primary" disabled={busy || !consent || days < 1 || days > 90} onClick={scan}>Scan and queue for review</button>
              <button className="btn btn-outline" disabled={busy} onClick={() => { BankSms.openSettings().catch(err => setError(errorText(err))); }}>Manage SMS permission</button>
            </div>
            <p className="text-muted">Scan is user-triggered, not a background listener. Rescan for new messages. Revoke permission in Android settings at any time.</p>
          </div>}
          <details><summary>Paste a message manually</summary>
            <p className="text-muted">Do not paste OTPs. A pasted message may duplicate a scanned/manual record: inspect existing records before confirming.</p>
            <div className="form-group"><label htmlFor="sms-sender">Sender (e.g. VM-HDFCBK)</label><input id="sms-sender" value={sender} onChange={e => setSender(e.target.value)} /></div>
            <div className="form-group"><label htmlFor="sms-date">SMS received date/time</label><input id="sms-date" type="datetime-local" value={receivedAt} onChange={e => setReceivedAt(e.target.value)} /></div>
            <div className="form-group"><label htmlFor="sms-body">Message (not stored)</label><textarea id="sms-body" maxLength={5000} rows={4} value={body} onChange={e => setBody(e.target.value)} /></div>
            <div className="ledger-actions"><button className="btn btn-outline" disabled={busy} onClick={previewSms}>Preview in transaction form</button><button className="btn btn-outline" onClick={() => setBody('')}>Clear message</button></div>
          </details>
        </section>
      </>}

      <section className="card">
        <h2>Transactions and review inbox</h2>
        <div className="form-row">
          <div className="form-group"><label htmlFor="ledger-month">Month (clear for all months)</label><input id="ledger-month" type="month" value={month} onChange={e => setMonth(e.target.value)} /></div>
          <div className="form-group"><label htmlFor="ledger-filter">Status</label><select id="ledger-filter" value={status} onChange={e => setStatus(e.target.value)}><option value="all">All</option><option value="pending">Pending review</option><option value="confirmed">Confirmed</option><option value="rejected">Rejected</option></select></div>
        </div>
        {loadError ? <p role="alert" className="text-danger">{loadError} <button className="btn btn-outline" onClick={refresh}>Retry</button></p>
          : snapshot.owner !== owner ? <p>Loading ledger...</p>
          : filtered.length === 0 ? <p>No matching records. Pending imports are not included in totals.</p>
          : <div className="table-container"><table><thead><tr><th>Date</th><th>Description</th><th>Type / status</th><th>Amount</th><th>Account / due</th>{!isViewOnly && <th>Action</th>}</tr></thead>
            <tbody>{filtered.map(entry => <tr key={entry.id}>
              <td>{entry.date}</td><td>{entry.description}<br /><small>{entry.category} / {entry.source}</small></td>
              <td>{entry.kind}<br />{entry.status}</td><td>{money(entry.amount_paise)}</td>
              <td>{entry.bank} {entry.account && `XX${entry.account}`}<br />{entry.due_date && `Due ${entry.due_date}`}</td>
              {!isViewOnly && <td><button className="btn btn-outline btn-sm" disabled={busy} onClick={() => edit(entry)}>Review / edit</button></td>}
            </tr>)}</tbody></table></div>}
      </section>
    </div>
  );
}
