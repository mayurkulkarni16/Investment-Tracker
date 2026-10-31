import { Link } from 'react-router-dom';
import { Area, AreaChart, ResponsiveContainer, Tooltip } from 'recharts';
import { ArrowDownRight, ArrowRight, ArrowUpRight } from 'lucide-react';
import type { FinanceOverview } from '../../api/finance';
import type { DashboardSummary, NetWorthSnapshot } from '../../types';
import { formatCompactINR, formatCurrency, formatPercent } from '../../utils/format';
import InfoTooltip from '../InfoTooltip';
import { ChartTooltip } from '../charts';
import { monthLabel } from './history';

function ChangeChip({ change, base }: { change: number; base: number }) {
  const up = change >= 0;
  const pct = base > 0 ? (change / base) * 100 : null;
  const Icon = up ? ArrowUpRight : ArrowDownRight;
  return (
    <span className="hero-chip" title="Change since last month's snapshot">
      <Icon size={14} aria-hidden="true" />
      {up ? '+' : '-'}{formatCompactINR(Math.abs(change))}
      {pct !== null && ` (${pct >= 0 ? '+' : ''}${pct.toFixed(1)}%)`}
      <span className="fw-medium" style={{ opacity: 0.85 }}>&nbsp;this month</span>
    </span>
  );
}

export function NetWorthHero({ finance, history }: { finance: FinanceOverview; history: NetWorthSnapshot[] }) {
  const spark = history.slice(-12).map(s => ({ month: s.month, value: s.net_worth }));
  const change = finance.net_worth_change_month;
  const assets = finance.total_assets;
  const liabilities = finance.total_liabilities;
  const debtShare = assets > 0 ? Math.min(100, (liabilities / assets) * 100) : 0;

  return (
    <section className="hero-card" aria-label="Net worth">
      <div className="hero-top">
        <div>
          <div className="hero-label">Net worth</div>
          <div className="hero-value">{formatCurrency(finance.net_worth)}</div>
          <div className="mt-2" style={{ minHeight: 24 }}>
            {change !== undefined && change !== null
              ? <ChangeChip change={change} base={finance.net_worth - change} />
              : <span className="hero-chip fw-medium">Monthly change shows after your first snapshot</span>}
          </div>
        </div>
        <Link to="/net-worth" className="hero-link">Details <ArrowRight size={14} aria-hidden="true" /></Link>
      </div>

      <div className="hero-meta">
        <div><span>Assets</span><strong>{formatCompactINR(assets)}</strong></div>
        <div><span>Liabilities</span><strong>{formatCompactINR(liabilities)}</strong></div>
        <div className="hero-debt">
          <span>Debt to assets</span>
          <div className="hero-debt-bar" title={`${debtShare.toFixed(0)}% of assets`}><i style={{ width: `${debtShare}%` }} /></div>
          <strong>{debtShare.toFixed(0)}%</strong>
        </div>
      </div>

      <div className="hero-spark">
        {spark.length >= 2 && (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={spark} margin={{ top: 6, right: 0, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="hero-spark-fill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#fff" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="#fff" stopOpacity={0} />
                </linearGradient>
              </defs>
              <Tooltip
                cursor={{ stroke: 'rgba(255,255,255,0.5)', strokeWidth: 1 }}
                content={<ChartTooltip labelFormatter={l => monthLabel(String(l))} />}
              />
              <Area type="monotone" dataKey="value" name="Net worth" stroke="#fff" strokeWidth={2} fill="url(#hero-spark-fill)" dot={false} activeDot={{ r: 4, fill: '#fff', stroke: '#fff' }} />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </section>
  );
}

export function InvestmentsCard({ data }: { data: DashboardSummary }) {
  const holdings = [
    data.mutual_fund_summary, data.stock_summary, data.fixed_deposit_summary, data.corporate_bond_summary,
    data.provident_fund_summary, data.nps_summary, data.vehicle_rental_summary,
  ].reduce((n, s) => n + (s?.count || 0), 0);
  const gainUp = data.total_gains >= 0;

  return (
    <section className="card invest-card" aria-label="Investments">
      <div className="card-head">
        <h3>Investments</h3>
        <span className="text-muted" style={{ fontSize: 12.5 }}>{holdings} holding{holdings === 1 ? '' : 's'}</span>
      </div>
      <div className="big-number">{formatCurrency(data.current_value)}</div>
      <div className="mt-1 mb-4">
        <span className={`pill ${gainUp ? 'pill-up' : 'pill-down'}`}>
          {gainUp ? <ArrowUpRight size={13} /> : <ArrowDownRight size={13} />}
          {formatPercent(data.overall_return_percent)} overall
        </span>
      </div>
      <div className="metric-list">
        <div className="metric-row"><span className="k">Invested</span><span className="v">{formatCurrency(data.total_invested)}</span></div>
        <div className="metric-row">
          <span className="k">Total gains<InfoTooltip tip="Current value minus amount invested, plus cash already paid out to you (bond interest and principal, FD interest and maturities, rental payouts)" /></span>
          <span className={`v ${gainUp ? 'positive' : 'negative'}`}>{gainUp ? '+' : ''}{formatCurrency(data.total_gains)}</span>
        </div>
        <div className="metric-row">
          <span className="k">Portfolio XIRR<InfoTooltip tip="Extended Internal Rate of Return — annualized return accounting for the timing of each cash flow" /></span>
          <span className={`v ${data.portfolio_xirr >= 0 ? 'positive' : 'negative'}`}>{formatPercent(data.portfolio_xirr)}</span>
        </div>
      </div>
    </section>
  );
}
