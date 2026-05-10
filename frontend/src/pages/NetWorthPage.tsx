import { useEffect, useState } from 'react';
import { getNetWorthCurrent, getNetWorthHistory, takeNetWorthSnapshot } from '../api/netWorth';
import { getBenchmarks } from '../api/benchmarks';
import type { NetWorthCurrent, NetWorthSnapshot, BenchmarkData } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { AreaChart, Area, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, LineChart, Brush } from 'recharts';

export default function NetWorthPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [current, setCurrent] = useState<NetWorthCurrent | null>(null);
  const [history, setHistory] = useState<NetWorthSnapshot[]>([]);
  const [benchmarks, setBenchmarks] = useState<BenchmarkData[]>([]);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<string>('all');
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

  if (loading) return <div className="loading">Loading...</div>;

  // Filter history by range
  const filteredHistory = (() => {
    if (range === 'all' || history.length === 0) return history;
    const months = range === '6m' ? 6 : range === '1y' ? 12 : range === '2y' ? 24 : range === '3y' ? 36 : history.length;
    return history.slice(-months);
  })();

  // Normalize benchmark data to overlay with portfolio
  const benchmarkOverlay = (() => {
    if (filteredHistory.length < 2 || benchmarks.length === 0) return null;
    const nifty = benchmarks.find(b => b.symbol === '^NSEI');
    if (!nifty || !nifty.points || nifty.points.length === 0) return null;

    const startNW = filteredHistory[0].net_worth;
    const niftyStart = nifty.points[0].value;

    return filteredHistory.map(snap => {
      // Find closest benchmark point to this month
      const snapDate = new Date(snap.month + '-15');
      let closest = nifty.points[0];
      let minDiff = Math.abs(new Date(closest.date).getTime() - snapDate.getTime());
      for (const p of nifty.points) {
        const diff = Math.abs(new Date(p.date).getTime() - snapDate.getTime());
        if (diff < minDiff) { minDiff = diff; closest = p; }
      }
      return {
        month: snap.month,
        portfolio: startNW > 0 ? ((snap.net_worth / startNW - 1) * 100) : 0,
        nifty50: niftyStart > 0 ? ((closest.value / niftyStart - 1) * 100) : 0,
      };
    });
  })();

  return (
    <div>
      <div className="page-header">
        <h1>Net Worth</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={handleSnapshot}>Take Snapshot</button>}
      </div>

      {current && (
        <div className="card-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', marginBottom: 24 }}>
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
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
          <div className="card">
            <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Assets Breakdown</h3>
            <div className="payout-list">
              {Object.entries(current.assets || {}).map(([k, v]) => (
                <div key={k} className="payout-item">
                  <span style={{ textTransform: 'capitalize' }}>{k.replace(/_/g, ' ')}</span>
                  <span style={{ fontWeight: 600 }}>{formatCurrency(v)}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="card">
            <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Liabilities Breakdown</h3>
            <div className="payout-list">
              {Object.entries(current.liabilities || {}).map(([k, v]) => (
                <div key={k} className="payout-item">
                  <span style={{ textTransform: 'capitalize' }}>{k.replace(/_/g, ' ')}</span>
                  <span style={{ fontWeight: 600 }} className="text-danger">{formatCurrency(v)}</span>
                </div>
              ))}
              {Object.keys(current.liabilities || {}).length === 0 && <p className="text-muted">No liabilities</p>}
            </div>
          </div>
        </div>
      )}

      {history.length > 0 && (
        <div className="card" style={{ marginBottom: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h3 style={{ fontSize: 16, fontWeight: 600, margin: 0 }}>Net Worth Over Time</h3>
            <div style={{ display: 'flex', gap: 0 }}>
              {[{ key: '6m', label: '6M' }, { key: '1y', label: '1Y' }, { key: '2y', label: '2Y' }, { key: '3y', label: '3Y' }, { key: 'all', label: 'All' }].map(r => (
                <button key={r.key} onClick={() => setRange(r.key)}
                  style={{
                    padding: '4px 12px', fontSize: 12, border: '1px solid var(--border-color)', cursor: 'pointer',
                    background: range === r.key ? 'var(--primary)' : 'var(--bg-primary)',
                    color: range === r.key ? '#fff' : 'var(--text-primary)',
                    borderRadius: r.key === '6m' ? '6px 0 0 6px' : r.key === 'all' ? '0 6px 6px 0' : '0',
                    fontWeight: range === r.key ? 600 : 400,
                  }}>{r.label}</button>
              ))}
            </div>
          </div>
          <ResponsiveContainer width="100%" height={320}>
            <AreaChart data={filteredHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" fontSize={11} />
              <YAxis tickFormatter={(v: number) => `₹${(v / 100000).toFixed(0)}L`} fontSize={11} />
              <Tooltip formatter={(v: any) => formatCurrency(v)} />
              <Legend />
              <Area type="monotone" dataKey="total_assets" name="Assets" stroke="#0f9d58" fill="#e6f4ea" />
              <Area type="monotone" dataKey="total_liabilities" name="Liabilities" stroke="#ea4335" fill="#fce8e6" />
              <Line type="monotone" dataKey="net_worth" name="Net Worth" stroke="#1a73e8" strokeWidth={2} />
              {filteredHistory.length > 12 && <Brush dataKey="month" height={20} stroke="#1a73e8" />}
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}

      {benchmarkOverlay && benchmarkOverlay.length > 1 && (
        <div className="card">
          <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 16 }}>Portfolio vs Nifty 50 (Normalized)</h3>
          <p className="text-muted" style={{ fontSize: 12, marginBottom: 12 }}>Both normalized to 0% at start of period for comparison</p>
          <ResponsiveContainer width="100%" height={280}>
            <LineChart data={benchmarkOverlay}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" fontSize={11} />
              <YAxis tickFormatter={(v: number) => `${v.toFixed(0)}%`} fontSize={11} />
              <Tooltip formatter={(v: any) => `${Number(v).toFixed(1)}%`} />
              <Legend />
              <Line type="monotone" dataKey="portfolio" name="Your Portfolio" stroke="#1a73e8" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="nifty50" name="Nifty 50" stroke="#ea4335" strokeWidth={2} dot={false} strokeDasharray="5 5" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
