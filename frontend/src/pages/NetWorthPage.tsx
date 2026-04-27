import { useEffect, useState } from 'react';
import { getNetWorthCurrent, getNetWorthHistory, takeNetWorthSnapshot } from '../api/netWorth';
import type { NetWorthCurrent, NetWorthSnapshot } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { AreaChart, Area, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

export default function NetWorthPage() {
  const [current, setCurrent] = useState<NetWorthCurrent | null>(null);
  const [history, setHistory] = useState<NetWorthSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const { toast } = useToast();

  const load = () => {
    Promise.all([
      getNetWorthCurrent().then(r => setCurrent(r.data)),
      getNetWorthHistory().then(r => setHistory(r.data || [])),
    ]).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, []);

  const handleSnapshot = async () => {
    try { await takeNetWorthSnapshot(); toast('Snapshot taken'); load(); }
    catch { toast('Failed to take snapshot', 'error'); }
  };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Net Worth</h1>
        <button className="btn btn-primary" onClick={handleSnapshot}>Take Snapshot</button>
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
        <div className="card">
          <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 16 }}>Net Worth Over Time</h3>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={history}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" />
              <YAxis tickFormatter={(v: number) => `₹${(v / 100000).toFixed(0)}L`} />
              <Tooltip formatter={(v) => formatCurrency(Number(v))} />
              <Legend />
              <Area type="monotone" dataKey="total_assets" name="Assets" stroke="#0f9d58" fill="#e6f4ea" />
              <Area type="monotone" dataKey="total_liabilities" name="Liabilities" stroke="#ea4335" fill="#fce8e6" />
              <Line type="monotone" dataKey="net_worth" name="Net Worth" stroke="#1a73e8" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
