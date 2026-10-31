import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Clock, TriangleAlert } from 'lucide-react';
import { getDashboard } from '../api/dashboard';
import { getMutualFunds } from '../api/mutualFunds';
import { getBenchmarks } from '../api/benchmarks';
import { getInsights, type InsightsResponse } from '../api/insights';
import { getStocks } from '../api/stocks';
import { getFinanceOverview, type FinanceOverview } from '../api/finance';
import { getNetWorthHistory } from '../api/netWorth';
import { useAuth } from '../context/AuthContext';
import type { BenchmarkData, DashboardSummary, MutualFund, NetWorthSnapshot, Stock } from '../types';
import { formatDate } from '../utils/format';
import { apiErrorMessage } from '../utils/errors';
import MoneyOverview from '../components/MoneyOverview';
import { InvestmentsCard, NetWorthHero } from '../components/dashboard/Hero';
import HealthRow from '../components/dashboard/HealthRow';
import PortfolioHistoryChart from '../components/dashboard/PortfolioHistoryChart';
import AllocationChart from '../components/dashboard/AllocationChart';
import { ContributionChart, FundPerformanceChart } from '../components/dashboard/PerformanceCharts';
import { HoldingsGrid, LiabilitiesGrid } from '../components/dashboard/Holdings';
import { BenchmarksCard, InsightsCard, PayoutsCard } from '../components/dashboard/Lists';

const STALE_AFTER_MS = 4 * 24 * 3600 * 1000; // covers weekends and market holidays

export default function DashboardPage() {
  const { viewAsUserId, user } = useAuth();
  return <Dashboard key={viewAsUserId || user?.id} />;
}

function greeting(hour: number) {
  if (hour < 12) return 'Good morning';
  if (hour < 17) return 'Good afternoon';
  return 'Good evening';
}

function SectionTitle({ title, to, linkLabel }: { title: string; to?: string; linkLabel?: string }) {
  return (
    <div className="section-title">
      <h2>{title}</h2>
      {to && <Link to={to} className="card-link">{linkLabel} <ArrowRight size={14} /></Link>}
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading dashboard">
      <div className="hero-grid">
        <div className="skeleton" style={{ height: 250, borderRadius: 16 }} />
        <div className="skeleton" style={{ height: 250, borderRadius: 12 }} />
      </div>
      <div className="kpi-grid">
        {[0, 1, 2].map(i => <div key={i} className="skeleton" style={{ height: 140, borderRadius: 12 }} />)}
      </div>
      <div className="money-grid" style={{ marginTop: 40 }}>
        {[0, 1, 2, 3].map(i => <div key={i} className="skeleton" style={{ height: 150, borderRadius: 12 }} />)}
      </div>
      <div className="dash-grid-2-1" style={{ marginTop: 40 }}>
        <div className="skeleton" style={{ height: 340, borderRadius: 12 }} />
        <div className="skeleton" style={{ height: 340, borderRadius: 12 }} />
      </div>
    </div>
  );
}

