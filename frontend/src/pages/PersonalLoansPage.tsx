import { useEffect, useState } from 'react';
import { getPersonalLoans, createPersonalLoan, deletePersonalLoan, updatePersonalLoan, recordPersonalLoanEMI, addPersonalLoanPrepayment, changePersonalLoanRate, getPersonalLoanAmortization } from '../api/personalLoans';
import type { PersonalLoan, CreatePersonalLoanRequest, UpdatePersonalLoanRequest, AddEMIPaymentRequest, AddPrepaymentRequest, ChangeRateRequest, LoanRateType, AmortizationEntry } from '../types';
import { formatCurrency, formatDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, Line } from 'recharts';

const PURPOSES: Record<string, string> = {
  personal: 'Personal', education: 'Education', vehicle: 'Vehicle',
  medical: 'Medical', wedding: 'Wedding', travel: 'Travel', other: 'Other',
};

export default function PersonalLoansPage() {
  const { isViewOnly } = useAuth();
  const [loans, setLoans] = useState<PersonalLoan[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<PersonalLoan | null>(null);
  const [showEMI, setShowEMI] = useState<string | null>(null);
  const [showPrepay, setShowPrepay] = useState<string | null>(null);
  const [showRate, setShowRate] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [expandedLoan, setExpandedLoan] = useState<string | null>(null);
  const [amortization, setAmortization] = useState<AmortizationEntry[] | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [showAmort, setShowAmort] = useState<string | null>(null);

  const [form, setForm] = useState<CreatePersonalLoanRequest>({
    lender_name: '', principal_amount: 0, disbursed_amount: 0,
    interest_rate: 0, rate_type: 'fixed', tenure_months: 36,
    emi_start_date: '', disbursement_date: '',
  });
  const [editForm, setEditForm] = useState<UpdatePersonalLoanRequest>({});
  const [emiForm, setEmiForm] = useState<AddEMIPaymentRequest>({ month: '', paid_date: '' });
  const [prepayForm, setPrepayForm] = useState<AddPrepaymentRequest>({ date: '', amount: 0, type: 'part_payment' });
  const [rateForm, setRateForm] = useState<ChangeRateRequest>({ effective_date: '', new_rate: 0 });

  const load = () => {
    getPersonalLoans().then(r => setLoans(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createPersonalLoan(form);
      setShowAdd(false);
      setForm({ lender_name: '', principal_amount: 0, disbursed_amount: 0, interest_rate: 0, rate_type: 'fixed', tenure_months: 36, emi_start_date: '', disbursement_date: '' });
      toast('Personal loan added');
      load();
    } catch { toast('Failed to add loan', 'error'); }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try { await updatePersonalLoan(showEdit.id, editForm); setShowEdit(null); toast('Loan updated'); load(); }
    catch { toast('Failed to update loan', 'error'); }
  };

  const openEdit = (l: PersonalLoan) => {
    setEditForm({ lender_name: l.lender_name, loan_account_number: l.loan_account_number, loan_purpose: l.loan_purpose, notes: l.notes });
    setShowEdit(l);
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Delete this personal loan?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deletePersonalLoan(id); toast('Loan deleted'); load(); }
    catch { toast('Failed to delete loan', 'error'); }
  };

  const handleRecordEMI = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEMI) return;
    try { await recordPersonalLoanEMI(showEMI, emiForm); setShowEMI(null); setEmiForm({ month: '', paid_date: '' }); toast('EMI recorded'); load(); }
    catch { toast('Failed to record EMI', 'error'); }
  };

  const handlePrepay = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showPrepay) return;
    try { await addPersonalLoanPrepayment(showPrepay, prepayForm); setShowPrepay(null); setPrepayForm({ date: '', amount: 0, type: 'part_payment' }); toast('Prepayment added'); load(); }
    catch { toast('Failed to add prepayment', 'error'); }
  };

  const handleRateChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showRate) return;
    try { await changePersonalLoanRate(showRate, rateForm); setShowRate(null); setRateForm({ effective_date: '', new_rate: 0 }); toast('Rate updated'); load(); }
    catch { toast('Failed to update rate', 'error'); }
  };

  const handleShowAmort = async (id: string) => {
    if (showAmort === id) { setShowAmort(null); setAmortization(null); return; }
    try { const res = await getPersonalLoanAmortization(id); setAmortization(res.data); setShowAmort(id); }
    catch { toast('Failed to load amortization', 'error'); }
  };

  const activeLoans = loans.filter(l => l.status === 'active');
  const totalOutstanding = activeLoans.reduce((s, l) => s + l.outstanding_principal, 0);
  const totalEMI = activeLoans.reduce((s, l) => s + l.emi_amount, 0);
  const totalInterestPaid = loans.reduce((s, l) => s + l.total_interest_paid, 0);
  const totalPrepayments = loans.reduce((s, l) => s + l.total_prepayments, 0);

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Personal Loans</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Loan</button>}
      </div>

      <div className="card-grid" style={{ marginBottom: 20 }}>
        <div className="stat-card"><div className="label">Outstanding Principal</div><div className="value">{formatCurrency(totalOutstanding)}</div></div>
        <div className="stat-card"><div className="label">Monthly EMI</div><div className="value">{formatCurrency(totalEMI)}</div></div>
        <div className="stat-card"><div className="label">Total Interest Paid</div><div className="value negative">{formatCurrency(totalInterestPaid)}</div></div>
        <div className="stat-card"><div className="label">Total Prepayments</div><div className="value positive">{formatCurrency(totalPrepayments)}</div></div>
      </div>

      {loans.length === 0 ? (
        <div className="empty-state">
          <h3>No personal loans yet</h3>
          <p>Add your first personal loan to start tracking.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Loan</button>}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <div className="search-bar" style={{ flex: 1, marginBottom: 0 }}>
              <input type="search" placeholder="Search by lender name..." value={search} onChange={e => setSearch(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: 4 }}>
              {['all', 'active', 'closed'].map(s => (
                <button key={s} className={`btn btn-sm ${statusFilter === s ? 'btn-primary' : 'btn-outline'}`} onClick={() => setStatusFilter(s)}>{s.charAt(0).toUpperCase() + s.slice(1)}</button>
              ))}
            </div>
          </div>
          {loans.filter(l => {
            const matchSearch = !search || l.lender_name.toLowerCase().includes(search.toLowerCase()) || (l.loan_purpose || '').toLowerCase().includes(search.toLowerCase());
            const matchStatus = statusFilter === 'all' || l.status === statusFilter;
            return matchSearch && matchStatus;
          }).map(l => (
            <div key={l.id} className="card" style={{ padding: 20 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                <div>
                  <h3 style={{ margin: 0 }}>{l.lender_name}</h3>
                  <div className="text-muted" style={{ fontSize: 12 }}>
                    A/C: {l.loan_account_number || '-'} | Purpose: {PURPOSES[l.loan_purpose] || l.loan_purpose || '-'}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                  <span style={{
                    padding: '3px 10px', borderRadius: 12, fontSize: 11, fontWeight: 600,
                    background: l.status === 'active' ? '#e8f5e9' : l.status === 'closed' ? '#e3f2fd' : '#fff3e0',
                    color: l.status === 'active' ? '#2e7d32' : l.status === 'closed' ? '#1565c0' : '#e65100',
                  }}>{l.status.toUpperCase()}</span>
                  <span style={{
                    padding: '3px 10px', borderRadius: 12, fontSize: 11, fontWeight: 600,
                    background: l.rate_type === 'floating' ? '#fce4ec' : '#e8eaf6',
                    color: l.rate_type === 'floating' ? '#c62828' : '#283593',
                  }}>{l.rate_type.toUpperCase()}</span>
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12, marginBottom: 12 }}>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Disbursed</div><strong>{formatCurrency(l.disbursed_amount)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Outstanding</div><strong>{formatCurrency(l.outstanding_principal)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Interest Rate</div><strong>{l.interest_rate}%</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>EMI</div><strong>{formatCurrency(l.emi_amount)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Tenure</div><strong>{Math.floor(l.tenure_months / 12)}y {l.tenure_months % 12}m</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Remaining</div><strong>{Math.floor(l.remaining_tenure_months / 12)}y {l.remaining_tenure_months % 12}m</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Principal Paid</div><strong>{formatCurrency(l.total_principal_paid)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Interest Paid</div><strong>{formatCurrency(l.total_interest_paid)}</strong></div>
              </div>

              {(l.processing_fee > 0 || l.foreclosure_charges > 0) && (
                <div style={{ background: '#fff8e1', padding: '8px 12px', borderRadius: 6, marginBottom: 12, fontSize: 13 }}>
                  {l.processing_fee > 0 && <span>Processing Fee: {formatCurrency(l.processing_fee)}</span>}
                  {l.processing_fee > 0 && l.foreclosure_charges > 0 && <span> | </span>}
                  {l.foreclosure_charges > 0 && <span>Foreclosure Charges: {l.foreclosure_charges}%</span>}
                </div>
              )}

              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {l.status === 'active' && (
                  <>
                    {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowEMI(l.id)}>Record EMI</button>}
                    {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowPrepay(l.id)}>Prepayment</button>}
                    {!isViewOnly && l.rate_type === 'floating' && <button className="btn btn-sm btn-outline" onClick={() => setShowRate(l.id)}>Change Rate</button>}
                  </>
                )}
                <button className="btn btn-sm btn-outline" onClick={() => handleShowAmort(l.id)}>
                  {showAmort === l.id ? 'Hide' : 'Amortization'}
                </button>
                <button className="btn btn-sm btn-outline" onClick={() => setExpandedLoan(expandedLoan === l.id ? null : l.id)}>
                  History ({(l.emis_paid?.length || 0) + (l.prepayments?.length || 0)})
                </button>
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => openEdit(l)}>Edit</button>}
                {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDelete(l.id)}>Del</button>}
              </div>

              {expandedLoan === l.id && (
                <div style={{ marginTop: 12 }}>
                  {l.emis_paid?.length > 0 && (
                    <div style={{ marginBottom: 12 }}>
                      <strong style={{ fontSize: 13 }}>EMI History</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Month</th><th>EMI</th><th>Principal</th><th>Interest</th><th>Outstanding</th><th>Paid On</th></tr></thead>
                        <tbody>{[...l.emis_paid].reverse().map(e => (
                          <tr key={e.emi_id}><td>{e.month}</td><td>{formatCurrency(e.emi_amount)}</td><td>{formatCurrency(e.principal_portion)}</td><td>{formatCurrency(e.interest_portion)}</td><td>{formatCurrency(e.outstanding_after)}</td><td>{e.paid_date ? formatDate(e.paid_date) : '-'}</td></tr>
                        ))}</tbody>
                      </table>
                    </div>
                  )}
                  {l.prepayments?.length > 0 && (
                    <div style={{ marginBottom: 12 }}>
                      <strong style={{ fontSize: 13 }}>Prepayments</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Date</th><th>Amount</th><th>Type</th><th>Notes</th></tr></thead>
                        <tbody>{l.prepayments.map(p => (
                          <tr key={p.prepayment_id}><td>{formatDate(p.date)}</td><td>{formatCurrency(p.amount)}</td><td>{p.type.replace('_', ' ').toUpperCase()}</td><td>{p.notes || '-'}</td></tr>
                        ))}</tbody>
                      </table>
                    </div>
                  )}
                  {l.rate_change_history?.length > 0 && (
                    <div>
                      <strong style={{ fontSize: 13 }}>Rate Change History</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Date</th><th>Old Rate</th><th>New Rate</th><th>New EMI</th></tr></thead>
                        <tbody>{l.rate_change_history.map((r, i) => (
                          <tr key={i}><td>{formatDate(r.effective_date)}</td><td>{r.old_rate}%</td><td>{r.new_rate}%</td><td>{formatCurrency(r.new_emi)}</td></tr>
                        ))}</tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {showAmort === l.id && amortization && (
                <div style={{ marginTop: 12 }}>
                  <strong style={{ fontSize: 13 }}>Amortization Schedule (Remaining)</strong>
                  <div style={{ marginTop: 12, marginBottom: 16 }}>
                    <ResponsiveContainer width="100%" height={250}>
                      <AreaChart data={amortization}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="month" fontSize={11} />
                        <YAxis tickFormatter={(v: number) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                        <Tooltip formatter={(v) => formatCurrency(v as number)} />
                        <Legend />
                        <Area type="monotone" dataKey="principal_portion" name="Principal" stackId="emi" stroke="#1a73e8" fill="#bbdefb" />
                        <Area type="monotone" dataKey="interest_portion" name="Interest" stackId="emi" stroke="#ea4335" fill="#ffcdd2" />
                        <Line type="monotone" dataKey="outstanding_after" name="Outstanding" stroke="#f57f17" strokeWidth={2} dot={false} />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                  <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                  <table style={{ fontSize: 12, marginTop: 6 }}>
                    <thead><tr><th>#</th><th>EMI</th><th>Principal</th><th>Interest</th><th>Outstanding</th></tr></thead>
                    <tbody>{amortization.map(a => (
                      <tr key={a.month}><td>{a.month}</td><td>{formatCurrency(a.emi)}</td><td>{formatCurrency(a.principal_portion)}</td><td>{formatCurrency(a.interest_portion)}</td><td>{formatCurrency(a.outstanding_after)}</td></tr>
                    ))}</tbody>
                  </table>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Add Modal */}
      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 600 }}>
            <h2>Add Personal Loan</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group"><label>Lender / Bank</label><input required value={form.lender_name} onChange={e => setForm({ ...form, lender_name: e.target.value })} /></div>
                <div className="form-group"><label>Account Number</label><input value={form.loan_account_number || ''} onChange={e => setForm({ ...form, loan_account_number: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Principal Amount (₹)</label><input type="number" required value={form.principal_amount || ''} onChange={e => setForm({ ...form, principal_amount: parseFloat(e.target.value) })} /></div>
                <div className="form-group"><label>Disbursed Amount (₹)</label><input type="number" required value={form.disbursed_amount || ''} onChange={e => setForm({ ...form, disbursed_amount: parseFloat(e.target.value) })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Interest Rate (%)</label><input type="number" step="0.01" required value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) })} /></div>
                <div className="form-group"><label>Rate Type</label>
                  <select value={form.rate_type} onChange={e => setForm({ ...form, rate_type: e.target.value as LoanRateType })}>
                    <option value="fixed">Fixed</option>
                    <option value="floating">Floating</option>
                  </select>
                </div>
                <div className="form-group"><label>Tenure (months)</label><input type="number" required value={form.tenure_months || ''} onChange={e => setForm({ ...form, tenure_months: parseInt(e.target.value) })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Disbursement Date</label><input type="date" required value={form.disbursement_date} onChange={e => setForm({ ...form, disbursement_date: e.target.value })} /></div>
                <div className="form-group"><label>EMI Start Date</label><input type="date" required value={form.emi_start_date} onChange={e => setForm({ ...form, emi_start_date: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Loan Purpose</label>
                  <select value={form.loan_purpose || 'personal'} onChange={e => setForm({ ...form, loan_purpose: e.target.value })}>
                    {Object.entries(PURPOSES).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                  </select>
                </div>
                <div className="form-group"><label>Processing Fee (₹)</label><input type="number" value={form.processing_fee || ''} onChange={e => setForm({ ...form, processing_fee: parseFloat(e.target.value) })} /></div>
                <div className="form-group"><label>Foreclosure Charges (%)</label><input type="number" step="0.01" value={form.foreclosure_charges || ''} onChange={e => setForm({ ...form, foreclosure_charges: parseFloat(e.target.value) })} /></div>
              </div>
              <div className="form-group"><label>Notes</label><textarea value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Loan</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Personal Loan</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group"><label>Lender / Bank</label><input value={editForm.lender_name || ''} onChange={e => setEditForm({ ...editForm, lender_name: e.target.value })} /></div>
              <div className="form-group"><label>Account Number</label><input value={editForm.loan_account_number || ''} onChange={e => setEditForm({ ...editForm, loan_account_number: e.target.value })} /></div>
              <div className="form-group"><label>Purpose</label><input value={editForm.loan_purpose || ''} onChange={e => setEditForm({ ...editForm, loan_purpose: e.target.value })} /></div>
              <div className="form-group"><label>Notes</label><textarea value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Changes</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Record EMI Modal */}
      {showEMI && (
        <div className="modal-overlay" onClick={() => setShowEMI(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Record EMI Payment</h2>
            <form onSubmit={handleRecordEMI}>
              <div className="form-group"><label>Month (e.g., 2026-04)</label><input required placeholder="2026-04" value={emiForm.month} onChange={e => setEmiForm({ ...emiForm, month: e.target.value })} /></div>
              <div className="form-group"><label>Paid Date</label><input type="date" required value={emiForm.paid_date} onChange={e => setEmiForm({ ...emiForm, paid_date: e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEMI(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Record Payment</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Prepayment Modal */}
      {showPrepay && (
        <div className="modal-overlay" onClick={() => setShowPrepay(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Prepayment</h2>
            <form onSubmit={handlePrepay}>
              <div className="form-group"><label>Date</label><input type="date" required value={prepayForm.date} onChange={e => setPrepayForm({ ...prepayForm, date: e.target.value })} /></div>
              <div className="form-group"><label>Amount (₹)</label><input type="number" step="0.01" required value={prepayForm.amount || ''} onChange={e => setPrepayForm({ ...prepayForm, amount: parseFloat(e.target.value) })} /></div>
              <div className="form-group"><label>Type</label>
                <select value={prepayForm.type} onChange={e => setPrepayForm({ ...prepayForm, type: e.target.value as 'part_payment' | 'foreclosure' })}>
                  <option value="part_payment">Part Payment</option>
                  <option value="foreclosure">Foreclosure</option>
                </select>
              </div>
              <div className="form-group"><label>Notes</label><textarea value={prepayForm.notes || ''} onChange={e => setPrepayForm({ ...prepayForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowPrepay(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Prepayment</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Rate Change Modal */}
      {showRate && (
        <div className="modal-overlay" onClick={() => setShowRate(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Change Interest Rate</h2>
            <form onSubmit={handleRateChange}>
              <div className="form-group"><label>Effective Date</label><input type="date" required value={rateForm.effective_date} onChange={e => setRateForm({ ...rateForm, effective_date: e.target.value })} /></div>
              <div className="form-group"><label>New Rate (%)</label><input type="number" step="0.01" required value={rateForm.new_rate || ''} onChange={e => setRateForm({ ...rateForm, new_rate: parseFloat(e.target.value) })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowRate(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Update Rate</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
