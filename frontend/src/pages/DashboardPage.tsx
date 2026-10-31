import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getDashboard } from '../api/dashboard';
import { getMutualFunds } from '../api/mutualFunds';
import { getBenchmarks } from '../api/benchmarks';
import { getInsights, type InsightsResponse } from '../api/insights';
import { useAuth } from '../context/AuthContext';
import { getStocks } from '../api/stocks';
import type { DashboardSummary, MutualFund, BenchmarkData, Stock } from '../types';
import { formatCurrency, formatPercent, formatDate } from '../utils/format';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Legend } from 'recharts';
import InfoTooltip from '../components/InfoTooltip';
import MoneyOverview from '../components/MoneyOverview';

const COLORS = ['#1a73e8', '#0f9d58', '#f9ab00', '#ea4335', '#9c27b0', '#00bcd4', '#ff5722', '#607d8b'];
const LABELS: Record<string, string> = {
  mutual_funds: 'Mutual Funds',
  corporate_bonds: 'Corporate Bonds',
  fixed_deposits: 'Fixed Deposits',
  provident_fund: 'Provident Fund',
  stocks: 'Stocks',
  nps: 'NPS',
  vehicle_rentals: 'Vehicle Rentals',
  home_loans: 'Home Loans',
  personal_loans: 'Personal Loans',
};

export default function DashboardPage() {
  const { viewAsUserId, user } = useAuth();
  return (
    <div>
      <div className="page-header"><h1>Dashboard</h1></div>
      <MoneyOverview key={`money-${viewAsUserId || user?.id}`} />
      <InvestmentDashboard key={`investments-${viewAsUserId || user?.id}`} />
    </div>
  );
}

