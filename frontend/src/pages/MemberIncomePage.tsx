import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { familyTaxApi, fyOptions, sectionLabel } from '../api/tax';
import type { FamilyTaxReport, ManualDeductions, MemberIncome, OtherIncome, Payslip, TaxMeta } from '../api/tax';
import { getProfiles } from '../api/profiles';
import type { Occupation, Profile } from '../types';
import { OCCUPATION_LABELS } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { PageSkeleton, Tabs } from '../components/ui';
import InfoTooltip from '../components/InfoTooltip';

const emptyPayslip = (month: string): Payslip => ({
  month, projected: false, employer: '', basic: 0, da: 0, hra: 0, special_allowance: 0, lta: 0, other_allowances: 0, perquisites: 0,
  bonus: 0, incentives: 0, esop_perquisite: 0, other_variable: 0, employee_pf: 0, employer_nps: 0, professional_tax: 0, tds: 0, rent_paid: 0,
});

const emptyOther = (): OtherIncome => ({
  pension: 0, family_pension: 0, business_scheme: '', business_receipts: 0, business_profit: 0, savings_interest: 0, other_interest: 0,
  dividends: 0, rental_income: 0, municipal_tax: 0, other_income: 0, stcg_equity: 0, ltcg_equity: 0, other_gains_slab: 0,
});

const emptyDeductions = (): ManualDeductions => ({
  sec_80c_other: 0, sec_80d_self: 0, sec_80d_parents: 0, parents_senior: false, sec_80e: 0, sec_80g: 0, nps_other: 0,
  lta_exempt: 0, home_loan_interest: 0, other_tds: 0, advance_tax: 0,
});

const FIXED: [keyof Payslip, string][] = [
  ['basic', 'Basic'], ['da', 'Dearness allowance'], ['hra', 'HRA received'], ['special_allowance', 'Special allowance'],
  ['lta', 'LTA'], ['other_allowances', 'Other taxable allowances'], ['perquisites', 'Perquisites'],
];
const VARIABLE: [keyof Payslip, string][] = [
  ['bonus', 'Bonus (performance / joining / retention)'], ['incentives', 'Incentives / commission'],
  ['esop_perquisite', 'ESOP / RSU perquisite'], ['other_variable', 'Other variable pay'],
];
const CONTRIB: [keyof Payslip, string][] = [
  ['employee_pf', 'Employee PF + VPF'], ['employer_nps', 'Employer NPS'], ['professional_tax', 'Professional tax'], ['tds', 'TDS deducted'],
];

const fixedOf = (p: Payslip) => FIXED.reduce((s, [k]) => s + (Number(p[k]) || 0), 0);
const variableOf = (p: Payslip) => VARIABLE.reduce((s, [k]) => s + (Number(p[k]) || 0), 0);

function fyMonths(fy: string): string[] {
  const start = parseInt(fy.slice(0, 4), 10);
  if (!start) return [];
  return Array.from({ length: 12 }, (_, i) => {
    const m = (3 + i) % 12;
    const y = start + (i >= 9 ? 1 : 0);
    return `${y}-${String(m + 1).padStart(2, '0')}`;
  });
}

const monthLabel = (m: string) => new Date(`${m}-01T00:00:00`).toLocaleDateString('en-IN', { month: 'short', year: 'numeric' });

function NumField({ label, value, onChange, tip, disabled }: { label: string; value: number; onChange: (v: number) => void; tip?: string; disabled?: boolean }) {
  return (
    <div className="form-group">
      <label>{label}{tip && <> <InfoTooltip tip={tip} /></>}</label>
      <input type="number" min="0" step="any" value={value || ''} disabled={disabled} placeholder="0"
        onChange={e => onChange(Math.max(0, parseFloat(e.target.value) || 0))} />
    </div>
  );
}

