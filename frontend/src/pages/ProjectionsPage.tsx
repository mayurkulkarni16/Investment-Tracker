import { useEffect, useState } from 'react';
import { getProjections } from '../api/projections';
import type { ProjectionsResponse } from '../types';
import { formatCurrency } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, AreaChart, Area, BarChart, Bar, Cell } from 'recharts';

const COLORS = ['#1a73e8', '#0f9d58', '#f9ab00', '#ea4335', '#9c27b0', '#00bcd4', '#ff5722', '#607d8b', '#795548', '#3f51b5'];
const CATEGORY_COLORS: Record<string, string> = {
  mutual_fund: '#1a73e8',
  fixed_deposit: '#f9ab00',
  provident_fund: '#0f9d58',
  stock: '#ea4335',
  corporate_bond: '#9c27b0',
};
const CATEGORY_LABELS: Record<string, string> = {
  mutual_fund: 'Mutual Fund',
  fixed_deposit: 'Fixed Deposit',
  provident_fund: 'Provident Fund',
  stock: 'Stock',
  corporate_bond: 'Corporate Bond',
};

const MONTH_NAMES = ['', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

export default function ProjectionsPage() {
  const { viewAsUserId } = useAuth();
  const [data, setData] = useState<ProjectionsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [years, setYears] = useState(5);
  const [scenario, setScenario] = useState<string>('base');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  const load = () => {
    setLoading(true);
    getProjections(years, scenario).then(r => setData(r.data)).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [years, scenario, viewAsUserId]);

  if (loading) return <div className="loading">Loading projections...</div>;
  if (!data || data.investments.length === 0) {
    return (
      <div className="empty-state">
        <h3>No investments to project</h3>
        <p>Add some investments to see future growth projections.</p>
      </div>
    );
  }

  // Filter by category
  const filtered = selectedCategory === 'all'
    ? data.investments
    : data.investments.filter(inv => inv.category === selectedCategory);

  // Aggregate line chart data (quarterly points for readability)
  const aggregateChartData = data.aggregate_monthly
    .filter((_, i) => i % 3 === 0 || i === data.aggregate_monthly.length - 1)
    .map(pt => ({
      label: `${MONTH_NAMES[pt.month]} ${pt.year}`,
      value: pt.value,
    }));

  // Per-category aggregate data
  const categories = [...new Set(data.investments.map(i => i.category))];
  const categoryAggData = data.aggregate_monthly
    .filter((_, i) => i % 3 === 0 || i === data.aggregate_monthly.length - 1)
    .map((pt, idx) => {
      const row: Record<string, string | number> = { label: `${MONTH_NAMES[pt.month]} ${pt.year}` };
      for (const cat of categories) {
        const invs = data.investments.filter(inv => inv.category === cat);
        let total = 0;
        for (const inv of invs) {
          const qIdx = idx * 3 < inv.projections.length ? idx * 3 : inv.projections.length - 1;
          if (qIdx >= 0 && inv.projections[qIdx]) {
            total += inv.projections[qIdx].value;
          }
        }
        row[cat] = Math.round(total);
      }
      return row;
    });

  // Individual investment comparison bar (current vs projected)
  const comparisonData = filtered.map(inv => {
    const lastProjection = inv.projections[inv.projections.length - 1];
    return {
      name: inv.name.length > 18 ? inv.name.substring(0, 18) + '...' : inv.name,
      current: Math.round(inv.current_value),
      projected: lastProjection ? Math.round(lastProjection.value) : Math.round(inv.current_value),
      growth: lastProjection ? Math.round(((lastProjection.value - inv.current_value) / inv.current_value) * 100) : 0,
      rate: inv.assumed_rate_pct,
      category: inv.category,
    };
  }).sort((a, b) => b.growth - a.growth);

  // Growth percentage (projected total gain)
  const projectedGrowth = data.total_current > 0
    ? ((data.total_projected_5y - data.total_current) / data.total_current * 100)
    : 0;

  return (
    <div>
      <div className="page-header">
        <h1>Investment Projections</h1>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <label style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Scenario:</label>
          <div style={{ display: 'flex', gap: 0 }}>
            {[{ key: 'bear', label: '🐻 Bear', color: '#ea4335' }, { key: 'base', label: '📊 Base', color: '#1a73e8' }, { key: 'bull', label: '🐂 Bull', color: '#0f9d58' }].map(s => (
              <button key={s.key} onClick={() => setScenario(s.key)}
                style={{
                  padding: '6px 14px', fontSize: 13, border: '1px solid var(--border-color)', cursor: 'pointer',
                  background: scenario === s.key ? s.color : 'var(--bg-primary)',
                  color: scenario === s.key ? '#fff' : 'var(--text-primary)',
                  borderRadius: s.key === 'bear' ? '6px 0 0 6px' : s.key === 'bull' ? '0 6px 6px 0' : '0',
                  fontWeight: scenario === s.key ? 600 : 400,
                }}>{s.label}</button>
            ))}
          </div>
          <label style={{ fontSize: 13, color: 'var(--text-secondary)', marginLeft: 12 }}>Period:</label>
          <select value={years} onChange={e => setYears(parseInt(e.target.value))} style={{ padding: '6px 12px', borderRadius: 6 }}>
            <option value={1}>1 Year</option>
            <option value={3}>3 Years</option>
            <option value={5}>5 Years</option>
            <option value={10}>10 Years</option>
            <option value={15}>15 Years</option>
            <option value={20}>20 Years</option>
          </select>
        </div>
      </div>

      {/* Summary cards */}
      <div className="card-grid" style={{ marginBottom: 20 }}>
        <div className="stat-card">
          <div className="label">Current Portfolio</div>
          <div className="value">{formatCurrency(data.total_current)}</div>
        </div>
        {data.total_projected_1y > 0 && (
          <div className="stat-card">
            <div className="label">Projected (1Y)</div>
            <div className="value positive">{formatCurrency(data.total_projected_1y)}</div>
          </div>
        )}
        {data.total_projected_3y > 0 && (
          <div className="stat-card">
            <div className="label">Projected (3Y)</div>
            <div className="value positive">{formatCurrency(data.total_projected_3y)}</div>
          </div>
        )}
        {data.total_projected_5y > 0 && (
          <div className="stat-card">
            <div className="label">Projected ({years}Y)</div>
            <div className="value positive">
              {formatCurrency(data.total_projected_5y)}
              <span style={{ fontSize: 13, marginLeft: 6, color: '#0f9d58' }}>
                (+{projectedGrowth.toFixed(1)}%)
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Category filter */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' }}>
        <button className={`btn btn-sm ${selectedCategory === 'all' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setSelectedCategory('all')}>All</button>
        {categories.map(cat => (
          <button key={cat} className={`btn btn-sm ${selectedCategory === cat ? 'btn-primary' : 'btn-outline'}`} onClick={() => setSelectedCategory(cat)}>
            {CATEGORY_LABELS[cat] || cat}
          </button>
        ))}
      </div>

      {/* Charts Row */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 20 }}>
        {/* Total Portfolio Growth Area Chart */}
        <div className="card">
          <h3 style={{ marginBottom: 16 }}>Total Portfolio Growth</h3>
          <ResponsiveContainer width="100%" height={320}>
            <AreaChart data={aggregateChartData}>
              <defs>
                <linearGradient id="growthGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#1a73e8" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#1a73e8" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="label" fontSize={11} angle={-45} textAnchor="end" height={60} interval={Math.floor(aggregateChartData.length / 8)} />
              <YAxis tickFormatter={v => `₹${(v / 100000).toFixed(1)}L`} fontSize={11} />
              <Tooltip formatter={(val) => formatCurrency(val as number)} />
              <Area type="monotone" dataKey="value" name="Portfolio Value" stroke="#1a73e8" fill="url(#growthGradient)" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Category-wise Stacked Area */}
        <div className="card">
          <h3 style={{ marginBottom: 16 }}>Growth by Category</h3>
          <ResponsiveContainer width="100%" height={320}>
            <AreaChart data={categoryAggData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="label" fontSize={11} angle={-45} textAnchor="end" height={60} interval={Math.floor(categoryAggData.length / 8)} />
              <YAxis tickFormatter={v => `₹${(v / 100000).toFixed(1)}L`} fontSize={11} />
              <Tooltip formatter={(val) => formatCurrency(val as number)} />
              <Legend />
              {categories.map(cat => (
                <Area key={cat} type="monotone" dataKey={cat} name={CATEGORY_LABELS[cat] || cat} stroke={CATEGORY_COLORS[cat] || '#999'} fill={CATEGORY_COLORS[cat] || '#999'} fillOpacity={0.15} strokeWidth={2} stackId="1" />
              ))}
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Individual Investment Projections */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 20 }}>
        {/* Individual growth lines */}
        <div className="card">
          <h3 style={{ marginBottom: 16 }}>Individual Investment Growth</h3>
          <ResponsiveContainer width="100%" height={350}>
            <LineChart>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="label"
                type="category"
                allowDuplicatedCategory={false}
                fontSize={11}
                angle={-45}
                textAnchor="end"
                height={60}
              />
              <YAxis tickFormatter={v => `₹${(v / 100000).toFixed(1)}L`} fontSize={11} />
              <Tooltip formatter={(val) => formatCurrency(val as number)} />
              <Legend />
              {filtered.slice(0, 8).map((inv, i) => {
                const lineData = inv.projections
                  .filter((_, idx) => idx % 3 === 0 || idx === inv.projections.length - 1)
                  .map(pt => ({
                    label: `${MONTH_NAMES[pt.month]} ${pt.year}`,
                    value: pt.value,
                  }));
                return (
                  <Line
                    key={inv.name}
                    data={lineData}
                    type="monotone"
                    dataKey="value"
                    name={inv.name.length > 20 ? inv.name.substring(0, 20) + '...' : inv.name}
                    stroke={COLORS[i % COLORS.length]}
                    strokeWidth={2}
                    dot={false}
                  />
                );
              })}
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Growth comparison bar */}
        <div className="card">
          <h3 style={{ marginBottom: 16 }}>Projected Growth (%)</h3>
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={comparisonData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis type="number" tickFormatter={v => `${v}%`} fontSize={11} />
              <YAxis type="category" dataKey="name" width={140} fontSize={11} />
              <Tooltip formatter={(val, name) => name === 'growth' ? `${val}%` : formatCurrency(val as number)} />
              <Bar dataKey="growth" name="Growth %" radius={[0, 4, 4, 0]}>
                {comparisonData.map((entry, i) => (
                  <Cell key={i} fill={CATEGORY_COLORS[entry.category] || COLORS[i % COLORS.length]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Current vs Projected Comparison */}
      <div className="card" style={{ marginBottom: 20 }}>
        <h3 style={{ marginBottom: 16 }}>Current vs Projected Value</h3>
        <ResponsiveContainer width="100%" height={Math.max(300, comparisonData.length * 45)}>
          <BarChart data={comparisonData} layout="vertical">
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis type="number" tickFormatter={v => `₹${(v / 100000).toFixed(1)}L`} fontSize={11} />
            <YAxis type="category" dataKey="name" width={140} fontSize={11} />
            <Tooltip formatter={(val) => formatCurrency(val as number)} />
            <Legend />
            <Bar dataKey="current" name="Current Value" fill="#1a73e8" radius={[0, 4, 4, 0]} />
            <Bar dataKey="projected" name={`Projected (${years}Y)`} fill="#0f9d58" radius={[0, 4, 4, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>

      {/* Assumptions table */}
      <div className="card">
        <h3 style={{ marginBottom: 12 }}>Projection Assumptions</h3>
        <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 12 }}>
          Projections are based on historical CAGR where available, otherwise category-specific defaults. These are estimates — actual returns may vary.
        </p>
        <table style={{ fontSize: 13 }}>
          <thead>
            <tr><th>Investment</th><th>Category</th><th>Current Value</th><th>Assumed Rate</th><th>Projected ({years}Y)</th><th>Expected Growth</th></tr>
          </thead>
          <tbody>
            {data.investments.map((inv, i) => {
              const last = inv.projections[inv.projections.length - 1];
              const growth = last ? last.value - inv.current_value : 0;
              return (
                <tr key={i}>
                  <td>{inv.name}</td>
                  <td><span style={{ padding: '2px 8px', borderRadius: 10, fontSize: 11, fontWeight: 600, background: CATEGORY_COLORS[inv.category] + '22', color: CATEGORY_COLORS[inv.category] }}>{CATEGORY_LABELS[inv.category] || inv.category}</span></td>
                  <td>{formatCurrency(inv.current_value)}</td>
                  <td>{inv.assumed_rate_pct}% p.a.</td>
                  <td>{last ? formatCurrency(last.value) : '-'}</td>
                  <td className={growth >= 0 ? 'positive' : 'negative'}>{formatCurrency(growth)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
