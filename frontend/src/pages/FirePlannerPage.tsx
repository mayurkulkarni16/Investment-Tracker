import React, { useEffect, useState } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, ReferenceLine } from 'recharts';
import { fireApi } from '../api/planning';
import type { FIREDefaults, FIREPlan, FIREPlanRequest } from '../api/planning';
import { formatCurrency, formatCompactINR } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { SERIES } from '../components/chartUtils';

const AGE_STORAGE_KEY = 'fire_planner_ages';

type FormState = Record<keyof FIREPlanRequest, string>;

interface StoredAges {
  current_age?: number;
  retirement_age?: number;
  life_expectancy?: number;
}

function loadStoredAges(): StoredAges {
  try {
    const raw = localStorage.getItem(AGE_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as StoredAges) : {};
  } catch {
    return {};
  }
}

function initialForm(): FormState {
  const ages = loadStoredAges();
  return {
    current_age: String(ages.current_age ?? 30),
    retirement_age: String(ages.retirement_age ?? 50),
    life_expectancy: String(ages.life_expectancy ?? 85),
    monthly_expenses: '',
    current_corpus: '',
    monthly_savings: '',
    annual_step_up_pct: '5',
    pre_retirement_return: '12',
    post_retirement_return: '8',
    inflation: '6',
    retirement_expense_pct: '100',
  };
}

function expensesSourceText(src: string): string {
  if (src === 'ledger') return 'from your last 6 months of expenses';
  return 'no expense data found — enter manually';
}

function savingsSourceText(src: string): string {
  if (src === 'ledger') return 'from income minus expenses in your ledger';
  if (src === 'sips') return 'from your active SIPs';
  return 'no savings data found — enter manually';
}

const num = (s: string) => {
  const n = parseFloat(s);
  return Number.isFinite(n) ? n : 0;
};