export default function MemberIncomePage() {
  const { profileId = '' } = useParams();
  const [params, setParams] = useSearchParams();
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const [meta, setMeta] = useState<TaxMeta | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [record, setRecord] = useState<MemberIncome | null>(null);
  const [report, setReport] = useState<FamilyTaxReport | null>(null);
  const [selected, setSelected] = useState<string>('');
  const [tab, setTab] = useState<'salary' | 'other' | 'deductions'>('salary');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const fy = params.get('fy') || meta?.current_fy || '';

  useEffect(() => {
    familyTaxApi.meta().then(r => setMeta(r.data)).catch(() => {});
    getProfiles().then(r => setProfile((r.data || []).find(p => p.id === profileId) || null)).catch(() => {});
  }, [profileId, viewAsUserId]);

  const loadReport = useCallback(() => {
    if (!fy) return;
    familyTaxApi.report(fy).then(r => setReport(r.data)).catch(() => {});
  }, [fy]);

  useEffect(() => {
    if (!fy) return;
    let cancelled = false;
    familyTaxApi.getIncome(profileId, fy)
      .then(r => {
        if (cancelled) return;
        const rec = r.data;
        setRecord({ ...rec, payslips: rec.payslips || [], other: { ...emptyOther(), ...rec.other }, deductions: { ...emptyDeductions(), ...rec.deductions } });
        setDirty(false);
        setError('');
      })
      .catch(err => { if (!cancelled) setError(apiErrorMessage(err, 'Failed to load income')); });
    loadReport();
    return () => { cancelled = true; };
  }, [profileId, fy, viewAsUserId, loadReport]);

  const months = useMemo(() => fyMonths(fy), [fy]);
  const byMonth = useMemo(() => new Map((record?.payslips || []).map(p => [p.month, p])), [record]);
  const member = report?.members.find(m => m.profile_id === profileId);
  const readOnly = isViewOnly;

  const patch = (fn: (r: MemberIncome) => MemberIncome) => {
    setRecord(r => (r ? fn(r) : r));
    setDirty(true);
  };
  const setPayslip = (p: Payslip) => patch(r => ({
    ...r, payslips: [...r.payslips.filter(x => x.month !== p.month), p].sort((a, b) => a.month.localeCompare(b.month)),
  }));
  const removePayslip = (month: string) => patch(r => ({ ...r, payslips: r.payslips.filter(x => x.month !== month) }));
  const addPayslip = (month: string) => {
    const prev = [...(record?.payslips || [])].filter(p => p.month < month).pop();
    const base = prev ? { ...prev, month, bonus: 0, incentives: 0, esop_perquisite: 0, other_variable: 0, projected: false } : emptyPayslip(month);
    setPayslip(base);
    setSelected(month);
  };
  const fillLater = (src: Payslip) => {
    const later = months.filter(m => m > src.month);
    patch(r => {
      const keep = r.payslips.filter(p => p.month <= src.month || !p.projected);
      const existing = new Set(keep.map(p => p.month));
      const added = later.filter(m => !existing.has(m)).map(m => ({ ...src, month: m, projected: true, bonus: 0, incentives: 0, esop_perquisite: 0, other_variable: 0 }));
      return { ...r, payslips: [...keep, ...added].sort((a, b) => a.month.localeCompare(b.month)) };
    });
    toast(`Projected fixed pay for ${later.length} later month(s)`);
  };

  const save = async () => {
    if (!record) return;
    setSaving(true);
    try {
      const r = await familyTaxApi.saveIncome(profileId, fy, {
        payslips: record.payslips, project_remaining: record.project_remaining, other: record.other, deductions: record.deductions, notes: record.notes,
      });
      setRecord({ ...r.data, payslips: r.data.payslips || [] });
      setDirty(false);
      toast('Income saved');
      loadReport();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save income'), 'error');
    } finally {
      setSaving(false);
    }
  };

  if (error && !record) {
    return (
      <div className="empty-state">
        <h3>Couldn't load income</h3>
        <p>{error}</p>
        <Link className="btn btn-primary" to="/family">Back to Family</Link>
      </div>
    );
  }
  if (!record || !fy) return <PageSkeleton label="Loading income" />;

  const sel = selected ? byMonth.get(selected) : undefined;
  const totals = record.payslips.reduce((t, p) => ({
    fixed: t.fixed + fixedOf(p), variable: t.variable + variableOf(p), tds: t.tds + (p.tds || 0), pf: t.pf + (p.employee_pf || 0),
  }), { fixed: 0, variable: 0, tds: 0, pf: 0 });
  const other = record.other;
  const ded = record.deductions;
  const setOther = (p: Partial<OtherIncome>) => patch(r => ({ ...r, other: { ...r.other, ...p } }));
  const setDed = (p: Partial<ManualDeductions>) => patch(r => ({ ...r, deductions: { ...r.deductions, ...p } }));
  const cmp = member?.comparison;

  return (
    <div>
      <div className="page-header">
        <div>
          <Link to="/family" className="text-sm text-muted">← Family</Link>
          <h1 className="mt-0">{profile?.name || 'Member'} — Income &amp; tax</h1>
          {profile && (
            <div className="text-muted text-sm">
              {[profile.occupation ? OCCUPATION_LABELS[profile.occupation as Occupation] : 'Occupation not set', profile.age ? `${profile.age} yrs` : ''].filter(Boolean).join(' · ')}
            </div>
          )}
        </div>
        <div className="flex items-center gap-2">
          <select value={fy} onChange={e => setParams({ fy: e.target.value })} aria-label="Financial year">
            {fyOptions(meta?.supported_fys, fy).map(f => <option key={f} value={f}>FY {f}</option>)}
          </select>
          {!readOnly && <button className="btn btn-primary" disabled={!dirty || saving} onClick={save}>{saving ? 'Saving…' : dirty ? 'Save' : 'Saved'}</button>}
        </div>
      </div>

      {cmp && member && (
        <div className="card-grid stat-grid mb-4">
          <div className="stat-card">
            <div className="label">Old regime</div>
            <div className="value">{formatCurrency(cmp.old.total_tax)}</div>
            <div className="text-muted text-sm">Taxable {formatCurrency(cmp.old.total_income)}</div>
          </div>
          <div className="stat-card">
            <div className="label">New regime</div>
            <div className="value">{formatCurrency(cmp.new.total_tax)}</div>
            <div className="text-muted text-sm">Taxable {formatCurrency(cmp.new.total_income)}</div>
          </div>
          <div className="stat-card" style={{ background: 'var(--success-soft)' }}>
            <div className="label">Recommended</div>
            <div className="value text-success capitalize">{cmp.recommended} regime</div>
            <div className="text-muted text-sm">Saves {formatCurrency(cmp.savings)}</div>
          </div>
          <div className="stat-card">
            <div className="label">{member.balance_payable >= 0 ? 'Still payable' : 'Refund due'} ({member.chosen_regime})</div>
            <div className={`value ${member.balance_payable > 0 ? 'negative' : ''}`}>{formatCurrency(Math.abs(member.balance_payable))}</div>
            <div className="text-muted text-sm">TDS / tax paid {formatCurrency(member.tax_paid)}</div>
          </div>
        </div>
      )}
      {dirty && <div className="card mb-4 text-sm" style={{ borderLeft: '4px solid var(--warning)' }}>Unsaved changes — save to update the tax estimate.</div>}

      <Tabs ariaLabel="Income sections" value={tab} onChange={setTab}
        tabs={[{ value: 'salary', label: 'Salary (monthly)', count: record.payslips.length }, { value: 'other', label: 'Other income' }, { value: 'deductions', label: 'Deductions & tax paid' }]} />

      {tab === 'salary' && (
        <div>
          <label className="flex items-center gap-2 mb-3 text-md">
            <input type="checkbox" disabled={readOnly} checked={record.project_remaining} onChange={e => patch(r => ({ ...r, project_remaining: e.target.checked }))} />
            Estimate months after the latest payslip using its fixed pay (variable pay isn't projected)
          </label>
          <div className="table-container mb-4">
            <table>
              <thead>
                <tr>
                  <th>Month</th>
                  <th>Status</th>
                  <th className="text-right">Fixed</th>
                  <th className="text-right">Variable</th>
                  <th className="text-right">Employee PF</th>
                  <th className="text-right">TDS</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {months.map(m => {
                  const p = byMonth.get(m);
                  return (
                    <tr key={m} style={selected === m ? { background: 'var(--primary-soft)' } : undefined}>
                      <td>{monthLabel(m)}</td>
                      <td>{p ? (p.projected ? <span className="badge badge-pending">Expected</span> : <span className="badge badge-active">Actual</span>) : <span className="text-muted text-sm">—</span>}</td>
                      <td className="text-right">{p ? formatCurrency(fixedOf(p)) : ''}</td>
                      <td className="text-right">{p ? formatCurrency(variableOf(p)) : ''}</td>
                      <td className="text-right">{p ? formatCurrency(p.employee_pf) : ''}</td>
                      <td className="text-right">{p ? formatCurrency(p.tds) : ''}</td>
                      <td className="text-right">
                        {p ? (
                          <span className="inline-flex gap-2">
                            <button type="button" className="btn btn-sm btn-outline" onClick={() => setSelected(selected === m ? '' : m)}>{selected === m ? 'Close' : readOnly ? 'View' : 'Edit'}</button>
                            {!readOnly && <button type="button" className="btn btn-sm btn-ghost text-danger" onClick={() => { removePayslip(m); if (selected === m) setSelected(''); }}>Remove</button>}
                          </span>
                        ) : !readOnly && (
                          <button type="button" className="btn btn-sm btn-ghost" onClick={() => addPayslip(m)}>Add payslip</button>
                        )}
                      </td>
                    </tr>
                  );
                })}
                <tr className="fw-semibold">
                  <td colSpan={2}>Entered total</td>
                  <td className="text-right">{formatCurrency(totals.fixed)}</td>
                  <td className="text-right">{formatCurrency(totals.variable)}</td>
                  <td className="text-right">{formatCurrency(totals.pf)}</td>
                  <td className="text-right">{formatCurrency(totals.tds)}</td>
                  <td />
                </tr>
              </tbody>
            </table>
          </div>
          {member && member.income.projected_months > 0 && (
            <p className="text-muted text-sm mb-4">
              Annual estimate includes {member.income.projected_months} projected month(s): fixed {formatCurrency(member.income.salary_fixed)}, variable {formatCurrency(member.income.salary_variable)}.
            </p>
          )}

          {sel && (
            <div className="card mb-4">
              <div className="flex items-center gap-2 mb-3">
                <h3 className="text-lg fw-semibold m-0">{monthLabel(sel.month)}</h3>
                <label className="flex items-center gap-2 text-md ml-auto">
                  <input type="checkbox" disabled={readOnly} checked={sel.projected} onChange={e => setPayslip({ ...sel, projected: e.target.checked })} />
                  Expected (not yet paid)
                </label>
              </div>
              <div className="form-group">
                <label>Employer</label>
                <input value={sel.employer || ''} disabled={readOnly} onChange={e => setPayslip({ ...sel, employer: e.target.value })} />
              </div>
              <h4 className="fw-semibold mb-2">Fixed pay</h4>
              <div className="form-row-3">
                {FIXED.map(([k, label]) => <NumField key={k} label={label} value={sel[k] as number} disabled={readOnly} onChange={v => setPayslip({ ...sel, [k]: v })} />)}
              </div>
              <h4 className="fw-semibold mb-2">Variable pay</h4>
              <div className="form-row-3">
                {VARIABLE.map(([k, label]) => <NumField key={k} label={label} value={sel[k] as number} disabled={readOnly} onChange={v => setPayslip({ ...sel, [k]: v })} />)}
              </div>
              <h4 className="fw-semibold mb-2">Contributions &amp; deductions on the payslip</h4>
              <div className="form-row-3">
                {CONTRIB.map(([k, label]) => <NumField key={k} label={label} value={sel[k] as number} disabled={readOnly} onChange={v => setPayslip({ ...sel, [k]: v })} />)}
                <NumField label="Rent paid this month" value={sel.rent_paid} disabled={readOnly} onChange={v => setPayslip({ ...sel, rent_paid: v })}
                  tip="Used for the HRA exemption (old regime), or 80GG if you get no HRA." />
              </div>
              <p className="text-muted text-sm">
                If EPF or employer NPS are also tracked in the PF / NPS pages for this member, the larger figure is used, so nothing is counted twice.
              </p>
              {!readOnly && (
                <div className="flex gap-2">
                  <button type="button" className="btn btn-sm btn-outline" onClick={() => fillLater(sel)}>Use this fixed pay for later months</button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'other' && (
        <div className="card">
          <p className="text-muted text-sm mb-3">
            Interest, dividends, rent and capital gains from investments tracked in the app and assigned to this member are added automatically.
            Enter only income from outside the app here (annual amounts).
          </p>
          <div className="form-row-3">
            <NumField label="Pension (annual)" value={other.pension} disabled={readOnly} onChange={v => setOther({ pension: v })} />
            <NumField label="Family pension (annual)" value={other.family_pension} disabled={readOnly} onChange={v => setOther({ family_pension: v })} />
            <NumField label="Savings account interest" value={other.savings_interest} disabled={readOnly} onChange={v => setOther({ savings_interest: v })} tip="80TTA (₹10K) / 80TTB for seniors under the old regime." />
          </div>
          <div className="form-row-3">
            <div className="form-group">
              <label>Business / profession</label>
              <select value={other.business_scheme} disabled={readOnly} onChange={e => setOther({ business_scheme: e.target.value as OtherIncome['business_scheme'] })}>
                <option value="">None</option>
                <option value="44ADA">Presumptive — professional (44ADA, 50%)</option>
                <option value="44AD">Presumptive — business (44AD, 6%)</option>
                <option value="actual">Actual profit (books of account)</option>
              </select>
            </div>
            {(other.business_scheme === '44AD' || other.business_scheme === '44ADA') && (
              <NumField label="Gross receipts / turnover" value={other.business_receipts} disabled={readOnly} onChange={v => setOther({ business_receipts: v })} />
            )}
            {other.business_scheme === 'actual' && (
              <NumField label="Net profit" value={other.business_profit} disabled={readOnly} onChange={v => setOther({ business_profit: v })} />
            )}
          </div>
          <div className="form-row-3">
            <NumField label="Other interest (FD/RD not in app)" value={other.other_interest} disabled={readOnly} onChange={v => setOther({ other_interest: v })} />
            <NumField label="Dividends (not in app)" value={other.dividends} disabled={readOnly} onChange={v => setOther({ dividends: v })} />
            <NumField label="Other income" value={other.other_income} disabled={readOnly} onChange={v => setOther({ other_income: v })} tip="Freelance (not business), gifts over ₹50K from non-relatives, etc." />
          </div>
          <div className="form-row-3">
            <NumField label="Rent received (not in app)" value={other.rental_income} disabled={readOnly} onChange={v => setOther({ rental_income: v })} />
            <NumField label="Municipal tax paid" value={other.municipal_tax} disabled={readOnly} onChange={v => setOther({ municipal_tax: v })} />
          </div>
          <h4 className="fw-semibold mb-2">Capital gains outside the app</h4>
          <div className="form-row-3">
            <NumField label="Equity STCG (20%)" value={other.stcg_equity} disabled={readOnly} onChange={v => setOther({ stcg_equity: v })} />
            <NumField label="Equity LTCG (12.5% over ₹1.25L)" value={other.ltcg_equity} disabled={readOnly} onChange={v => setOther({ ltcg_equity: v })} />
            <NumField label="Gains taxed at slab (debt MF…)" value={other.other_gains_slab} disabled={readOnly} onChange={v => setOther({ other_gains_slab: v })} />
          </div>
        </div>
      )}

      {tab === 'deductions' && (
        <div>
          <div className="card mb-4">
            <p className="text-muted text-sm mb-3">
              EPF, PPF, ELSS, life &amp; health insurance, NPS and home loans tracked in the app count automatically for the member who owns or paid for them.
              Add anything else below (annual amounts; most apply to the old regime only).
            </p>
            <div className="form-row-3">
              <NumField label="Other 80C" value={ded.sec_80c_other} disabled={readOnly} onChange={v => setDed({ sec_80c_other: v })} tip="Children's tuition fees, SSY, NSC, 5-yr FD, stamp duty, LIC not in app." />
              <NumField label="80D — self & family (not in app)" value={ded.sec_80d_self} disabled={readOnly} onChange={v => setDed({ sec_80d_self: v })} tip="Includes preventive health check-up up to ₹5K." />
              <NumField label="80D — parents" value={ded.sec_80d_parents} disabled={readOnly} onChange={v => setDed({ sec_80d_parents: v })} />
            </div>
            <label className="flex items-center gap-2 mb-3 text-md">
              <input type="checkbox" disabled={readOnly} checked={ded.parents_senior} onChange={e => setDed({ parents_senior: e.target.checked })} />
              Parents are senior citizens (₹50K 80D limit)
            </label>
            <div className="form-row-3">
              <NumField label="NPS Tier-1 (not in app)" value={ded.nps_other} disabled={readOnly} onChange={v => setDed({ nps_other: v })} tip="80CCD(1) and the extra ₹50K under 80CCD(1B)." />
              <NumField label="80E education loan interest" value={ded.sec_80e} disabled={readOnly} onChange={v => setDed({ sec_80e: v })} />
              <NumField label="80G donations (eligible amount)" value={ded.sec_80g} disabled={readOnly} onChange={v => setDed({ sec_80g: v })} />
            </div>
            <div className="form-row-3">
              <NumField label="LTA exemption claimed" value={ded.lta_exempt} disabled={readOnly} onChange={v => setDed({ lta_exempt: v })} />
              <NumField label="Home loan interest (not in app)" value={ded.home_loan_interest} disabled={readOnly} onChange={v => setDed({ home_loan_interest: v })} tip="Self-occupied property; capped at ₹2L in the old regime." />
            </div>
          </div>
          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Tax already paid</h3>
            <div className="form-row-3">
              <NumField label="Other TDS (FD, rent… from Form 26AS)" value={ded.other_tds} disabled={readOnly} onChange={v => setDed({ other_tds: v })} />
              <NumField label="Advance / self-assessment tax" value={ded.advance_tax} disabled={readOnly} onChange={v => setDed({ advance_tax: v })} />
            </div>
          </div>
        </div>
      )}

      {member && (
        <div className="card mt-4">
          <h3 className="text-lg fw-semibold mb-3">How the estimate is built (FY {report?.fy})</h3>
          {member.warnings.length > 0 && (
            <ul className="text-sm mb-3" style={{ paddingLeft: 18 }}>
              {member.warnings.map((w, i) => <li key={i} className="text-warning">{w}</li>)}
            </ul>
          )}
          {cmp && (
            <div className="payout-list mb-3">
              {cmp[member.chosen_regime].deductions.map((d, i) => (
                <div className="payout-item" key={i}>
                  <span>{sectionLabel(d.code, d.new_section)} — {d.label}</span>
                  <span>{formatCurrency(d.allowed)}{d.claimed > d.allowed ? <span className="text-muted text-xs"> of {formatCurrency(d.claimed)}</span> : null}</span>
                </div>
              ))}
            </div>
          )}
          {member.lines.length > 0 && (
            <>
              <h4 className="fw-semibold mb-2">From tracked investments</h4>
              <div className="payout-list">
                {member.lines.map((l, i) => (
                  <div className="payout-item" key={i}>
                    <span>
                      {l.name || l.asset_type} <span className="text-muted text-xs">({l.kind.replace(/_/g, ' ')}{l.share_pct && l.share_pct < 100 ? `, ${l.share_pct}% share` : ''})</span>
                      {l.clubbed && <span className="badge badge-pending ml-1">clubbed</span>}
                      {l.clubbing_ignored && <span className="badge badge-overdue ml-1" title="Clubbing is turned off in Tax Center">not clubbed</span>}
                    </span>
                    <span>{formatCurrency(l.amount)}</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
