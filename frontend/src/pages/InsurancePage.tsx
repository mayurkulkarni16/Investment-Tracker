import { useCallback, useEffect, useState } from 'react';
import type React from 'react';
import {
  getInsurancePolicies, getInsuranceSummary, createInsurancePolicy, updateInsurancePolicy, deleteInsurancePolicy,
} from '../api/insurance';
import type {
  InsurancePolicy, InsurancePolicyRequest, InsuranceSummary, PolicyType, PremiumFrequency, PolicyStatus,
} from '../api/insurance';
import { formatCurrency, formatDate, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { apiErrorMessage } from '../utils/errors';
import { Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

const POLICY_TYPES: { value: PolicyType; label: string; color: string }[] = [
  { value: 'term', label: 'Term', color: '#2563eb' },
  { value: 'health', label: 'Health', color: '#16a34a' },
  { value: 'life', label: 'Life', color: '#7c3aed' },
  { value: 'ulip', label: 'ULIP', color: '#db2777' },
  { value: 'vehicle', label: 'Vehicle', color: '#ea580c' },
  { value: 'home', label: 'Home', color: '#0891b2' },
  { value: 'personal_accident', label: 'Personal Accident', color: '#ca8a04' },
  { value: 'other', label: 'Other', color: '#64748b' },
];

const FREQUENCIES: { value: PremiumFrequency; label: string }[] = [
  { value: 'monthly', label: 'Monthly' },
  { value: 'quarterly', label: 'Quarterly' },
  { value: 'half_yearly', label: 'Half-yearly' },
  { value: 'yearly', label: 'Yearly' },
  { value: 'single', label: 'Single' },
];

const STATUSES: PolicyStatus[] = ['active', 'lapsed', 'matured', 'surrendered', 'expired'];

const STATUS_COLORS: Record<PolicyStatus, string> = {
  active: '#16a34a',
  lapsed: '#dc2626',
  matured: '#2563eb',
  surrendered: '#64748b',
  expired: '#94a3b8',
};

const typeMeta = (t: string) => POLICY_TYPES.find(p => p.value === t) || POLICY_TYPES[POLICY_TYPES.length - 1];
const freqLabel = (f: string) => FREQUENCIES.find(x => x.value === f)?.label || f;
const hasCashValue = (t: PolicyType) => t === 'life' || t === 'ulip';

const emptyForm = (): InsurancePolicyRequest => ({
  policy_type: 'term', insurer: '', policy_name: '', policy_number: '', insured_persons: '',
  sum_assured: 0, premium_amount: 0, premium_frequency: 'yearly',
  start_date: '', end_date: '', premium_end_date: '', current_value: 0,
  nominee: '', status: 'active', notes: '',
});

const pill = (text: string, color: string) => (
  <span className="badge nowrap" style={{ background: `${color}22`, color, border: `1px solid ${color}55` }}>{text}</span>
);

export default function InsurancePage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [policies, setPolicies] = useState<InsurancePolicy[]>([]);
  const [summary, setSummary] = useState<InsuranceSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [incomeInput, setIncomeInput] = useState('');
  const [appliedIncome, setAppliedIncome] = useState<number | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<InsurancePolicy | null>(null);
  const [form, setForm] = useState<InsurancePolicyRequest>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | PolicyStatus>('all');

  const loadSummary = useCallback((income?: number) => {
    return getInsuranceSummary(income).then(r => setSummary(r.data)).catch(() => setSummary(null));
  }, []);

  const loadPolicies = useCallback(() => {
    return getInsurancePolicies().then(r => setPolicies(r.data || [])).catch(() => toast('Failed to load policies', 'error'));
  }, [toast]);

  useEffect(() => {
    Promise.all([loadPolicies(), loadSummary(appliedIncome)]).finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewAsUserId]);

  const reload = () => { loadPolicies(); loadSummary(appliedIncome); };

  const applyIncome = (e: React.FormEvent) => {
    e.preventDefault();
    const v = parseFloat(incomeInput);
    const income = Number.isFinite(v) && v > 0 ? v : undefined;
    setAppliedIncome(income);
    loadSummary(income);
  };

  const clearIncome = () => {
    setIncomeInput('');
    setAppliedIncome(undefined);
    loadSummary(undefined);
  };

  const openAdd = () => { setEditing(null); setForm(emptyForm()); setModalOpen(true); };

  const openEdit = (p: InsurancePolicy) => {
    setEditing(p);
    setForm({
      policy_type: p.policy_type,
      insurer: p.insurer,
      policy_name: p.policy_name || '',
      policy_number: p.policy_number || '',
      insured_persons: p.insured_persons || '',
      health_cover_for: p.health_cover_for || 'self_family',
      insured_senior: !!p.insured_senior,
      sum_assured: p.sum_assured,
      premium_amount: p.premium_amount,
      premium_frequency: p.premium_frequency,
      start_date: toInputDate(p.start_date),
      end_date: toInputDate(p.end_date),
      premium_end_date: toInputDate(p.premium_end_date),
      current_value: p.current_value,
      nominee: p.nominee || '',
      status: p.status,
      notes: p.notes || '',
    });
    setModalOpen(true);
  };

  const closeModal = () => { setModalOpen(false); setEditing(null); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload: InsurancePolicyRequest = {
      ...form,
      current_value: hasCashValue(form.policy_type) ? form.current_value || 0 : 0,
      premium_end_date: form.premium_frequency === 'single' ? '' : form.premium_end_date,
      health_cover_for: form.policy_type === 'health' ? form.health_cover_for || 'self_family' : '',
      insured_senior: form.policy_type === 'health' ? !!form.insured_senior : false,
    };
    setSaving(true);
    try {
      if (editing) { await updateInsurancePolicy(editing.id, payload); toast('Policy updated'); }
      else { await createInsurancePolicy(payload); toast('Policy added'); }
      closeModal();
      reload();
    } catch (err) {
      toast(apiErrorMessage(err, editing ? 'Failed to update policy' : 'Failed to add policy'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (p: InsurancePolicy) => {
    const name = p.policy_name || p.insurer;
    if (!await confirm({ message: `Delete policy "${name}"?`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteInsurancePolicy(p.id); toast('Policy deleted'); reload(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete policy'), 'error'); }
  };

  const num = (v: string) => { const n = parseFloat(v); return Number.isFinite(n) ? n : 0; };

  if (loading) return <PageSkeleton label="Loading" />;

  const upcoming = summary?.upcoming_premiums || [];
  const suggestions = summary?.suggestions || [];
  const incomeSourceLabel = (src?: string) => {
    if (src === 'ledger') return 'from last 12 months of ledger income';
    if (src === 'query') return 'entered manually';
    return 'not available — enter your annual income';
  };

  const filtered = policies.filter(p => {
    const q = search.trim().toLowerCase();
    const matchSearch = !q || [p.insurer, p.policy_name, p.policy_number, p.insured_persons].some(s => (s || '').toLowerCase().includes(q));
    const matchStatus = statusFilter === 'all' || p.status === statusFilter;
    return matchSearch && matchStatus;
  });

  const coverRow = (label: string, actual: number, recommended: number, gap: number) => {
    const pct = recommended > 0 ? Math.min(100, (actual / recommended) * 100) : (actual > 0 ? 100 : 0);
    const ok = gap <= 0;
    return (
      <div className="mb-4">
        <div className="flex justify-between flex-wrap gap-2 mb-2">
          <strong>{label}</strong>
          <span className="text-muted">
            Actual {formatCurrency(actual)} / Recommended {formatCurrency(recommended)}
          </span>
        </div>
        <div style={{ height: 10, borderRadius: 5, background: 'var(--border, #e2e8f0)', overflow: 'hidden' }}>
          <div style={{ width: `${pct}%`, height: '100%', background: ok ? '#16a34a' : '#f59e0b' }} />
        </div>
        <div className={`${ok ? 'text-success' : 'text-danger'} mt-1 text-md`}>
          {ok ? 'Adequately covered' : `Gap: ${formatCurrency(gap)}`}
        </div>
      </div>
    );
  };

  const isSingle = form.premium_frequency === 'single';

  return (
    <div>
      <div className="page-header">
        <h1>Insurance</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Policy</button>}
      </div>

      {summary && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card"><div className="label">Life Cover</div><div className="value">{formatCurrency(summary.total_life_cover)}</div></div>
          <div className="stat-card"><div className="label">Health Cover</div><div className="value">{formatCurrency(summary.total_health_cover)}</div></div>
          <div className="stat-card"><div className="label">Annual Premium</div><div className="value">{formatCurrency(summary.annual_premium)}</div></div>
          <div className="stat-card"><div className="label">80C Premium (this FY)</div><div className="value">{formatCurrency(summary.premium_80c_this_fy)}</div></div>
          <div className="stat-card"><div className="label">80D Premium (this FY)</div><div className="value">{formatCurrency(summary.premium_80d_this_fy)}</div></div>
          <div className="stat-card"><div className="label">Cash Value</div><div className="value">{formatCurrency(summary.cash_value)}</div></div>
        </div>
      )}

      {summary && (
        <div className="card-grid stat-grid mb-5">
          <div className="card">
            <h3 className="mb-3">Cover adequacy</h3>
            <form className="flex gap-2 items-end flex-wrap mb-2" onSubmit={applyIncome}>
              <div className="form-group mb-0 flex-1" style={{ minWidth: 160 }}>
                <label>Annual income (₹)</label>
                <input type="number" min="0" step="1000" placeholder={summary.annual_income > 0 ? String(Math.round(summary.annual_income)) : 'e.g. 1200000'}
                  value={incomeInput} onChange={e => setIncomeInput(e.target.value)} />
              </div>
              <button type="submit" className="btn btn-sm btn-primary">Recalculate</button>
              {appliedIncome !== undefined && <button type="button" className="btn btn-sm btn-outline" onClick={clearIncome}>Reset</button>}
            </form>
            <div className="text-muted text-md mb-4">
              Using income {formatCurrency(summary.annual_income)} ({incomeSourceLabel(summary.annual_income_source)})
              {summary.outstanding_loans > 0 && <> · Outstanding loans {formatCurrency(summary.outstanding_loans)}</>}
            </div>
            {coverRow('Life cover', summary.total_life_cover, summary.recommended_life_cover, summary.life_cover_gap)}
            {coverRow('Health cover', summary.total_health_cover, summary.recommended_health_cover, summary.health_cover_gap)}
            {suggestions.length > 0 && (
              <>
                <h4 style={{ margin: '12px 0 6px' }}>Suggestions</h4>
                <ul className="m-0" style={{ paddingLeft: 20 }}>
                  {suggestions.map((s, i) => <li className="mb-1" key={i}>{s}</li>)}
                </ul>
              </>
            )}
          </div>

          <div className="card">
            <h3 className="mb-3">Upcoming premiums (next 60 days)</h3>
            {upcoming.length === 0 ? (
              <div className="text-muted">No premiums due in the next 60 days.</div>
            ) : (
              <div className="table-container">
                <table>
                  <thead><tr><th>Policy</th><th>Due</th><th className="num">Amount</th></tr></thead>
                  <tbody>
                    {upcoming.map(u => (
                      <tr key={`${u.policy_id}-${u.date}`}>
                        <td>
                          <div>{u.policy_name || u.insurer}</div>
                          {u.policy_name && <div className="text-muted text-sm">{u.insurer}</div>}
                        </td>
                        <td>
                          <div>{formatDate(u.date)}</div>
                          <div className={`${u.days_left <= 7 ? 'text-danger' : 'text-muted'} text-sm`}>
                            {u.days_left <= 0 ? 'Due today' : `in ${u.days_left} day${u.days_left === 1 ? '' : 's'}`}
                          </div>
                        </td>
                        <td className="num">{formatCurrency(u.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr style={{ fontWeight: 'bold' }}>
                      <td colSpan={2}>Total</td>
                      <td>{formatCurrency(upcoming.reduce((s, u) => s + u.amount, 0))}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {policies.length === 0 ? (
        <div className="empty-state">
          <h3>No insurance policies yet</h3>
          <p>Add your term, health, life and other policies to track cover and premiums.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Policy</button>}
        </div>
      ) : (
        <>
          <div className="flex gap-3 mb-4 items-center flex-wrap">
            <div className="search-bar flex-1 mb-0" style={{ minWidth: 200 }}>
              <input type="search" placeholder="Search by insurer, policy name/number, insured..." value={search} onChange={e => setSearch(e.target.value)} />
            </div>
            <div className="flex gap-1 flex-wrap">
              {(['all', ...STATUSES] as const).map(s => (
                <button key={s} className={`chip${statusFilter === s ? ' active' : ''}`} aria-pressed={statusFilter === s} onClick={() => setStatusFilter(s)}>
                  {s.charAt(0).toUpperCase() + s.slice(1)}
                </button>
              ))}
            </div>
          </div>
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Insurer / Policy</th>
                  <th>Sum Assured</th>
                  <th>Premium</th>
                  <th>Next Premium</th>
                  <th>End Date</th>
                  <th>Tax</th>
                  <th>Status</th>
                  {!isViewOnly && <th>Actions</th>}
                </tr>
              </thead>
              <tbody>
                {filtered.map(p => {
                  const meta = typeMeta(p.policy_type);
                  return (
                    <tr key={p.id}>
                      <td>{pill(meta.label, meta.color)}</td>
                      <td>
                        <div className="fw-medium">{p.insurer}</div>
                        <div className="text-muted text-sm">
                          {[p.policy_name, p.policy_number && `#${p.policy_number}`].filter(Boolean).join(' · ') || '-'}
                        </div>
                        {p.insured_persons && <div className="text-muted text-sm">Insured: {p.insured_persons}</div>}
                      </td>
                      <td>
                        <div>{formatCurrency(p.sum_assured)}</div>
                        {hasCashValue(p.policy_type) && p.current_value > 0 && (
                          <div className="text-muted text-sm">Value {formatCurrency(p.current_value)}</div>
                        )}
                      </td>
                      <td>
                        <div>{formatCurrency(p.premium_amount)} <span className="text-muted text-sm">{freqLabel(p.premium_frequency)}</span></div>
                        {p.premium_frequency !== 'single' && p.premium_frequency !== 'yearly' && (
                          <div className="text-muted text-sm">{formatCurrency(p.annual_premium)}/yr</div>
                        )}
                      </td>
                      <td>{p.next_premium_date ? formatDate(p.next_premium_date) : '-'}</td>
                      <td>
                        <div>{p.end_date ? formatDate(p.end_date) : '-'}</div>
                        {p.days_to_expiry !== undefined && p.days_to_expiry !== null && (
                          <div className={`${p.days_to_expiry < 0 ? 'text-danger' : p.days_to_expiry <= 60 ? 'text-danger' : 'text-muted'} text-sm`}>
                            {p.days_to_expiry < 0 ? `Expired ${-p.days_to_expiry}d ago` : `${p.days_to_expiry}d left`}
                          </div>
                        )}
                      </td>
                      <td>{p.tax_section ? pill(p.tax_section, p.tax_section === '80D' ? '#16a34a' : '#2563eb') : '-'}</td>
                      <td>{pill(p.status, STATUS_COLORS[p.status] || '#64748b')}</td>
                      {!isViewOnly && (
                        <td className="cell-actions">
                          <RowActions
                            inline={1}
                            label={`Actions for ${p.policy_name || p.insurer}`}
                            actions={[
                              { label: 'Edit', icon: Pencil, onClick: () => openEdit(p) },
                              { label: 'Delete', icon: Trash2, onClick: () => handleDelete(p), danger: true },
                            ]}
                          />
                        </td>
                      )}
                    </tr>
                  );
                })}
                {filtered.length === 0 && (
                  <tr><td colSpan={isViewOnly ? 8 : 9} className="text-center text-muted">No policies match your filters.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {modalOpen && !isViewOnly && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing ? 'Edit Policy' : 'Add Policy'}</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Policy Type</label>
                  <select value={form.policy_type} onChange={e => setForm({ ...form, policy_type: e.target.value as PolicyType })}>
                    {POLICY_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                  </select>
                </div>
                <div className="form-group">
                  <label>Insurer</label>
                  <input required value={form.insurer} onChange={e => setForm({ ...form, insurer: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Policy Name</label>
                  <input value={form.policy_name || ''} onChange={e => setForm({ ...form, policy_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Policy Number</label>
                  <input value={form.policy_number || ''} onChange={e => setForm({ ...form, policy_number: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>{form.policy_type === 'vehicle' ? 'IDV / Sum Insured (₹)' : 'Sum Assured (₹)'}</label>
                  <input type="number" min="0" step="1" required value={form.sum_assured || ''} onChange={e => setForm({ ...form, sum_assured: num(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Insured Persons</label>
                  <input placeholder="e.g. Self, Spouse" value={form.insured_persons || ''} onChange={e => setForm({ ...form, insured_persons: e.target.value })} />
                </div>
              </div>
              {form.policy_type === 'health' && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Covers (for 80D)</label>
                    <select value={form.health_cover_for || 'self_family'} onChange={e => setForm({ ...form, health_cover_for: e.target.value as 'self_family' | 'parents' })}>
                      <option value="self_family">Self, spouse &amp; children</option>
                      <option value="parents">Parents</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label className="flex items-center gap-2" style={{ marginTop: 28 }}>
                      <input type="checkbox" checked={!!form.insured_senior} onChange={e => setForm({ ...form, insured_senior: e.target.checked })} />
                      A covered person is 60 or older (₹50K limit)
                    </label>
                  </div>
                </div>
              )}
              <div className="form-row">
                <div className="form-group">
                  <label>Premium Amount (₹)</label>
                  <input type="number" min="0" step="0.01" required value={form.premium_amount || ''} onChange={e => setForm({ ...form, premium_amount: num(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Premium Frequency</label>
                  <select value={form.premium_frequency} onChange={e => setForm({ ...form, premium_frequency: e.target.value as PremiumFrequency })}>
                    {FREQUENCIES.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
                  </select>
                </div>
              </div>
              <div className={isSingle ? 'form-row' : 'form-row-3'}>
                <div className="form-group">
                  <label>Start Date</label>
                  <input type="date" required value={form.start_date} onChange={e => setForm({ ...form, start_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>{hasCashValue(form.policy_type) ? 'Maturity / End Date' : 'Cover End Date'}</label>
                  <input type="date" min={form.start_date || undefined} value={form.end_date || ''} onChange={e => setForm({ ...form, end_date: e.target.value })} />
                </div>
                {!isSingle && (
                  <div className="form-group">
                    <label>Premium End Date <span className="text-muted">(optional)</span></label>
                    <input type="date" min={form.start_date || undefined} value={form.premium_end_date || ''} onChange={e => setForm({ ...form, premium_end_date: e.target.value })} />
                  </div>
                )}
              </div>
              <div className={hasCashValue(form.policy_type) ? 'form-row-3' : 'form-row'}>
                {hasCashValue(form.policy_type) && (
                  <div className="form-group">
                    <label>Current Value (₹) <span className="text-muted">(surrender/fund)</span></label>
                    <input type="number" min="0" step="0.01" value={form.current_value || ''} onChange={e => setForm({ ...form, current_value: num(e.target.value) })} />
                  </div>
                )}
                <div className="form-group">
                  <label>Nominee</label>
                  <input value={form.nominee || ''} onChange={e => setForm({ ...form, nominee: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Status</label>
                  <select value={form.status || 'active'} onChange={e => setForm({ ...form, status: e.target.value as PolicyStatus })}>
                    {STATUSES.map(s => <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={closeModal}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : editing ? 'Save Changes' : 'Add Policy'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
