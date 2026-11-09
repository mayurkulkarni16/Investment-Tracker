import { useEffect, useState } from 'react';
import { getInsights, type InsightsResponse, type Insight } from '../api/insights';
import { getRebalanceSuggestions, type RebalanceSuggestion, type RebalanceResponse } from '../api/dashboard';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import SeverityIcon from '../components/SeverityIcon';
import { PageSkeleton } from '../components/ui';

const severityConfig: Record<string, { color: string; bg: string; label: string }> = {
  critical: { color: 'var(--danger-text)', bg: 'var(--danger-soft)', label: 'Critical' },
  warning: { color: 'var(--warning-text)', bg: 'var(--warning-soft)', label: 'Warning' },
  info: { color: 'var(--primary-text)', bg: 'var(--primary-soft)', label: 'Info' },
  positive: { color: 'var(--success-text)', bg: 'var(--success-soft)', label: 'Positive' },
};

const categoryLabels: Record<string, string> = {
  risk: 'Risk',
  performance: 'Performance',
  tax: 'Tax',
  diversification: 'Diversification',
  loan: 'Loans',
  opportunity: 'Opportunity',
  goal: 'Goals',
};

export default function InsightsPage() {
  const { viewAsUserId } = useAuth();
  const [result, setResult] = useState<{ key: string; data: InsightsResponse | null } | null>(null);
  const [filter, setFilter] = useState<string>('all');
  const { toast } = useToast();

  // Rebalancing state
  const CATEGORIES = ['Mutual Funds', 'Stocks', 'Corporate Bonds', 'Fixed Deposits', 'Provident Fund', 'NPS', 'Alternate Investments'];
  const [targets, setTargets] = useState<Record<string, number>>(() => {
    const t: Record<string, number> = {};
    CATEGORIES.forEach(c => t[c] = 0);
    return t;
  });
  const [rebalanceResult, setRebalanceResult] = useState<RebalanceResponse | null>(null);
  const [rebalanceLoading, setRebalanceLoading] = useState(false);

  const requestKey = viewAsUserId ?? '';
  useEffect(() => {
    let cancelled = false;
    getInsights()
      .then(d => { if (!cancelled) setResult({ key: requestKey, data: d }); })
      .catch(() => {
        if (cancelled) return;
        toast('Failed to load insights', 'error');
        setResult({ key: requestKey, data: null });
      });
    return () => { cancelled = true; };
  }, [requestKey, toast]);

  if (result?.key !== requestKey) return <PageSkeleton label="Analyzing your portfolio" />;
  const data = result.data;
  if (!data) return <div className="card">Failed to load insights.</div>;

  const filtered = filter === 'all' ? data.insights : data.insights.filter(i => i.severity === filter);

  return (
    <div>
      <div className="flex justify-between items-center mb-5">
        <h1>Portfolio Insights</h1>
        <span className="text-muted text-md">{data.insights.length} insights generated</span>
      </div>

      <div className="card-grid stat-grid mb-5">
        <div className="stat-card cursor-pointer" style={{ outline: filter === 'critical' ? '2px solid #c62828' : undefined }} onClick={() => setFilter(filter === 'critical' ? 'all' : 'critical')}>
          <div className="label">Critical</div>
          <div className="value" style={{ color: 'var(--danger-text)' }}>{data.critical_count}</div>
        </div>
        <div className="stat-card cursor-pointer" style={{ outline: filter === 'warning' ? '2px solid #e65100' : undefined }} onClick={() => setFilter(filter === 'warning' ? 'all' : 'warning')}>
          <div className="label">Warnings</div>
          <div className="value text-warning">{data.warning_count}</div>
        </div>
        <div className="stat-card cursor-pointer" style={{ outline: filter === 'info' ? '2px solid #1565c0' : undefined }} onClick={() => setFilter(filter === 'info' ? 'all' : 'info')}>
          <div className="label">Info</div>
          <div className="value" style={{ color: 'var(--primary-text)' }}>{data.info_count}</div>
        </div>
        <div className="stat-card cursor-pointer" style={{ outline: filter === 'positive' ? '2px solid #2e7d32' : undefined }} onClick={() => setFilter(filter === 'positive' ? 'all' : 'positive')}>
          <div className="label">Positive</div>
          <div className="value" style={{ color: 'var(--success-text)' }}>{data.positive_count}</div>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="card text-center" style={{ padding: 40 }}>
          <p className="text-muted">No insights in this category.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {filtered.map(insight => (
            <InsightCard key={insight.id} insight={insight} />
          ))}
        </div>
      )}

      {/* Rebalancing Tool */}
      <div className="mt-8">
        <h2 className="mb-4">Portfolio Rebalancing Tool</h2>
        <div className="card p-5">
          <p className="text-muted text-md mb-4">
            Set your target allocation percentages. Total must equal 100%. The tool will compare against your actual portfolio and suggest adjustments.
          </p>
          <div className="grid gap-3 mb-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))' }}>
            {CATEGORIES.map(cat => (
              <div key={cat} className="form-group mb-0">
                <label className="text-md">{cat}</label>
                <div className="flex items-center gap-1">
                  <input
                    type="number" min="0" max="100" step="1"
                    value={targets[cat] || ''}
                    onChange={e => setTargets(prev => ({ ...prev, [cat]: parseFloat(e.target.value) || 0 }))}
                    style={{ width: 80 }}
                  />
                  <span className="text-muted">%</span>
                </div>
              </div>
            ))}
          </div>
          <div className="flex justify-between items-center">
            <span className="text-md" style={{ color: Math.abs(Object.values(targets).reduce((s, v) => s + v, 0) - 100) > 0.01 ? 'var(--danger-text)' : undefined }}>
              Total: {Object.values(targets).reduce((s, v) => s + v, 0).toFixed(0)}%
              {Math.abs(Object.values(targets).reduce((s, v) => s + v, 0) - 100) > 0.01 && ' (must equal 100%)'}
            </span>
            <button
              className="btn btn-primary"
              disabled={rebalanceLoading || Math.abs(Object.values(targets).reduce((s, v) => s + v, 0) - 100) > 0.01}
              onClick={async () => {
                setRebalanceLoading(true);
                try {
                  const active: Record<string, number> = {};
                  Object.entries(targets).forEach(([k, v]) => { if (v > 0) active[k] = v; });
                  const res = await getRebalanceSuggestions(active);
                  setRebalanceResult(res.data);
                } catch { toast('Failed to get rebalancing suggestions', 'error'); }
                finally { setRebalanceLoading(false); }
              }}
            >
              {rebalanceLoading ? 'Analyzing...' : 'Analyze'}
            </button>
          </div>
        </div>

        {rebalanceResult && rebalanceResult.suggestions && (
          <div className="card mt-4 p-5">
            <div className="flex justify-between items-center mb-4">
              <h3 className="m-0">Rebalancing Suggestions</h3>
              <span className="text-muted text-md">Portfolio: {formatCurrency(rebalanceResult.total_value)}</span>
            </div>
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Category</th>
                    <th className="num">Current %</th>
                    <th className="num">Target %</th>
                    <th className="num">Difference</th>
                    <th className="num">Current Value</th>
                    <th className="num">Target Value</th>
                    <th className="num">Adjustment</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {rebalanceResult.suggestions.map((s: RebalanceSuggestion) => (
                    <tr key={s.category}>
                      <td className="fw-semibold">{s.category}</td>
                      <td className="num">{s.current_pct.toFixed(1)}%</td>
                      <td className="num">{s.target_pct.toFixed(1)}%</td>
                      <td className="num" style={{ color: s.diff_pct > 1 ? 'var(--success-text)' : s.diff_pct < -1 ? 'var(--danger-text)' : undefined }}>
                        {s.diff_pct > 0 ? '+' : ''}{s.diff_pct.toFixed(1)}%
                      </td>
                      <td className="num">{formatCurrency(s.current_value)}</td>
                      <td className="num">{formatCurrency(s.target_value)}</td>
                      <td className="fw-semibold num" style={{ color: s.adjustment_amount > 0 ? 'var(--success-text)' : s.adjustment_amount < 0 ? 'var(--danger-text)' : undefined }}>
                        {s.adjustment_amount > 0 ? '+' : ''}{formatCurrency(Math.abs(s.adjustment_amount))}
                      </td>
                      <td>
                        <span className="text-xs fw-semibold" style={{ padding: '2px 8px', borderRadius: 10, background: s.action === 'buy_more' ? 'var(--success-soft)' : s.action === 'reduce' ? 'var(--danger-soft)' : 'var(--surface-muted)', color: s.action === 'buy_more' ? 'var(--success-text)' : s.action === 'reduce' ? 'var(--danger-text)' : 'var(--text-secondary)' }}>
                          {s.action === 'buy_more' ? 'Buy More' : s.action === 'reduce' ? 'Reduce' : 'On Target'}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function InsightCard({ insight }: { insight: Insight }) {
  const config = severityConfig[insight.severity] || severityConfig.info;

  return (
    <div className="card" style={{
      borderLeft: `4px solid ${config.color}`,
      padding: '16px 20px',
    }}>
      <div className="flex justify-between items-start mb-2">
        <div className="flex items-center gap-2">
          <SeverityIcon severity={insight.severity} size={18} />
          <h3 className="fw-semibold m-0" style={{ fontSize: 15 }}>{insight.title}</h3>
        </div>
        <div className="flex gap-2">
          <span className="text-xs fw-semibold" style={{ padding: '2px 8px', borderRadius: 10, background: config.bg, color: config.color }}>{config.label}</span>
          <span className="text-xs fw-medium text-muted" style={{ padding: '2px 8px', borderRadius: 10, background: 'var(--surface-muted)' }}>{categoryLabels[insight.category] || insight.category}</span>
        </div>
      </div>
      <p className="text-base text-muted" style={{ margin: '0 0 8px 26px', lineHeight: 1.5 }}>{insight.description}</p>
      {insight.action && (
        <div className="text-md" style={{ margin: '0 0 0 26px', padding: '8px 12px', borderRadius: 6, background: 'var(--surface-muted)', color: 'var(--text)', lineHeight: 1.5 }}>
          <strong>Action:</strong> {insight.action}
        </div>
      )}
    </div>
  );
}
