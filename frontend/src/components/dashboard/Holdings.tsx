import { Link } from 'react-router-dom';
import { ArrowDownRight, ArrowUpRight, CreditCard, HandCoins, House, Plus } from 'lucide-react';
import type { DashboardSummary, InvestmentSummary } from '../../types';
import { formatCurrency, formatPercent } from '../../utils/format';
import { INVESTMENT_CATEGORIES } from './categories';

const SUMMARY_FIELD: Record<string, keyof DashboardSummary> = {
  mutual_funds: 'mutual_fund_summary',
  stocks: 'stock_summary',
  fixed_deposits: 'fixed_deposit_summary',
  corporate_bonds: 'corporate_bond_summary',
  provident_fund: 'provident_fund_summary',
  nps: 'nps_summary',
  alternate_investments: 'alternate_investment_summary',
};

export function HoldingsGrid({ data }: { data: DashboardSummary }) {
  const items = INVESTMENT_CATEGORIES
    .map(meta => ({ meta, s: data[SUMMARY_FIELD[meta.key]] as InvestmentSummary | undefined }))
    .filter(({ s }) => s && s.count > 0)
    .sort((a, b) => (b.s!.current_value) - (a.s!.current_value));
  const totalValue = items.reduce((t, { s }) => t + (s?.current_value || 0), 0);

  if (items.length === 0) {
    return (
      <div className="card onboarding">
        <h3>Start tracking your investments</h3>
        <p className="text-muted">Add your first holding and FinBoard will show returns, allocation and trends here.</p>
        <div className="onboarding-actions">
          {INVESTMENT_CATEGORIES.slice(0, 4).map(c => (
            <Link key={c.key} to={c.to} className="btn btn-outline"><Plus size={15} /> {c.label}</Link>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="holdings-grid">
      {items.map(({ meta, s }) => {
        const summary = s!;
        const gain = summary.current_value + (summary.income_received || 0) - summary.total_invested;
        const gainPct = summary.total_invested > 0 ? (gain / summary.total_invested) * 100 : 0;
        const share = totalValue > 0 ? (summary.current_value / totalValue) * 100 : 0;
        const Icon = meta.icon;
        const up = gain >= 0;
        return (
          <Link key={meta.key} to={meta.to} className="holding-card">
            <div className="holding-head">
              <span className="tile-icon" style={{ background: `${meta.color}1f`, color: meta.color }}><Icon size={17} aria-hidden="true" /></span>
              <div style={{ minWidth: 0 }}>
                <div className="holding-name">{meta.label}</div>
                <div className="holding-count">{summary.count} holding{summary.count === 1 ? '' : 's'}</div>
              </div>
              {summary.total_invested > 0 && (
                <span className={`pill ${up ? 'pill-up' : 'pill-down'} ml-auto`}>
                  {up ? <ArrowUpRight size={12} /> : <ArrowDownRight size={12} />}{formatPercent(gainPct)}
                </span>
              )}
            </div>
            <div className="holding-value">{formatCurrency(summary.current_value)}</div>
            <div className="holding-sub">
              Invested {formatCurrency(summary.total_invested)}
              {!!summary.income_received && <> · Received {formatCurrency(summary.income_received)}</>}
            </div>
            <div className="meter meter-thin" title={`${share.toFixed(1)}% of your investments`}>
              <span style={{ width: `${share}%`, background: meta.color }} />
            </div>
            <div className="holding-share">{share.toFixed(1)}% of portfolio</div>
          </Link>
        );
      })}
    </div>
  );
}

export function LiabilitiesGrid({ data }: { data: DashboardSummary }) {
  const loans = [
    { key: 'home', label: 'Home loans', icon: House, to: '/home-loans', s: data.home_loan_summary, noun: 'loan' },
    { key: 'personal', label: 'Personal loans', icon: HandCoins, to: '/personal-loans', s: data.personal_loan_summary, noun: 'loan' },
  ].filter(l => l.s && l.s.count > 0);
  const cards = data.credit_card_summary && data.credit_card_summary.count > 0 ? data.credit_card_summary : null;
  if (loans.length === 0 && !cards) return null;

  return (
    <div className="holdings-grid">
      {loans.map(({ key, label, icon: Icon, to, s, noun }) => (
        <Link key={key} to={to} className="holding-card">
          <div className="holding-head">
            <span className="tile-icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger-text)' }}><Icon size={17} aria-hidden="true" /></span>
            <div>
              <div className="holding-name">{label}</div>
              <div className="holding-count">{s.count} {noun}{s.count === 1 ? '' : 's'}</div>
            </div>
          </div>
          <div className="holding-value">{formatCurrency(s.total_outstanding)}</div>
          <div className="holding-sub">Outstanding · EMI {formatCurrency(s.monthly_emi)}/month</div>
          {s.total_disbursed > 0 && (
            <>
              <div className="meter meter-thin" title="Share of the loan already repaid">
                <span style={{ width: `${Math.max(0, Math.min(100, (1 - s.total_outstanding / s.total_disbursed) * 100))}%`, background: 'var(--success)' }} />
              </div>
              <div className="holding-share">{Math.max(0, (1 - s.total_outstanding / s.total_disbursed) * 100).toFixed(0)}% repaid</div>
            </>
          )}
        </Link>
      ))}
      {cards && (
        <Link to="/credit-cards" className="holding-card">
          <div className="holding-head">
            <span className="tile-icon" style={{ background: 'var(--danger-soft)', color: 'var(--danger-text)' }}><CreditCard size={17} aria-hidden="true" /></span>
            <div>
              <div className="holding-name">Credit cards</div>
              <div className="holding-count">{cards.count} card{cards.count === 1 ? '' : 's'}</div>
            </div>
          </div>
          <div className="holding-value">{formatCurrency(cards.total_outstanding)}</div>
          <div className="holding-sub">Outstanding of {formatCurrency(cards.total_limit)} limit</div>
          <div className="meter meter-thin" title="Credit utilisation">
            <span style={{ width: `${Math.min(100, cards.avg_utilization)}%`, background: cards.avg_utilization > 30 ? 'var(--danger)' : 'var(--success)' }} />
          </div>
          <div className="holding-share">{cards.avg_utilization.toFixed(0)}% utilisation{cards.avg_utilization > 30 ? ' · aim for under 30%' : ''}</div>
        </Link>
      )}
    </div>
  );
}
