import { useEffect, useState } from 'react';
import { getOverdrafts, createOverdraft, updateOverdraft, deleteOverdraft, addOverdraftTransaction, deleteOverdraftTransaction } from '../api/overdrafts';
import type { Overdraft, CreateOverdraftRequest, UpdateOverdraftRequest, AddOverdraftTransactionRequest, OverdraftTxnType } from '../types';
import { formatCurrency, formatDate } from '../utils/format';
import { apiErrorMessage } from '../utils/errors';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

const TXN_LABELS: Record<OverdraftTxnType, string> = {
  withdrawal: 'Withdrawal (money drawn)',
  repayment: 'Principal repayment',
  interest: 'Interest payment',
};

// Local (not UTC) date as YYYY-MM-DD, so "today" is correct in IST.
const today = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};

const emptyForm = (): CreateOverdraftRequest => ({
  lender_name: '', account_number: '', sanctioned_limit: 0, interest_rate: 0, open_date: today(), opening_balance: 0, notes: '',
});

export default function OverdraftsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [items, setItems] = useState<Overdraft[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [editing, setEditing] = useState<Overdraft | null>(null);
  const [txnFor, setTxnFor] = useState<Overdraft | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [form, setForm] = useState<CreateOverdraftRequest>(emptyForm);
  const [editForm, setEditForm] = useState<UpdateOverdraftRequest | null>(null);
  const [txnForm, setTxnForm] = useState<AddOverdraftTransactionRequest>({ type: 'withdrawal', date: today(), amount: 0, notes: '' });

  const load = () => {
    getOverdrafts().then(r => setItems(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, [viewAsUserId]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createOverdraft(form);
      setShowAdd(false);
      setForm(emptyForm());
      toast('Overdraft added');
      load();
    } catch (err) { toast(apiErrorMessage(err, 'Failed to add overdraft'), 'error'); }
  };

  const openEdit = (o: Overdraft) => {
    setEditForm({
      lender_name: o.lender_name, account_number: o.account_number, sanctioned_limit: o.sanctioned_limit,
      interest_rate: o.interest_rate, open_date: o.open_date.slice(0, 10), status: o.status, notes: o.notes,
    });
    setEditing(o);
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editing || !editForm) return;
    try { await updateOverdraft(editing.id, editForm); setEditing(null); toast('Overdraft updated'); load(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to update overdraft'), 'error'); }
  };

  const handleDelete = async (o: Overdraft) => {
    if (!await confirm({ message: `Delete the ${o.lender_name} overdraft and its history?`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteOverdraft(o.id); toast('Overdraft deleted'); load(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete overdraft'), 'error'); }
  };

  const openTxn = (o: Overdraft, type: OverdraftTxnType) => {
    const amount = type === 'interest' ? Math.round(o.accrued_interest) : 0;
    setTxnForm({ type, date: today(), amount, notes: '' });
    setTxnFor(o);
  };

  const handleTxn = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!txnFor) return;
    try {
      await addOverdraftTransaction(txnFor.id, txnForm);
      setTxnFor(null);
      toast('Transaction recorded');
      load();
    } catch (err) { toast(apiErrorMessage(err, 'Failed to record transaction'), 'error'); }
  };

  const handleDeleteTxn = async (o: Overdraft, txnId: string) => {
    if (!await confirm({ message: 'Delete this transaction?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteOverdraftTransaction(o.id, txnId); toast('Transaction deleted'); load(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete transaction'), 'error'); }
  };

  const active = items.filter(o => o.status === 'active');
  const totalOutstanding = active.reduce((s, o) => s + o.outstanding, 0);
  const totalLimit = active.reduce((s, o) => s + o.sanctioned_limit, 0);
  const totalMonthlyInterest = active.reduce((s, o) => s + o.monthly_interest, 0);
  const totalInterestPaid = items.reduce((s, o) => s + o.total_interest_paid, 0);

  if (loading) return <PageSkeleton label="Loading" />;

  return (
    <div>
      <div className="page-header">
        <h1>Overdrafts</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Overdraft</button>}
      </div>
      <p className="text-muted text-sm mb-4">
        Interest-only credit lines: you pay interest on what you've drawn, and repay principal whenever you want.
      </p>

      <div className="card-grid mb-5">
        <div className="stat-card"><div className="label">Outstanding</div><div className="value">{formatCurrency(totalOutstanding)}</div></div>
        <div className="stat-card"><div className="label">Available Limit</div><div className="value positive">{formatCurrency(Math.max(totalLimit - totalOutstanding, 0))}</div></div>
        <div className="stat-card"><div className="label">Monthly Interest (est.)</div><div className="value negative">{formatCurrency(totalMonthlyInterest)}</div></div>
        <div className="stat-card"><div className="label">Total Interest Paid</div><div className="value">{formatCurrency(totalInterestPaid)}</div></div>
      </div>

      {items.length === 0 ? (
        <div className="empty-state">
          <h3>No overdrafts yet</h3>
          <p>Add an overdraft account to track withdrawals, interest and repayments.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Overdraft</button>}
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {items.map(o => (
            <div key={o.id} className="card p-5">
              <div className="flex justify-between items-start mb-3">
                <div>
                  <h3 className="m-0">{o.lender_name}</h3>
                  <div className="text-muted text-sm">
                    A/C: {o.account_number || '-'} | Opened {formatDate(o.open_date)} | {o.interest_rate}% p.a.
                  </div>
                </div>
                <span className="text-xs fw-semibold" style={{ padding: '3px 10px', borderRadius: 12, background: o.status === 'active' ? 'var(--success-soft)' : 'var(--primary-soft)', color: o.status === 'active' ? 'var(--success-text)' : 'var(--primary-text)' }}>{o.status.toUpperCase()}</span>
              </div>

              <div className="grid gap-3 mb-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))' }}>
                <div><div className="text-muted text-sm">Sanctioned Limit</div><strong>{formatCurrency(o.sanctioned_limit)}</strong></div>
                <div><div className="text-muted text-sm">Outstanding</div><strong>{formatCurrency(o.outstanding)}</strong></div>
                <div><div className="text-muted text-sm">Available</div><strong>{formatCurrency(o.available_limit)}</strong></div>
                <div><div className="text-muted text-sm">Utilization</div><strong>{o.utilization_pct.toFixed(1)}%</strong></div>
                <div><div className="text-muted text-sm">Monthly Interest (est.)</div><strong>{formatCurrency(o.monthly_interest)}</strong></div>
                <div>
                  <div className="text-muted text-sm">Interest Due (est.)</div>
                  <strong>{formatCurrency(o.accrued_interest)}</strong>
                  <div className="text-muted text-xs">since {o.last_interest_paid_date ? `last paid ${formatDate(o.last_interest_paid_date)}` : 'opening'}</div>
                </div>
                <div><div className="text-muted text-sm">Interest Paid</div><strong>{formatCurrency(o.total_interest_paid)}</strong></div>
                <div><div className="text-muted text-sm">Principal Repaid</div><strong>{formatCurrency(o.total_repaid)}</strong></div>
              </div>

              <div className="flex flex-wrap gap-2">
                {!isViewOnly && o.status === 'active' && (
                  <>
                    <button className="btn btn-sm btn-outline" onClick={() => openTxn(o, 'withdrawal')}>Withdraw</button>
                    <button className="btn btn-sm btn-outline" onClick={() => openTxn(o, 'interest')}>Pay Interest</button>
                    <button className="btn btn-sm btn-outline" onClick={() => openTxn(o, 'repayment')}>Repay Principal</button>
                  </>
                )}
                <button className="btn btn-sm btn-outline" onClick={() => setExpanded(expanded === o.id ? null : o.id)}>
                  History ({o.transactions?.length || 0})
                </button>
                {!isViewOnly && (
                  <RowActions
                    label={`More actions for ${o.lender_name} overdraft`}
                    actions={[
                      { label: 'Edit overdraft', icon: Pencil, onClick: () => openEdit(o) },
                      { label: 'Delete overdraft', icon: Trash2, onClick: () => handleDelete(o), danger: true },
                    ]}
                  />
                )}
              </div>

              {expanded === o.id && (
                <div className="mt-3" style={{ maxHeight: 320, overflowY: 'auto' }}>
                  {(o.transactions?.length || 0) === 0 ? (
                    <p className="text-muted text-sm">No transactions yet.</p>
                  ) : (
                    <table className="text-sm">
                      <thead><tr><th>Date</th><th>Type</th><th className="num">Amount</th><th className="num">Principal After</th><th>Notes</th>{!isViewOnly && <th />}</tr></thead>
                      <tbody>{[...o.transactions].reverse().map(t => (
                        <tr key={t.txn_id}>
                          <td>{formatDate(t.date)}</td>
                          <td>{TXN_LABELS[t.type] || t.type}</td>
                          <td className={`num ${t.type === 'withdrawal' ? 'negative' : ''}`}>{formatCurrency(t.amount)}</td>
                          <td className="num">{formatCurrency(t.balance_after)}</td>
                          <td>{t.notes || '-'}</td>
                          {!isViewOnly && (
                            <td><button className="btn btn-sm btn-outline" aria-label="Delete transaction" onClick={() => handleDeleteTxn(o, t.txn_id)}><Trash2 size={14} aria-hidden="true" /></button></td>
                          )}
                        </tr>
                      ))}</tbody>
                    </table>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 600 }}>
            <h2>Add Overdraft</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group"><label>Lender / Bank</label><input required value={form.lender_name} onChange={e => setForm({ ...form, lender_name: e.target.value })} /></div>
                <div className="form-group"><label>Account Number</label><input value={form.account_number || ''} onChange={e => setForm({ ...form, account_number: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Sanctioned Limit (₹)</label><input type="number" min="1" step="0.01" required value={form.sanctioned_limit || ''} onChange={e => setForm({ ...form, sanctioned_limit: parseFloat(e.target.value) || 0 })} /></div>
                <div className="form-group"><label>Interest Rate (% p.a.)</label><input type="number" min="0" max="60" step="0.01" required value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) || 0 })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Open Date</label><input type="date" required value={form.open_date} onChange={e => setForm({ ...form, open_date: e.target.value })} /></div>
                <div className="form-group">
                  <label>Amount Already Drawn (₹)</label>
                  <input type="number" min="0" step="0.01" value={form.opening_balance || ''} onChange={e => setForm({ ...form, opening_balance: parseFloat(e.target.value) || 0 })} />
                  <small className="text-muted">Recorded as a withdrawal on the open date. Add later ones from History.</small>
                </div>
              </div>
              <div className="form-group"><label>Notes</label><textarea value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Overdraft</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {editing && editForm && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 600 }}>
            <h2>Edit Overdraft</h2>
            <form onSubmit={handleEdit}>
              <div className="form-row">
                <div className="form-group"><label>Lender / Bank</label><input required value={editForm.lender_name} onChange={e => setEditForm({ ...editForm, lender_name: e.target.value })} /></div>
                <div className="form-group"><label>Account Number</label><input value={editForm.account_number || ''} onChange={e => setEditForm({ ...editForm, account_number: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Sanctioned Limit (₹)</label><input type="number" min="1" step="0.01" required value={editForm.sanctioned_limit || ''} onChange={e => setEditForm({ ...editForm, sanctioned_limit: parseFloat(e.target.value) || 0 })} /></div>
                <div className="form-group">
                  <label>Interest Rate (% p.a.)</label>
                  <input type="number" min="0" max="60" step="0.01" required value={editForm.interest_rate || ''} onChange={e => setEditForm({ ...editForm, interest_rate: parseFloat(e.target.value) || 0 })} />
                  <small className="text-muted">Interest estimates use the current rate.</small>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Open Date</label><input type="date" required value={editForm.open_date} onChange={e => setEditForm({ ...editForm, open_date: e.target.value })} /></div>
                <div className="form-group"><label>Status</label>
                  <select value={editForm.status} onChange={e => setEditForm({ ...editForm, status: e.target.value as 'active' | 'closed' })}>
                    <option value="active">Active</option>
                    <option value="closed">Closed</option>
                  </select>
                </div>
              </div>
              <div className="form-group"><label>Notes</label><textarea value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setEditing(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Changes</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {txnFor && (
        <div className="modal-overlay" onClick={() => setTxnFor(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{txnFor.lender_name} — {TXN_LABELS[txnForm.type]}</h2>
            <form onSubmit={handleTxn}>
              <div className="form-group"><label>Type</label>
                <select value={txnForm.type} onChange={e => setTxnForm({ ...txnForm, type: e.target.value as OverdraftTxnType })}>
                  {(Object.keys(TXN_LABELS) as OverdraftTxnType[]).map(k => <option key={k} value={k}>{TXN_LABELS[k]}</option>)}
                </select>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Date</label><input type="date" required value={txnForm.date} onChange={e => setTxnForm({ ...txnForm, date: e.target.value })} /></div>
                <div className="form-group"><label>Amount (₹)</label><input type="number" min="0.01" step="0.01" required value={txnForm.amount || ''} onChange={e => setTxnForm({ ...txnForm, amount: parseFloat(e.target.value) || 0 })} /></div>
              </div>
              <small className="text-muted">
                {txnForm.type === 'withdrawal' && `Available limit: ${formatCurrency(txnFor.available_limit)}.`}
                {txnForm.type === 'repayment' && `Outstanding principal: ${formatCurrency(txnFor.outstanding)}.`}
                {txnForm.type === 'interest' && `Estimated interest due: ${formatCurrency(txnFor.accrued_interest)}. Interest payments don't reduce the principal.`}
              </small>
              <div className="form-group mt-3"><label>Notes</label><input value={txnForm.notes || ''} onChange={e => setTxnForm({ ...txnForm, notes: e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setTxnFor(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Record</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
