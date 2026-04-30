import { useEffect, useState } from 'react';
import { getCorporateBonds, createCorporateBond, deleteCorporateBond, markPayoutReceived, updateCorporateBond } from '../api/corporateBonds';
import type { CorporateBond, CreateCorporateBondRequest, PrincipalRepaymentInput, PayoutFrequency, MaturityType } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';

export default function CorporateBondsPage() {
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

  useEffect(() => { load(); }, []);

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

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Corporate Bonds</h1>
        <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Bond</button>
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
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(5, 1fr)', marginBottom: 20 }}>
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
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Bond</button>
        </div>
      ) : (
        <>
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, alignItems: 'center' }}>
          <div className="search-bar" style={{ flex: 1, marginBottom: 0 }}>
            <input type="search" placeholder="Search by bond name or issuer..." value={search} onChange={e => setSearch(e.target.value)} />
          </div>
          <div style={{ display: 'flex', gap: 4 }}>
            {['all', 'active', 'partially_matured', 'matured'].map(s => (
              <button key={s} className={`btn btn-sm ${statusFilter === s ? 'btn-primary' : 'btn-outline'}`} onClick={() => setStatusFilter(s)}>{s === 'partially_matured' ? 'Partial' : s.charAt(0).toUpperCase() + s.slice(1)}</button>
            ))}
          </div>
        </div>
        {bonds.filter(b => {
          const matchSearch = !search || b.bond_name.toLowerCase().includes(search.toLowerCase()) || b.issuer.toLowerCase().includes(search.toLowerCase());
          const matchStatus = statusFilter === 'all' || b.status === statusFilter;
          return matchSearch && matchStatus;
        }).map(bond => (
          <div key={bond.id} className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <h3 style={{ fontSize: 16 }}>
                  {bond.bond_name}
                  <span className={`badge badge-${bond.status}`} style={{ marginLeft: 8 }}>{bond.status}</span>
                </h3>
                <div className="text-muted" style={{ fontSize: 13 }}>{bond.issuer}</div>
              </div>
              <div style={{ display: 'flex', gap: 4 }}>
                <button className="btn btn-sm btn-outline" onClick={() => openEdit(bond)}>Edit</button>
                <button className="btn btn-sm btn-outline" onClick={() => setExpanded(expanded === bond.id ? null : bond.id)}>
                  {expanded === bond.id ? 'Hide' : 'Details'}
                </button>
                <button className="btn btn-sm btn-danger" onClick={() => handleDelete(bond.id)}>Del</button>
              </div>
            </div>

            <div className="card-grid" style={{ marginTop: 12 }}>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Investment</span><div style={{ fontWeight: 600 }}>{formatCurrency(bond.investment_amount)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Coupon Rate</span><div style={{ fontWeight: 600 }}>{bond.coupon_rate}%</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Frequency</span><div style={{ fontWeight: 600 }}>{bond.interest_payout_frequency}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Maturity</span><div style={{ fontWeight: 600 }}>{formatDate(bond.maturity_date)} ({bond.maturity_type})</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Remaining Principal</span><div style={{ fontWeight: 600 }}>{formatCurrency(bond.remaining_principal)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Total Interest Earned</span><div style={{ fontWeight: 600, color: 'var(--success)' }}>{formatCurrency(bond.total_interest_earned)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>XIRR</span><div style={{ fontWeight: 600, color: bond.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatPercent(bond.xirr)}</div></div>
            </div>

            {expanded === bond.id && (
              <div style={{ marginTop: 16 }}>
                <h4 style={{ marginBottom: 8 }}>Interest Payouts</h4>
                <div className="table-container" style={{ marginBottom: 16 }}>
                  <table>
                    <thead><tr><th>Date</th><th>Principal at Time</th><th>Amount</th><th>Status</th><th>Action</th></tr></thead>
                    <tbody>
                      {(bond.interest_payouts || []).map(p => (
                        <tr key={p.payout_id}>
                          <td>{formatDate(p.scheduled_date)}</td>
                          <td>{formatCurrency(p.principal_at_time)}</td>
                          <td>{formatCurrency(p.amount)}</td>
                          <td><span className={`badge badge-${p.status}`}>{p.status}</span></td>
                          <td>
                            {p.status === 'pending' && (
                              <button className="btn btn-sm btn-success" onClick={() => handleMarkReceived(bond.id, p.payout_id)}>Mark Received</button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <h4 style={{ marginBottom: 8 }}>Principal Repayments</h4>
                <div className="table-container">
                  <table>
                    <thead><tr><th>Date</th><th>Amount</th><th>Status</th><th>Action</th></tr></thead>
                    <tbody>
                      {(bond.principal_repayments || []).map(p => (
                        <tr key={p.repayment_id}>
                          <td>{formatDate(p.scheduled_date)}</td>
                          <td>{formatCurrency(p.amount)}</td>
                          <td><span className={`badge badge-${p.status}`}>{p.status}</span></td>
                          <td>
                            {p.status === 'pending' && (
                              <button className="btn btn-sm btn-success" onClick={() => handleMarkReceived(bond.id, p.repayment_id)}>Mark Received</button>
                            )}
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
                  <input type="number" step="0.01" required value={form.coupon_rate || ''} onChange={e => setForm({ ...form, coupon_rate: parseFloat(e.target.value) })} />
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
                <div style={{ background: '#f8f9fa', padding: 16, borderRadius: 8, marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                    <label style={{ fontWeight: 600, fontSize: 14 }}>Principal Repayment Schedule</label>
                    <button type="button" className="btn btn-sm btn-outline" onClick={addRepayment}>+ Add Repayment</button>
                  </div>
                  {repayments.map((r, i) => (
                    <div key={i} className="form-row" style={{ marginBottom: 8 }}>
                      <div className="form-group" style={{ marginBottom: 0 }}>
                        <input type="date" placeholder="Date" value={r.scheduled_date} onChange={e => {
                          const copy = [...repayments]; copy[i].scheduled_date = e.target.value; setRepayments(copy);
                        }} />
                      </div>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <div className="form-group" style={{ marginBottom: 0, flex: 1 }}>
                          <input type="number" step="0.01" placeholder="Amount" value={r.amount || ''} onChange={e => {
                            const copy = [...repayments]; copy[i].amount = parseFloat(e.target.value); setRepayments(copy);
                          }} />
                        </div>
                        <button type="button" className="btn btn-sm btn-danger" onClick={() => removeRepayment(i)}>X</button>
                      </div>
                    </div>
                  ))}
                  {repayments.length === 0 && <div className="text-muted" style={{ fontSize: 13 }}>No repayments added. Click "+ Add Repayment" to define the schedule.</div>}
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
                  <input type="number" step="0.01" required value={editForm.coupon_rate || ''} onChange={e => setEditForm({ ...editForm, coupon_rate: parseFloat(e.target.value) })} />
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
                <div style={{ background: '#f8f9fa', padding: 16, borderRadius: 8, marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                    <label style={{ fontWeight: 600, fontSize: 14 }}>Principal Repayment Schedule</label>
                    <button type="button" className="btn btn-sm btn-outline" onClick={() => setEditRepayments([...editRepayments, { scheduled_date: '', amount: 0 }])}>+ Add</button>
                  </div>
                  {editRepayments.map((r, i) => (
                    <div key={i} className="form-row" style={{ marginBottom: 8 }}>
                      <div className="form-group" style={{ marginBottom: 0 }}>
                        <input type="date" value={r.scheduled_date} onChange={e => {
                          const copy = [...editRepayments]; copy[i].scheduled_date = e.target.value; setEditRepayments(copy);
                        }} />
                      </div>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <div className="form-group" style={{ marginBottom: 0, flex: 1 }}>
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
