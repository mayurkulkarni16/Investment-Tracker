import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { getTaxSummary, getCapitalGains, familyTaxApi, fyOptions, sectionLabel } from '../api/tax';
import type { FamilyTaxReport, MemberTaxReport, PlacementResult, RegimeResult, TaxMeta, TaxSettings } from '../api/tax';
import type { TaxSummary, CapitalGainsSummary, Occupation } from '../types';
import { OCCUPATION_LABELS } from '../types';
import { formatCurrency } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { Tabs, PageSkeleton } from '../components/ui';
import InfoTooltip from '../components/InfoTooltip';

type Tab = 'family' | 'advisor' | 'deductions' | 'gains';

const DEFAULT_RETURNS: Record<string, number> = {
  fd: 7, debt_mf: 7, bond: 8.5, scss: 8.2, equity_mf: 12, elss: 12, ppf: 7.1, ssy: 8.2, nps: 10, savings: 3,
};

const CATEGORY_LABEL: Record<string, string> = {
  regime: 'Regime', deduction: 'Deduction', placement: 'Placement', clubbing: 'Clubbing', data: 'Data', capital_gains: 'Capital gains',
};

function BreakdownTable({ old, nw, chosen }: { old: RegimeResult; nw: RegimeResult; chosen: string }) {
  const rows: [string, (r: RegimeResult) => number][] = [
    ['Gross salary / pension', r => r.gross_salary],
    ['Standard deduction', r => -r.standard_deduction],
    ['HRA & LTA exemptions', r => -r.exemptions],
    ['Professional tax', r => -r.professional_tax],
    ['House property', r => r.house_property],
    ['Business / profession', r => r.business_income],
    ['Other sources', r => r.other_sources],
    ['Capital gains at special rates', r => r.special_income],
    ['Gross total income', r => r.gross_total_income],
    ['Chapter VI-A deductions', r => -r.total_deductions],
    ['Total income', r => r.total_income],
    ['Tax at slab rates', r => r.slab_tax],
    ['Tax on capital gains', r => r.special_tax],
    ['87A rebate', r => -r.rebate_87a],
    ['Surcharge', r => r.surcharge],
    ['Health & education cess (4%)', r => r.cess],
    ['Total tax', r => r.total_tax],
  ];
  return (
    <div className="table-container flat mb-3">
      <table>
        <thead>
          <tr><th /><th className="text-right">Old regime{chosen === 'old' ? ' ✓' : ''}</th><th className="text-right">New regime{chosen === 'new' ? ' ✓' : ''}</th></tr>
        </thead>
        <tbody>
          {rows.map(([label, f]) => {
            const a = f(old), b = f(nw);
            if (a === 0 && b === 0 && label !== 'Total tax') return null;
            const bold = label === 'Total tax' || label === 'Total income' || label === 'Gross total income';
            const cell = (v: number) => (v === 0 && label !== 'Total tax' ? '—' : formatCurrency(v));
            return (
              <tr key={label} className={bold ? 'fw-semibold' : undefined}>
                <td>{label}</td>
                <td className="text-right">{cell(a)}</td>
                <td className="text-right">{cell(b)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function MemberRegimeCard({ m }: { m: MemberTaxReport }) {
  const [open, setOpen] = useState(false);
  const c = m.comparison;
  const max = Math.max(c.old.total_tax, c.new.total_tax, 1);
  const h = c.headroom;
  const bar = (label: string, value: number, best: boolean) => (
    <div className="mb-2">
      <div className="flex justify-between text-sm">
        <span>{label}{best && <span className="badge badge-active ml-1">Best</span>}</span>
        <span className="fw-semibold">{formatCurrency(value)}</span>
      </div>
      <div style={{ height: 6, background: 'var(--border, #e0e0e0)', borderRadius: 3 }}>
        <div style={{ width: `${(value / max) * 100}%`, height: 6, borderRadius: 3, background: best ? 'var(--success, #16a34a)' : 'var(--text-muted, #9aa0a6)' }} />
      </div>
    </div>
  );
  return (
    <div className="card" style={{ borderTop: `4px solid ${m.color || 'var(--primary)'}` }}>
      <div className="flex items-center gap-2 mb-1">
        <strong className="text-lg">{m.name}</strong>
        <span className="badge text-muted capitalize" style={{ background: 'var(--bg-secondary, #f1f3f4)' }}>{m.relationship}</span>
        <Link className="ml-auto text-sm" to={`/family/${m.profile_id}/income?fy=${c.fy}`}>Edit income →</Link>
      </div>
      <div className="text-muted text-sm mb-3">
        {[m.occupation ? OCCUPATION_LABELS[m.occupation as Occupation] : 'Occupation not set', m.age ? `${m.age} yrs` : '', !m.resident ? 'NRI' : '',
          `Income ${formatCurrency(m.income.total)}`].filter(Boolean).join(' · ')}
      </div>
      {bar('Old regime', c.old.total_tax, c.recommended === 'old')}
      {bar('New regime', c.new.total_tax, c.recommended === 'new')}
      <div className="text-md mt-2">
        {c.savings > 0
          ? <>The <strong className="capitalize">{c.recommended}</strong> regime saves <strong className="text-success">{formatCurrency(c.savings)}</strong>.</>
          : <>Both regimes cost the same; the new regime is the default.</>}
        {m.preferred_regime && m.preferred_regime !== c.recommended && m.extra_tax_vs_best > 0 && (
          <div className="text-danger text-sm">Declared {m.preferred_regime} regime costs {formatCurrency(m.extra_tax_vs_best)} extra.</div>
        )}
        {c.recommended === 'new' && c.break_even_deductions > 0 && (
          <div className="text-muted text-sm">Old regime needs {formatCurrency(c.break_even_deductions)} more deductions to break even.</div>
        )}
        {c.recommended === 'new' && c.break_even_deductions < 0 && (
          <div className="text-muted text-sm">No realistic level of deductions makes the old regime cheaper.</div>
        )}
      </div>
      <div className="flex gap-4 text-sm mt-2 flex-wrap">
        <span>Marginal rate: {(c[m.chosen_regime].marginal_rate || 0).toFixed(1)}%</span>
        <span>Tax paid: {formatCurrency(m.tax_paid)}</span>
        <span className={m.balance_payable > 0 ? 'text-danger' : 'text-success'}>
          {m.balance_payable >= 0 ? 'Payable' : 'Refund'}: {formatCurrency(Math.abs(m.balance_payable))}
        </span>
      </div>
      {(h.sec_80c > 0 || h.sec_80ccd1b > 0 || h.sec_80d_self > 0) && (c.recommended === 'old' || c.break_even_deductions > 0) && (
        <div className="text-sm text-muted mt-2">
          Room left (old regime): 80C {formatCurrency(h.sec_80c)} · NPS 80CCD(1B) {formatCurrency(h.sec_80ccd1b)} · 80D {formatCurrency(h.sec_80d_self)}
        </div>
      )}
      {m.warnings.length > 0 && (
        <ul className="text-sm mt-2 mb-0" style={{ paddingLeft: 18 }}>
          {m.warnings.slice(0, open ? undefined : 2).map((w, i) => <li key={i} className="text-warning">{w}</li>)}
        </ul>
      )}
      <button type="button" className="btn btn-sm btn-ghost mt-2" onClick={() => setOpen(o => !o)}>{open ? 'Hide details' : 'Show calculation'}</button>
      {open && (
        <div className="mt-2">
          <BreakdownTable old={c.old} nw={c.new} chosen={m.chosen_regime} />
          {c.old.deductions.length > 0 && (
            <>
              <h4 className="fw-semibold mb-2">Old-regime deductions</h4>
              <div className="payout-list mb-3">
                {c.old.deductions.map((d, i) => (
                  <div className="payout-item" key={i}>
                    <span>{sectionLabel(d.code, d.new_section)} — {d.label}</span>
                    <span>{formatCurrency(d.allowed)}{d.limit ? <span className="text-muted text-xs"> / {formatCurrency(d.limit)}</span> : null}</span>
                  </div>
                ))}
              </div>
            </>
          )}
          {c.new.deductions.length > 0 && (
            <div className="text-sm text-muted">New regime: {c.new.deductions.map(d => `${sectionLabel(d.code, d.new_section)} ${formatCurrency(d.allowed)}`).join(', ')}</div>
          )}
        </div>
      )}
    </div>
  );
}

function ClubbingCard({ report, readOnly, onChanged }: { report: FamilyTaxReport; readOnly: boolean; onChanged: () => void }) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [s, setS] = useState<TaxSettings>(report.settings);
  const update = async (patch: Partial<TaxSettings>) => {
    const prev = s;
    const next = { ...s, ...patch };
    setS(next);
    setSaving(true);
    try {
      await familyTaxApi.saveSettings({ ignore_spouse_clubbing: next.ignore_spouse_clubbing, ignore_minor_clubbing: next.ignore_minor_clubbing });
      setError('');
      onChanged();
    } catch (err) {
      setS(prev);
      setError(apiErrorMessage(err, 'Failed to save setting'));
    } finally {
      setSaving(false);
    }
  };
  const off = s.ignore_spouse_clubbing || s.ignore_minor_clubbing;
  return (
    <div className="card mb-4" style={off ? { borderLeft: '4px solid var(--warning)' } : undefined}>
      <h3 className="text-lg fw-semibold mb-1">
        Clubbing of income{' '}
        <InfoTooltip tip="By law, income from money you give your spouse, and income of a minor child, is added to your income (Sec 64). Turn a switch off to see tax as if each person were taxed on what is held in their name." />
      </h3>
      <p className="text-muted text-sm mb-3">Choose whether income from investments in a spouse's or child's name is added back to the person who paid for them.</p>
      <label className="flex items-center gap-2 mb-2 text-md">
        <input type="checkbox" disabled={readOnly || saving} checked={!s.ignore_spouse_clubbing}
          onChange={e => update({ ignore_spouse_clubbing: !e.target.checked })} />
        Add income from money given to a spouse back to the giver
      </label>
      <label className="flex items-center gap-2 mb-2 text-md">
        <input type="checkbox" disabled={readOnly || saving} checked={!s.ignore_minor_clubbing}
          onChange={e => update({ ignore_minor_clubbing: !e.target.checked })} />
        Add a minor child's investment income to the parent's income
      </label>
      {off && (
        <div className="text-sm text-warning mt-2">
          {report.clubbing_ignored > 0
            ? <>{formatCurrency(report.clubbing_ignored)} of income is being taxed in a spouse's or child's hands. </>
            : <>Clubbing is off. </>}
          The tax department can add this back to the giver's income, so actual tax may be higher. It's genuinely separate only if the spouse invests
          their own money (own savings or income, or gifts from their own relatives) or money lent to them at a fair interest rate.
        </div>
      )}
      {error && <div className="text-danger text-sm mt-1">{error}</div>}
    </div>
  );
}

function FamilyTab({ report, readOnly, onSettingsChanged }: { report: FamilyTaxReport; readOnly: boolean; onSettingsChanged: () => void }) {
  if (report.members.length === 0) {
    return (
      <div className="empty-state">
        <h3>No family members yet</h3>
        <p>Add yourself and your family (with occupation and date of birth), then enter each earning member's income.</p>
        <Link className="btn btn-primary" to="/family">Go to Family</Link>
      </div>
    );
  }
  const savers = report.suggestions.filter(s => s.category !== 'data');
  const data = report.suggestions.filter(s => s.category === 'data');
  return (
    <div>
      <div className="card-grid stat-grid mb-4">
        <div className="stat-card">
          <div className="label">Household tax (best regimes)</div>
          <div className="value">{formatCurrency(report.total_tax_best)}</div>
          <div className="text-muted text-sm">FY {report.fy} · {report.act}</div>
        </div>
        <div className="stat-card">
          <div className="label">With declared regimes</div>
          <div className="value">{formatCurrency(report.total_tax_preferred)}</div>
        </div>
        <div className="stat-card" style={{ background: report.regime_savings > 0 ? 'var(--warning-soft)' : undefined }}>
          <div className="label">Saving from switching regimes</div>
          <div className={`value ${report.regime_savings > 0 ? 'text-warning' : ''}`}>{formatCurrency(report.regime_savings)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Members</div>
          <div className="value">{report.members.length}</div>
          {report.unassigned_items > 0 && <div className="text-muted text-sm">{report.unassigned_items} item(s) unassigned → {report.default_member_name}</div>}
        </div>
      </div>

      <ClubbingCard report={report} readOnly={readOnly} onChanged={onSettingsChanged} />

      {savers.length > 0 && (
        <div className="card mb-4">
          <h3 className="text-lg fw-semibold mb-3">Ways to reduce family tax</h3>
          <div className="payout-list">
            {savers.map(s => (
              <div className="payout-item" key={s.id} style={{ alignItems: 'flex-start' }}>
                <div>
                  <div className="fw-semibold">{s.title} <span className="badge badge-neutral ml-1">{CATEGORY_LABEL[s.category] || s.category}</span></div>
                  <div className="text-muted text-sm">{s.detail}</div>
                </div>
                {s.est_saving > 0 && <span className="text-success fw-semibold" style={{ whiteSpace: 'nowrap' }}>~{formatCurrency(s.est_saving)}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="card-grid mb-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))' }}>
        {report.members.map(m => <MemberRegimeCard key={m.profile_id} m={m} />)}
      </div>

      {data.length > 0 && (
        <div className="card mb-4" style={{ borderLeft: '4px solid var(--warning)' }}>
          {data.map(s => <div key={s.id} className="text-sm mb-1"><strong>{s.title}.</strong> <span className="text-muted">{s.detail}</span></div>)}
        </div>
      )}
      <p className="text-muted text-xs">{report.disclaimer}</p>
    </div>
  );
}

function AdvisorTab({ report, meta, fy, readOnly, onSettingsChanged }: {
  report: FamilyTaxReport; meta: TaxMeta | null; fy: string; readOnly: boolean; onSettingsChanged: () => void;
}) {
  const instruments = meta?.instruments || {};
  const [amount, setAmount] = useState(500000);
  const [instrument, setInstrument] = useState('fd');
  const [ret, setRet] = useState(DEFAULT_RETURNS.fd);
  const [fundedBy, setFundedBy] = useState('');
  const [result, setResult] = useState<PlacementResult | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const run = async (e?: React.FormEvent) => {
    e?.preventDefault();
    setLoading(true);
    try {
      const r = await familyTaxApi.placement({ fy, amount, instrument, expected_return_pct: ret, funded_by: fundedBy });
      setResult(r.data);
      setError('');
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to compute placement'));
    } finally {
      setLoading(false);
    }
  };

  if (report.members.length === 0) {
    return <div className="empty-state"><h3>Add family members first</h3><Link className="btn btn-primary" to="/family">Go to Family</Link></div>;
  }
  return (
    <div>
      <ClubbingCard report={report} readOnly={readOnly} onChanged={() => { setResult(null); onSettingsChanged(); }} />
      <div className="card mb-4">
        <h3 className="text-lg fw-semibold mb-1">Whose name should this investment be in?</h3>
        <p className="text-muted text-sm mb-3">
          Compares the extra household tax if the investment is held by each member, using their income this year, best regime and clubbing rules.
        </p>
        <form onSubmit={run}>
          <div className="form-row">
            <div className="form-group">
              <label>Amount (₹)</label>
              <input type="number" min="1" step="any" required value={amount || ''} onChange={e => setAmount(parseFloat(e.target.value) || 0)} />
            </div>
            <div className="form-group">
              <label>Investment</label>
              <select value={instrument} onChange={e => { setInstrument(e.target.value); setRet(DEFAULT_RETURNS[e.target.value] ?? ret); }}>
                {Object.entries(instruments).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Expected return (% a year)</label>
              <input type="number" min="0" max="50" step="0.1" value={ret} onChange={e => setRet(parseFloat(e.target.value) || 0)} />
            </div>
            <div className="form-group">
              <label>Whose money? <InfoTooltip tip="Income from money given to a spouse or minor child is taxed back to the giver. Gifts to adult children, parents and siblings are not clubbed." /></label>
              <select value={fundedBy} onChange={e => setFundedBy(e.target.value)}>
                <option value="">Each member's own money</option>
                {report.members.map(m => <option key={m.profile_id} value={m.profile_id}>{m.name}</option>)}
              </select>
            </div>
          </div>
          <div className="modal-actions" style={{ justifyContent: 'flex-start' }}>
            <button type="submit" className="btn btn-primary" disabled={loading || amount <= 0}>{loading ? 'Calculating…' : 'Compare members'}</button>
          </div>
        </form>
        {error && <div className="text-danger text-sm mt-2">{error}</div>}
      </div>

      {result && (
        <div>
          <div className="table-container mb-3">
            <table>
              <thead>
                <tr>
                  <th>Hold in name of</th>
                  <th>Taxed in hands of</th>
                  <th className="text-right">Yearly return</th>
                  <th className="text-right">Tax on it</th>
                  <th className="text-right">Deduction saving</th>
                  <th className="text-right">Post-tax return</th>
                  <th className="text-right">Post-tax yield</th>
                </tr>
              </thead>
              <tbody>
                {result.options.map(o => (
                  <tr key={o.profile_id} style={{ opacity: o.eligible ? 1 : 0.55, background: o.profile_id === result.best_profile_id ? 'var(--success-soft)' : undefined }}>
                    <td>
                      <strong>{o.name}</strong>
                      {o.profile_id === result.best_profile_id && <span className="badge badge-active ml-1">Best</span>}
                      {!o.eligible && <span className="badge badge-overdue ml-1">Not eligible</span>}
                      {o.notes.length > 0 && <div className="text-muted text-xs">{o.notes.join(' ')}</div>}
                    </td>
                    <td>{o.taxpayer_name}</td>
                    <td className="text-right">{formatCurrency(o.annual_income)}</td>
                    <td className="text-right">{formatCurrency(o.incremental_tax)}</td>
                    <td className="text-right text-success">{o.deduction_saving > 0 ? formatCurrency(o.deduction_saving) : '—'}</td>
                    <td className="text-right fw-semibold">{formatCurrency(o.post_tax_income)}</td>
                    <td className="text-right">{o.post_tax_yield_pct.toFixed(2)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {result.notes.length > 0 && (
            <ul className="text-sm text-muted" style={{ paddingLeft: 18 }}>
              {result.notes.map((n, i) => <li key={i}>{n}</li>)}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

export default function TaxCenterPage() {
  const { viewAsUserId, isViewOnly } = useAuth();
  const [meta, setMeta] = useState<TaxMeta | null>(null);
  const [fy, setFy] = useState('');
  const [memberId, setMemberId] = useState('');
  const [report, setReport] = useState<FamilyTaxReport | null>(null);
  const [summary, setSummary] = useState<TaxSummary | null>(null);
  const [cg, setCG] = useState<CapitalGainsSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<Tab>('family');
  const [reloadTick, setReloadTick] = useState(0);
  const [metaDone, setMetaDone] = useState(false);

  useEffect(() => {
    familyTaxApi.meta()
      .then(r => { setMeta(r.data); setFy(f => f || r.data.current_fy); })
      .catch(() => {})
      .finally(() => setMetaDone(true));
  }, []);

  useEffect(() => {
    if (!metaDone) return;
    let cancelled = false;
    Promise.all([
      familyTaxApi.report(fy || undefined).then(r => { if (!cancelled) setReport(r.data); }),
      getTaxSummary(fy || undefined, memberId).then(r => { if (!cancelled) setSummary(r.data); }),
      getCapitalGains(fy || undefined, memberId).then(r => { if (!cancelled) setCG(r.data); }),
    ])
      .then(() => { if (!cancelled) setError(''); })
      .catch(err => { if (!cancelled) setError(apiErrorMessage(err, 'Failed to load tax data')); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [viewAsUserId, fy, memberId, metaDone, reloadTick]);

  const fys = useMemo(() => fyOptions(meta?.supported_fys || report?.supported_fys, fy || report?.fy), [meta, report, fy]);
  const members = report?.members || [];

  if (loading && !report) return <PageSkeleton label="Loading" />;

  return (
    <div>
      <div className="page-header">
        <h1>Tax Center</h1>
        <div className="flex items-center gap-2">
          {(tab === 'deductions' || tab === 'gains') && members.length > 0 && (
            <select value={memberId} onChange={e => setMemberId(e.target.value)} aria-label="Family member">
              <option value="">Whole household</option>
              {members.map(m => <option key={m.profile_id} value={m.profile_id}>{m.name}</option>)}
            </select>
          )}
          <select value={fy || report?.fy || ''} onChange={e => setFy(e.target.value)} aria-label="Financial year">
            {fys.map(f => <option key={f} value={f}>FY {f}</option>)}
          </select>
        </div>
      </div>
      {error && <div className="card mb-4 text-danger">{error}</div>}

      <Tabs
        ariaLabel="Tax views"
        value={tab}
        onChange={setTab}
        tabs={[
          { value: 'family', label: 'Family & regimes' },
          { value: 'advisor', label: 'Where to invest' },
          { value: 'deductions', label: 'Tax deductions' },
          { value: 'gains', label: 'Capital gains' },
        ]}
      />

      {tab === 'family' && report && <FamilyTab report={report} readOnly={isViewOnly} onSettingsChanged={() => setReloadTick(t => t + 1)} />}
      {tab === 'advisor' && report && (
        <AdvisorTab report={report} meta={meta} fy={fy || report.fy} readOnly={isViewOnly} onSettingsChanged={() => setReloadTick(t => t + 1)} />
      )}

      {tab === 'deductions' && summary && (
        <div>
          <div className="stat-card mb-4" style={{ background: 'var(--primary-soft)' }}>
            <div className="label">Total Deductions (FY {summary.fy || 'Current'}){memberId ? ` · ${members.find(m => m.profile_id === memberId)?.name}` : ''}</div>
            <div className="value" style={{ color: '#4f46e5' }}>{formatCurrency(summary.total_deductions)}</div>
            <div className="text-muted text-sm">From tracked investments; old regime only except employer NPS.</div>
          </div>

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 80C <span className="text-muted text-md fw-normal">(Limit: {formatCurrency(summary.section_80c.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_80c.epf_contribution > 0 && <div className="payout-item"><span>EPF / VPF Contribution</span><span>{formatCurrency(summary.section_80c.epf_contribution)}</span></div>}
              {summary.section_80c.ppf_contribution > 0 && <div className="payout-item"><span>PPF Contribution</span><span>{formatCurrency(summary.section_80c.ppf_contribution)}</span></div>}
              {summary.section_80c.elss_investment > 0 && <div className="payout-item"><span>ELSS Investment</span><span>{formatCurrency(summary.section_80c.elss_investment)}</span></div>}
              {summary.section_80c.home_loan_principal > 0 && <div className="payout-item"><span>Home Loan Principal</span><span>{formatCurrency(summary.section_80c.home_loan_principal)}</span></div>}
              {summary.section_80c.life_insurance > 0 && <div className="payout-item"><span>Life Insurance</span><span>{formatCurrency(summary.section_80c.life_insurance)}</span></div>}
              <div className="payout-item fw-semibold" style={{ borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_80c.total)} / {formatCurrency(summary.section_80c.deduction)}</span>
              </div>
            </div>
          </div>

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 80CCD (NPS Tier-1)</h3>
            <div className="payout-list">
              <div className="payout-item"><span>80CCD(1) - Own contributions</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1)}</span></div>
              <div className="payout-item"><span>80CCD(1B) - Additional ₹50K</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1b)}</span></div>
              <div className="payout-item"><span>80CCD(2) - Employer (both regimes)</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd2)}</span></div>
            </div>
          </div>

          {summary.section_80d && (summary.section_80d.health_insurance > 0 || summary.section_80d.parents > 0) && (
            <div className="card mb-4">
              <h3 className="text-lg fw-semibold mb-3">Section 80D (Health insurance)</h3>
              <div className="payout-list">
                <div className="payout-item"><span>Self, spouse &amp; children <span className="text-muted text-xs">(limit {formatCurrency(summary.section_80d.limit)})</span></span><span>{formatCurrency(summary.section_80d.health_insurance)}</span></div>
                <div className="payout-item"><span>Parents <span className="text-muted text-xs">(limit {formatCurrency(summary.section_80d.parents_limit)})</span></span><span>{formatCurrency(summary.section_80d.parents)}</span></div>
                <div className="payout-item fw-semibold" style={{ borderTop: '2px solid var(--border)', paddingTop: 8 }}><span>Deduction</span><span>{formatCurrency(summary.section_80d.deduction)}</span></div>
              </div>
            </div>
          )}

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 24(b) <span className="text-muted text-md fw-normal">(Self-occupied limit: {formatCurrency(summary.section_24b.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_24b.home_loan_interest > 0 && <div className="payout-item"><span>Home Loan Interest</span><span>{formatCurrency(summary.section_24b.home_loan_interest)}</span></div>}
              {summary.section_24b.pre_emi_interest > 0 && <div className="payout-item"><span>Pre-EMI Interest</span><span>{formatCurrency(summary.section_24b.pre_emi_interest)}</span></div>}
              <div className="payout-item fw-semibold" style={{ borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_24b.total)} / {formatCurrency(summary.section_24b.deduction)}</span>
              </div>
            </div>
          </div>

          {(summary.interest_income > 0 || summary.dividend_income > 0 || (summary.rental_income || 0) > 0 || (summary.other_rental_income || 0) > 0) && (
            <div className="card">
              <h3 className="text-lg fw-semibold mb-3">Taxable Income from Investments</h3>
              <div className="payout-list">
                {summary.interest_income > 0 && <div className="payout-item"><span>Interest (FD, bonds)</span><span>{formatCurrency(summary.interest_income)}</span></div>}
                {summary.dividend_income > 0 && <div className="payout-item"><span>Dividends</span><span>{formatCurrency(summary.dividend_income)}</span></div>}
                {(summary.rental_income || 0) > 0 && <div className="payout-item"><span>Rent from let-out property</span><span>{formatCurrency(summary.rental_income || 0)}</span></div>}
                {(summary.other_rental_income || 0) > 0 && <div className="payout-item"><span>Vehicle leasing payouts</span><span>{formatCurrency(summary.other_rental_income || 0)}</span></div>}
              </div>
            </div>
          )}
        </div>
      )}

      {tab === 'gains' && cg && (
        <div>
          <div className="card-grid stat-grid mb-4">
            <div className="stat-card">
              <div className="label">Equity LTCG</div>
              <div className="value">{formatCurrency(cg.ltcg)}</div>
              <div className="text-muted text-sm">Tax: {formatCurrency(cg.ltcg_tax)} (Exempt: {formatCurrency(cg.ltcg_exemption)})</div>
            </div>
            <div className="stat-card">
              <div className="label">Equity STCG</div>
              <div className="value">{formatCurrency(cg.stcg)}</div>
              <div className="text-muted text-sm">Tax: {formatCurrency(cg.stcg_tax)}</div>
            </div>
            {(cg.slab_gains || 0) !== 0 && (
              <div className="stat-card">
                <div className="label">Debt fund gains</div>
                <div className="value">{formatCurrency(cg.slab_gains || 0)}</div>
                <div className="text-muted text-sm">Taxed at the holder's slab rate</div>
              </div>
            )}
            <div className="stat-card">
              <div className="label">Equity Gains Tax</div>
              <div className="value negative">{formatCurrency(cg.total_tax)}</div>
            </div>
          </div>

          {cg.harvesting_tips && cg.harvesting_tips.length > 0 && (
            <div className="card mb-4" style={{ background: 'var(--success-soft)', border: '1px solid var(--success)' }}>
              <h3 className="fw-semibold mb-2" style={{ fontSize: 15, color: 'var(--success-text)' }}>Tax Harvesting Tips</h3>
              <ul className="m-0 text-md" style={{ paddingLeft: 18 }}>
                {cg.harvesting_tips.map((tip, i) => (
                  <li className="mb-1" key={i}>{tip}</li>
                ))}
              </ul>
            </div>
          )}

          {cg.entries?.length > 0 && (
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Investment</th>
                    <th>Buy Date</th>
                    <th className="text-right">Buy</th>
                    <th className="text-right">Sell</th>
                    <th className="text-right">Gain</th>
                    <th>Holding</th>
                    <th>Type</th>
                    <th className="text-right">Tax</th>
                  </tr>
                </thead>
                <tbody>
                  {cg.entries.map((e, i) => (
                    <tr key={i}>
                      <td>
                        {e.investment_name} <span className="text-muted text-xs">({e.investment_type})</span>
                        {e.grandfathered && <span className="text-warning ml-1" style={{ fontSize: 10, background: 'var(--warning-soft)', borderRadius: 4, padding: '1px 5px' }}>GF</span>}
                      </td>
                      <td className="text-muted text-sm">{e.buy_date}</td>
                      <td className="text-right">{formatCurrency(e.buy_amount)}</td>
                      <td className="text-right">{formatCurrency(e.sell_amount)}</td>
                      <td className={`text-right ${e.gain >= 0 ? 'text-success' : 'text-danger'}`}>{formatCurrency(e.gain)}</td>
                      <td className="text-muted text-sm">{e.holding_days > 365 ? `${Math.floor(e.holding_days / 365)}y ${Math.floor((e.holding_days % 365) / 30)}m` : `${e.holding_days}d`}</td>
                      <td>
                        {e.is_equity === false ? 'Slab' : e.is_long_term ? 'LTCG' : 'STCG'}
                        <span className="text-muted text-xs"> @{e.is_equity === false ? 'slab' : `${e.tax_rate}%`}</span>
                      </td>
                      <td className="text-right">{e.is_equity === false ? '—' : formatCurrency(e.tax_liability)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
