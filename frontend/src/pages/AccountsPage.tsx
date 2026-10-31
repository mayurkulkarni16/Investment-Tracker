import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import {
  getAccounts, createAccount, updateAccount, deleteAccount, updateAccountBalance,
} from '../api/accounts';
import type { BankAccount, BankAccountRequest, AccountType } from '../api/accounts';
import { formatCurrency, formatDate, toInputDate } from '../utils/format';
import { apiErrorMessage } from '../utils/errors';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';

const STALE_DAYS = 30;

const TYPE_LABELS: Record<AccountType, string> = {
  savings: 'Savings',
  current: 'Current',
  salary: 'Salary',
  cash: 'Cash in Hand',
  wallet: 'Wallet',
  other: 'Other',
};
const TYPE_ORDER: AccountType[] = ['savings', 'salary', 'current', 'wallet', 'cash', 'other'];
const BANK_TYPES: AccountType[] = ['savings', 'current', 'salary'];

const today = () => new Date().toISOString().split('T')[0];

const emptyForm = (): BankAccountRequest => ({
  name: '', institution: '', account_type: 'savings', last_four: '', balance: 0,
  balance_as_of: today(), interest_rate: 0, is_emergency_fund: false, status: 'active', notes: '',
});

function BalanceSparkline({ account }: { account: BankAccount }) {
  const data = (account.balance_history || [])
    .slice()
    .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
    .map(p => ({ date: formatDate(p.date), balance: p.balance }));
  if (data.length < 2) return null;
  return (
    <div style={{ width: 160, height: 40 }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 4, right: 4, bottom: 4, left: 4 }}>
          <XAxis dataKey="date" hide />
          <YAxis hide domain={['auto', 'auto']} />
          <Tooltip formatter={(v) => formatCurrency(Number(v))} />
          <Line type="monotone" dataKey="balance" stroke="#1a73e8" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

export default function AccountsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [showClosed, setShowClosed] = useState(false);
  const [editing, setEditing] = useState<BankAccount | 'new' | null>(null);
  const [form, setForm] = useState<BankAccountRequest>(emptyForm);
  const [balanceFor, setBalanceFor] = useState<BankAccount | null>(null);
  const [balanceForm, setBalanceForm] = useState({ balance: 0, as_of: today() });
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    getAccounts()
      .then(r => setAccounts(r.data || []))
      .catch(err => toast(apiErrorMessage(err, 'Failed to load accounts'), 'error'))
      .finally(() => setLoading(false));
  }, [toast]);

  useEffect(() => { load(); }, [load, viewAsUserId]);

  const active = useMemo(() => accounts.filter(a => a.status !== 'closed'), [accounts]);

  const summary = useMemo(() => {
    const sum = (list: BankAccount[]) => list.reduce((s, a) => s + (a.balance || 0), 0);
    return {
      bank: sum(active.filter(a => BANK_TYPES.includes(a.account_type) || a.account_type === 'other')),
      cash: sum(active.filter(a => a.account_type === 'cash')),
      wallet: sum(active.filter(a => a.account_type === 'wallet')),
      emergency: sum(active.filter(a => a.is_emergency_fund)),
      stale: active.filter(a => a.days_stale > STALE_DAYS).length,
      total: sum(active),
    };
  }, [active]);

  const grouped = useMemo(() => {
    const visible = showClosed ? accounts : active;
    return TYPE_ORDER
      .map(t => ({ type: t, items: visible.filter(a => (TYPE_ORDER.includes(a.account_type) ? a.account_type : 'other') === t) }))
      .filter(g => g.items.length > 0);
  }, [accounts, active, showClosed]);

  const openAdd = () => { setForm(emptyForm()); setEditing('new'); };

  const openEdit = (a: BankAccount) => {
    setForm({
      name: a.name,
      institution: a.institution || '',
      account_type: a.account_type,
      last_four: a.last_four || '',
      balance: a.balance,
      balance_as_of: toInputDate(a.balance_as_of) || today(),
      interest_rate: a.interest_rate || 0,
      is_emergency_fund: a.is_emergency_fund,
      status: a.status === 'closed' ? 'closed' : 'active',
      notes: a.notes || '',
    });
    setEditing(a);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editing) return;
    const payload: BankAccountRequest = { ...form };
    if (payload.account_type === 'cash') {
      payload.institution = '';
      payload.last_four = '';
    }
    if (payload.last_four && !/^\d{4}$/.test(payload.last_four)) {
      toast('Last four must be exactly 4 digits', 'error');
      return;
    }
    setSaving(true);
    try {
      if (editing === 'new') {
        await createAccount(payload);
        toast('Account added');
      } else {
        await updateAccount(editing.id, payload);
        toast('Account updated');
      }
      setEditing(null);
      load();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save account'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (a: BankAccount) => {
    if (!await confirm({ title: 'Delete account', message: `Delete "${a.name}" and its balance history?`, danger: true, confirmLabel: 'Delete' })) return;
    try {
      await deleteAccount(a.id);
      toast('Account deleted');
      load();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to delete account'), 'error');
    }
  };

  const openBalance = (a: BankAccount) => {
    setBalanceForm({ balance: a.balance, as_of: today() });
    setBalanceFor(a);
  };

  const handleBalance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!balanceFor) return;
    setSaving(true);
    try {
      await updateAccountBalance(balanceFor.id, { balance: balanceForm.balance, as_of: balanceForm.as_of, source: 'manual' });
      toast('Balance updated');
      setBalanceFor(null);
      load();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to update balance'), 'error');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="loading">Loading...</div>;

  const isCash = form.account_type === 'cash';

  return (
    <div>
      <div className="page-header">
        <h1>Bank &amp; Cash</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}>+ Add Account</button>}
      </div>

      {accounts.length > 0 && (
        <div className="card-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', marginBottom: 20 }}>
          <div className="stat-card">
            <div className="label">Total Bank Balance</div>
            <div className="value">{formatCurrency(summary.bank)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Cash in Hand</div>
            <div className="value">{formatCurrency(summary.cash)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Wallets</div>
            <div className="value">{formatCurrency(summary.wallet)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Emergency Fund</div>
            <div className="value positive">{formatCurrency(summary.emergency)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Stale Balances (&gt;{STALE_DAYS}d)</div>
            <div className={`value ${summary.stale > 0 ? 'negative' : ''}`}>{summary.stale}</div>
          </div>
        </div>
      )}

      {accounts.length === 0 ? (
        <div className="empty-state">
          <h3>No bank accounts or cash yet</h3>
          <p>Track savings accounts, cash in hand and wallets to see your liquid money in one place.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}>+ Add Account</button>}
        </div>
      ) : (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <span className="text-muted">Total liquid: <strong>{formatCurrency(summary.total)}</strong> across {active.length} active account{active.length === 1 ? '' : 's'}</span>
            {accounts.some(a => a.status === 'closed') && (
              <label style={{ fontSize: 13 }}>
                <input type="checkbox" checked={showClosed} onChange={e => setShowClosed(e.target.checked)} style={{ marginRight: 6 }} />
                Show closed
              </label>
            )}
          </div>

          {grouped.map(g => {
            const groupTotal = g.items.filter(a => a.status !== 'closed').reduce((s, a) => s + a.balance, 0);
            const showBankCols = g.type !== 'cash';
            return (
              <div className="card" key={g.type}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <h3 style={{ margin: 0 }}>{TYPE_LABELS[g.type]}</h3>
                  <strong>{formatCurrency(groupTotal)}</strong>
                </div>
                <div className="table-container">
                  <table>
                    <thead>
                      <tr>
                        <th>Name</th>
                        {showBankCols && <th>Institution</th>}
                        {showBankCols && <th>A/c</th>}
                        <th className="text-right">Balance</th>
                        <th>As of</th>
                        <th className="text-right">This month in</th>
                        <th className="text-right">This month out</th>
                        <th>Trend</th>
                        {!isViewOnly && <th>Actions</th>}
                      </tr>
                    </thead>
                    <tbody>
                      {g.items.map(a => (
                        <tr key={a.id} style={a.status === 'closed' ? { opacity: 0.6 } : undefined}>
                          <td>
                            {a.name}
                            {a.is_emergency_fund && <span className="badge badge-active" style={{ marginLeft: 6 }} title="Counted as emergency fund">Emergency</span>}
                            {a.status === 'closed' && <span className="badge badge-matured" style={{ marginLeft: 6 }}>closed</span>}
                            {a.interest_rate > 0 && <div className="text-muted" style={{ fontSize: 11 }}>{a.interest_rate}% p.a.</div>}
                          </td>
                          {showBankCols && <td>{a.institution || '-'}</td>}
                          {showBankCols && <td>{a.last_four ? `••${a.last_four}` : '-'}</td>}
                          <td className="text-right"><strong>{formatCurrency(a.balance)}</strong></td>
                          <td>
                            {formatDate(a.balance_as_of)}
                            {a.days_stale > STALE_DAYS && (
                              <span className="badge badge-overdue" style={{ marginLeft: 6 }} title={`Balance last updated ${a.days_stale} days ago`}>
                                stale {a.days_stale}d
                              </span>
                            )}
                          </td>
                          <td className="text-right text-success">{a.month_inflow ? formatCurrency(a.month_inflow) : '-'}</td>
                          <td className="text-right text-danger">{a.month_outflow ? formatCurrency(a.month_outflow) : '-'}</td>
                          <td><BalanceSparkline account={a} /></td>
                          {!isViewOnly && (
                            <td>
                              <div style={{ display: 'flex', gap: 4 }}>
                                <button className="btn btn-sm btn-primary" onClick={() => openBalance(a)}>Update balance</button>
                                <button className="btn btn-sm btn-outline" onClick={() => openEdit(a)}>Edit</button>
                                <button className="btn btn-sm btn-danger" onClick={() => handleDelete(a)}>Del</button>
                              </div>
                            </td>
                          )}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })}
          <p className="text-muted" style={{ fontSize: 12 }}>
            Monthly inflow/outflow is matched from confirmed Income &amp; Expenses entries using the account's last four digits.
          </p>
        </>
      )}

      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing === 'new' ? 'Add Account' : 'Edit Account'}</h2>
            <form onSubmit={handleSave}>
              <div className="form-row">
                <div className="form-group">
                  <label>Name</label>
                  <input required value={form.name} placeholder={isCash ? 'e.g. Cash at home' : 'e.g. HDFC Savings'} onChange={e => setForm({ ...form, name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Type</label>
                  <select value={form.account_type} onChange={e => setForm({ ...form, account_type: e.target.value as AccountType })}>
                    {(Object.keys(TYPE_LABELS) as AccountType[]).map(t => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
                  </select>
                </div>
              </div>
              {!isCash && (
                <div className="form-row">
                  <div className="form-group">
                    <label>{form.account_type === 'wallet' ? 'Provider' : 'Bank / Institution'}</label>
                    <input value={form.institution || ''} onChange={e => setForm({ ...form, institution: e.target.value })} />
                  </div>
                  <div className="form-group">
                    <label>Last 4 digits</label>
                    <input value={form.last_four || ''} maxLength={4} inputMode="numeric" pattern="\d{4}" placeholder="1234"
                      onChange={e => setForm({ ...form, last_four: e.target.value.replace(/\D/g, '').slice(0, 4) })} />
                  </div>
                </div>
              )}
              <div className="form-row-3">
                <div className="form-group">
                  <label>Balance (₹)</label>
                  <input type="number" step="0.01" required value={Number.isFinite(form.balance) ? form.balance : ''} onChange={e => setForm({ ...form, balance: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Balance as of</label>
                  <input type="date" value={form.balance_as_of || ''} onChange={e => setForm({ ...form, balance_as_of: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Interest rate (%)</label>
                  <input type="number" step="0.01" min="0" value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Status</label>
                  <select value={form.status || 'active'} onChange={e => setForm({ ...form, status: e.target.value as 'active' | 'closed' })}>
                    <option value="active">Active</option>
                    <option value="closed">Closed</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>&nbsp;</label>
                  <label style={{ display: 'flex', alignItems: 'center' }}>
                    <input type="checkbox" checked={!!form.is_emergency_fund} onChange={e => setForm({ ...form, is_emergency_fund: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                    Part of emergency fund
                  </label>
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setEditing(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : editing === 'new' ? 'Add Account' : 'Save'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {balanceFor && (
        <div className="modal-overlay" onClick={() => setBalanceFor(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Update balance — {balanceFor.name}</h2>
            <p className="text-muted" style={{ marginBottom: 12 }}>
              Current: {formatCurrency(balanceFor.balance)} as of {formatDate(balanceFor.balance_as_of)}
            </p>
            <form onSubmit={handleBalance}>
              <div className="form-row">
                <div className="form-group">
                  <label>New balance (₹)</label>
                  <input type="number" step="0.01" required autoFocus value={Number.isFinite(balanceForm.balance) ? balanceForm.balance : ''}
                    onChange={e => setBalanceForm({ ...balanceForm, balance: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>As of</label>
                  <input type="date" required max={today()} value={balanceForm.as_of} onChange={e => setBalanceForm({ ...balanceForm, as_of: e.target.value })} />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setBalanceFor(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Update'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