function Dashboard() {
  const { user } = useAuth();
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [finance, setFinance] = useState<FinanceOverview | null>(null);
  const [financeError, setFinanceError] = useState('');
  const [history, setHistory] = useState<NetWorthSnapshot[]>([]);
  const [funds, setFunds] = useState<MutualFund[]>([]);
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [benchmarks, setBenchmarks] = useState<BenchmarkData[]>([]);
  const [insights, setInsights] = useState<InsightsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [renderedAt] = useState(() => Date.now());

  const loadFinance = useCallback(() =>
    getFinanceOverview()
      .then(r => { setFinance(r.data); setFinanceError(''); })
      .catch(err => setFinanceError(apiErrorMessage(err, 'Failed to load money overview'))),
  []);

  useEffect(() => {
    Promise.all([
      getDashboard().then(r => setData(r.data)).catch(() => {}),
      loadFinance(),
      getNetWorthHistory().then(r => setHistory(Array.isArray(r.data) ? r.data : [])).catch(() => {}),
      getMutualFunds().then(r => setFunds(Array.isArray(r.data) ? r.data : [])).catch(() => {}),
      getStocks().then(r => setStocks(Array.isArray(r.data) ? r.data : [])).catch(() => {}),
      getBenchmarks().then(r => setBenchmarks(r.data?.indices || [])).catch(() => {}),
      getInsights().then(r => setInsights(r)).catch(() => {}),
    ]).finally(() => setLoading(false));
  }, [loadFinance]);

  const retryFinance = () => {
    setFinanceError('');
    loadFinance();
  };

  // How fresh the market data behind current values is
  const freshness = data?.data_freshness;
  const isStale = (iso: string) => renderedAt - new Date(iso).getTime() > STALE_AFTER_MS;
  const freshnessParts: { label: string; stale: boolean }[] = [];
  if (freshness?.has_mutual_funds) {
    freshnessParts.push(freshness.navs_as_of
      ? { label: `NAVs ${formatDate(freshness.navs_as_of)}${freshness.navs_missing ? ` (${freshness.navs_missing} never fetched)` : ''}`, stale: isStale(freshness.navs_as_of) || freshness.navs_missing > 0 }
      : { label: 'NAVs never fetched', stale: true });
  }
  if (freshness?.has_stocks) {
    freshnessParts.push(freshness.prices_as_of
      ? { label: `Stock prices ${formatDate(freshness.prices_as_of)}${freshness.prices_missing ? ` (${freshness.prices_missing} never fetched)` : ''}`, stale: isStale(freshness.prices_as_of) || freshness.prices_missing > 0 }
      : { label: 'Stock prices never fetched', stale: true });
  }
  const anyStale = freshnessParts.some(p => p.stale);

  const firstName = user?.name?.split(' ')[0];
  const hasLiabilities = !!data && (
    (data.home_loan_summary?.count || 0) + (data.personal_loan_summary?.count || 0) + (data.credit_card_summary?.count || 0) > 0
  );
  const hasFunds = funds.some(f => f.total_invested > 0);

  return (
    <div className="dashboard">
      <div className="dash-header">
        <div>
          <h1>{greeting(new Date(renderedAt).getHours())}{firstName ? `, ${firstName}` : ''}</h1>
          <p>Here's where your money stands today.</p>
        </div>
        {!loading && data && (
          <span
            className={`freshness-chip${anyStale ? ' stale' : ''}`}
            title={anyStale ? 'Some market prices are out of date. Refresh them from the Mutual Funds or Stocks page.' : 'Dates of the latest market prices behind current values'}
          >
            {anyStale ? <TriangleAlert size={13} aria-hidden="true" /> : <Clock size={13} aria-hidden="true" />}
            {freshnessParts.length === 0 ? 'Values from your recorded data' : freshnessParts.map(p => p.label).join(' · ')}
          </span>
        )}
      </div>

      {loading ? <DashboardSkeleton /> : (
        <>
          <div className="hero-grid">
            {finance ? <NetWorthHero finance={finance} history={history} /> : (
              <section className="hero-card" aria-label="Net worth">
                <div className="hero-label">Net worth</div>
                <div className="hero-value">—</div>
                <p className="mt-2 text-md" style={{ opacity: 0.85 }}>Couldn't load your net worth right now.</p>
              </section>
            )}
            {data ? <InvestmentsCard data={data} /> : (
              <section className="card" role="alert">
                <h3>Investment summary unavailable</h3>
                <p className="text-muted" style={{ margin: '8px 0 16px' }}>We couldn't load your portfolio. Check your connection and try again.</p>
                <button className="btn btn-primary" onClick={() => window.location.reload()}>Reload</button>
              </section>
            )}
          </div>

          {data && <HealthRow data={data} />}

          <SectionTitle title="Your money" to="/ledger" linkLabel="Income & expenses" />
          <MoneyOverview data={finance} error={financeError} onRetry={retryFinance} />

          {data && (
            <>
              <SectionTitle title="Portfolio" />
              <div className="dash-grid-2-1">
                <PortfolioHistoryChart history={history} />
                <AllocationChart allocation={data.asset_allocation} funds={funds} stocks={stocks} />
              </div>
              <div className={hasFunds ? 'chart-grid' : undefined}>
                <ContributionChart data={data} />
                {hasFunds && <FundPerformanceChart funds={funds} />}
              </div>

              <SectionTitle title="Holdings" />
              <HoldingsGrid data={data} />

              {hasLiabilities && (
                <>
                  <SectionTitle title="Loans & cards" />
                  <LiabilitiesGrid data={data} />
                </>
              )}

              <SectionTitle title="Coming up" />
              <div className="chart-grid">
                <PayoutsCard payouts={data.upcoming_payouts || []} />
                <InsightsCard insights={insights} />
              </div>
            </>
          )}

          <BenchmarksCard benchmarks={benchmarks} />
        </>
      )}
    </div>
  );
}
