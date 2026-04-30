import { useEffect, useState } from 'react';
import { getHomeLoans, createHomeLoan, deleteHomeLoan, updateHomeLoan, recordEMI, addPrepayment, changeRate, getAmortization, addDisbursement, markConstructionComplete, recalculateLoan } from '../api/homeLoans';
import type { HomeLoan, CreateHomeLoanRequest, UpdateHomeLoanRequest, AddEMIPaymentRequest, AddPrepaymentRequest, ChangeRateRequest, LoanRateType, AmortizationEntry, AddDisbursementRequest } from '../types';
import { formatCurrency, formatDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, Line } from 'recharts';

export default function HomeLoansPage() {
  const { isViewOnly } = useAuth();
  const [loans, setLoans] = useState<HomeLoan[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<HomeLoan | null>(null);
  const [showEMI, setShowEMI] = useState<string | null>(null);
  const [showPrepay, setShowPrepay] = useState<string | null>(null);
  const [showRate, setShowRate] = useState<string | null>(null);
  const [expandedLoan, setExpandedLoan] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [amortization, setAmortization] = useState<AmortizationEntry[] | null>(null);
  const [showAmort, setShowAmort] = useState<string | null>(null);
  const [showDisb, setShowDisb] = useState<string | null>(null);
  const [showMarkComplete, setShowMarkComplete] = useState<string | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();

  const [form, setForm] = useState<CreateHomeLoanRequest>({
    bank_name: '', sanctioned_amount: 0, disbursed_amount: 0,
    interest_rate: 0, rate_type: 'floating', tenure_months: 240,
    emi_start_date: '', disbursement_date: '',
  });
  const [editForm, setEditForm] = useState<UpdateHomeLoanRequest>({});
  const [emiForm, setEmiForm] = useState<AddEMIPaymentRequest>({ month: '', paid_date: '' });
  const [prepayForm, setPrepayForm] = useState<AddPrepaymentRequest>({ date: '', amount: 0, type: 'part_payment' });
  const [rateForm, setRateForm] = useState<ChangeRateRequest>({ effective_date: '', new_rate: 0 });
  const [disbForm, setDisbForm] = useState<AddDisbursementRequest>({ date: '', amount: 0 });

  const load = () => {
    getHomeLoans().then(r => setLoans(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createHomeLoan(form);
      setShowAdd(false);
      setForm({ bank_name: '', sanctioned_amount: 0, disbursed_amount: 0, interest_rate: 0, rate_type: 'floating', tenure_months: 240, emi_start_date: '', disbursement_date: '' });
      toast('Home loan added');
      load();
    } catch { toast('Failed to add home loan', 'error'); }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try { await updateHomeLoan(showEdit.id, editForm); setShowEdit(null); toast('Home loan updated'); load(); }
    catch { toast('Failed to update home loan', 'error'); }
  };

  const openEdit = (l: HomeLoan) => {
    setEditForm({ bank_name: l.bank_name, loan_account_number: l.loan_account_number, property_address: l.property_address, loan_purpose: l.loan_purpose, co_borrower: l.co_borrower, notes: l.notes });
    setShowEdit(l);
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Delete this home loan?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteHomeLoan(id); toast('Home loan deleted'); load(); }
    catch { toast('Failed to delete home loan', 'error'); }
  };

  const handleRecordEMI = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEMI) return;
    try { await recordEMI(showEMI, emiForm); setShowEMI(null); setEmiForm({ month: '', paid_date: '' }); toast('EMI recorded'); load(); }
    catch { toast('Failed to record EMI', 'error'); }
  };

  const handlePrepay = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showPrepay) return;
    try { await addPrepayment(showPrepay, prepayForm); setShowPrepay(null); setPrepayForm({ date: '', amount: 0, type: 'part_payment' }); toast('Prepayment added'); load(); }
    catch { toast('Failed to add prepayment', 'error'); }
  };

  const handleRateChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showRate) return;
    try { await changeRate(showRate, rateForm); setShowRate(null); setRateForm({ effective_date: '', new_rate: 0 }); toast('Rate updated'); load(); }
    catch { toast('Failed to update rate', 'error'); }
  };

  const handleShowAmort = async (id: string) => {
    if (showAmort === id) { setShowAmort(null); setAmortization(null); return; }
    try { const res = await getAmortization(id); setAmortization(res.data); setShowAmort(id); }
    catch { toast('Failed to load amortization', 'error'); }
  };

  const handleAddDisbursement = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showDisb) return;
    try { await addDisbursement(showDisb, disbForm); setShowDisb(null); setDisbForm({ date: '', amount: 0 }); toast('Disbursement added'); load(); }
    catch { toast('Failed to add disbursement', 'error'); }
  };

  const handleMarkComplete = async () => {
    if (!showMarkComplete) return;
    try { await markConstructionComplete(showMarkComplete); setShowMarkComplete(null); toast('Marked construction complete'); load(); }
    catch { toast('Failed to mark complete', 'error'); }
  };

  const handleRecalculate = async (id: string) => {
    if (!await confirm({ message: 'Recalculate all EMIs and pre-EMIs using the correct interest rate at each point in time? This replays all transactions chronologically.', danger: false, confirmLabel: 'Confirm' })) return;
    try { await recalculateLoan(id); toast('Loan recalculated'); load(); }
    catch { toast('Failed to recalculate', 'error'); }
  };

  // Totals
  const activeLoans = loans.filter(l => l.status === 'active');
  const totalOutstanding = activeLoans.reduce((s, l) => s + l.outstanding_principal, 0);
  const totalEMI = activeLoans.reduce((s, l) => s + l.emi_amount, 0);
  const totalInterestPaid = loans.reduce((s, l) => s + l.total_interest_paid, 0);
  const totalPrepayments = loans.reduce((s, l) => s + l.total_prepayments, 0);
  const totalPreEMI = loans.reduce((s, l) => s + l.total_pre_emi_paid, 0);

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Home Loans</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Loan</button>}
      </div>

      {/* Summary cards */}
      <div className="card-grid" style={{ marginBottom: 20 }}>
        <div className="stat-card">
          <div className="label">Outstanding Principal</div>
          <div className="value">{formatCurrency(totalOutstanding)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Monthly EMI</div>
          <div className="value">{formatCurrency(totalEMI)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Total Interest Paid</div>
          <div className="value negative">{formatCurrency(totalInterestPaid)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Total Prepayments</div>
          <div className="value positive">{formatCurrency(totalPrepayments)}</div>
        </div>
        {totalPreEMI > 0 && (
          <div className="stat-card">
            <div className="label">Total Pre-EMI Paid</div>
            <div className="value negative">{formatCurrency(totalPreEMI)}</div>
          </div>
        )}
      </div>

      {loans.length === 0 ? (
        <div className="empty-state">
          <h3>No home loans yet</h3>
          <p>Add your first home loan to start tracking.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Loan</button>}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <div className="search-bar" style={{ flex: 1, marginBottom: 0 }}>
              <input type="search" placeholder="Search by bank name..." value={search} onChange={e => setSearch(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: 4 }}>
              {['all', 'active', 'closed'].map(s => (
                <button key={s} className={`btn btn-sm ${statusFilter === s ? 'btn-primary' : 'btn-outline'}`} onClick={() => setStatusFilter(s)}>{s.charAt(0).toUpperCase() + s.slice(1)}</button>
              ))}
            </div>
          </div>
          {loans.filter(l => {
            const matchSearch = !search || l.bank_name.toLowerCase().includes(search.toLowerCase());
            const matchStatus = statusFilter === 'all' || l.status === statusFilter;
            return matchSearch && matchStatus;
          }).map(l => (
            <div key={l.id} className="card" style={{ padding: 20 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                <div>
                  <h3 style={{ margin: 0 }}>{l.bank_name}</h3>
                  <div className="text-muted" style={{ fontSize: 12 }}>
                    A/C: {l.loan_account_number || '-'} | {l.property_address || 'No address'}
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
                  {l.is_under_construction && (
                    <span style={{
                      padding: '3px 10px', borderRadius: 12, fontSize: 11, fontWeight: 600,
                      background: '#fff3e0', color: '#e65100',
                    }}>CONSTRUCTION</span>
                  )}
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12, marginBottom: 12 }}>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Sanctioned</div><strong>{formatCurrency(l.sanctioned_amount)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Disbursed</div><strong>{formatCurrency(l.disbursed_amount)}</strong>{l.is_under_construction && <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}> ({Math.round(l.disbursed_amount / l.sanctioned_amount * 100)}%)</span>}</div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Outstanding</div><strong>{formatCurrency(l.outstanding_principal)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Interest Rate</div><strong>{l.interest_rate}%</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>EMI</div><strong>{formatCurrency(l.emi_amount)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Tenure</div><strong>{Math.floor(l.tenure_months / 12)}y {l.tenure_months % 12}m</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Remaining</div><strong>{Math.floor(l.remaining_tenure_months / 12)}y {l.remaining_tenure_months % 12}m</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Principal Paid</div><strong>{formatCurrency(l.total_principal_paid)}</strong></div>
                <div><div className="text-muted" style={{ fontSize: 12 }}>Interest Paid</div><strong>{formatCurrency(l.total_interest_paid)}</strong></div>
                {l.total_pre_emi_paid > 0 && <div><div className="text-muted" style={{ fontSize: 12 }}>Pre-EMI Paid</div><strong className="negative">{formatCurrency(l.total_pre_emi_paid)}</strong></div>}
              </div>

              {/* Disbursement progress for under-construction */}
              {l.is_under_construction && l.disbursements?.length > 0 && (
                <div style={{ background: '#fff8e1', padding: '8px 12px', borderRadius: 6, marginBottom: 12, fontSize: 13 }}>
                  <strong>Disbursements ({l.disbursements.length} tranche{l.disbursements.length > 1 ? 's' : ''}):</strong>
                  <div style={{ display: 'flex', gap: 12, marginTop: 4, flexWrap: 'wrap' }}>
                    {l.disbursements.map(d => (
                      <span key={d.disbursement_id} style={{ fontSize: 12 }}>
                        T{d.tranche}: {formatCurrency(d.amount)} ({formatDate(d.date)})
                        {d.pre_emi_amount > 0 && <span style={{ color: '#e65100' }}> — Pre-EMI: {formatCurrency(d.pre_emi_amount)} ({d.pre_emi_days}/{d.days_in_month} days)</span>}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Tax benefits */}
              {(l.interest_paid_this_fy > 0 || l.principal_paid_this_fy > 0 || l.pre_emi_paid_this_fy > 0) && (
                <div style={{ background: '#f0f7ff', padding: '8px 12px', borderRadius: 6, marginBottom: 12, fontSize: 13 }}>
                  <strong>Tax Benefits (Current FY):</strong>{' '}
                  Sec 24(b) Interest: {formatCurrency(Math.min(l.interest_paid_this_fy, 200000))} (max ₹2L)
                  {l.pre_emi_paid_this_fy > 0 && <span> (incl. Pre-EMI: {formatCurrency(l.pre_emi_paid_this_fy)})</span>}
                  {' '}| Sec 80C Principal: {formatCurrency(Math.min(l.principal_paid_this_fy, 150000))} (max ₹1.5L)
                </div>
              )}

              {l.co_borrower && (
                <div style={{ fontSize: 12, marginBottom: 8 }} className="text-muted">Co-borrower: {l.co_borrower}</div>
              )}

              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {l.status === 'active' && (
                  <>
                    {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowEMI(l.id)}>Record EMI</button>}
                    {l.is_under_construction && (
                      <>
                        {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowDisb(l.id)}>Add Disbursement</button>}
                        {!isViewOnly && <button className="btn btn-sm btn-primary" onClick={() => setShowMarkComplete(l.id)}>Mark Construction Complete</button>}
                      </>
                    )}
                    {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowPrepay(l.id)}>Prepayment</button>}
                    {!isViewOnly && l.rate_type === 'floating' && <button className="btn btn-sm btn-outline" onClick={() => setShowRate(l.id)}>Change Rate</button>}
                  </>
                )}
                <button className="btn btn-sm btn-outline" onClick={() => handleShowAmort(l.id)}>
                  {showAmort === l.id ? 'Hide' : 'Amortization'}
                </button>
                <button className="btn btn-sm btn-outline" onClick={() => setExpandedLoan(expandedLoan === l.id ? null : l.id)}>
                  History ({(l.emis_paid?.length || 0) + (l.prepayments?.length || 0) + (l.pre_emis_paid?.length || 0)})
                </button>
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => openEdit(l)}>Edit</button>}
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => handleRecalculate(l.id)} title="Replay all transactions with correct rates">Recalculate</button>}
                {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDelete(l.id)}>Del</button>}
              </div>

              {/* EMI + Prepayment History */}
              {expandedLoan === l.id && (
                <div style={{ marginTop: 12 }}>
                  {l.emis_paid?.length > 0 && (
                    <div style={{ marginBottom: 12 }}>
                      <strong style={{ fontSize: 13 }}>EMI History</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Month</th><th>EMI</th><th>Principal</th><th>Interest</th><th>Outstanding</th><th>Paid On</th></tr></thead>
                        <tbody>
                          {[...l.emis_paid].reverse().map(e => (
                            <tr key={e.emi_id}>
                              <td>{e.month}</td>
                              <td>{formatCurrency(e.emi_amount)}</td>
                              <td>{formatCurrency(e.principal_portion)}</td>
                              <td>{formatCurrency(e.interest_portion)}</td>
                              <td>{formatCurrency(e.outstanding_after)}</td>
                              <td>{e.paid_date ? formatDate(e.paid_date) : '-'}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  {l.prepayments?.length > 0 && (
                    <div style={{ marginBottom: 12 }}>
                      <strong style={{ fontSize: 13 }}>Prepayments</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Date</th><th>Amount</th><th>Type</th><th>Notes</th></tr></thead>
                        <tbody>
                          {l.prepayments.map(p => (
                            <tr key={p.prepayment_id}>
                              <td>{formatDate(p.date)}</td>
                              <td>{formatCurrency(p.amount)}</td>
                              <td><span style={{ padding: '2px 6px', borderRadius: 4, fontSize: 10, fontWeight: 600, background: p.type === 'foreclosure' ? '#fde8e8' : '#e8f5e9', color: p.type === 'foreclosure' ? '#c62828' : '#2e7d32' }}>{p.type.replace('_', ' ').toUpperCase()}</span></td>
                              <td>{p.notes || '-'}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  {l.rate_change_history?.length > 0 && (
                    <div>
                      <strong style={{ fontSize: 13 }}>Rate Change History</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Date</th><th>Old Rate</th><th>New Rate</th><th>New EMI</th></tr></thead>
                        <tbody>
                          {l.rate_change_history.map((r, i) => (
                            <tr key={i}>
                              <td>{formatDate(r.effective_date)}</td>
                              <td>{r.old_rate}%</td>
                              <td>{r.new_rate}%</td>
                              <td>{formatCurrency(r.new_emi)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  {l.pre_emis_paid?.length > 0 && (
                    <div style={{ marginTop: 12 }}>
                      <strong style={{ fontSize: 13 }}>Pre-EMI History (auto-generated per disbursement)</strong>
                      <table style={{ fontSize: 12, marginTop: 6 }}>
                        <thead><tr><th>Month</th><th>Tranche Amt</th><th>Rate</th><th>Days</th><th>Pre-EMI Interest</th><th>Paid On</th></tr></thead>
                        <tbody>
                          {[...l.pre_emis_paid].reverse().map(pe => (
                            <tr key={pe.pre_emi_id}>
                              <td>{pe.month}</td>
                              <td>{formatCurrency(pe.tranche_amount)}</td>
                              <td>{pe.interest_rate}%</td>
                              <td>{pe.days_charged}/{pe.days_in_month}</td>
                              <td>{formatCurrency(pe.interest_amount)}</td>
                              <td>{pe.paid_date ? formatDate(pe.paid_date) : '-'}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {/* Amortization Schedule */}
              {showAmort === l.id && amortization && (
                <div style={{ marginTop: 12 }}>
                  <strong style={{ fontSize: 13 }}>Amortization Schedule (Remaining)</strong>
                  <div style={{ marginTop: 12, marginBottom: 16 }}>
                    <ResponsiveContainer width="100%" height={250}>
                      <AreaChart data={amortization}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="month" fontSize={11} />
                        <YAxis tickFormatter={(v: number) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                        <Tooltip formatter={(v: any) => formatCurrency(v)} />
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
                    <tbody>
                      {amortization.map(a => (
                        <tr key={a.month}>
                          <td>{a.month}</td>
                          <td>{formatCurrency(a.emi)}</td>
                          <td>{formatCurrency(a.principal_portion)}</td>
                          <td>{formatCurrency(a.interest_portion)}</td>
                          <td>{formatCurrency(a.outstanding_after)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Add Loan Modal */}
      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 600 }}>
            <h2>Add Home Loan</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group"><label>Bank / Lender</label><input required value={form.bank_name} onChange={e => setForm({ ...form, bank_name: e.target.value })} /></div>
                <div className="form-group"><label>Loan Account Number</label><input value={form.loan_account_number || ''} onChange={e => setForm({ ...form, loan_account_number: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Sanctioned Amount (₹)</label><input type="number" required value={form.sanctioned_amount || ''} onChange={e => setForm({ ...form, sanctioned_amount: parseFloat(e.target.value) })} /></div>
                <div className="form-group"><label>Disbursed Amount (₹)</label><input type="number" required value={form.disbursed_amount || ''} onChange={e => setForm({ ...form, disbursed_amount: parseFloat(e.target.value) })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Interest Rate (%)</label><input type="number" step="0.01" required value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) })} /></div>
                <div className="form-group"><label>Rate Type</label>
                  <select value={form.rate_type} onChange={e => setForm({ ...form, rate_type: e.target.value as LoanRateType })}>
                    <option value="floating">Floating</option>
                    <option value="fixed">Fixed</option>
                  </select>
                </div>
                <div className="form-group"><label>Tenure (months)</label><input type="number" required value={form.tenure_months || ''} onChange={e => setForm({ ...form, tenure_months: parseInt(e.target.value) })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Disbursement Date</label><input type="date" required value={form.disbursement_date} onChange={e => setForm({ ...form, disbursement_date: e.target.value })} /></div>
                <div className="form-group"><label>EMI Start Date</label><input type="date" required value={form.emi_start_date} onChange={e => setForm({ ...form, emi_start_date: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Property Address</label><input value={form.property_address || ''} onChange={e => setForm({ ...form, property_address: e.target.value })} /></div>
                <div className="form-group"><label>Loan Purpose</label>
                  <select value={form.loan_purpose || 'purchase'} onChange={e => setForm({ ...form, loan_purpose: e.target.value })}>
                    <option value="purchase">Purchase</option>
                    <option value="construction">Construction</option>
                    <option value="renovation">Renovation</option>
                    <option value="balance_transfer">Balance Transfer</option>
                  </select>
                </div>
              </div>
              <div className="form-group"><label>Co-Borrower</label><input value={form.co_borrower || ''} onChange={e => setForm({ ...form, co_borrower: e.target.value })} /></div>
              <div className="form-group" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input type="checkbox" id="under-construction" checked={form.is_under_construction || false} onChange={e => setForm({ ...form, is_under_construction: e.target.checked })} />
                <label htmlFor="under-construction" style={{ margin: 0 }}>Under Construction (disbursements in tranches)</label>
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

      {/* Edit Loan Modal */}
      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Home Loan</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group"><label>Bank / Lender</label><input value={editForm.bank_name || ''} onChange={e => setEditForm({ ...editForm, bank_name: e.target.value })} /></div>
              <div className="form-group"><label>Loan Account Number</label><input value={editForm.loan_account_number || ''} onChange={e => setEditForm({ ...editForm, loan_account_number: e.target.value })} /></div>
              <div className="form-group"><label>Property Address</label><input value={editForm.property_address || ''} onChange={e => setEditForm({ ...editForm, property_address: e.target.value })} /></div>
              <div className="form-group"><label>Co-Borrower</label><input value={editForm.co_borrower || ''} onChange={e => setEditForm({ ...editForm, co_borrower: e.target.value })} /></div>
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

      {/* Add Disbursement Modal */}
      {showDisb && (
        <div className="modal-overlay" onClick={() => setShowDisb(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Disbursement Tranche</h2>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>A pro-rata pre-EMI (interest on the new tranche for remaining days of the month) will be auto-calculated and added to your next EMI.</p>
            <form onSubmit={handleAddDisbursement}>
              <div className="form-group"><label>Disbursement Date</label><input type="date" required value={disbForm.date} onChange={e => setDisbForm({ ...disbForm, date: e.target.value })} /></div>
              <div className="form-group"><label>Amount (₹)</label><input type="number" step="0.01" required value={disbForm.amount || ''} onChange={e => setDisbForm({ ...disbForm, amount: parseFloat(e.target.value) })} /></div>
              <div className="form-group"><label>Notes</label><textarea value={disbForm.notes || ''} onChange={e => setDisbForm({ ...disbForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowDisb(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Disbursement</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Mark Construction Complete Modal */}
      {showMarkComplete && (
        <div className="modal-overlay" onClick={() => setShowMarkComplete(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Mark Construction Complete</h2>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>All tranches disbursed? Mark construction as complete. No more disbursements will be expected and the "Add Disbursement" option will be removed.</p>
            <div className="modal-actions">
              <button type="button" className="btn btn-outline" onClick={() => setShowMarkComplete(null)}>Cancel</button>
              <button type="button" className="btn btn-primary" onClick={handleMarkComplete}>Mark Complete</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
