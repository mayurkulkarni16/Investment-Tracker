import { useEffect, useState } from 'react';
import { getMonthlyCashflows, getMonthDetail, type MonthlyCashflow, type CashflowDetail } from '../api/cashflow';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, AreaChart, Area } from 'recharts';

export default function CashflowPage() {
  const [data, setData] = useState<MonthlyCashflow[]>([]);
  const [months, setMonths] = useState(12);
  const [loading, setLoading] = useState(true);
  const [selectedMonth, setSelectedMonth] = useState<string | null>(null);
  const [details, setDetails] = useState<CashflowDetail[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const { toast } = useToast();

  const load = async () => {
    setLoading(true);
    try {
      const res = await getMonthlyCashflows(months);
      setData(res || []);
    } catch {
      toast('Failed to load cashflow data', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [months]);

  if (loading) return <div className="loading">Loading cashflow data...</div>;

  const totalInflow = data.reduce((s, d) => s + d.inflow, 0);
  const totalOutflow = data.reduce((s, d) => s + d.outflow, 0);
  const totalNet = totalInflow - totalOutflow;

  // Cumulative net for area chart
  let cumulative = 0;
  const cumulativeData = data.map(d => {
    cumulative += d.net;
    return { ...d, cumulative };
  });

  const categoryColor = (cat: string) => {
    switch (cat) {
      case 'Mutual Fund': return '#1a73e8';
      case 'Stock': return '#e8710a';
      case 'Corporate Bond': return '#9c27b0';
      case 'Fixed Deposit': return '#0f9d58';
      case 'Provident Fund': return '#ea4335';
      default: return '#666';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h1>Cashflow Analysis</h1>
        <select
          value={months}
          onChange={e => setMonths(Number(e.target.value))}
          style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid var(--border-color)', fontSize: 14 }}
        >
          <option value={6}>Last 6 months</option>
          <option value={12}>Last 12 months</option>
          <option value={24}>Last 24 months</option>
          <option value={36}>Last 3 years</option>
          <option value={60}>Last 5 years</option>
        </select>
      </div>

      <div className="card-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', marginBottom: 20 }}>
        <div className="stat-card">
          <div className="label">Total Inflows</div>
          <div className="value positive">{formatCurrency(totalInflow)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Total Outflows</div>
          <div className="value negative">{formatCurrency(totalOutflow)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Net Cashflow</div>
          <div className={`value ${totalNet >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(totalNet)}</div>
        </div>
      </div>

      {data.length > 0 && (
        <>
          <div className="card" style={{ marginBottom: 20 }}>
            <h3 style={{ marginBottom: 16 }}>Monthly Inflows vs Outflows</h3>
            <ResponsiveContainer width="100%" height={320}>
              <BarChart data={data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" fontSize={12} />
                <YAxis tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <Tooltip formatter={(val) => formatCurrency(val as number)} />
                <Legend />
                <Bar dataKey="inflow" name="Inflows" fill="#0f9d58" radius={[4, 4, 0, 0]} />
                <Bar dataKey="outflow" name="Outflows" fill="#ea4335" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="card" style={{ marginBottom: 20 }}>
            <h3 style={{ marginBottom: 16 }}>Cumulative Net Cashflow</h3>
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={cumulativeData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" fontSize={12} />
                <YAxis tickFormatter={(v) => `₹${(v / 1000).toFixed(0)}K`} fontSize={11} />
                <Tooltip formatter={(val) => formatCurrency(val as number)} />
                <Area type="monotone" dataKey="cumulative" name="Cumulative Net" stroke="#1a73e8" fill="#1a73e8" fillOpacity={0.15} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          <div className="card">
            <h3 style={{ marginBottom: 16 }}>Monthly Breakdown</h3>
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Month</th>
                    <th className="text-right">Inflows</th>
                    <th className="text-right">Outflows</th>
                    <th className="text-right">Net</th>
                  </tr>
                </thead>
                <tbody>
                  {[...data].reverse().map(d => (
                    <tr
                      key={d.month}
                      onClick={async () => {
                        if (selectedMonth === d.month) {
                          setSelectedMonth(null);
                          setDetails([]);
                          return;
                        }
                        setSelectedMonth(d.month);
                        setDetailLoading(true);
                        try {
                          const res = await getMonthDetail(d.month);
                          setDetails(res || []);
                        } catch {
                          toast('Failed to load details', 'error');
                          setDetails([]);
                        } finally {
                          setDetailLoading(false);
                        }
                      }}
                      style={{ cursor: 'pointer', background: selectedMonth === d.month ? 'var(--bg-hover, #f0f4ff)' : undefined }}
                    >
                      <td>{d.month} {selectedMonth === d.month ? '▼' : '▶'}</td>
                      <td className="text-right text-success">{formatCurrency(d.inflow)}</td>
                      <td className="text-right text-danger">{formatCurrency(d.outflow)}</td>
                      <td className={`text-right ${d.net >= 0 ? 'text-success' : 'text-danger'}`}>{formatCurrency(d.net)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {selectedMonth && (
              <div style={{ marginTop: 16, padding: 16, background: 'var(--bg-secondary, #f8f9fa)', borderRadius: 8 }}>
                <h4 style={{ marginBottom: 12, fontSize: 15 }}>Transactions for {selectedMonth}</h4>
                {detailLoading ? (
                  <div className="text-muted">Loading...</div>
                ) : details.length === 0 ? (
                  <div className="text-muted">No transactions found.</div>
                ) : (
                  <div className="table-container">
                    <table>
                      <thead>
                        <tr>
                          <th>Date</th>
                          <th>Category</th>
                          <th>Name</th>
                          <th>Type</th>
                          <th className="text-right">Amount</th>
                          <th>Direction</th>
                        </tr>
                      </thead>
                      <tbody>
                        {details.map((d, i) => (
                          <tr key={i}>
                            <td style={{ fontSize: 13 }}>{d.date}</td>
                            <td><span style={{ fontSize: 12, background: categoryColor(d.category), color: '#fff', borderRadius: 4, padding: '2px 6px' }}>{d.category}</span></td>
                            <td>{d.name}</td>
                            <td className="text-muted" style={{ fontSize: 13, textTransform: 'capitalize' }}>{d.type.replace(/_/g, ' ')}</td>
                            <td className="text-right">{formatCurrency(d.amount)}</td>
                            <td><span className={d.direction === 'inflow' ? 'text-success' : 'text-danger'} style={{ fontWeight: 600, fontSize: 12 }}>{d.direction === 'inflow' ? '↓ IN' : '↑ OUT'}</span></td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>
        </>
      )}

      {data.length === 0 && (
        <div className="card" style={{ textAlign: 'center', padding: 40 }}>
          <p className="text-muted">No cashflow data found for the selected period.</p>
        </div>
      )}
    </div>
  );
}