function InvestmentDashboard() {
  const { viewAsUserId } = useAuth();
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [funds, setFunds] = useState<MutualFund[]>([]);
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [benchmarks, setBenchmarks] = useState<BenchmarkData[]>([]);
  const [insightsData, setInsightsData] = useState<InsightsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [drillCategory, setDrillCategory] = useState<string | null>(null);
  const [renderedAt] = useState(() => Date.now());

  useEffect(() => {
    Promise.all([
      getDashboard().then(r => setData(r.data)).catch(() => {}),
      getMutualFunds().then(r => setFunds(r.data || [])).catch(() => {}),
      getStocks().then(r => setStocks(r.data || [])).catch(() => {}),
      getBenchmarks().then(r => setBenchmarks(r.data?.indices || [])).catch(() => {}),
      getInsights().then(r => setInsightsData(r)).catch(() => {}),
    ]).finally(() => setLoading(false));
  }, [viewAsUserId]);

  if (loading) return <div className="loading">Loading dashboard...</div>;
  if (!data) return <div className="empty-state"><h3>No data yet</h3><p>Add some investments to see your dashboard.</p></div>;

  const allocationEntries = data.asset_allocation ? Object.entries(data.asset_allocation) : [];
  const total = allocationEntries.reduce((s, [, v]) => s + v, 0);

  // Pie chart data
  const pieData = allocationEntries.map(([key, val]) => ({
    name: LABELS[key] || key,
    key,
    value: val,
  }));

  // Drilldown data for clicked category
  const drillData: { name: string; value: number }[] = (() => {
    if (!drillCategory) return [];
    switch (drillCategory) {
      case 'mutual_funds':
        return funds.filter(f => f.current_value > 0).map(f => ({ name: f.fund_name.length > 25 ? f.fund_name.substring(0, 25) + '...' : f.fund_name, value: f.current_value }));
      case 'stocks':
        return stocks.filter(s => s.current_value > 0).map(s => ({ name: s.symbol, value: s.current_value }));
      default:
        return [];
    }
  })();

  // MF performance bar chart
  const mfBarData = funds
    .filter(f => f.total_invested > 0)
    .map(f => ({
      name: f.fund_name.length > 20 ? f.fund_name.substring(0, 20) + '...' : f.fund_name,
      invested: f.total_invested,
      current: f.current_value,
      gain: f.gain_loss,
    }));

  // Investment summary bar chart. Gain includes cash already paid out (bond
  // interest/principal, FD payouts and maturities, rental payouts).
  const gainOf = (s?: { total_invested: number; current_value: number; income_received?: number }) =>
    s ? s.current_value + (s.income_received || 0) - s.total_invested : 0;
  const summaryBarData = [
    { name: 'Mutual Funds', s: data.mutual_fund_summary },
    { name: 'Bonds', s: data.corporate_bond_summary },
    { name: 'FDs', s: data.fixed_deposit_summary },
    { name: 'Stocks', s: data.stock_summary },
    { name: 'PF', s: data.provident_fund_summary },
    { name: 'NPS', s: data.nps_summary },
    { name: 'Vehicle Rentals', s: data.vehicle_rental_summary },
  ].map(({ name, s }) => ({ name, invested: s?.total_invested || 0, value: s?.current_value || 0, gain: gainOf(s) }))
    .filter(d => d.invested > 0 || d.value > 0);

  // How fresh the market data behind current values is
  const freshness = data.data_freshness;
  const staleAfterMs = 4 * 24 * 3600 * 1000; // covers weekends and market holidays
  const isStale = (iso: string) => renderedAt - new Date(iso).getTime() > staleAfterMs;
  const freshnessParts: { label: string; stale: boolean }[] = [];
  if (freshness?.has_mutual_funds) {
    freshnessParts.push(freshness.navs_as_of
      ? { label: `NAVs as of ${formatDate(freshness.navs_as_of)}${freshness.navs_missing ? ` (${freshness.navs_missing} never fetched)` : ''}`, stale: isStale(freshness.navs_as_of) || freshness.navs_missing > 0 }
      : { label: 'NAVs never fetched', stale: true });
  }
  if (freshness?.has_stocks) {
    freshnessParts.push(freshness.prices_as_of
      ? { label: `Stock prices as of ${formatDate(freshness.prices_as_of)}${freshness.prices_missing ? ` (${freshness.prices_missing} never fetched)` : ''}`, stale: isStale(freshness.prices_as_of) || freshness.prices_missing > 0 }
      : { label: 'Stock prices never fetched', stale: true });
  }
  const tax80c = data.tax_80c;

  // MF gain/loss data for line visualization
  const mfGainData = funds
    .filter(f => f.total_invested > 0)
    .map(f => ({
      name: f.fund_name.length > 15 ? f.fund_name.substring(0, 15) + '...' : f.fund_name,
      gainPercent: f.gain_loss_percent,
    }))
    .sort((a, b) => b.gainPercent - a.gainPercent);

  return (
    <div>
      <div className="page-header">
        <span className="text-muted" style={{ fontSize: 12, textAlign: 'right' }}>
          {freshnessParts.length === 0
            ? 'Values from your recorded data'
            : freshnessParts.map((p, i) => (
              <span key={p.label} style={{ color: p.stale ? 'var(--warning)' : undefined }}>
                {i > 0 && ' · '}{p.label}
              </span>
            ))}
        </span>
      </div>

      <div className="card-grid">
        <div className="stat-card">
          <div className="label">Total Invested</div>
          <div className="value">{formatCurrency(data.total_invested)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Current Value</div>
          <div className="value">{formatCurrency(data.current_value)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Total Gains<InfoTooltip tip="Current value minus amount invested, plus cash already paid out to you (bond interest and principal, FD interest and maturities, rental payouts)" /></div>
          <div className={`value ${data.total_gains >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(data.total_gains)} ({formatPercent(data.overall_return_percent)})
          </div>
        </div>
        <div className="stat-card">
          <div className="label">Portfolio XIRR<InfoTooltip tip="Extended Internal Rate of Return — annualized return accounting for the timing of each cash flow" /></div>
          <div className={`value ${data.portfolio_xirr >= 0 ? 'positive' : 'negative'}`}>
            {formatPercent(data.portfolio_xirr)}
          </div>
        </div>
        <div className="stat-card">
          <div className="label">Risk Score<InfoTooltip tip="1-10 scale based on equity/debt ratio and asset concentration. Higher = more risk" /></div>
          <div className="value">
            <span style={{ display: 'inline-block', padding: '2px 10px', borderRadius: 12, fontSize: 14, fontWeight: 600, background: data.risk_metrics?.risk_score <= 3 ? 'var(--success-soft)' : data.risk_metrics?.risk_score <= 6 ? 'var(--warning-soft)' : 'var(--danger-soft)', color: data.risk_metrics?.risk_score <= 3 ? 'var(--success-text)' : data.risk_metrics?.risk_score <= 6 ? 'var(--warning-text)' : 'var(--danger-text)' }}>
              {data.risk_metrics?.risk_score ?? '-'}/10
            </span>
          </div>
        </div>
        <div className="stat-card">
          <div className="label">Section 80C{tax80c ? ` (FY ${tax80c.fy})` : ''}<InfoTooltip tip="Deductions used this financial year: EPF, ELSS (incl. SIPs) and home loan principal. Applies under the old tax regime only." /></div>
          {tax80c ? (
            <>
              <div className="value">{formatCurrency(Math.min(tax80c.used, tax80c.limit))} <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>/ {formatCurrency(tax80c.limit)}</span></div>
              <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 4 }}>
                {tax80c.used >= tax80c.limit ? 'Limit fully used' : `${formatCurrency(tax80c.limit - tax80c.used)} room left`}
                {tax80c.elss > 0 && ` · ELSS ${formatCurrency(tax80c.elss)}`}
              </div>
            </>
          ) : (
            <div className="value">{formatCurrency(data.elss_tax_saving)}</div>
          )}
        </div>
        <div className="stat-card">
          <div className="label">Diversification<InfoTooltip tip="Percentage of your portfolio spread across different asset classes. Higher = more diversified" /></div>
          <div className="value">{(data.risk_metrics?.diversification ?? 0).toFixed(0)}%</div>
        </div>
      </div>

      {benchmarks.length > 0 && (
        <div className="card" style={{ marginBottom: 20 }}>
          <h3 style={{ marginBottom: 16 }}>Market Benchmarks</h3>
          <div className="table-container" style={{ boxShadow: 'none' }}>
            <table>
              <thead>
                <tr>
                  <th>Index</th>
                  <th className="text-right">Current</th>
                  <th className="text-right">1Y Return</th>
                  <th className="text-right">3Y Return</th>
                  <th className="text-right">5Y Return</th>
                </tr>
              </thead>
              <tbody>
                {benchmarks.map(b => {
                  const has = (k: string) => !b.available || b.available.includes(k);
                  const cell = (k: string, change: number, cagr?: number) => has(k)
                    ? <td className={`text-right ${change >= 0 ? 'text-success' : 'text-danger'}`} title={cagr !== undefined ? `${cagr.toFixed(1)}% per year` : undefined}>
                        {change.toFixed(1)}%{cagr !== undefined && <div className="text-muted" style={{ fontSize: 11 }}>{cagr.toFixed(1)}% p.a.</div>}
                      </td>
                    : <td className="text-right text-muted">—</td>;
                  return (
                    <tr key={b.symbol}>
                      <td>
                        {b.index_name}
                        <div className="text-muted" style={{ fontSize: 11 }}>{b.symbol}{b.as_of ? ` · ${formatDate(b.as_of)}` : ''}</div>
                      </td>
                      <td className="text-right" style={{ fontWeight: 600 }}>{b.current?.toLocaleString('en-IN', { maximumFractionDigits: 0 })}</td>
                      {cell('1y', b.change_1y)}
                      {cell('3y', b.change_3y, b.cagr_3y)}
                      {cell('5y', b.change_5y, b.cagr_5y)}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Charts Row */}
      <div className="chart-grid">
        {/* Asset Allocation Pie Chart */}
        {total > 0 && (
          <div className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <h3 style={{ margin: 0 }}>{drillCategory ? `${LABELS[drillCategory] || drillCategory} Breakdown` : 'Asset Allocation'}</h3>
              {drillCategory && (
                <button className="btn btn-sm btn-outline" onClick={() => setDrillCategory(null)}>← Back</button>
              )}
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <PieChart>
                <Pie
                  data={drillCategory && drillData.length > 0 ? drillData : pieData}
                  cx="50%" cy="50%" outerRadius={100} innerRadius={50} dataKey="value"
                  label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`} labelLine={true}
                  onClick={(_: unknown, index: number) => {
                    if (!drillCategory && pieData[index]) {
                      const key = pieData[index].key;
                      if (key === 'mutual_funds' || key === 'stocks') {
                        setDrillCategory(key);
                      }
                    }
                  }}
                  style={{ cursor: drillCategory ? 'default' : 'pointer' }}
                >
                  {(drillCategory && drillData.length > 0 ? drillData : pieData).map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(val) => formatCurrency(Number(val))} />
              </PieChart>
            </ResponsiveContainer>
            {!drillCategory && <p className="text-muted" style={{ fontSize: 11, textAlign: 'center', marginTop: 4 }}>Click Mutual Funds or Stocks to drill down</p>}
          </div>
        )}

        {/* Contribution vs Gains (Stacked) */}
        {summaryBarData.length > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Contribution vs Gains</h3>
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={summaryBarData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" fontSize={12} />
                <YAxis tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <Tooltip formatter={(val) => formatCurrency(Number(val))} />
                <Legend />
                <Bar dataKey="invested" name="Contributed" stackId="a" fill="#1a73e8" radius={[0, 0, 0, 0]} />
                <Bar dataKey="gain" name="Gains" stackId="a" fill="#0f9d58" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* MF Performance Charts */}
      {mfBarData.length > 0 && (
        <div className="chart-grid">
          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Mutual Fund Performance</h3>
            <ResponsiveContainer width="100%" height={Math.max(280, mfBarData.length * 40)}>
              <BarChart data={mfBarData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <YAxis type="category" dataKey="name" width={130} fontSize={11} />
                <Tooltip formatter={(val) => formatCurrency(Number(val))} />
                <Legend />
                <Bar dataKey="invested" name="Invested" fill="#1a73e8" radius={[0, 4, 4, 0]} />
                <Bar dataKey="current" name="Current Value" fill="#0f9d58" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Fund Returns (%)</h3>
            <ResponsiveContainer width="100%" height={Math.max(280, mfGainData.length * 40)}>
              <BarChart data={mfGainData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" tickFormatter={(v) => `${v.toFixed(1)}%`} fontSize={11} />
                <YAxis type="category" dataKey="name" width={130} fontSize={11} />
                <Tooltip formatter={(val) => `${Number(val).toFixed(2)}%`} />
                <Bar dataKey="gainPercent" name="Return %" fill="#f9ab00" radius={[0, 4, 4, 0]}>
                  {mfGainData.map((entry, i) => (
                    <Cell key={i} fill={entry.gainPercent >= 0 ? '#0f9d58' : '#ea4335'} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Category Summaries */}
      <div className="card-grid">
        {[
          { label: 'Mutual Funds', s: data.mutual_fund_summary },
          { label: 'Corporate Bonds', s: data.corporate_bond_summary },
          { label: 'Fixed Deposits', s: data.fixed_deposit_summary },
          { label: 'Stocks', s: data.stock_summary },
          { label: 'Provident Fund', s: data.provident_fund_summary },
          { label: 'NPS', s: data.nps_summary },
          { label: 'Vehicle Rentals', s: data.vehicle_rental_summary },
        ].filter(({ s }) => s && s.count > 0).map(({ label, s }) => s && (
          <div key={label} className="card">
            <h3 style={{ marginBottom: 8 }}>{label}</h3>
            <div className="text-muted" style={{ fontSize: 13 }}>{s.count} holding(s)</div>
            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Invested: </span>
              <strong>{formatCurrency(s.total_invested)}</strong>
            </div>
            <div>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Value: </span>
              <strong>{formatCurrency(s.current_value)}</strong>
            </div>
            {!!s.income_received && (
              <div>
                <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Cash returned: </span>
                <strong>{formatCurrency(s.income_received)}</strong>
              </div>
            )}
          </div>
        ))}
        {data.home_loan_summary && data.home_loan_summary.count > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 8 }}>Home Loans</h3>
            <div className="text-muted" style={{ fontSize: 13 }}>{data.home_loan_summary.count} loan(s)</div>
            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Outstanding: </span>
              <strong>{formatCurrency(data.home_loan_summary.total_outstanding)}</strong>
            </div>
            <div>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Monthly EMI: </span>
              <strong>{formatCurrency(data.home_loan_summary.monthly_emi)}</strong>
            </div>
          </div>
        )}
        {data.personal_loan_summary && data.personal_loan_summary.count > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 8 }}>Personal Loans</h3>
            <div className="text-muted" style={{ fontSize: 13 }}>{data.personal_loan_summary.count} loan(s)</div>
            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Outstanding: </span>
              <strong>{formatCurrency(data.personal_loan_summary.total_outstanding)}</strong>
            </div>
            <div>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Monthly EMI: </span>
              <strong>{formatCurrency(data.personal_loan_summary.monthly_emi)}</strong>
            </div>
          </div>
        )}
        {data.credit_card_summary && data.credit_card_summary.count > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 8 }}>Credit Cards</h3>
            <div className="text-muted" style={{ fontSize: 13 }}>{data.credit_card_summary.count} card(s)</div>
            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Outstanding: </span>
              <strong className="negative">{formatCurrency(data.credit_card_summary.total_outstanding)}</strong>
            </div>
            <div>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Utilization: </span>
              <strong>{data.credit_card_summary.avg_utilization.toFixed(0)}%</strong>
            </div>
          </div>
        )}
      </div>

      {data.upcoming_payouts && data.upcoming_payouts.length > 0 && (
        <div className="card">
          <h3 style={{ marginBottom: 12 }}>Upcoming Payouts (Next 30 Days)</h3>
          <ul className="payout-list">
            {data.upcoming_payouts.map((p, i) => (
              <li key={i} className="payout-item">
                <div>
                  <div style={{ fontWeight: 500, fontSize: 14 }}>{p.name}</div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{p.type} - {p.payout_type}</div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontWeight: 600 }}>{formatCurrency(p.amount)}</div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{formatDate(p.date)}</div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {insightsData && insightsData.insights.length > 0 && (
        <div className="card" style={{ marginTop: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <h3>Key Insights</h3>
            <Link to="/insights" style={{ fontSize: 13, color: 'var(--primary)', textDecoration: 'none' }}>View All ({insightsData.insights.length}) →</Link>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {insightsData.insights.slice(0, 5).map(ins => {
              const colors: Record<string, { color: string; icon: string }> = {
                critical: { color: 'var(--danger-text)', icon: '🚨' },
                warning: { color: 'var(--warning-text)', icon: '⚠️' },
                info: { color: 'var(--primary-text)', icon: 'ℹ️' },
                positive: { color: 'var(--success-text)', icon: '✅' },
              };
              const c = colors[ins.severity] || colors.info;
              return (
                <div key={ins.id} style={{ display: 'flex', gap: 8, alignItems: 'flex-start', padding: '8px 12px', borderRadius: 6, background: 'var(--surface-muted)', borderLeft: `3px solid ${c.color}` }}>
                  <span>{c.icon}</span>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600 }}>{ins.title}</div>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{ins.description}</div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
