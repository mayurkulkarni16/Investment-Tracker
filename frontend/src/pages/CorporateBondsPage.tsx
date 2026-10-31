import { useEffect, useState } from 'react';
import { getCorporateBonds, createCorporateBond, deleteCorporateBond, markPayoutReceived, updateCorporateBond } from '../api/corporateBonds';
import type { CorporateBond, CreateCorporateBondRequest, PrincipalRepaymentInput, PayoutFrequency, MaturityType } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate, statusLabel } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { ChevronDown, Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { FilterChips, PageSkeleton } from '../components/ui';

export default function CorporateBondsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [bonds, setBonds] = useState<CorporateBond[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<CorporateBond | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [form, setForm] = useState<CreateCorporateBondRequest>({
    bond_name: '', issuer: '', purchase_date: '', investment_amount: 0,
    coupon_rate: 0, interest_payout_frequency: 'quarterly', maturity_date: '',
    maturity_type: 'bullet', principal_repayments: [],
  });
  const [editForm, setEditForm] = useState<CreateCorporateBondRequest>({
    bond_name: '', issuer: '', purchase_date: '', investment_amount: 0,
    coupon_rate: 0, interest_payout_frequency: 'quarterly', maturity_date: '',
    maturity_type: 'bullet', principal_repayments: [],
  });
  const [repayments, setRepayments] = useState<PrincipalRepaymentInput[]>([]);
  const [editRepayments, setEditRepayments] = useState<PrincipalRepaymentInput[]>([]);

  const load = () => {
    getCorporateBonds().then(r => setBonds(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [viewAsUserId]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const data = { ...form };
      if (form.maturity_type === 'staggered') {
        data.principal_repayments = repayments;
      }
      await createCorporateBond(data);
      setShowAdd(false);
      setForm({ bond_name: '', issuer: '', purchase_date: '', investment_amount: 0, coupon_rate: 0, interest_payout_frequency: 'quarterly', maturity_date: '', maturity_type: 'bullet', principal_repayments: [] });
      setRepayments([]);
      toast('Bond added');
      load();
    } catch { toast('Failed to add bond', 'error'); }
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Delete this bond?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteCorporateBond(id); toast('Bond deleted'); load(); }
    catch { toast('Failed to delete bond', 'error'); }
  };

  const openEdit = (bond: CorporateBond) => {
    setEditForm({
      bond_name: bond.bond_name,
      issuer: bond.issuer,
      purchase_date: toInputDate(bond.purchase_date),
      investment_amount: bond.investment_amount,
      coupon_rate: bond.coupon_rate,
      interest_payout_frequency: bond.interest_payout_frequency,
      maturity_date: toInputDate(bond.maturity_date),
      maturity_type: bond.maturity_type,
      notes: bond.notes,
    });
    setEditRepayments(
      bond.maturity_type === 'staggered'
        ? (bond.principal_repayments || []).map(r => ({ scheduled_date: toInputDate(r.scheduled_date), amount: r.amount }))
        : []
    );
    setShowEdit(bond);
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try {
      const data = { ...editForm };
      if (editForm.maturity_type === 'staggered') {
        data.principal_repayments = editRepayments;
      }
      await updateCorporateBond(showEdit.id, data);
      setShowEdit(null);
      toast('Bond updated');
      load();
    } catch { toast('Failed to update bond', 'error'); }
  };

  const handleMarkReceived = async (bondId: string, payoutId: string) => {
    try { await markPayoutReceived(bondId, payoutId); toast('Payout marked received'); load(); }
    catch { toast('Failed to mark payout', 'error'); }
  };

  const addRepayment = () => setRepayments([...repayments, { scheduled_date: '', amount: 0 }]);
  const removeRepayment = (i: number) => setRepayments(repayments.filter((_, idx) => idx !== i));

  if (loading) return <PageSkeleton label="Loading" />;

  return (
    <div>
      <div className="page-header">
        <h1>Corporate Bonds</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Bond</button>}
      </div>

      {bonds.length > 0 && (() => {
        const totalInvested = bonds.reduce((s, b) => s + b.investment_amount, 0);
        const totalInterest = bonds.reduce((s, b) => s + b.total_interest_earned, 0);
        const totalPrincipalReturned = bonds.reduce((s, b) => s + b.total_principal_returned, 0);
        const remaining = bonds.reduce((s, b) => s + b.remaining_principal, 0);
        const activeBonds = bonds.filter(b => b.status !== 'matured').length;
        const avgCoupon = bonds.length > 0 ? bonds.reduce((s, b) => s + b.coupon_rate, 0) / bonds.length : 0;
        const now = new Date();
        const thisMonth = now.getMonth();
        const thisYear = now.getFullYear();
        const interestThisMonth = bonds.reduce((s, b) => s + (b.interest_payouts || []).filter(p => { const d = new Date(p.scheduled_date); return d.getMonth() === thisMonth && d.getFullYear() === thisYear; }).reduce((s2, p) => s2 + p.amount, 0), 0);
        return (
          <div className="card-grid stat-grid mb-5">
            <div className="stat-card">
              <div className="label">Active / Total Bonds</div>
              <div className="value">{activeBonds} / {bonds.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Invested</div>
              <div className="value">{formatCurrency(totalInvested)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Interest Earned</div>
              <div className="value positive">{formatCurrency(totalInterest)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Interest This Month</div>
              <div className="value positive">{formatCurrency(interestThisMonth)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Principal Returned</div>
              <div className="value positive">{formatCurrency(totalPrincipalReturned)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Remaining Principal</div>
              <div className="value">{formatCurrency(remaining)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg Coupon Rate</div>
              <div className="value">{avgCoupon.toFixed(2)}%</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg XIRR</div>
              <div className={`value ${(() => { const w = bonds.filter(b => b.investment_amount > 0); const t = w.reduce((s, b) => s + b.investment_amount, 0); const x = t > 0 ? w.reduce((s, b) => s + b.xirr * b.investment_amount, 0) / t : 0; return x >= 0 ? 'positive' : 'negative'; })()}`}>
                {formatPercent((() => { const w = bonds.filter(b => b.investment_amount > 0); const t = w.reduce((s, b) => s + b.investment_amount, 0); return t > 0 ? w.reduce((s, b) => s + b.xirr * b.investment_amount, 0) / t : 0; })())}
              </div>
            </div>
          </div>
        );
      })()}

      {bonds.length === 0 ? (
        <div className="empty-state">
          <h3>No corporate bonds yet</h3>
          <p>Add your first corporate bond to track interest payouts and principal repayments.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Bond</button>}
        </div>
      ) : (
        <>
        <div className="toolbar">
          <div className="search-bar">
            <input type="search" placeholder="Search by bond name or issuer" aria-label="Search bonds" value={search} onChange={e => setSearch(e.target.value)} />
          </div>
          <FilterChips
            ariaLabel="Filter by status"
            value={statusFilter}
            onChange={setStatusFilter}
            options={[
              { value: 'all', label: 'All', count: bonds.length },
              { value: 'active', label: 'Active', count: bonds.filter(b => b.status === 'active').length },
              { value: 'partially_matured', label: 'Partially matured', count: bonds.filter(b => b.status === 'partially_matured').length },
              { value: 'matured', label: 'Matured', count: bonds.filter(b => b.status === 'matured').length },
            ]}
          />
        </div>
        {bonds.filter(b => {
          const matchSearch = !search || b.bond_name.toLowerCase().includes(search.toLowerCase()) || b.issuer.toLowerCase().includes(search.toLowerCase());
          const matchStatus = statusFilter === 'all' || b.status === statusFilter;
          return matchSearch && matchStatus;
        }).map(bond => (
          <div key={bond.id} className="card">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="text-lg">
                  {bond.bond_name}
                  <span className={`badge badge-${bond.status} ml-2`}>{statusLabel(bond.status)}</span>
                </h3>
                <div className="text-muted text-md">{bond.issuer}</div>
              </div>
              <div className="flex gap-1 items-center">
                <button className="btn btn-sm btn-ghost" aria-expanded={expanded === bond.id} onClick={() => setExpanded(expanded === bond.id ? null : bond.id)}>
                  <ChevronDown size={14} className={`row-chevron${expanded === bond.id ? ' flip' : ''}`} aria-hidden="true" />{expanded === bond.id ? 'Hide details' : 'Details'}
                </button>
                {!isViewOnly && (
                  <RowActions
                    label={`More actions for ${bond.bond_name}`}
                    actions={[
                      { label: 'Edit bond', icon: Pencil, onClick: () => openEdit(bond) },
                      { label: 'Delete bond', icon: Trash2, onClick: () => handleDelete(bond.id), danger: true },
                    ]}
                  />
                )}
              </div>
            </div>

            <div className="card-grid mt-3">
              <div><span className="text-muted text-sm">Investment</span><div className="fw-semibold">{formatCurrency(bond.investment_amount)}</div></div>
              <div><span className="text-muted text-sm">Coupon Rate</span><div className="fw-semibold">{bond.coupon_rate}%</div></div>
              <div><span className="text-muted text-sm">Frequency</span><div className="fw-semibold">{bond.interest_payout_frequency}</div></div>
              <div><span className="text-muted text-sm">Maturity</span><div className="fw-semibold">{formatDate(bond.maturity_date)} ({bond.maturity_type})</div></div>
              <div><span className="text-muted text-sm">Remaining Principal</span><div className="fw-semibold">{formatCurrency(bond.remaining_principal)}</div></div>
              <div><span className="text-muted text-sm">Total Interest Earned</span><div className="fw-semibold text-success">{formatCurrency(bond.total_interest_earned)}</div></div>
              <div><span className="text-muted text-sm">XIRR</span><div className="fw-semibold" style={{ color: bond.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatPercent(bond.xirr)}</div></div>
            </div>

            {expanded === bond.id && (
              <div className="mt-4">
                <h4 className="mb-2">Interest Payouts</h4>
                <div className="table-container mb-4">
                  <table>
                    <thead><tr><th>Date</th><th className="num">Principal at Time</th><th className="num">Amount</th><th>Status</th><th>Action</th></tr></thead>
                    <tbody>
                      {(bond.interest_payouts || []).map(p => (
                        <tr key={p.payout_id}>
                          <td>{formatDate(p.scheduled_date)}</td>
                          <td className="num">{formatCurrency(p.principal_at_time)}</td>
                          <td className="num">{formatCurrency(p.amount)}</td>
                          <td><span className={`badge badge-${p.status}`}>{statusLabel(p.status)}</span></td>
                          <td>
                            {!isViewOnly && p.status === 'pending' && <button className="btn btn-sm btn-success" onClick={() => handleMarkReceived(bond.id, p.payout_id)}>Mark Received</button>}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <h4 className="mb-2">Principal Repayments</h4>
                <div className="table-container">
                  <table>
                    <thead><tr><th>Date</th><th className="num">Amount</th><th>Status</th><th>Action</th></tr></thead>
                    <tbody>
                      {(bond.principal_repayments || []).map(p => (
                        <tr key={p.repayment_id}>
                          <td>{formatDate(p.scheduled_date)}</td>
                          <td className="num">{formatCurrency(p.amount)}</td>
                          <td><span className={`badge badge-${p.status}`}>{statusLabel(p.status)}</span></td>
                          <td>
                            {!isViewOnly && p.status === 'pending' && <button className="btn btn-sm btn-success" onClick={() => handleMarkReceived(bond.id, p.repayment_id)}>Mark Received</button>}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        ))}
        </>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Corporate Bond</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group">
                  <label>Bond Name</label>
                  <input required value={form.bond_name} onChange={e => setForm({ ...form, bond_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Issuer</label>
                  <input required value={form.issuer} onChange={e => setForm({ ...form, issuer: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Purchase Date</label>
                  <input type="date" required value={form.purchase_date} onChange={e => setForm({ ...form, purchase_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Investment Amount (₹)</label>
                  <input type="number" step="0.01" required value={form.investment_amount || ''} onChange={e => setForm({ ...form, investment_amount: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Coupon Rate (%)</label>
                  <input type="number" step="0.01" min={0} placeholder="0 for zero-coupon" value={form.coupon_rate || ''} onChange={e => setForm({ ...form, coupon_rate: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Interest Payout</label>
                  <select value={form.interest_payout_frequency} onChange={e => setForm({ ...form, interest_payout_frequency: e.target.value as PayoutFrequency })}>
                    <option value="monthly">Monthly</option>
                    <option value="quarterly">Quarterly</option>
                    <option value="biannually">Biannually</option>
                    <option value="annually">Annually</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Maturity Date</label>
                  <input type="date" required value={form.maturity_date} onChange={e => setForm({ ...form, maturity_date: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>Maturity Type</label>
                <select value={form.maturity_type} onChange={e => setForm({ ...form, maturity_type: e.target.value as MaturityType })}>
                  <option value="bullet">Bullet (Full amount at maturity)</option>
                  <option value="staggered">Staggered (Multiple repayments)</option>
                </select>
              </div>

              {form.maturity_type === 'staggered' && (
                <div className="p-4 mb-4" style={{ background: 'var(--surface-muted)', borderRadius: 8 }}>
                  <div className="flex justify-between items-center mb-3">
                    <label className="fw-semibold text-base">Principal Repayment Schedule</label>
                    <button type="button" className="btn btn-sm btn-outline" onClick={addRepayment}><Plus size={14} aria-hidden="true" />Add Repayment</button>
                  </div>
                  {repayments.map((r, i) => (
                    <div key={i} className="form-row mb-2">
                      <div className="form-group mb-0">
                        <input type="date" placeholder="Date" value={r.scheduled_date} onChange={e => {
                          const copy = [...repayments]; copy[i].scheduled_date = e.target.value; setRepayments(copy);
                        }} />
                      </div>
                      <div className="flex gap-2">
                        <div className="form-group mb-0 flex-1">
                          <input type="number" step="0.01" placeholder="Amount" value={r.amount || ''} onChange={e => {
                            const copy = [...repayments]; copy[i].amount = parseFloat(e.target.value); setRepayments(copy);
                          }} />
                        </div>
                        <button type="button" className="btn btn-sm btn-danger" onClick={() => removeRepayment(i)}>X</button>
                      </div>
                    </div>
                  ))}
                  {repayments.length === 0 && <div className="text-muted text-md">No repayments added. Click "+ Add Repayment" to define the schedule.</div>}
                </div>
              )}

              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Bond</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Bond Modal */}
      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Corporate Bond</h2>
            <form onSubmit={handleEdit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Bond Name</label>
                  <input required value={editForm.bond_name} onChange={e => setEditForm({ ...editForm, bond_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Issuer</label>
                  <input required value={editForm.issuer} onChange={e => setEditForm({ ...editForm, issuer: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Purchase Date</label>
                  <input type="date" required value={editForm.purchase_date} onChange={e => setEditForm({ ...editForm, purchase_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Investment Amount (₹)</label>
                  <input type="number" step="0.01" required value={editForm.investment_amount || ''} onChange={e => setEditForm({ ...editForm, investment_amount: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Coupon Rate (%)</label>
                  <input type="number" step="0.01" min={0} placeholder="0 for zero-coupon" value={editForm.coupon_rate || ''} onChange={e => setEditForm({ ...editForm, coupon_rate: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Interest Payout</label>
                  <select value={editForm.interest_payout_frequency} onChange={e => setEditForm({ ...editForm, interest_payout_frequency: e.target.value as PayoutFrequency })}>
                    <option value="monthly">Monthly</option>
                    <option value="quarterly">Quarterly</option>
                    <option value="biannually">Biannually</option>
                    <option value="annually">Annually</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Maturity Date</label>
                  <input type="date" required value={editForm.maturity_date} onChange={e => setEditForm({ ...editForm, maturity_date: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>Maturity Type</label>
                <select value={editForm.maturity_type} onChange={e => setEditForm({ ...editForm, maturity_type: e.target.value as MaturityType })}>
                  <option value="bullet">Bullet</option>
                  <option value="staggered">Staggered</option>
                </select>
              </div>
              {editForm.maturity_type === 'staggered' && (
                <div className="p-4 mb-4" style={{ background: 'var(--surface-muted)', borderRadius: 8 }}>
                  <div className="flex justify-between items-center mb-3">
                    <label className="fw-semibold text-base">Principal Repayment Schedule</label>
                    <button type="button" className="btn btn-sm btn-outline" onClick={() => setEditRepayments([...editRepayments, { scheduled_date: '', amount: 0 }])}><Plus size={14} aria-hidden="true" />Add</button>
                  </div>
                  {editRepayments.map((r, i) => (
                    <div key={i} className="form-row mb-2">
                      <div className="form-group mb-0">
                        <input type="date" value={r.scheduled_date} onChange={e => {
                          const copy = [...editRepayments]; copy[i].scheduled_date = e.target.value; setEditRepayments(copy);
                        }} />
                      </div>
                      <div className="flex gap-2">
                        <div className="form-group mb-0 flex-1">
                          <input type="number" step="0.01" value={r.amount || ''} onChange={e => {
                            const copy = [...editRepayments]; copy[i].amount = parseFloat(e.target.value); setEditRepayments(copy);
                          }} />
                        </div>
                        <button type="button" className="btn btn-sm btn-danger" onClick={() => setEditRepayments(editRepayments.filter((_, idx) => idx !== i))}>X</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Changes</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
