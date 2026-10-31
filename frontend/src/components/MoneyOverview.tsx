import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getFinanceOverview, type FinanceOverview } from '../api/finance';
import { formatCurrency, formatDate, formatPercent } from '../utils/format';
import { apiErrorMessage } from '../utils/errors';

export default function MoneyOverview() {
  const [data, setData] = useState<FinanceOverview | null>(null);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    getFinanceOverview()
      .then(response => { if (active) setData(response.data); })
      .catch(err => {
        if (active) setError(apiErrorMessage(err, 'Failed to load money overview'));
      });
    return () => { active = false; };
  }, [attempt]);

  return (
    <section aria-label="Money overview" style={{ marginBottom: 24 }}>
      <h2>Money overview</h2>
      {error ? (
        <div className="card" role="alert">
          <p>{error}</p>
          <button className="btn btn-secondary" onClick={() => {
            setError('');
            setAttempt(value => value + 1);
          }}>Retry</button>
        </div>
      ) : !data ? (
        <div className="loading" role="status">Loading money overview...</div>
      ) : (
        <div className="card-grid">
          <div className="stat-card">
            <div className="label"><Link to="/accounts">Bank &amp; cash</Link></div>
            <div className="value">{formatCurrency(data.liquid_balance)}</div>
            <p className="text-muted">
              Bank / wallets: {formatCurrency(data.bank_balance)} | Cash: {formatCurrency(data.cash_in_hand)}
            </p>
            <p className="text-muted">Recorded balances, not a live bank connection.</p>
            {data.stale_balances > 0 && (
              <p style={{ color: 'var(--warning)' }}>{data.stale_balances} account balance(s) over 30 days old</p>
            )}
            <Link to="/accounts">Manage accounts</Link>
          </div>
          <div className="stat-card">
            <div className="label"><Link to="/ledger">This month's income &amp; expenses</Link></div>
            <div className="value">{formatCurrency(data.month_income)} income</div>
            <p>{formatCurrency(data.month_expense)} expenses</p>
            <p className={data.month_savings_rate_pct < 0 ? 'negative' : 'text-muted'}>
              Savings rate: {data.month_income > 0 ? formatPercent(data.month_savings_rate_pct) : 'N/A (no income recorded)'}
            </p>
            {data.month_income === 0 && data.month_expense === 0 && <p className="text-muted">No confirmed income or expenses this month.</p>}
            {data.pending_review > 0 && <Link to="/ledger">{data.pending_review} transaction(s) need review</Link>}
          </div>
          <div className="stat-card">
            <div className="label"><Link to="/budgets">Budget alerts</Link></div>
            <div className={`value ${data.budgets_over > 0 ? 'negative' : ''}`}>{data.budgets_over} over budget</div>
            <p>{data.budgets_warning} nearing their limit (80% or more)</p>
            {(data.budgets || []).length === 0 ? (
              <p className="text-muted">No budgets set. <Link to="/budgets">Set category limits</Link>.</p>
            ) : data.budgets_over === 0 && data.budgets_warning === 0 ? (
              <p className="text-muted">All budgets are within their limits.</p>
            ) : (
              <ul>
                {(data.budgets || []).filter(b => b.status === 'over' || b.status === 'warning').map(b => (
                  <li key={b.category}>{b.category}: {formatCurrency(b.spent)} / {formatCurrency(b.limit)}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="stat-card">
            <div className="label"><Link to="/insurance">Insurance cover</Link></div>
            <div className="value">{formatCurrency(data.life_cover)} life</div>
            <p>{formatCurrency(data.health_cover)} health</p>
            {data.life_cover === 0 && data.health_cover === 0 && <p className="text-muted">No life or health cover recorded. <Link to="/insurance">Review policies</Link>.</p>}
            <div className="label">Premiums due in the next 30 days</div>
            {(data.upcoming_premiums || []).length === 0 ? (
              <p className="text-muted">No upcoming premiums.</p>
            ) : (
              <ul>
                {(data.upcoming_premiums || []).map(p => (
                  <li key={`${p.policy_id}-${p.date}`}>
                    {p.policy_name || p.insurer}: {formatCurrency(p.amount)} on {formatDate(p.date)}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </section>
  );
}
