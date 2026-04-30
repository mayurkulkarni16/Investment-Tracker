import { useEffect, useState } from 'react';
import { getInsights, type InsightsResponse, type Insight } from '../api/insights';
import { getRebalanceSuggestions, type RebalanceSuggestion, type RebalanceResponse } from '../api/dashboard';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';

const severityConfig: Record<string, { color: string; bg: string; icon: string; label: string }> = {
  critical: { color: '#c62828', bg: '#ffebee', icon: '🚨', label: 'Critical' },
  warning: { color: '#e65100', bg: '#fff3e0', icon: '⚠️', label: 'Warning' },
  info: { color: '#1565c0', bg: '#e3f2fd', icon: 'ℹ️', label: 'Info' },
  positive: { color: '#2e7d32', bg: '#e8f5e9', icon: '✅', label: 'Positive' },
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
  const [data, setData] = useState<InsightsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');
  const { toast } = useToast();

  // Rebalancing state
  const CATEGORIES = ['Mutual Funds', 'Stocks', 'Corporate Bonds', 'Fixed Deposits', 'Provident Fund', 'NPS'];
  const [targets, setTargets] = useState<Record<string, number>>(() => {
    const t: Record<string, number> = {};
    CATEGORIES.forEach(c => t[c] = 0);
    return t;
  });
  const [rebalanceResult, setRebalanceResult] = useState<RebalanceResponse | null>(null);
  const [rebalanceLoading, setRebalanceLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    getInsights()
      .then(setData)
      .catch(() => toast('Failed to load insights', 'error'))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="loading">Analyzing your portfolio...</div>;
  if (!data) return <div className="card">Failed to load insights.</div>;

  const filtered = filter === 'all' ? data.insights : data.insights.filter(i => i.severity === filter);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h1>Portfolio Insights</h1>
        <span className="text-muted" style={{ fontSize: 13 }}>{data.insights.length} insights generated</span>
      </div>

      <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
        <div className="stat-card" style={{ cursor: 'pointer', outline: filter === 'critical' ? '2px solid #c62828' : undefined }} onClick={() => setFilter(filter === 'critical' ? 'all' : 'critical')}>
          <div className="label">Critical</div>
          <div className="value" style={{ color: '#c62828' }}>{data.critical_count}</div>
        </div>
        <div className="stat-card" style={{ cursor: 'pointer', outline: filter === 'warning' ? '2px solid #e65100' : undefined }} onClick={() => setFilter(filter === 'warning' ? 'all' : 'warning')}>
          <div className="label">Warnings</div>
          <div className="value" style={{ color: '#e65100' }}>{data.warning_count}</div>
        </div>
        <div className="stat-card" style={{ cursor: 'pointer', outline: filter === 'info' ? '2px solid #1565c0' : undefined }} onClick={() => setFilter(filter === 'info' ? 'all' : 'info')}>
          <div className="label">Info</div>
          <div className="value" style={{ color: '#1565c0' }}>{data.info_count}</div>
        </div>
        <div className="stat-card" style={{ cursor: 'pointer', outline: filter === 'positive' ? '2px solid #2e7d32' : undefined }} onClick={() => setFilter(filter === 'positive' ? 'all' : 'positive')}>
          <div className="label">Positive</div>
          <div className="value" style={{ color: '#2e7d32' }}>{data.positive_count}</div>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: 40 }}>
          <p className="text-muted">No insights in this category.</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {filtered.map(insight => (
            <InsightCard key={insight.id} insight={insight} />
          ))}
        </div>
      )}

      {/* Rebalancing Tool */}
      <div style={{ marginTop: 32 }}>
        <h2 style={{ marginBottom: 16 }}>Portfolio Rebalancing Tool</h2>
        <div className="card" style={{ padding: 20 }}>
          <p className="text-muted" style={{ fontSize: 13, marginBottom: 16 }}>
            Set your target allocation percentages. Total must equal 100%. The tool will compare against your actual portfolio and suggest adjustments.
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12, marginBottom: 16 }}>
            {CATEGORIES.map(cat => (
              <div key={cat} className="form-group" style={{ marginBottom: 0 }}>
                <label style={{ fontSize: 13 }}>{cat}</label>
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
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
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: 13, color: Math.abs(Object.values(targets).reduce((s, v) => s + v, 0) - 100) > 0.01 ? '#c62828' : undefined }}>
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
          <div className="card" style={{ marginTop: 16, padding: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <h3 style={{ margin: 0 }}>Rebalancing Suggestions</h3>
              <span className="text-muted" style={{ fontSize: 13 }}>Portfolio: {formatCurrency(rebalanceResult.total_value)}</span>
            </div>
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Category</th>
                    <th>Current %</th>
                    <th>Target %</th>
                    <th>Difference</th>
                    <th>Current Value</th>
                    <th>Target Value</th>
                    <th>Adjustment</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {rebalanceResult.suggestions.map((s: RebalanceSuggestion) => (
                    <tr key={s.category}>
                      <td style={{ fontWeight: 600 }}>{s.category}</td>
                      <td>{s.current_pct.toFixed(1)}%</td>
                      <td>{s.target_pct.toFixed(1)}%</td>
                      <td style={{ color: s.diff_pct > 1 ? '#2e7d32' : s.diff_pct < -1 ? '#c62828' : undefined }}>
                        {s.diff_pct > 0 ? '+' : ''}{s.diff_pct.toFixed(1)}%
                      </td>
                      <td>{formatCurrency(s.current_value)}</td>
                      <td>{formatCurrency(s.target_value)}</td>
                      <td style={{ fontWeight: 600, color: s.adjustment_amount > 0 ? '#2e7d32' : s.adjustment_amount < 0 ? '#c62828' : undefined }}>
                        {s.adjustment_amount > 0 ? '+' : ''}{formatCurrency(Math.abs(s.adjustment_amount))}
                      </td>
                      <td>
                        <span style={{
                          padding: '2px 8px', borderRadius: 10, fontSize: 11, fontWeight: 600,
                          background: s.action === 'buy_more' ? '#e8f5e9' : s.action === 'reduce' ? '#ffebee' : '#f5f5f5',
                          color: s.action === 'buy_more' ? '#2e7d32' : s.action === 'reduce' ? '#c62828' : '#666',
                        }}>
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
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 18 }}>{config.icon}</span>
          <h3 style={{ fontSize: 15, fontWeight: 600, margin: 0 }}>{insight.title}</h3>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <span style={{
            padding: '2px 8px', borderRadius: 10, fontSize: 11, fontWeight: 600,
            background: config.bg, color: config.color,
          }}>{config.label}</span>
          <span style={{
            padding: '2px 8px', borderRadius: 10, fontSize: 11, fontWeight: 500,
            background: '#f5f5f5', color: '#666',
          }}>{categoryLabels[insight.category] || insight.category}</span>
        </div>
      </div>
      <p style={{ fontSize: 14, color: '#555', margin: '0 0 8px 26px', lineHeight: 1.5 }}>{insight.description}</p>
      {insight.action && (
        <div style={{
          margin: '0 0 0 26px', padding: '8px 12px', borderRadius: 6,
          background: '#f8f9fa', fontSize: 13, color: '#333', lineHeight: 1.5,
        }}>
          <strong>Action:</strong> {insight.action}
        </div>
      )}
    </div>
  );
}
