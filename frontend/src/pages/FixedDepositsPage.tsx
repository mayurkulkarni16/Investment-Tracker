import { useEffect, useState } from 'react';
import { getFixedDeposits, createFixedDeposit, deleteFixedDeposit, updateFixedDeposit } from '../api/fixedDeposits';
import type { FixedDeposit, CreateFixedDepositRequest, InterestType, PayoutFrequency } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';

export default function FixedDepositsPage() {
  const [fds, setFDs] = useState<FixedDeposit[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<FixedDeposit | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [sortCol, setSortCol] = useState<string>('');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [form, setForm] = useState<CreateFixedDepositRequest>({
    bank_name: '', principal_amount: 0, interest_rate: 0,
    start_date: '', maturity_date: '', tenure_months: 12,
    interest_type: 'cumulative', is_auto_renewed: false,
  });
  const [editForm, setEditForm] = useState<CreateFixedDepositRequest>({
    bank_name: '', principal_amount: 0, interest_rate: 0,
    start_date: '', maturity_date: '', tenure_months: 12,
    interest_type: 'cumulative', is_auto_renewed: false,
  });

  const load = () => {
    getFixedDeposits().then(r => setFDs(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await createFixedDeposit(form); setShowAdd(false); setForm({ bank_name: '', principal_amount: 0, interest_rate: 0, start_date: '', maturity_date: '', tenure_months: 12, interest_type: 'cumulative', is_auto_renewed: false }); toast('FD added'); load(); }
    catch { toast('Failed to add FD', 'error'); }
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Are you sure you want to delete this FD?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteFixedDeposit(id); toast('FD deleted'); load(); }
    catch { toast('Failed to delete FD', 'error'); }
  };

  const openEdit = (fd: FixedDeposit) => {
    setEditForm({
      bank_name: fd.bank_name,
      fd_number: fd.fd_number,
      principal_amount: fd.principal_amount,
      interest_rate: fd.interest_rate,
      start_date: toInputDate(fd.start_date),
      maturity_date: toInputDate(fd.maturity_date),
      tenure_months: fd.tenure_months,
      interest_type: fd.interest_type,
      payout_frequency: fd.payout_frequency,
      is_auto_renewed: fd.is_auto_renewed,
      notes: fd.notes,
    });
    setShowEdit(fd);
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try { await updateFixedDeposit(showEdit.id, editForm); setShowEdit(null); toast('FD updated'); load(); }
    catch { toast('Failed to update FD', 'error'); }
  };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Fixed Deposits</h1>
        <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add FD</button>
      </div>

      {fds.length > 0 && (() => {
        const activeFDs = fds.filter(f => f.status === 'active');
        const totalPrincipal = activeFDs.reduce((s, f) => s + f.principal_amount, 0);
        const totalMaturity = activeFDs.reduce((s, f) => s + f.maturity_amount, 0);
        const totalInterest = fds.reduce((s, f) => s + f.interest_earned, 0);
        const avgRate = activeFDs.length > 0 ? activeFDs.reduce((s, f) => s + f.interest_rate, 0) / activeFDs.length : 0;
        return (
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
            <div className="stat-card">
              <div className="label">Active / Total FDs</div>
              <div className="value">{activeFDs.length} / {fds.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Principal</div>
              <div className="value">{formatCurrency(totalPrincipal)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Interest Earned</div>
              <div className="value positive">{formatCurrency(totalInterest)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg Interest Rate</div>
              <div className="value">{avgRate.toFixed(2)}%</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg XIRR</div>
              <div className={`value ${(() => { const w = fds.filter(f => f.principal_amount > 0); const t = w.reduce((s, f) => s + f.principal_amount, 0); const x = t > 0 ? w.reduce((s, f) => s + f.xirr * f.principal_amount, 0) / t : 0; return x >= 0 ? 'positive' : 'negative'; })()}`}>
                {formatPercent((() => { const w = fds.filter(f => f.principal_amount > 0); const t = w.reduce((s, f) => s + f.principal_amount, 0); return t > 0 ? w.reduce((s, f) => s + f.xirr * f.principal_amount, 0) / t : 0; })())}
              </div>
            </div>
          </div>
        );
      })()}

      {fds.length === 0 ? (
        <div className="empty-state">
          <h3>No fixed deposits yet</h3>
          <p>Add your first FD to start tracking.</p>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add FD</button>
        </div>
      ) : (() => {
        const filtered = fds.filter(fd => {
          const matchSearch = !search || fd.bank_name.toLowerCase().includes(search.toLowerCase()) || (fd.fd_number || '').toLowerCase().includes(search.toLowerCase());
          const matchStatus = statusFilter === 'all' || fd.status === statusFilter;
          return matchSearch && matchStatus;
        });
        const toggleSort = (col: string) => { if (sortCol === col) setSortDir(d => d === 'asc' ? 'desc' : 'asc'); else { setSortCol(col); setSortDir('asc'); } };
        const sorted = [...filtered].sort((a, b) => {
          if (!sortCol) return 0;
          const dir = sortDir === 'asc' ? 1 : -1;
          const av = (a as any)[sortCol], bv = (b as any)[sortCol];
          if (typeof av === 'string') return av.localeCompare(bv) * dir;
          return ((av ?? 0) - (bv ?? 0)) * dir;
        });
        return (
          <>
            <div style={{ display: 'flex', gap: 12, marginBottom: 16, alignItems: 'center' }}>
              <div className="search-bar" style={{ flex: 1, marginBottom: 0 }}>
                <input type="search" placeholder="Search by bank name or FD number..." value={search} onChange={e => setSearch(e.target.value)} />
              </div>
              <div style={{ display: 'flex', gap: 4 }}>
                {['all', 'active', 'matured'].map(s => (
                  <button key={s} className={`btn btn-sm ${statusFilter === s ? 'btn-primary' : 'btn-outline'}`} onClick={() => setStatusFilter(s)}>{s.charAt(0).toUpperCase() + s.slice(1)}</button>
                ))}
              </div>
            </div>
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('bank_name')}>Bank {sortCol === 'bank_name' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th>FD Number</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('principal_amount')}>Principal {sortCol === 'principal_amount' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('interest_rate')}>Rate {sortCol === 'interest_rate' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('tenure_months')}>Tenure {sortCol === 'tenure_months' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('start_date')}>Start {sortCol === 'start_date' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('maturity_date')}>Maturity {sortCol === 'maturity_date' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('maturity_amount')}>Maturity Amt {sortCol === 'maturity_amount' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th>Type</th>
                <th>Status</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('xirr')}>XIRR {sortCol === 'xirr' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map(fd => (
                <tr key={fd.id}>
                  <td>{fd.bank_name}</td>
                  <td>{fd.fd_number || '-'}</td>
                  <td>{formatCurrency(fd.principal_amount)}</td>
                  <td>{fd.interest_rate}%</td>
                  <td>{fd.tenure_months}m</td>
                  <td>{formatDate(fd.start_date)}</td>
                  <td>{formatDate(fd.maturity_date)}</td>
                  <td>{formatCurrency(fd.maturity_amount)}</td>
                  <td>{fd.interest_type === 'cumulative' ? 'Cumulative' : 'Non-Cumulative'}</td>
                  <td><span className={`badge badge-${fd.status}`}>{fd.status}</span></td>
                  <td className={fd.xirr >= 0 ? 'text-success' : 'text-danger'}>{formatPercent(fd.xirr)}</td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }}>
                      <button className="btn btn-sm btn-outline" onClick={() => openEdit(fd)}>Edit</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(fd.id)}>Del</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
          </>
        );
      })()}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Fixed Deposit</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group">
                  <label>Bank Name</label>
                  <input required value={form.bank_name} onChange={e => setForm({ ...form, bank_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>FD Number</label>
                  <input value={form.fd_number || ''} onChange={e => setForm({ ...form, fd_number: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Principal (₹)</label>
                  <input type="number" step="0.01" required value={form.principal_amount || ''} onChange={e => setForm({ ...form, principal_amount: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Interest Rate (%)</label>
                  <input type="number" step="0.01" required value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Tenure (months)</label>
                  <input type="number" required value={form.tenure_months || ''} onChange={e => setForm({ ...form, tenure_months: parseInt(e.target.value) })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Start Date</label>
                  <input type="date" required value={form.start_date} onChange={e => setForm({ ...form, start_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Maturity Date</label>
                  <input type="date" required value={form.maturity_date} onChange={e => setForm({ ...form, maturity_date: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Interest Type</label>
                  <select value={form.interest_type} onChange={e => setForm({ ...form, interest_type: e.target.value as InterestType })}>
                    <option value="cumulative">Cumulative</option>
                    <option value="non_cumulative">Non-Cumulative</option>
                  </select>
                </div>
                {form.interest_type === 'non_cumulative' && (
                  <div className="form-group">
                    <label>Payout Frequency</label>
                    <select value={form.payout_frequency || 'quarterly'} onChange={e => setForm({ ...form, payout_frequency: e.target.value as PayoutFrequency })}>
                      <option value="monthly">Monthly</option>
                      <option value="quarterly">Quarterly</option>
                      <option value="annually">Annually</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="form-group">
                <label>
                  <input type="checkbox" checked={form.is_auto_renewed} onChange={e => setForm({ ...form, is_auto_renewed: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                  Auto-Renew on Maturity
                </label>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add FD</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Fixed Deposit</h2>
            <form onSubmit={handleEdit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Bank Name</label>
                  <input required value={editForm.bank_name} onChange={e => setEditForm({ ...editForm, bank_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>FD Number</label>
                  <input value={editForm.fd_number || ''} onChange={e => setEditForm({ ...editForm, fd_number: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Principal (₹)</label>
                  <input type="number" step="0.01" required value={editForm.principal_amount || ''} onChange={e => setEditForm({ ...editForm, principal_amount: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Interest Rate (%)</label>
                  <input type="number" step="0.01" required value={editForm.interest_rate || ''} onChange={e => setEditForm({ ...editForm, interest_rate: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Tenure (months)</label>
                  <input type="number" required value={editForm.tenure_months || ''} onChange={e => setEditForm({ ...editForm, tenure_months: parseInt(e.target.value) })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Start Date</label>
                  <input type="date" required value={editForm.start_date} onChange={e => setEditForm({ ...editForm, start_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Maturity Date</label>
                  <input type="date" required value={editForm.maturity_date} onChange={e => setEditForm({ ...editForm, maturity_date: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Interest Type</label>
                  <select value={editForm.interest_type} onChange={e => setEditForm({ ...editForm, interest_type: e.target.value as InterestType })}>
                    <option value="cumulative">Cumulative</option>
                    <option value="non_cumulative">Non-Cumulative</option>
                  </select>
                </div>
                {editForm.interest_type === 'non_cumulative' && (
                  <div className="form-group">
                    <label>Payout Frequency</label>
                    <select value={editForm.payout_frequency || 'quarterly'} onChange={e => setEditForm({ ...editForm, payout_frequency: e.target.value as PayoutFrequency })}>
                      <option value="monthly">Monthly</option>
                      <option value="quarterly">Quarterly</option>
                      <option value="annually">Annually</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="form-group">
                <label>
                  <input type="checkbox" checked={editForm.is_auto_renewed} onChange={e => setEditForm({ ...editForm, is_auto_renewed: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                  Auto-Renew on Maturity
                </label>
              </div>
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
