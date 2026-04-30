import { useEffect, useState } from 'react';
import { getNPSAccounts, createNPSAccount, updateNPSAccount, deleteNPSAccount, addNPSContribution } from '../api/nps';
import type { NPSAccount, CreateNPSAccountRequest, UpdateNPSAccountRequest, AddNPSContributionRequest } from '../types';
import { formatCurrency, formatPercent } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';

export default function NPSPage() {
  const { isViewOnly } = useAuth();
  const [accounts, setAccounts] = useState<NPSAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<NPSAccount | null>(null);
  const [showContrib, setShowContrib] = useState<string | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [form, setForm] = useState<CreateNPSAccountRequest>({
    account_holder_name: '', pran: '', account_type: 'tier_1', fund_manager: '',
    date_of_joining: '', equity_pct: 50, corporate_bond_pct: 30, govt_sec_pct: 20,
    alternate_pct: 0, current_value: 0,
  });
  const [contribForm, setContribForm] = useState<AddNPSContributionRequest>({
    date: '', amount: 0, type: 'self',
  });
  const [editForm, setEditForm] = useState<UpdateNPSAccountRequest>({});

  const load = () => {
    getNPSAccounts().then(r => setAccounts(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    const sum = form.equity_pct + form.corporate_bond_pct + form.govt_sec_pct + form.alternate_pct;
    if (sum !== 100) { toast(`Allocation must total 100% (currently ${sum}%)`, 'error'); return; }
    try { await createNPSAccount(form); setShowAdd(false); toast('NPS account added'); load(); }
    catch { toast('Failed to add NPS account', 'error'); }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    const sum = (editForm.equity_pct ?? 0) + (editForm.corporate_bond_pct ?? 0) + (editForm.govt_sec_pct ?? 0) + (editForm.alternate_pct ?? 0);
    if (sum !== 100) { toast(`Allocation must total 100% (currently ${sum}%)`, 'error'); return; }
    try { await updateNPSAccount(showEdit.id, editForm); setShowEdit(null); toast('NPS account updated'); load(); }
    catch { toast('Failed to update NPS account', 'error'); }
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Delete this NPS account?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteNPSAccount(id); toast('NPS account deleted'); load(); }
    catch { toast('Failed to delete', 'error'); }
  };

  const handleContrib = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showContrib) return;
    try { await addNPSContribution(showContrib, contribForm); setShowContrib(null); setContribForm({ date: '', amount: 0, type: 'self' }); toast('Contribution added'); load(); }
    catch { toast('Failed to add contribution', 'error'); }
  };

  const totalValue = accounts.reduce((s, a) => s + a.current_value, 0);
  const totalContrib = accounts.reduce((s, a) => s + a.total_contribution, 0);

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>NPS Accounts</h1>
          <p className="text-muted" style={{ fontSize: 14 }}>Total Value: {formatCurrency(totalValue)} | Total Contributions: {formatCurrency(totalContrib)}</p>
        </div>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add NPS Account</button>}
      </div>

      {accounts.length > 0 && (() => {
        const gainLoss = totalValue - totalContrib;
        const gainPct = totalContrib > 0 ? (gainLoss / totalContrib) * 100 : 0;
        return (
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
            <div className="stat-card">
              <div className="label">NPS Accounts</div>
              <div className="value">{accounts.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Contributions</div>
              <div className="value">{formatCurrency(totalContrib)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Current Value</div>
              <div className="value">{formatCurrency(totalValue)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Gain/Loss</div>
              <div className={`value ${gainLoss >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(gainLoss)} ({gainPct.toFixed(1)}%)</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg XIRR</div>
              <div className={`value ${(() => { const w = accounts.filter(a => a.current_value > 0); const t = w.reduce((s, a) => s + a.current_value, 0); const x = t > 0 ? w.reduce((s, a) => s + a.xirr * a.current_value, 0) / t : 0; return x >= 0 ? 'positive' : 'negative'; })()}`}>
                {formatPercent((() => { const w = accounts.filter(a => a.current_value > 0); const t = w.reduce((s, a) => s + a.current_value, 0); return t > 0 ? w.reduce((s, a) => s + a.xirr * a.current_value, 0) / t : 0; })())}
              </div>
            </div>
          </div>
        );
      })()}

      {accounts.length === 0 ? (
        <div className="empty-state">
          <h3>No NPS accounts</h3>
          <p>Add one to get started.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add NPS Account</button>}
        </div>
      ) : (
        <div className="card-grid">
          <div style={{ gridColumn: '1 / -1' }}>
            <div className="search-bar" style={{ marginBottom: 0 }}>
              <input type="search" placeholder="Search by name or PRAN..." value={search} onChange={e => setSearch(e.target.value)} />
            </div>
          </div>
          {accounts.filter(a => !search || a.account_holder_name.toLowerCase().includes(search.toLowerCase()) || a.pran.toLowerCase().includes(search.toLowerCase())).map(a => (
            <div key={a.id} className="card" style={{ cursor: 'pointer' }} onClick={() => setExpanded(expanded === a.id ? null : a.id)}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <h3 style={{ fontSize: 16, fontWeight: 600 }}>{a.account_holder_name}</h3>
                  <p className="text-muted" style={{ fontSize: 13 }}>PRAN: {a.pran} | {a.account_type.toUpperCase()} | {a.fund_manager}</p>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: 20, fontWeight: 700 }}>{formatCurrency(a.current_value)}</div>
                  <div className="text-muted" style={{ fontSize: 13 }}>Contributed: {formatCurrency(a.total_contribution)}</div>
                  <div style={{ fontSize: 13, fontWeight: 600, color: a.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>XIRR: {formatPercent(a.xirr)}</div>
                </div>
              </div>

              <div className="allocation-bar" style={{ marginTop: 12 }}>
                <div className="allocation-segment" style={{ width: `${a.equity_pct}%`, background: '#1a73e8' }}>E {a.equity_pct}%</div>
                <div className="allocation-segment" style={{ width: `${a.corporate_bond_pct}%`, background: '#0f9d58' }}>C {a.corporate_bond_pct}%</div>
                <div className="allocation-segment" style={{ width: `${a.govt_sec_pct}%`, background: '#f9ab00' }}>G {a.govt_sec_pct}%</div>
                {a.alternate_pct > 0 && <div className="allocation-segment" style={{ width: `${a.alternate_pct}%`, background: '#9c27b0' }}>A {a.alternate_pct}%</div>}
              </div>

              {a.section_80ccd1 > 0 && (
                <div style={{ marginTop: 8, padding: '8px 12px', background: '#e6f4ea', borderRadius: 6, fontSize: 13, color: '#1e8e3e' }}>
                  Tax: 80CCD(1): {formatCurrency(a.section_80ccd1)} | 80CCD(1B): {formatCurrency(a.section_80ccd1b)} | 80CCD(2): {formatCurrency(a.section_80ccd2)}
                </div>
              )}

              <div style={{ marginTop: 12, display: 'flex', gap: 8 }} onClick={e => e.stopPropagation()}>
                {!isViewOnly && <button className="btn btn-sm btn-success" onClick={() => { setShowContrib(a.id); setContribForm({ date: '', amount: 0, type: 'self' }); }}>Add Contribution</button>}
                {!isViewOnly && <button className="btn btn-sm btn-primary" onClick={() => { setShowEdit(a); setEditForm({ fund_manager: a.fund_manager, equity_pct: a.equity_pct, corporate_bond_pct: a.corporate_bond_pct, govt_sec_pct: a.govt_sec_pct, alternate_pct: a.alternate_pct, current_value: a.current_value }); }}>Edit</button>}
                {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDelete(a.id)}>Delete</button>}
              </div>

              {expanded === a.id && a.contributions?.length > 0 && (
                <div style={{ marginTop: 16 }}>
                  <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8 }}>Contributions</h4>
                  <table>
                    <thead><tr><th>Date</th><th>Type</th><th className="text-right">Amount</th><th>FY</th></tr></thead>
                    <tbody>
                      {a.contributions.map((c, i) => (
                        <tr key={i}><td>{new Date(c.date).toLocaleDateString()}</td><td>{c.type}</td><td className="text-right">{formatCurrency(c.amount)}</td><td>{c.fy}</td></tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add NPS Account</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group"><label>Account Holder Name</label><input required value={form.account_holder_name} onChange={e => setForm({ ...form, account_holder_name: e.target.value })} /></div>
                <div className="form-group"><label>PRAN</label><input required value={form.pran} onChange={e => setForm({ ...form, pran: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Account Type</label>
                  <select value={form.account_type} onChange={e => setForm({ ...form, account_type: e.target.value as 'tier_1' | 'tier_2' })}>
                    <option value="tier_1">Tier 1</option><option value="tier_2">Tier 2</option>
                  </select>
                </div>
                <div className="form-group"><label>Fund Manager</label><input value={form.fund_manager} onChange={e => setForm({ ...form, fund_manager: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Date of Joining</label><input type="date" required value={form.date_of_joining} onChange={e => setForm({ ...form, date_of_joining: e.target.value })} /></div>
                <div className="form-group"><label>Current Value (₹)</label><input type="number" value={form.current_value || ''} onChange={e => setForm({ ...form, current_value: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Equity %</label><input type="number" value={form.equity_pct} onChange={e => setForm({ ...form, equity_pct: +e.target.value })} /></div>
                <div className="form-group"><label>Corporate Bond %</label><input type="number" value={form.corporate_bond_pct} onChange={e => setForm({ ...form, corporate_bond_pct: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Govt Securities %</label><input type="number" value={form.govt_sec_pct} onChange={e => setForm({ ...form, govt_sec_pct: +e.target.value })} /></div>
                <div className="form-group"><label>Alternate %</label><input type="number" value={form.alternate_pct} onChange={e => setForm({ ...form, alternate_pct: +e.target.value })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit NPS Account</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group"><label>Fund Manager</label><input value={editForm.fund_manager || ''} onChange={e => setEditForm({ ...editForm, fund_manager: e.target.value })} /></div>
              <div className="form-group"><label>Current Value (₹)</label><input type="number" value={editForm.current_value || ''} onChange={e => setEditForm({ ...editForm, current_value: +e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Equity %</label><input type="number" value={editForm.equity_pct ?? ''} onChange={e => setEditForm({ ...editForm, equity_pct: +e.target.value })} /></div>
                <div className="form-group"><label>Corporate Bond %</label><input type="number" value={editForm.corporate_bond_pct ?? ''} onChange={e => setEditForm({ ...editForm, corporate_bond_pct: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Govt Securities %</label><input type="number" value={editForm.govt_sec_pct ?? ''} onChange={e => setEditForm({ ...editForm, govt_sec_pct: +e.target.value })} /></div>
                <div className="form-group"><label>Alternate %</label><input type="number" value={editForm.alternate_pct ?? ''} onChange={e => setEditForm({ ...editForm, alternate_pct: +e.target.value })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showContrib && (
        <div className="modal-overlay" onClick={() => setShowContrib(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 400 }}>
            <h2>Add Contribution</h2>
            <form onSubmit={handleContrib}>
              <div className="form-group"><label>Date</label><input type="date" required value={contribForm.date} onChange={e => setContribForm({ ...contribForm, date: e.target.value })} /></div>
              <div className="form-group"><label>Amount (₹)</label><input type="number" required value={contribForm.amount || ''} onChange={e => setContribForm({ ...contribForm, amount: +e.target.value })} /></div>
              <div className="form-group">
                <label>Type</label>
                <select value={contribForm.type} onChange={e => setContribForm({ ...contribForm, type: e.target.value })}>
                  <option value="self">Self</option><option value="employer">Employer</option>
                </select>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowContrib(null)}>Cancel</button>
                <button type="submit" className="btn btn-success">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
