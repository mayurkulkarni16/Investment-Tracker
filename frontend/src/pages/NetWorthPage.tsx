import { useEffect, useState } from 'react';
import { getNetWorthCurrent, getNetWorthHistory, takeNetWorthSnapshot } from '../api/netWorth';
import { getBenchmarks } from '../api/benchmarks';
import type { NetWorthCurrent, NetWorthSnapshot, BenchmarkData } from '../types';
import { formatCurrency, formatCompactINR } from '../utils/format';
import { useToast } from '../components/Toast';
import { SegmentedControl } from '../components/charts';
import { useAuth } from '../context/AuthContext';
import { AreaChart, Area, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, LineChart, Brush } from 'recharts';
import { PageSkeleton } from '../components/ui';
import { SERIES } from '../components/chartUtils';

export default function NetWorthPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [current, setCurrent] = useState<NetWorthCurrent | null>(null);
  const [history, setHistory] = useState<NetWorthSnapshot[]>([]);
  const [benchmarks, setBenchmarks] = useState<BenchmarkData[]>([]);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<'6m' | '1y' | '2y' | '3y' | 'all'>('all');
  const { toast } = useToast();

  const load = () => {
    Promise.all([
      getNetWorthCurrent().then(r => setCurrent(r.data)),
      getNetWorthHistory().then(r => setHistory(r.data || [])),
      getBenchmarks('5y').then(r => setBenchmarks(r.data?.indices || [])).catch(() => {}),
    ]).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, [viewAsUserId]);

  const handleSnapshot = async () => {
    try { await takeNetWorthSnapshot(); toast('Snapshot taken'); load(); }
    catch { toast('Failed to take snapshot', 'error'); }
  };

  if (loading) return <PageSkeleton label="Loading" />;

  // Filter history by range
  const filteredHistory = (() => {
    if (range === 'all' || history.length === 0) return history;
    const months = range === '6m' ? 6 : range === '1y' ? 12 : range === '2y' ? 24 : range === '3y' ? 36 : history.length;
    return history.slice(-months);
  })();

  // Net worth growth overlaid with Nifty 50 (both rebased to 0%). This is not an
  // investment-return comparison: net worth also grows with new savings and loan
  // repayments. Only meaningful when the starting net worth is positive.
  const benchmarkOverlay = (() => {
    if (filteredHistory.length < 2 || benchmarks.length === 0) return null;
    const nifty = benchmarks.find(b => b.symbol === '^NSEI');
    if (!nifty || !nifty.points || nifty.points.length === 0) return null;

    const startNW = filteredHistory[0].net_worth;
    if (startNW <= 0) return null;
    const startDate = new Date(filteredHistory[0].month + '-15');
    const niftyAt = (d: Date) => {
      let closest = nifty.points[0];
      let minDiff = Infinity;
      for (const p of nifty.points) {
        const diff = Math.abs(new Date(p.date).getTime() - d.getTime());
        if (diff < minDiff) { minDiff = diff; closest = p; }
      }
      return closest.value;
    };
    const niftyStart = niftyAt(startDate);

    return filteredHistory.map(snap => ({
      month: snap.month,
      portfolio: (snap.net_worth / startNW - 1) * 100,
      nifty50: niftyStart > 0 ? ((niftyAt(new Date(snap.month + '-15')) / niftyStart - 1) * 100) : 0,
    }));
  })();

  return (
    <div>
      <div className="page-header">
        <h1>Net Worth</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={handleSnapshot}>Take Snapshot</button>}
      </div>

      {current && (
        <div className="card-grid stat-grid mb-6">
          <div className="stat-card">
            <div className="label">Total Assets</div>
            <div className="value positive">{formatCurrency(current.total_assets)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Total Liabilities</div>
            <div className="value negative">{formatCurrency(current.total_liabilities)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Net Worth</div>
            <div className={`value ${current.net_worth >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(current.net_worth)}</div>
          </div>
        </div>
      )}

      {current && (
        <div className="chart-grid gap-4 mb-6">
          <div className="card">
            <h3 className="text-lg fw-semibold mb-3">Assets Breakdown</h3>
            <div className="payout-list">
              {Object.entries(current.assets || {}).map(([k, v]) => (
                <div key={k} className="payout-item">
                  <span className="capitalize">{k.replace(/_/g, ' ')}</span>
                  <span className="fw-semibold">{formatCurrency(v)}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="card">
            <h3 className="text-lg fw-semibold mb-3">Liabilities Breakdown</h3>
            <div className="payout-list">
              {Object.entries(current.liabilities || {}).map(([k, v]) => (
                <div key={k} className="payout-item">
                  <span className="capitalize">{k.replace(/_/g, ' ')}</span>
                  <span className="text-danger fw-semibold">{formatCurrency(v)}</span>
                </div>
              ))}
              {Object.keys(current.liabilities || {}).length === 0 && <p className="text-muted">No liabilities</p>}
            </div>
          </div>
        </div>
      )}

      {history.length > 0 && (
        <div className="card mb-5">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg fw-semibold m-0">Net Worth Over Time</h3>
            <SegmentedControl
              ariaLabel="Time range"
              value={range}
              onChange={setRange}
              options={[{ value: '6m', label: '6M' }, { value: '1y', label: '1Y' }, { value: '2y', label: '2Y' }, { value: '3y', label: '3Y' }, { value: 'all', label: 'All' }]}
            />
          </div>
          <ResponsiveContainer width="100%" height={320}>
            <AreaChart data={filteredHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" fontSize={11} />
              <YAxis tickFormatter={(v) => formatCompactINR(Number(v))} fontSize={11} />
              <Tooltip formatter={(v) => formatCurrency(Number(v))} />
              <Legend />
              <Area type="monotone" dataKey="total_assets" name="Assets" stroke="#10b981" fill="#10b981" fillOpacity={0.12} />
              <Area type="monotone" dataKey="total_liabilities" name="Liabilities" stroke="#ef4444" fill="#ef4444" fillOpacity={0.1} />
              <Line type="monotone" dataKey="net_worth" name="Net Worth" stroke={SERIES.primary} strokeWidth={2} />
              {filteredHistory.length > 12 && <Brush dataKey="month" height={20} stroke={SERIES.primary} />}
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}

      {benchmarkOverlay && benchmarkOverlay.length > 1 && (
        <div className="card">
          <h3 className="text-lg fw-semibold mb-4">Net Worth Growth vs Nifty 50</h3>
          <p className="text-muted text-sm mb-3">Both rebased to 0% at the start of the period. Net worth also grows with new savings and loan repayments, so this is not a measure of investment returns — see Portfolio XIRR on the dashboard for that.</p>
          <ResponsiveContainer width="100%" height={280}>
            <LineChart data={benchmarkOverlay}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" fontSize={11} />
              <YAxis tickFormatter={(v: number) => `${v.toFixed(0)}%`} fontSize={11} />
              <Tooltip formatter={(v) => `${Number(v).toFixed(1)}%`} />
              <Legend />
              <Line type="monotone" dataKey="portfolio" name="Net worth" stroke={SERIES.primary} strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="nifty50" name="Nifty 50" stroke={SERIES.negative} strokeWidth={2} dot={false} strokeDasharray="5 5" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
