import React, { Fragment, useEffect, useState } from 'react';
import { ChevronRight, List, Pencil, Plus, Trash2 } from 'lucide-react';
import { getFixedDeposits, createFixedDeposit, deleteFixedDeposit, updateFixedDeposit } from '../api/fixedDeposits';
import type { FixedDeposit, CreateFixedDepositRequest, InterestType, PayoutFrequency } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate, statusLabel } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import RowActions from '../components/RowActions';
import { FilterChips, PageSkeleton, SortableTh } from '../components/ui';
import { useSort } from '../utils/useSort';

const FD_SORT: Record<string, (f: FixedDeposit) => string | number> = {
  bank_name: f => f.bank_name,
  principal_amount: f => f.principal_amount,
  interest_rate: f => f.interest_rate,
  maturity_date: f => f.maturity_date,
  xirr: f => f.xirr,
};

export default function FixedDepositsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [fds, setFDs] = useState<FixedDeposit[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<FixedDeposit | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [expanded, setExpanded] = useState<string | null>(null);
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

  useEffect(() => { load(); }, [viewAsUserId]);

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

  const filtered = fds.filter(fd => {
    const matchSearch = !search || fd.bank_name.toLowerCase().includes(search.toLowerCase()) || (fd.fd_number || '').toLowerCase().includes(search.toLowerCase());
    const matchStatus = statusFilter === 'all' || fd.status === statusFilter;
    return matchSearch && matchStatus;
  });
  const { sorted, sortCol, sortDir, onSort } = useSort(filtered, FD_SORT);
  const sortProps = { sortCol, sortDir, onSort };

  if (loading) return <PageSkeleton label="Loading fixed deposits" />;

  const activeFDs = fds.filter(f => f.status === 'active');
  const totalPrincipal = activeFDs.reduce((s, f) => s + f.principal_amount, 0);
  const totalMaturity = activeFDs.reduce((s, f) => s + f.maturity_amount, 0);
  const totalInterest = fds.reduce((s, f) => s + f.interest_earned, 0);
  const avgRate = totalPrincipal > 0 ? activeFDs.reduce((s, f) => s + f.interest_rate * f.principal_amount, 0) / totalPrincipal : 0;
  const xirrBase = fds.filter(f => f.principal_amount > 0);
  const xirrWeight = xirrBase.reduce((s, f) => s + f.principal_amount, 0);
  const avgXirr = xirrWeight > 0 ? xirrBase.reduce((s, f) => s + f.xirr * f.principal_amount, 0) / xirrWeight : 0;
  const countBy = (status: string) => fds.filter(f => f.status === status).length;

  return (
    <div>
      <div className="page-header">
        <h1>Fixed Deposits</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add FD</button>}
      </div>

      {fds.length > 0 && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card">
            <div className="label">Active principal</div>
            <div className="value">{formatCurrency(totalPrincipal)}</div>
            <div className="text-muted text-sm" style={{ marginTop: 2 }}>{activeFDs.length} of {fds.length} FDs active</div>
          </div>
          <div className="stat-card">
            <div className="label">Maturity value</div>
            <div className="value">{formatCurrency(totalMaturity)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Interest earned</div>
            <div className="value positive">{formatCurrency(totalInterest)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Avg rate</div>
            <div className="value">{avgRate.toFixed(2)}%</div>
            <div className="text-muted text-sm" style={{ marginTop: 2 }}>Weighted by principal</div>
          </div>
          <div className="stat-card">
            <div className="label">Avg XIRR</div>
            <div className={`value ${avgXirr >= 0 ? 'positive' : 'negative'}`}>{formatPercent(avgXirr)}</div>
          </div>
        </div>
      )}

      {fds.length === 0 ? (
        <div className="empty-state">
          <h3>No fixed deposits yet</h3>
          <p>Add an FD to track its maturity date, interest payouts and returns.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add FD</button>}
        </div>
      ) : (
        <>
          <div className="toolbar">
            <div className="search-bar">
              <input type="search" placeholder="Search by bank or FD number" aria-label="Search fixed deposits" value={search} onChange={e => setSearch(e.target.value)} />
            </div>
            <FilterChips
              ariaLabel="Filter by status"
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: 'all', label: 'All', count: fds.length },
                { value: 'active', label: 'Active', count: countBy('active') },
                { value: 'matured', label: 'Matured', count: countBy('matured') },
              ]}
            />
          </div>

          {sorted.length === 0 ? (
            <div className="empty-state">
              <h3>No FDs match</h3>
              <p>Nothing matches your search or filter.</p>
              <button className="btn btn-outline" onClick={() => { setSearch(''); setStatusFilter('all'); }}>Clear filters</button>
            </div>
          ) : (
            <div className="table-container table-stack">
              <table>
                <thead>
                  <tr>
                    <SortableTh col="bank_name" {...sortProps}>Bank</SortableTh>
                    <SortableTh col="principal_amount" align="right" {...sortProps}>Principal</SortableTh>
                    <SortableTh col="interest_rate" align="right" {...sortProps}>Rate</SortableTh>
                    <SortableTh col="maturity_date" align="right" {...sortProps}>Maturity</SortableTh>
                    <th>Interest</th>
                    <th>Status</th>
                    <SortableTh col="xirr" align="right" {...sortProps}>XIRR</SortableTh>
                    <th><span className="sr-only">Actions</span></th>
                  </tr>
                </thead>
                <tbody>
                  {sorted.map(fd => {
                    const payouts = fd.interest_payouts || [];
                    const canExpand = fd.interest_type === 'non_cumulative' && payouts.length > 0;
                    const open = expanded === fd.id && canExpand;
                    const toggle = () => { if (canExpand) setExpanded(open ? null : fd.id); };
                    return (
                      <Fragment key={fd.id}>
                        <tr className={open ? 'is-expanded' : undefined} style={{ cursor: canExpand ? 'pointer' : undefined }} onClick={toggle}>
                          <td className="cell-primary">
                            <div className="cell-title">
                              {canExpand && <ChevronRight size={15} className={`row-chevron${open ? ' open' : ''}`} aria-hidden="true" />}
                              <span className="cell-main">{fd.bank_name}</span>
                              {fd.is_auto_renewed && <span className="badge badge-neutral">Auto-renew</span>}
                            </div>
                            <span className="cell-sub">{fd.fd_number ? `FD ${fd.fd_number} · ` : ''}{fd.tenure_months} mo from {formatDate(fd.start_date)}</span>
                          </td>
                          <td className="num cell-main" data-label="Principal">{formatCurrency(fd.principal_amount)}</td>
                          <td className="num" data-label="Rate">{fd.interest_rate}%</td>
                          <td className="num" data-label="Maturity">
                            {formatCurrency(fd.maturity_amount)}
                            <span className="cell-sub">on {formatDate(fd.maturity_date)}</span>
                          </td>
                          <td data-label="Interest">
                            {fd.interest_type === 'cumulative' ? 'Cumulative' : 'Paid out'}
                            {fd.interest_type !== 'cumulative' && fd.payout_frequency && <span className="cell-sub capitalize">{fd.payout_frequency.replace(/_/g, ' ')}</span>}
                          </td>
                          <td data-label="Status"><span className={`badge badge-${fd.status} capitalize`}>{statusLabel(fd.status)}</span></td>
                          <td className={`num ${fd.xirr >= 0 ? 'text-success' : 'text-danger'}`} data-label="XIRR">{formatPercent(fd.xirr)}</td>
                          <td className="cell-actions">
                            <RowActions
                              label={`Actions for ${fd.bank_name} FD`}
                              actions={[
                                { label: open ? 'Hide payouts' : `View payouts (${payouts.length})`, icon: List, onClick: toggle, hidden: !canExpand },
                                { label: 'Edit FD', icon: Pencil, onClick: () => openEdit(fd), hidden: isViewOnly },
                                { label: 'Delete FD', icon: Trash2, onClick: () => handleDelete(fd.id), danger: true, hidden: isViewOnly },
                              ]}
                            />
                          </td>
                        </tr>
                        {open && (
                          <tr className="row-detail">
                            <td className="p-0" colSpan={8}>
                              <div className="row-detail-panel">
                                <div className="row-detail-head">
                                  <strong className="text-base">Interest payouts</strong>
                                  <span className="text-muted text-sm">
                                    {payouts.filter(p => p.status === 'received').length} of {payouts.length} received · {formatCurrency(payouts.reduce((s, p) => s + p.amount, 0))} total
                                  </span>
                                </div>
                                <table>
                                  <thead><tr><th>Date</th><th className="num">Amount</th><th>Status</th></tr></thead>
                                  <tbody>
                                    {payouts.map(p => (
                                      <tr key={p.payout_id}>
                                        <td>{formatDate(p.scheduled_date)}</td>
                                        <td className="num">{formatCurrency(p.amount)}</td>
                                        <td><span className={`badge badge-${p.status} capitalize`}>{statusLabel(p.status)}</span></td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
                {sorted.length > 1 && (
                  <tfoot>
                    <tr>
                      <td className="cell-primary">Total · {sorted.length} FDs</td>
                      <td className="num">{formatCurrency(sorted.reduce((s, f) => s + f.principal_amount, 0))}</td>
                      <td />
                      <td className="num">{formatCurrency(sorted.reduce((s, f) => s + f.maturity_amount, 0))}</td>
                      <td /><td /><td /><td className="cell-actions" />
                    </tr>
                  </tfoot>
                )}
              </table>
            </div>
          )}
        </>
      )}

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
                      <option value="biannually">Bi-Annually</option>
                      <option value="annually">Annually</option>
                      <option value="at_maturity">At Maturity</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="form-group">
                <label>
                  <input className="w-auto mr-2" type="checkbox" checked={form.is_auto_renewed} onChange={e => setForm({ ...form, is_auto_renewed: e.target.checked })} />
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
                      <option value="biannually">Bi-Annually</option>
                      <option value="annually">Annually</option>
                      <option value="at_maturity">At Maturity</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="form-group">
                <label>
                  <input className="w-auto mr-2" type="checkbox" checked={editForm.is_auto_renewed} onChange={e => setEditForm({ ...editForm, is_auto_renewed: e.target.checked })} />
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
