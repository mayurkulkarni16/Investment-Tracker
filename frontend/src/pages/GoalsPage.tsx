import { useEffect, useState } from 'react';
import { getGoals, createGoal, updateGoal, deleteGoal, linkInvestment, unlinkInvestment } from '../api/goals';
import { getMutualFunds } from '../api/mutualFunds';
import { getStocks } from '../api/stocks';
import { getFixedDeposits } from '../api/fixedDeposits';
import { getNPSAccounts } from '../api/nps';
import { getProvidentFunds } from '../api/providentFund';
import { getCorporateBonds } from '../api/corporateBonds';
import type { Goal, CreateGoalRequest, UpdateGoalRequest, LinkInvestmentRequest } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';

const CATEGORIES = ['retirement', 'education', 'house', 'car', 'wedding', 'emergency', 'travel', 'other'];
const ICONS: Record<string, string> = { retirement: '🏖️', education: '🎓', house: '🏠', car: '🚗', wedding: '💍', emergency: '🆘', travel: '✈️', other: '🎯' };

export default function GoalsPage() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<Goal | null>(null);
  const [showLink, setShowLink] = useState<string | null>(null);
  const { toast } = useToast();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [form, setForm] = useState<CreateGoalRequest>({ name: '', category: 'retirement', target_amount: 0, target_date: '', assumed_return_rate: 12 });
  const [editForm, setEditForm] = useState<UpdateGoalRequest>({});
  const [linkForm, setLinkForm] = useState<LinkInvestmentRequest>({ investment_type: 'mutual_fund', investment_id: '', investment_name: '', allocated_pct: 100 });
  const [availableInvestments, setAvailableInvestments] = useState<{ id: string; name: string }[]>([]);
  const [loadingInv, setLoadingInv] = useState(false);

  const load = () => { getGoals().then(r => setGoals(r.data || [])).catch(() => {}).finally(() => setLoading(false)); };
  useEffect(() => { load(); }, []);

  const fetchInvestments = async (type: string) => {
    setLoadingInv(true);
    try {
      let items: { id: string; name: string }[] = [];
      switch (type) {
        case 'mutual_fund': { const r = await getMutualFunds(); items = (r.data || []).map(i => ({ id: i.id, name: i.fund_name })); break; }
        case 'stock': { const r = await getStocks(); items = (r.data || []).map(i => ({ id: i.id, name: `${i.stock_name} (${i.symbol})` })); break; }
        case 'fixed_deposit': { const r = await getFixedDeposits(); items = (r.data || []).map(i => ({ id: i.id, name: `${i.bank_name} - ${i.fd_number}` })); break; }
        case 'nps': { const r = await getNPSAccounts(); items = (r.data || []).map(i => ({ id: i.id, name: `${i.account_holder_name} (${i.pran})` })); break; }
        case 'provident_fund': { const r = await getProvidentFunds(); items = (r.data || []).map(i => ({ id: i.id, name: `${i.employer_name} - ${i.account_number}` })); break; }
        case 'corporate_bond': { const r = await getCorporateBonds(); items = (r.data || []).map(i => ({ id: i.id, name: `${i.bond_name} (${i.issuer})` })); break; }
      }
      setAvailableInvestments(items);
    } catch { setAvailableInvestments([]); }
    setLoadingInv(false);
  };

  const handleAdd = async (e: React.FormEvent) => { e.preventDefault(); try { await createGoal({ ...form, icon: ICONS[form.category] || '🎯' }); setShowAdd(false); toast('Goal added'); load(); } catch { toast('Failed to add goal', 'error'); } };
  const handleEdit = async (e: React.FormEvent) => { e.preventDefault(); if (!showEdit) return; try { await updateGoal(showEdit.id, editForm); setShowEdit(null); toast('Goal updated'); load(); } catch { toast('Failed to update goal', 'error'); } };
  const handleDelete = async (id: string) => { if (!confirm('Delete this goal?')) return; try { await deleteGoal(id); toast('Goal deleted'); load(); } catch { toast('Failed to delete goal', 'error'); } };
  const handleLink = async (e: React.FormEvent) => { e.preventDefault(); if (!showLink) return; try { await linkInvestment(showLink, linkForm); setShowLink(null); toast('Investment linked'); load(); } catch { toast('Failed to link investment', 'error'); } };
  const handleUnlink = async (goalId: string, invId: string) => { try { await unlinkInvestment(goalId, invId); toast('Investment unlinked'); load(); } catch { toast('Failed to unlink', 'error'); } };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Financial Goals</h1>
        <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Goal</button>
      </div>

      {goals.length > 0 && (() => {
        const totalTarget = goals.reduce((s, g) => s + g.target_amount, 0);
        const totalCurrent = goals.reduce((s, g) => s + g.current_value, 0);
        const onTrack = goals.filter(g => g.on_track).length;
        const overallProgress = totalTarget > 0 ? (totalCurrent / totalTarget) * 100 : 0;
        return (
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
            <div className="stat-card">
              <div className="label">Goals (On Track)</div>
              <div className="value">{onTrack} / {goals.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Target</div>
              <div className="value">{formatCurrency(totalTarget)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Current Value</div>
              <div className="value">{formatCurrency(totalCurrent)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Overall Progress</div>
              <div className="value">{overallProgress.toFixed(1)}%</div>
            </div>
          </div>
        );
      })()}

      {goals.length === 0 ? (
        <div className="empty-state">
          <h3>No goals set</h3>
          <p>Start planning your financial future!</p>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Goal</button>
        </div>
      ) : (
        <div className="card-grid">
          {goals.map(g => (
            <div key={g.id} className="card" style={{ cursor: 'pointer' }} onClick={() => setExpanded(expanded === g.id ? null : g.id)}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <h3 style={{ fontSize: 16, fontWeight: 600 }}>{g.icon} {g.name}</h3>
                  <p className="text-muted" style={{ fontSize: 13 }}>{g.category} | Target: {new Date(g.target_date).toLocaleDateString()}</p>
                </div>
                <span className={`badge ${g.on_track ? 'badge-active' : 'badge-matured'}`}>
                  {g.on_track ? 'On Track' : 'Behind'}
                </span>
              </div>

              <div style={{ marginTop: 12 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 4 }}>
                  <span>{formatCurrency(g.current_value)}</span>
                  <span>{formatCurrency(g.target_amount)}</span>
                </div>
                <div style={{ width: '100%', background: '#e0e0e0', borderRadius: 6, height: 10, overflow: 'hidden' }}>
                  <div style={{
                    width: `${Math.min(g.progress_pct, 100)}%`, height: '100%', borderRadius: 6,
                    background: g.progress_pct >= 100 ? '#0f9d58' : g.progress_pct >= 50 ? '#1a73e8' : '#f9ab00',
                  }}></div>
                </div>
                <p className="text-muted" style={{ fontSize: 12, marginTop: 4 }}>
                  {g.progress_pct?.toFixed(1)}% complete | {g.months_remaining} months left | Need {formatCurrency(g.monthly_needed)}/mo
                </p>
              </div>

              <div style={{ marginTop: 12, display: 'flex', gap: 8 }} onClick={e => e.stopPropagation()}>
                <button className="btn btn-sm btn-primary" onClick={() => { setShowLink(g.id); fetchInvestments(linkForm.investment_type); }}>Link Investment</button>
                <button className="btn btn-sm btn-outline" onClick={() => { setShowEdit(g); setEditForm({ name: g.name, category: g.category, target_amount: g.target_amount, target_date: g.target_date?.slice(0, 10), assumed_return_rate: g.assumed_return_rate, priority: g.priority, status: g.status, notes: g.notes }); }}>Edit</button>
                <button className="btn btn-sm btn-danger" onClick={() => handleDelete(g.id)}>Delete</button>
              </div>

              {expanded === g.id && g.linked_investments?.length > 0 && (
                <div style={{ marginTop: 12 }}>
                  <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 4 }}>Linked Investments</h4>
                  {g.linked_investments.map(li => (
                    <div key={li.investment_id} className="payout-item" style={{ fontSize: 13 }}>
                      <span>{li.investment_name} ({li.investment_type}) — {li.allocated_pct}%</span>
                      <button className="btn btn-sm btn-danger" onClick={(e) => { e.stopPropagation(); handleUnlink(g.id, li.investment_id); }}>Remove</button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 480 }}>
            <h2>Add Goal</h2>
            <form onSubmit={handleAdd}>
              <div className="form-group"><label>Goal Name</label><input required value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group">
                  <label>Category</label>
                  <select value={form.category} onChange={e => setForm({ ...form, category: e.target.value })}>
                    {CATEGORIES.map(c => <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>)}
                  </select>
                </div>
                <div className="form-group"><label>Target Amount (₹)</label><input type="number" required value={form.target_amount || ''} onChange={e => setForm({ ...form, target_amount: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Target Date</label><input type="date" required value={form.target_date} onChange={e => setForm({ ...form, target_date: e.target.value })} /></div>
                <div className="form-group"><label>Expected Return %</label><input type="number" step="0.1" value={form.assumed_return_rate} onChange={e => setForm({ ...form, assumed_return_rate: +e.target.value })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showLink && (
        <div className="modal-overlay" onClick={() => setShowLink(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 480 }}>
            <h2>Link Investment</h2>
            <form onSubmit={handleLink}>
              <div className="form-group">
                <label>Investment Type</label>
                <select value={linkForm.investment_type} onChange={e => { const type = e.target.value; setLinkForm({ ...linkForm, investment_type: type, investment_id: '', investment_name: '' }); fetchInvestments(type); }}>
                  <option value="mutual_fund">Mutual Fund</option><option value="stock">Stock</option><option value="fixed_deposit">Fixed Deposit</option>
                  <option value="nps">NPS</option><option value="provident_fund">Provident Fund</option><option value="corporate_bond">Corporate Bond</option>
                </select>
              </div>
              <div className="form-group">
                <label>Select Investment</label>
                {loadingInv ? <p className="text-muted" style={{ fontSize: 13 }}>Loading...</p> : availableInvestments.length === 0 ? (
                  <p className="text-muted" style={{ fontSize: 13 }}>No {linkForm.investment_type.replace(/_/g, ' ')}s found. Add one first.</p>
                ) : (
                  <select value={linkForm.investment_id} onChange={e => { const inv = availableInvestments.find(i => i.id === e.target.value); setLinkForm({ ...linkForm, investment_id: e.target.value, investment_name: inv?.name || '' }); }}>
                    <option value="">-- Select --</option>
                    {availableInvestments.map(inv => <option key={inv.id} value={inv.id}>{inv.name}</option>)}
                  </select>
                )}
              </div>
              <div className="form-group"><label>Allocation %</label><input type="number" min={1} max={100} value={linkForm.allocated_pct} onChange={e => setLinkForm({ ...linkForm, allocated_pct: +e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowLink(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={!linkForm.investment_id}>Link</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 480 }}>
            <h2>Edit Goal</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group"><label>Name</label><input value={editForm.name || ''} onChange={e => setEditForm({ ...editForm, name: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group">
                  <label>Category</label>
                  <select value={editForm.category || ''} onChange={e => setEditForm({ ...editForm, category: e.target.value })}>
                    {CATEGORIES.map(c => <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>)}
                  </select>
                </div>
                <div className="form-group">
                  <label>Priority</label>
                  <select value={editForm.priority || 'medium'} onChange={e => setEditForm({ ...editForm, priority: e.target.value })}>
                    <option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Target Amount (₹)</label><input type="number" value={editForm.target_amount || ''} onChange={e => setEditForm({ ...editForm, target_amount: +e.target.value })} /></div>
                <div className="form-group"><label>Target Date</label><input type="date" value={editForm.target_date || ''} onChange={e => setEditForm({ ...editForm, target_date: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Return Rate (%)</label><input type="number" step="0.1" value={editForm.assumed_return_rate || ''} onChange={e => setEditForm({ ...editForm, assumed_return_rate: +e.target.value })} /></div>
                <div className="form-group">
                  <label>Status</label>
                  <select value={editForm.status || 'active'} onChange={e => setEditForm({ ...editForm, status: e.target.value })}>
                    <option value="active">Active</option><option value="paused">Paused</option><option value="achieved">Achieved</option><option value="cancelled">Cancelled</option>
                  </select>
                </div>
              </div>
              <div className="form-group"><label>Notes</label><textarea value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