export default function FirePlannerPage() {
  const { viewAsUserId } = useAuth();
  const { toast } = useToast();
  const [form, setForm] = useState<FormState>(initialForm);
  const [defaults, setDefaults] = useState<FIREDefaults | null>(null);
  const [plan, setPlan] = useState<FIREPlan | null>(null);
  const [calculating, setCalculating] = useState(false);
  const [showTable, setShowTable] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fireApi.defaults()
      .then(r => {
        if (cancelled) return;
        const d = r.data;
        setDefaults(d);
        setForm(f => ({
          ...f,
          monthly_expenses: d.monthly_expenses > 0 ? String(Math.round(d.monthly_expenses)) : f.monthly_expenses,
          monthly_savings: d.monthly_savings > 0 ? String(Math.round(d.monthly_savings)) : f.monthly_savings,
          current_corpus: d.current_corpus > 0 ? String(Math.round(d.current_corpus)) : f.current_corpus,
          pre_retirement_return: d.pre_retirement_return > 0 ? String(d.pre_retirement_return) : f.pre_retirement_return,
          post_retirement_return: d.post_retirement_return > 0 ? String(d.post_retirement_return) : f.post_retirement_return,
          inflation: d.inflation > 0 ? String(d.inflation) : f.inflation,
        }));
      })
      .catch(() => { if (!cancelled) setDefaults(null); });
    return () => { cancelled = true; };
  }, [viewAsUserId]);

  const set = (key: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setForm(f => ({ ...f, [key]: value }));
    if (key === 'current_age' || key === 'retirement_age' || key === 'life_expectancy') {
      const stored = loadStoredAges();
      const n = parseInt(value, 10);
      if (Number.isFinite(n)) {
        localStorage.setItem(AGE_STORAGE_KEY, JSON.stringify({ ...stored, [key]: n }));
      }
    }
  };

  const handleCalculate = async (e: React.FormEvent) => {
    e.preventDefault();
    const req: FIREPlanRequest = {
      current_age: Math.round(num(form.current_age)),
      retirement_age: Math.round(num(form.retirement_age)),
      life_expectancy: Math.round(num(form.life_expectancy)),
      monthly_expenses: num(form.monthly_expenses),
      current_corpus: num(form.current_corpus),
      monthly_savings: num(form.monthly_savings),
      annual_step_up_pct: num(form.annual_step_up_pct),
      pre_retirement_return: num(form.pre_retirement_return),
      post_retirement_return: num(form.post_retirement_return),
      inflation: num(form.inflation),
      retirement_expense_pct: num(form.retirement_expense_pct),
    };
    if (req.monthly_expenses <= 0) { toast('Enter your monthly expenses', 'error'); return; }
    if (req.retirement_age <= req.current_age) { toast('Retirement age must be greater than current age', 'error'); return; }
    if (req.life_expectancy <= req.retirement_age) { toast('Life expectancy must be greater than retirement age', 'error'); return; }
    setCalculating(true);
    try {
      const r = await fireApi.plan(req);
      setPlan(r.data);
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to calculate FIRE plan'), 'error');
    } finally {
      setCalculating(false);
    }
  };

  const years = plan?.years || [];
  const chartData = years.map(y => ({ age: y.age, corpus: Math.round(y.end_corpus), required: Math.round(y.required_corpus) }));
  const retireAge = plan?.input.retirement_age;

  const field = (key: keyof FormState, label: string, opts: { step?: string; min?: number; max?: number; hint?: string; prefix?: string } = {}) => (
    <div className="form-group">
      <label>{label}</label>
      <input type="number" value={form[key]} onChange={set(key)} step={opts.step ?? 'any'} min={opts.min} max={opts.max} />
      {opts.hint && <div className="text-muted text-xs" style={{ marginTop: 2 }}>{opts.hint}</div>}
    </div>
  );

  return (
    <div>
      <div className="page-header">
        <h1>FIRE Planner</h1>
        <span className="text-muted text-md">Financial Independence, Retire Early</span>
      </div>

      <form className="card mb-5" onSubmit={handleCalculate}>
        <div className="form-row-3">
          {field('current_age', 'Current age', { step: '1', min: 15, max: 90 })}
          {field('retirement_age', 'Retirement age', { step: '1', min: 20, max: 95 })}
          {field('life_expectancy', 'Life expectancy', { step: '1', min: 40, max: 110 })}
        </div>
        <div className="form-row-3">
          {field('monthly_expenses', 'Monthly expenses (₹, today)', {
            min: 0,
            hint: defaults ? expensesSourceText(defaults.monthly_expenses_source) : undefined,
          })}
          {field('monthly_savings', 'Monthly savings / investments (₹)', {
            min: 0,
            hint: defaults ? savingsSourceText(defaults.monthly_savings_source) : undefined,
          })}
          {field('current_corpus', 'Current corpus (₹)', {
            min: 0,
            hint: defaults && defaults.current_corpus > 0 ? 'from your financial assets (excl. real estate)' : undefined,
          })}
        </div>
        <div className="form-row-3">
          {field('annual_step_up_pct', 'Annual savings step-up (%)', { min: 0, max: 50 })}
          {field('pre_retirement_return', 'Return before retirement (%)', { min: 0, max: 30, hint: defaults && defaults.pre_retirement_return > 0 ? 'based on your portfolio mix' : undefined })}
          {field('post_retirement_return', 'Return after retirement (%)', { min: 0, max: 30 })}
        </div>
        <div className="form-row-3">
          {field('inflation', 'Inflation (%)', { min: 0, max: 20 })}
          {field('retirement_expense_pct', 'Expenses after retirement (% of today)', { min: 0, max: 200, hint: '100% = same lifestyle as today' })}
          <div className="form-group flex items-end">
            <button type="submit" className="btn btn-primary btn-full" disabled={calculating}>
              {calculating ? 'Calculating…' : 'Calculate'}
            </button>
          </div>
        </div>
      </form>

      {!plan ? (
        <div className="empty-state">
          <h3>Plan your financial independence</h3>
          <p>Review the inputs above and click Calculate to see how much you need and when you can retire.</p>
        </div>
      ) : (
        <>
          <div className="card-grid stat-grid mb-5">
            <div className="stat-card">
              <div className="label">FI number (today)</div>
              <div className="value">{formatCurrency(plan.fi_number)}</div>
              <div className="text-muted text-sm mt-1">Corpus needed to retire right now</div>
            </div>
            <div className="stat-card">
              <div className="label">Target corpus at {plan.input.retirement_age}</div>
              <div className="value">{formatCurrency(plan.target_corpus)}</div>
              <div className="text-muted text-sm mt-1">
                {formatCurrency(plan.annual_expense_at_retirement)}/yr expenses · 25× rule: {formatCompactINR(plan.swr_corpus)}
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Projected corpus at {plan.input.retirement_age}</div>
              <div className={`value ${plan.on_track ? 'positive' : ''}`}>{formatCurrency(plan.projected_corpus)}</div>
              <div className="text-muted text-sm mt-1">In {plan.years_to_retirement} years</div>
            </div>
            <div className="stat-card">
              <div className="label">{plan.on_track ? 'On track' : 'Shortfall'}</div>
              <div className={`value ${plan.on_track ? 'positive' : 'negative'}`}>
                {plan.on_track ? `✓ ${formatCurrency(Math.abs(plan.gap))} surplus` : formatCurrency(plan.gap)}
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Required monthly savings</div>
              <div className="value">{formatCurrency(plan.required_monthly_savings)}</div>
              <div className="text-muted text-sm mt-1">
                With {plan.input.annual_step_up_pct}% yearly step-up (you save {formatCurrency(plan.input.monthly_savings)})
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Earliest retirement age</div>
              <div className={`value ${plan.earliest_retirement_age > 0 ? 'positive' : 'negative'}`}>
                {plan.earliest_retirement_age > 0 ? plan.earliest_retirement_age : 'Not reachable'}
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Corpus lasts until</div>
              <div className={`value ${plan.corpus_depletes_at_age > 0 ? 'negative' : 'positive'}`}>
                {plan.corpus_depletes_at_age > 0 ? `Depletes at ${plan.corpus_depletes_at_age}` : `Beyond ${plan.input.life_expectancy}`}
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Success probability</div>
              <div className={`value ${plan.success_probability_pct >= 80 ? 'positive' : plan.success_probability_pct < 50 ? 'negative' : ''}`}>
                {plan.success_probability_pct.toFixed(0)}%
              </div>
              <div className="text-muted text-sm mt-1">Monte Carlo simulation of market returns</div>
            </div>
            <div className="stat-card">
              <div className="label">Progress to FI</div>
              <div className="value">{plan.progress_pct.toFixed(1)}%</div>
              <div className="mt-2" style={{ height: 6, background: 'var(--border, #e0e0e0)', borderRadius: 3 }}>
                <div style={{ width: `${Math.min(100, Math.max(0, plan.progress_pct))}%`, height: 6, background: 'var(--success)', borderRadius: 3 }} />
              </div>
            </div>
          </div>

          {chartData.length > 0 && (
            <div className="card mb-5">
              <h3 className="mb-1">Corpus over time</h3>
              <p className="text-muted text-sm mt-0 mb-3">
                Projected corpus vs. the corpus required to retire at each age (nominal ₹).
              </p>
              <ResponsiveContainer width="100%" height={340}>
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="fireCorpus" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#4f46e5" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#4f46e5" stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="age" fontSize={11} />
                  <YAxis tickFormatter={v => formatCompactINR(v as number)} fontSize={11} width={70} />
                  <Tooltip
                    labelFormatter={l => `Age ${l}`}
                    formatter={(val, name) => [formatCurrency(val as number), name]}
                  />
                  <Legend />
                  {retireAge !== undefined && (
                    <ReferenceLine x={retireAge} stroke={SERIES.negative} strokeDasharray="4 4" label={{ value: `Retire @ ${retireAge}`, position: 'top', fontSize: 11 }} />
                  )}
                  <Area type="monotone" dataKey="corpus" name="Projected corpus" stroke={SERIES.primary} strokeWidth={2} fill="url(#fireCorpus)" />
                  <Area type="monotone" dataKey="required" name="Required corpus" stroke={SERIES.warning} strokeWidth={2} strokeDasharray="5 3" fill="none" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}

          {years.length > 0 && (
            <div className="card mb-5">
              <button type="button" className="btn btn-sm btn-outline" onClick={() => setShowTable(s => !s)}>
                {showTable ? '▼ Hide' : '▶ Show'} year-by-year table
              </button>
              {showTable && (
                <div className="table-container mt-3" style={{ maxHeight: 480, overflowY: 'auto' }}>
                  <table>
                    <thead>
                      <tr>
                        <th>Age</th>
                        <th>Year</th>
                        <th>Phase</th>
                        <th className="text-right">Contribution</th>
                        <th className="text-right">Withdrawal</th>
                        <th className="text-right">End corpus</th>
                        <th className="text-right">Required corpus</th>
                      </tr>
                    </thead>
                    <tbody>
                      {years.map(y => (
                        <tr key={y.year} style={y.age === retireAge ? { background: 'rgba(234,67,53,0.06)' } : undefined}>
                          <td>{y.age}</td>
                          <td>{y.year}</td>
                          <td>
                            <span className={`badge ${y.phase === 'retirement' ? 'badge-pending' : 'badge-completed'}`}>{y.phase}</span>
                          </td>
                          <td className="text-right">{y.contribution ? formatCurrency(y.contribution) : '-'}</td>
                          <td className="text-right">{y.withdrawal ? formatCurrency(y.withdrawal) : '-'}</td>
                          <td className={`text-right ${y.end_corpus < 0 ? 'text-danger' : ''}`}>{formatCurrency(y.end_corpus)}</td>
                          <td className="text-right text-muted">{y.required_corpus ? formatCurrency(y.required_corpus) : '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
