import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  ComposedChart, Bar, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  PieChart, Pie, Cell,
} from 'recharts';
import { getBudgets, saveBudget, deleteBudget, getLedgerSummary } from '../api/budgets';
import type { Budget, BudgetRequest, BudgetStatus, LedgerSummary } from '../api/budgets';
import { formatCurrency, formatCompactINR } from '../utils/format';
import { apiErrorMessage } from '../utils/errors';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';
import { Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';
import { CHART_COLORS, SERIES } from '../components/chartUtils';

const PIE_COLORS = CHART_COLORS;
const STATUS_COLORS: Record<string, string> = { ok: SERIES.positive, warning: SERIES.warning, over: SERIES.negative };

const monthLabel = (m: string) => {
  const [y, mo] = m.split('-').map(Number);
  if (!y || !mo) return m;
  return new Date(y, mo - 1, 1).toLocaleDateString('en-IN', { month: 'short', year: '2-digit' });
};

interface BudgetRow {
  budget?: Budget;
  status: BudgetStatus;
}

export default function BudgetsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [summary, setSummary] = useState<LedgerSummary | null>(null);
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Budget | 'new' | null>(null);
  const [form, setForm] = useState<BudgetRequest>({ category: '', monthly_limit: 0 });
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    Promise.all([getLedgerSummary(12), getBudgets()])
      .then(([s, b]) => {
        setSummary(s.data);
        setBudgets(b.data || []);
      })
      .catch(err => toast(apiErrorMessage(err, 'Failed to load budgets'), 'error'))
      .finally(() => setLoading(false));
  }, [toast]);

  useEffect(() => { load(); }, [load, viewAsUserId]);

  const months = useMemo(() => summary?.months || [], [summary]);
  const current = summary?.current_month;

  const chartData = useMemo(() => months.map(m => ({
    month: monthLabel(m.month),
    income: m.income,
    expense: m.expense,
    savingsRate: m.income > 0 ? Number(m.savings_rate_pct.toFixed(1)) : null,
  })), [months]);

  const topCategories = useMemo(() => {
    const top = summary?.top_categories;
    if (top && top.length > 0) return top;
    const byCat = current?.by_category || {};
    const total = Object.values(byCat).reduce((s, v) => s + v, 0);
    return Object.entries(byCat)
      .map(([category, amount]) => ({ category, amount, pct: total > 0 ? (amount / total) * 100 : 0 }))
      .sort((a, b) => b.amount - a.amount);
  }, [summary, current]);

  const pieData = useMemo(() => {
    if (topCategories.length <= 8) return topCategories;
    const head = topCategories.slice(0, 7);
    const rest = topCategories.slice(7);
    return [...head, {
      category: 'Others',
      amount: rest.reduce((s, c) => s + c.amount, 0),
      pct: rest.reduce((s, c) => s + c.pct, 0),
    }];
  }, [topCategories]);

  const categorySuggestions = useMemo(() => {
    const set = new Set<string>();
    months.forEach(m => Object.keys(m.by_category || {}).forEach(c => set.add(c)));
    Object.keys(current?.by_category || {}).forEach(c => set.add(c));
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [months, current]);

  const rows = useMemo<BudgetRow[]>(() => {
    const statusByCat = new Map((summary?.budgets || []).map(s => [s.category.toLowerCase(), s]));
    const spentByCat = new Map(Object.entries(current?.by_category || {}).map(([k, v]) => [k.toLowerCase(), v]));
    const list: BudgetRow[] = budgets.map(b => {
      const key = b.category.toLowerCase();
      const st = statusByCat.get(key);
      if (st) return { budget: b, status: st };
      const spent = spentByCat.get(key) || 0;
      const used = b.monthly_limit > 0 ? (spent / b.monthly_limit) * 100 : 0;
      return {
        budget: b,
        status: {
          category: b.category, limit: b.monthly_limit, spent, remaining: b.monthly_limit - spent,
          used_pct: used, status: used > 100 ? 'over' : used >= 80 ? 'warning' : 'ok',
        },
      };
    });
    const known = new Set(budgets.map(b => b.category.toLowerCase()));
    (summary?.budgets || []).forEach(s => { if (!known.has(s.category.toLowerCase())) list.push({ status: s }); });
    const rank: Record<string, number> = { over: 0, warning: 1, ok: 2 };
    return list.sort((a, b) => (rank[a.status.status] ?? 3) - (rank[b.status.status] ?? 3) || b.status.used_pct - a.status.used_pct);
  }, [budgets, summary, current]);

  const totals = useMemo(() => {
    const limit = rows.reduce((s, r) => s + r.status.limit, 0);
    const spent = rows.reduce((s, r) => s + r.status.spent, 0);
    return { limit, spent, over: rows.filter(r => r.status.status === 'over').length, warning: rows.filter(r => r.status.status === 'warning').length };
  }, [rows]);

  const openAdd = (category = '') => { setForm({ category, monthly_limit: 0 }); setEditing('new'); };
  const openEdit = (b: Budget) => { setForm({ category: b.category, monthly_limit: b.monthly_limit }); setEditing(b); };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    const category = form.category.trim();
    if (!category) { toast('Category is required', 'error'); return; }
    if (!(form.monthly_limit > 0)) { toast('Monthly limit must be greater than 0', 'error'); return; }
    setSaving(true);
    try {
      await saveBudget({ category, monthly_limit: form.monthly_limit });
      toast(editing === 'new' ? 'Budget saved' : 'Budget updated');
      setEditing(null);
      load();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save budget'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (b: Budget) => {
    if (!await confirm({ title: 'Delete budget', message: `Delete the budget for "${b.category}"?`, danger: true, confirmLabel: 'Delete' })) return;
    try {
      await deleteBudget(b.id);
      toast('Budget deleted');
      load();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to delete budget'), 'error');
    }
  };

  if (loading) return <PageSkeleton label="Loading" />;

  const hasEntries = months.some(m => m.entry_count > 0 || m.income > 0 || m.expense > 0) || (current?.entry_count ?? 0) > 0;
  const pending = summary?.pending_review_count ?? 0;
  const net = current?.net_savings ?? 0;
  const rate = current?.savings_rate_pct ?? 0;

  return (
    <div>
      <div className="page-header">
        <h1>Budgets &amp; Savings</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => openAdd()}><Plus size={16} aria-hidden="true" />Add Budget</button>}
      </div>

      {pending > 0 && (
        <div className="card flex justify-between items-center" style={{ borderLeft: '4px solid var(--warning)' }}>
          <span>
            <strong>{pending}</strong> ledger entr{pending === 1 ? 'y is' : 'ies are'} pending review and not counted below.
          </span>
          <Link to="/ledger" className="btn btn-sm btn-outline">Review entries</Link>
        </div>
      )}

      {!hasEntries ? (
        <div className="empty-state">
          <h3>No confirmed income/expense entries yet</h3>
          <p>No confirmed income/expense entries yet — add them in <Link to="/ledger">Income &amp; Expenses</Link>.</p>
        </div>
      ) : (
        <>
          <div className="card-grid stat-grid mb-5">
            <div className="stat-card">
              <div className="label">Income ({current ? monthLabel(current.month) : 'this month'})</div>
              <div className="value positive">{formatCurrency(current?.income ?? 0)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Expense</div>
              <div className="value negative">{formatCurrency(current?.expense ?? 0)}</div>
              {(current?.bills ?? 0) > 0 && <div className="text-muted text-xs">+ {formatCurrency(current?.bills ?? 0)} bills (excluded)</div>}
            </div>
            <div className="stat-card">
              <div className="label">Net Savings</div>
              <div className={`value ${net >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(net)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Savings Rate</div>
              <div className={`value ${rate >= 0 ? 'positive' : 'negative'}`}>{(current?.income ?? 0) > 0 ? `${rate.toFixed(1)}%` : '-'}</div>
              <div className="text-muted text-xs">Avg: {(summary?.avg_savings_rate_pct ?? 0).toFixed(1)}%</div>
            </div>
            <div className="stat-card">
              <div className="label">Pending Review</div>
              <div className="value">{pending}</div>
              <Link to="/ledger" style={{ fontSize: 12 }}>Go to Income &amp; Expenses →</Link>
            </div>
          </div>

          <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))' }}>
            <div className="card">
              <h3 className="mt-0">Income vs Expense (12 months)</h3>
              <div className="w-full" style={{ height: 300 }}>
                <ResponsiveContainer>
                  <ComposedChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="month" />
                    <YAxis yAxisId="amt" tickFormatter={(v) => formatCompactINR(Number(v))} />
                    <YAxis yAxisId="pct" orientation="right" unit="%" />
                    <Tooltip formatter={(v, name) => (name === 'Savings rate' ? `${Number(v).toFixed(1)}%` : formatCurrency(Number(v)))} />
                    <Legend />
                    <Bar yAxisId="amt" dataKey="income" name="Income" fill={SERIES.positive} />
                    <Bar yAxisId="amt" dataKey="expense" name="Expense" fill={SERIES.negative} />
                    <Line yAxisId="pct" type="monotone" dataKey="savingsRate" name="Savings rate" stroke={SERIES.primary} strokeWidth={2} connectNulls />
                  </ComposedChart>
                </ResponsiveContainer>
              </div>
              <div className="text-muted text-sm">
                Avg monthly income {formatCurrency(summary?.avg_monthly_income ?? 0)} · expense {formatCurrency(summary?.avg_monthly_expense ?? 0)}
              </div>
            </div>

            <div className="card">
              <h3 className="mt-0">Top categories this month</h3>
              {pieData.length === 0 ? (
                <p className="text-muted">No expenses recorded this month.</p>
              ) : (
                <div className="flex gap-4 flex-wrap items-center">
                  <div style={{ width: 220, height: 220 }}>
                    <ResponsiveContainer>
                      <PieChart>
                        <Pie data={pieData} dataKey="amount" nameKey="category" innerRadius={50} outerRadius={90}>
                          {pieData.map((c, i) => <Cell key={c.category} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
                        </Pie>
                        <Tooltip formatter={(v) => formatCurrency(Number(v))} />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                  <div className="flex-1" style={{ minWidth: 160 }}>
                    {pieData.map((c, i) => (
                      <div className="flex justify-between gap-2 text-md" key={c.category} style={{ padding: '3px 0' }}>
                        <span>
                          <span className="mr-2" style={{ display: 'inline-block', width: 10, height: 10, borderRadius: 2, background: PIE_COLORS[i % PIE_COLORS.length] }} />
                          {c.category}
                        </span>
                        <span>{formatCurrency(c.amount)} <span className="text-muted">({c.pct.toFixed(0)}%)</span></span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </>
      )}

      <div className="card">
        <div className="flex justify-between items-center mb-3">
          <h3 className="m-0">Monthly budgets</h3>
          {rows.length > 0 && (
            <span className="text-muted text-md">
              {formatCurrency(totals.spent)} of {formatCurrency(totals.limit)} used
              {totals.over > 0 && <span className="badge badge-overdue ml-2">{totals.over} over</span>}
              {totals.warning > 0 && <span className="badge badge-pending ml-2">{totals.warning} near limit</span>}
            </span>
          )}
        </div>
        {rows.length === 0 ? (
          <div className="text-muted">
            No budgets yet. {!isViewOnly && 'Set a monthly limit for a spending category to track it here.'}
          </div>
        ) : (
          rows.map(r => {
            const s = r.status;
            const color = STATUS_COLORS[s.status] || STATUS_COLORS.ok;
            return (
              <div key={s.category} style={{ padding: '10px 0', borderBottom: '1px solid var(--border)' }}>
                <div className="flex justify-between items-center gap-2 mb-2">
                  <div>
                    <strong>{s.category}</strong>
                    {s.status === 'over' && <span className="badge badge-overdue ml-2">over</span>}
                    {s.status === 'warning' && <span className="badge badge-pending ml-2">near limit</span>}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-md">
                      {formatCurrency(s.spent)} / {formatCurrency(s.limit)}
                      <span className={`${s.remaining < 0 ? 'text-danger' : 'text-muted'} ml-2`}>
                        ({s.remaining < 0 ? `${formatCurrency(-s.remaining)} over` : `${formatCurrency(s.remaining)} left`})
                      </span>
                    </span>
                    {!isViewOnly && r.budget && (
                      <RowActions
                        inline={1}
                        label={`Actions for ${s.category} budget`}
                        actions={[
                          { label: 'Edit', icon: Pencil, onClick: () => openEdit(r.budget as Budget) },
                          { label: 'Delete', icon: Trash2, onClick: () => handleDelete(r.budget as Budget), danger: true },
                        ]}
                      />
                    )}
                  </div>
                </div>
                <div style={{ height: 8, background: 'var(--border)', borderRadius: 4, overflow: 'hidden' }} title={`${s.used_pct.toFixed(0)}% used`}>
                  <div style={{ width: `${Math.min(100, Math.max(0, s.used_pct))}%`, height: '100%', background: color, transition: 'width 0.3s' }} />
                </div>
              </div>
            );
          })
        )}
      </div>

      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing === 'new' ? 'Add Budget' : 'Edit Budget'}</h2>
            <form onSubmit={handleSave}>
              <div className="form-group">
                <label>Category</label>
                <input required list="budget-category-suggestions" value={form.category} disabled={editing !== 'new'}
                  placeholder="e.g. Food" onChange={e => setForm({ ...form, category: e.target.value })} />
                <datalist id="budget-category-suggestions">
                  {categorySuggestions.map(c => <option key={c} value={c} />)}
                </datalist>
                {editing === 'new' && <div className="text-muted text-xs mt-1">Saving a category that already has a budget updates its limit.</div>}
              </div>
              <div className="form-group">
                <label>Monthly limit (₹)</label>
                <input type="number" step="1" min="1" required value={form.monthly_limit || ''}
                  onChange={e => setForm({ ...form, monthly_limit: parseFloat(e.target.value) || 0 })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setEditing(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
