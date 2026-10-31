import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, ChartColumn, CircleCheck, Receipt, ShieldCheck, TriangleAlert, Wallet, type LucideIcon } from 'lucide-react';
import type { FinanceOverview } from '../api/finance';
import { formatCompactINR, formatCurrency, formatDate } from '../utils/format';

function Tile({ icon: Icon, tint, title, to, children }: { icon: LucideIcon; tint: string; title: string; to: string; children: ReactNode }) {
  return (
    <section className="tile" aria-label={title}>
      <div className="tile-head">
        <span className="tile-icon" style={{ background: `${tint}1f`, color: tint }}><Icon size={17} aria-hidden="true" /></span>
        <span className="tile-title">{title}</span>
        <Link to={to} className="tile-more" aria-label={`Open ${title}`}><ArrowRight size={16} /></Link>
      </div>
      {children}
    </section>
  );
}

export default function MoneyOverview({ data, error, onRetry }: { data: FinanceOverview | null; error?: string; onRetry?: () => void }) {
  if (error) {
    return (
      <div className="card flex items-center gap-3 justify-between flex-wrap" role="alert">
        <span>{error}</span>
        {onRetry && <button className="btn btn-secondary btn-sm" onClick={onRetry}>Retry</button>}
      </div>
    );
  }
  if (!data) return null;

  const saved = data.month_income - data.month_expense;
  const spentPct = data.month_income > 0 ? Math.min(100, (data.month_expense / data.month_income) * 100) : 0;
  const flagged = (data.budgets || []).filter(b => b.status === 'over' || b.status === 'warning');
  const nextPremium = (data.upcoming_premiums || [])[0];

  return (
    <div className="money-grid">
      <Tile icon={Wallet} tint="#10b981" title="Bank & cash" to="/accounts">
        <div className="tile-value">{formatCurrency(data.liquid_balance)}</div>
        <div className="tile-sub">Bank {formatCompactINR(data.bank_balance)} · Cash {formatCompactINR(data.cash_in_hand)}</div>
        {data.stale_balances > 0 ? (
          <div className="tile-note text-warning">
            <TriangleAlert size={14} /> {data.stale_balances} balance{data.stale_balances === 1 ? '' : 's'} not updated in 30+ days
          </div>
        ) : (
          <div className="tile-note text-muted">Recorded balances, not a live bank feed</div>
        )}
      </Tile>

      <Tile icon={Receipt} tint="#6366f1" title="This month" to="/ledger">
        {data.month_income === 0 && data.month_expense === 0 ? (
          <>
            <div className="tile-value text-muted">No activity</div>
            <div className="tile-sub">No confirmed income or expenses yet</div>
          </>
        ) : (
          <>
            <div className={`tile-value ${saved < 0 ? 'negative' : ''}`}>{saved < 0 ? '-' : ''}{formatCurrency(Math.abs(saved))}<span className="tile-unit">{saved < 0 ? 'overspent' : 'saved'}</span></div>
            <div className="tile-sub">In {formatCompactINR(data.month_income)} · Out {formatCompactINR(data.month_expense)}</div>
            <div className="meter meter-thin mt-2" title={`${spentPct.toFixed(0)}% of income spent`}>
              <span style={{ width: `${spentPct}%`, background: spentPct > 90 ? 'var(--danger)' : 'var(--primary)' }} />
            </div>
            <div className="tile-sub mt-1">
              Savings rate {data.month_income > 0 ? `${data.month_savings_rate_pct.toFixed(0)}%` : 'N/A'}
            </div>
          </>
        )}
        {data.pending_review > 0 && (
          <Link to="/ledger" className="tile-note">{data.pending_review} transaction{data.pending_review === 1 ? '' : 's'} need review</Link>
        )}
      </Tile>

      <Tile icon={ChartColumn} tint="#f59e0b" title="Budgets" to="/budgets">
        {(data.budgets || []).length === 0 ? (
          <>
            <div className="tile-value text-muted">Not set</div>
            <div className="tile-sub"><Link to="/budgets">Set category limits</Link> to get alerts</div>
          </>
        ) : flagged.length === 0 ? (
          <>
            <div className="tile-value" style={{ color: 'var(--success-text)' }}>On track</div>
            <div className="tile-note text-muted"><CircleCheck size={14} /> All budgets within limits</div>
          </>
        ) : (
          <>
            <div className={`tile-value ${data.budgets_over > 0 ? 'negative' : ''}`} style={data.budgets_over > 0 ? undefined : { color: 'var(--warning-text)' }}>
              {data.budgets_over > 0 ? `${data.budgets_over} over` : `${data.budgets_warning} near limit`}
            </div>
            <ul className="tile-list">
              {flagged.slice(0, 3).map(b => {
                const pct = b.limit > 0 ? (b.spent / b.limit) * 100 : 0;
                return (
                  <li key={b.category}>
                    <div className="tile-list-row"><span>{b.category}</span><span>{formatCompactINR(b.spent)} / {formatCompactINR(b.limit)}</span></div>
                    <div className="meter meter-thin"><span style={{ width: `${Math.min(100, pct)}%`, background: b.status === 'over' ? 'var(--danger)' : 'var(--warning)' }} /></div>
                  </li>
                );
              })}
            </ul>
          </>
        )}
      </Tile>

      <Tile icon={ShieldCheck} tint="#8b5cf6" title="Insurance" to="/insurance">
        {data.life_cover === 0 && data.health_cover === 0 ? (
          <>
            <div className="tile-value text-muted">No cover</div>
            <div className="tile-sub"><Link to="/insurance">Add your policies</Link> to track cover and premiums</div>
          </>
        ) : (
          <>
            <div className="tile-value">{formatCompactINR(data.life_cover)}<span className="tile-unit">life</span></div>
            <div className="tile-sub">{formatCompactINR(data.health_cover)} health cover</div>
          </>
        )}
        <div className="tile-note text-muted">
          {nextPremium
            ? <>Next premium {formatCurrency(nextPremium.amount)} on {formatDate(nextPremium.date)}</>
            : 'No premiums due in 30 days'}
        </div>
      </Tile>
    </div>
  );
}
