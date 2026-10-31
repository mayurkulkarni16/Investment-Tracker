import { Link } from 'react-router-dom';
import { ArrowRight, CalendarCheck, Sparkles } from 'lucide-react';
import type { InsightsResponse } from '../../api/insights';
import type { BenchmarkData, UpcomingPayout } from '../../types';
import { formatCurrency, formatDate } from '../../utils/format';
import SeverityIcon from '../SeverityIcon';
import { ChartEmpty } from '../charts';

const SEVERITY_BG: Record<string, string> = {
  critical: 'var(--danger-soft)',
  warning: 'var(--warning-soft)',
  info: 'var(--primary-soft)',
  positive: 'var(--success-soft)',
};

function DateChip({ date }: { date: string }) {
  const d = new Date(date);
  return (
    <span className="date-chip" aria-hidden="true">
      <b>{d.getDate()}</b>
      <span>{d.toLocaleDateString('en-IN', { month: 'short' })}</span>
    </span>
  );
}

export function PayoutsCard({ payouts }: { payouts: UpcomingPayout[] }) {
  const total = payouts.reduce((s, p) => s + p.amount, 0);
  return (
    <section className="card" aria-label="Upcoming payouts">
      <div className="card-head">
        <div>
          <h3>Upcoming payouts</h3>
          <div className="card-subtitle">Next 30 days{payouts.length > 0 && <> · {formatCurrency(total)} expected</>}</div>
        </div>
      </div>
      {payouts.length === 0 ? (
        <ChartEmpty icon={CalendarCheck} title="Nothing due in the next 30 days">Bond coupons, FD maturities and rental payouts appear here.</ChartEmpty>
      ) : (
        <ul className="row-list">
          {payouts.map((p, i) => (
            <li key={`${p.name}-${p.date}-${i}`} className="list-row">
              <DateChip date={p.date} />
              <div className="list-main">
                <div className="list-title">{p.name}</div>
                <div className="list-sub">{p.type} · <span className="capitalize">{p.payout_type.replace(/_/g, ' ')}</span> · {formatDate(p.date)}</div>
              </div>
              <div className="list-amount positive">+{formatCurrency(p.amount)}</div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function InsightsCard({ insights }: { insights: InsightsResponse | null }) {
  const items = insights?.insights || [];
  return (
    <section className="card" aria-label="Key insights">
      <div className="card-head">
        <div>
          <h3>Key insights</h3>
          {items.length > 0 && insights && (
            <div className="card-subtitle">
              {insights.critical_count > 0 && <span className="text-danger">{insights.critical_count} critical · </span>}
              {insights.warning_count > 0 && <span className="text-warning">{insights.warning_count} to review · </span>}
              {items.length} total
            </div>
          )}
        </div>
        {items.length > 0 && <Link to="/insights" className="card-link">View all <ArrowRight size={14} /></Link>}
      </div>
      {items.length === 0 ? (
        <ChartEmpty icon={Sparkles} title="No insights right now">Suggestions appear as your portfolio data grows.</ChartEmpty>
      ) : (
        <ul className="row-list">
          {items.slice(0, 5).map(ins => (
            <li key={ins.id} className="list-row">
              <span className="row-icon" style={{ background: SEVERITY_BG[ins.severity] || SEVERITY_BG.info }}>
                <SeverityIcon severity={ins.severity} size={16} />
              </span>
              <div className="list-main">
                <div className="list-title">{ins.title}</div>
                <div className="list-sub">{ins.description}</div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function BenchmarksCard({ benchmarks }: { benchmarks: BenchmarkData[] }) {
  if (benchmarks.length === 0) return null;
  return (
    <section className="card" aria-label="Market benchmarks">
      <div className="card-head">
        <div>
          <h3>Market benchmarks</h3>
          <div className="card-subtitle">Index returns to compare your portfolio against</div>
        </div>
      </div>
      <div className="table-container flat">
        <table>
          <thead>
            <tr>
              <th>Index</th>
              <th className="text-right">Level</th>
              <th className="text-right">1Y</th>
              <th className="text-right">3Y</th>
              <th className="text-right">5Y</th>
            </tr>
          </thead>
          <tbody>
            {benchmarks.map(b => {
              const has = (k: string) => !b.available || b.available.includes(k);
              const cell = (k: string, change: number, cagr?: number) => has(k)
                ? (
                  <td className="text-right" title={cagr !== undefined ? `${cagr.toFixed(1)}% per year` : undefined}>
                    <span className={`${change >= 0 ? 'text-success' : 'text-danger'} fw-semibold`}>{change >= 0 ? '+' : ''}{change.toFixed(1)}%</span>
                    {cagr !== undefined && <div className="text-muted text-xs">{cagr.toFixed(1)}% p.a.</div>}
                  </td>
                )
                : <td className="text-right text-muted">—</td>;
              return (
                <tr key={b.symbol}>
                  <td>
                    <div className="fw-semibold">{b.index_name}</div>
                    <div className="text-muted text-xs">{b.symbol}{b.as_of ? ` · ${formatDate(b.as_of)}` : ''}</div>
                  </td>
                  <td className="text-right fw-semibold">{b.current?.toLocaleString('en-IN', { maximumFractionDigits: 0 })}</td>
                  {cell('1y', b.change_1y)}
                  {cell('3y', b.change_3y, b.cagr_3y)}
                  {cell('5y', b.change_5y, b.cagr_5y)}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
