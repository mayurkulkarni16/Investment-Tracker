import { useEffect, useState } from 'react';
import { getDashboard } from '../api/dashboard';
import { getMutualFunds } from '../api/mutualFunds';
import { getBenchmarks } from '../api/benchmarks';
import type { DashboardSummary, MutualFund, BenchmarkData } from '../types';
import { formatCurrency, formatPercent, formatDate } from '../utils/format';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Legend } from 'recharts';

const COLORS = ['#1a73e8', '#0f9d58', '#f9ab00', '#ea4335', '#9c27b0', '#00bcd4', '#ff5722', '#607d8b'];
const LABELS: Record<string, string> = {
  mutual_funds: 'Mutual Funds',
  corporate_bonds: 'Corporate Bonds',
  fixed_deposits: 'Fixed Deposits',
  provident_fund: 'Provident Fund',
  stocks: 'Stocks',
  home_loans: 'Home Loans',
  personal_loans: 'Personal Loans',
};

export default function DashboardPage() {
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [funds, setFunds] = useState<MutualFund[]>([]);
  const [benchmarks, setBenchmarks] = useState<BenchmarkData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      getDashboard().then(r => setData(r.data)).catch(() => {}),
      getMutualFunds().then(r => setFunds(r.data || [])).catch(() => {}),
      getBenchmarks().then(r => setBenchmarks(r.data?.indices || [])).catch(() => {}),
    ]).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="loading">Loading dashboard...</div>;
  if (!data) return <div className="empty-state"><h3>No data yet</h3><p>Add some investments to see your dashboard.</p></div>;

  const allocationEntries = data.asset_allocation ? Object.entries(data.asset_allocation) : [];
  const total = allocationEntries.reduce((s, [, v]) => s + v, 0);

  // Pie chart data
  const pieData = allocationEntries.map(([key, val]) => ({
    name: LABELS[key] || key,
    value: val,
  }));

  // MF performance bar chart
  const mfBarData = funds
    .filter(f => f.total_invested > 0)
    .map(f => ({
      name: f.fund_name.length > 20 ? f.fund_name.substring(0, 20) + '...' : f.fund_name,
      invested: f.total_invested,
      current: f.current_value,
      gain: f.gain_loss,
    }));

  // Investment summary bar chart
  const summaryBarData = [
    { name: 'Mutual Funds', invested: data.mutual_fund_summary.total_invested, value: data.mutual_fund_summary.current_value },
    { name: 'Bonds', invested: data.corporate_bond_summary.total_invested, value: data.corporate_bond_summary.current_value },
    { name: 'FDs', invested: data.fixed_deposit_summary.total_invested, value: data.fixed_deposit_summary.current_value },
    { name: 'Stocks', invested: data.stock_summary.total_invested, value: data.stock_summary.current_value },
    { name: 'PF', invested: data.provident_fund_summary.total_invested, value: data.provident_fund_summary.current_value },
  ].filter(d => d.invested > 0 || d.value > 0);

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
      <div className="page-header"><h1>Dashboard</h1></div>

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
          <div className="label">Total Gains</div>
          <div className={`value ${data.total_gains >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(data.total_gains)} ({formatPercent(data.overall_return_percent)})
          </div>
        </div>
        <div className="stat-card">
          <div className="label">ELSS Tax Saving (80C)</div>
          <div className="value">{formatCurrency(data.elss_tax_saving)}</div>
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
                {benchmarks.map(b => (
                  <tr key={b.symbol}>
                    <td>
                      {b.index_name}
                      <div className="text-muted" style={{ fontSize: 11 }}>{b.symbol}</div>
                    </td>
                    <td className="text-right" style={{ fontWeight: 600 }}>{b.current?.toLocaleString('en-IN', { maximumFractionDigits: 0 })}</td>
                    <td className={`text-right ${b.change_1y >= 0 ? 'text-success' : 'text-danger'}`}>{b.change_1y?.toFixed(1)}%</td>
                    <td className={`text-right ${b.change_3y >= 0 ? 'text-success' : 'text-danger'}`}>{b.change_3y?.toFixed(1)}%</td>
                    <td className={`text-right ${b.change_5y >= 0 ? 'text-success' : 'text-danger'}`}>{b.change_5y?.toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Charts Row */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 20 }}>
        {/* Asset Allocation Pie Chart */}
        {total > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Asset Allocation</h3>
            <ResponsiveContainer width="100%" height={280}>
              <PieChart>
                <Pie data={pieData} cx="50%" cy="50%" outerRadius={100} innerRadius={50} dataKey="value" label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`} labelLine={true}>
                  {pieData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(val: number) => formatCurrency(val)} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Investment Overview Bar Chart */}
        {summaryBarData.length > 0 && (
          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Investment Overview</h3>
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={summaryBarData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" fontSize={12} />
                <YAxis tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <Tooltip formatter={(val: number) => formatCurrency(val)} />
                <Legend />
                <Bar dataKey="invested" name="Invested" fill="#1a73e8" radius={[4, 4, 0, 0]} />
                <Bar dataKey="value" name="Current Value" fill="#0f9d58" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* MF Performance Charts */}
      {mfBarData.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 20 }}>
          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Mutual Fund Performance</h3>
            <ResponsiveContainer width="100%" height={Math.max(280, mfBarData.length * 40)}>
              <BarChart data={mfBarData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <YAxis type="category" dataKey="name" width={130} fontSize={11} />
                <Tooltip formatter={(val: number) => formatCurrency(val)} />
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
                <Tooltip formatter={(val: number) => `${val.toFixed(2)}%`} />
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
        ].map(({ label, s }) => (
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
    </div>
  );
}
