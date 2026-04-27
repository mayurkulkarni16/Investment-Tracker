import { useEffect, useState, useRef, useCallback } from 'react';
import { getStocks, createStock, deleteStock, addStockTransaction, refreshAllPrices, updateStock } from '../api/stocks';
import type { Stock, CreateStockRequest, AddStockTransactionRequest, StockExchange, UpdateStockRequest } from '../types';
import { formatCurrency, formatPercent, formatDate } from '../utils/format';
import { useToast } from '../components/Toast';

const EXCHANGES: StockExchange[] = ['NSE', 'BSE'];
const REFRESH_INTERVAL_MS = 60_000; // Auto-refresh every 60s during market hours

export default function StocksPage() {
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showTx, setShowTx] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<Stock | null>(null);
  const [expandedStock, setExpandedStock] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [form, setForm] = useState<CreateStockRequest>({ stock_name: '', symbol: '', exchange: 'NSE' });
  const [txForm, setTxForm] = useState<AddStockTransactionRequest>({ date: '', type: 'buy', quantity: 0, price_per_share: 0 });
  const [editForm, setEditForm] = useState<UpdateStockRequest>({});

  const isMarketOpen = stocks.length > 0 ? stocks[0].is_market_open : false;

  const load = useCallback(() => {
    getStocks().then(r => setStocks(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refreshAllPrices();
      toast('Prices refreshed');
      load();
    } catch { toast('Failed to refresh prices', 'error'); }
    finally { setRefreshing(false); }
  };

  // Auto-refresh during market hours
  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (autoRefresh && isMarketOpen) {
      intervalRef.current = setInterval(() => {
        refreshAllPrices().then(() => load()).catch(() => {});
      }, REFRESH_INTERVAL_MS);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, isMarketOpen, load]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await createStock(form); setShowAdd(false); setForm({ stock_name: '', symbol: '', exchange: 'NSE' }); toast('Stock added'); load(); }
    catch { toast('Failed to add stock', 'error'); }
  };

  const handleAddTx = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showTx) return;
    try { await addStockTransaction(showTx, txForm); setShowTx(null); setTxForm({ date: '', type: 'buy', quantity: 0, price_per_share: 0 }); toast('Transaction added'); load(); }
    catch { toast('Failed to add transaction', 'error'); }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try { await updateStock(showEdit.id, editForm); setShowEdit(null); toast('Stock updated'); load(); }
    catch { toast('Failed to update stock', 'error'); }
  };

  const openEdit = (s: Stock) => {
    setEditForm({ stock_name: s.stock_name, symbol: s.symbol, exchange: s.exchange, notes: s.notes });
    setShowEdit(s);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this stock?')) return;
    try { await deleteStock(id); toast('Stock deleted'); load(); }
    catch { toast('Failed to delete stock', 'error'); }
  };

  // Totals
  const totalInvested = stocks.reduce((s, st) => s + st.total_invested, 0);
  const totalValue = stocks.reduce((s, st) => s + st.current_value, 0);
  const totalGain = totalValue - totalInvested;
  const totalGainPct = totalInvested > 0 ? (totalGain / totalInvested) * 100 : 0;
  const totalDayChange = stocks.reduce((s, st) => s + (st.day_change * st.total_quantity), 0);

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Stocks</h1>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span style={{
            padding: '4px 10px', borderRadius: 12, fontSize: 12, fontWeight: 600,
            background: isMarketOpen ? '#e8f5e9' : '#fde8e8',
            color: isMarketOpen ? '#2e7d32' : '#c62828',
          }}>
            {isMarketOpen ? '● Market Open' : '○ Market Closed'}
          </span>
          {isMarketOpen && (
            <label style={{ fontSize: 12, display: 'flex', alignItems: 'center', gap: 4 }}>
              <input type="checkbox" checked={autoRefresh} onChange={e => setAutoRefresh(e.target.checked)} style={{ width: 'auto' }} />
              Auto-refresh
            </label>
          )}
          <button className="btn btn-outline" onClick={handleRefresh} disabled={refreshing}>
            {refreshing ? 'Refreshing...' : 'Refresh Prices'}
          </button>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Stock</button>
        </div>
      </div>

      {/* Summary cards */}
      <div className="card-grid" style={{ marginBottom: 20 }}>
        <div className="stat-card">
          <div className="label">Total Invested</div>
          <div className="value">{formatCurrency(totalInvested)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Current Value</div>
          <div className="value">{formatCurrency(totalValue)}</div>
        </div>
        <div className="stat-card">
          <div className="label">Total P&L</div>
          <div className={`value ${totalGain >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(totalGain)} ({formatPercent(totalGainPct)})
          </div>
        </div>
        <div className="stat-card">
          <div className="label">Day Change</div>
          <div className={`value ${totalDayChange >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(totalDayChange)}
          </div>
        </div>
      </div>

      {stocks.length === 0 ? (
        <div className="empty-state">
          <h3>No stocks yet</h3>
          <p>Add your first stock to start tracking.</p>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Stock</button>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Stock</th>
                <th>Exchange</th>
                <th>Qty</th>
                <th>Avg Price</th>
                <th>Invested</th>
                <th>LTP</th>
                <th>Current Value</th>
                <th>P&L</th>
                <th>Day Change</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {stocks.map(s => (
                <tr key={s.id} style={{ cursor: 'pointer' }} onClick={() => setExpandedStock(expandedStock === s.id ? null : s.id)}>
                  <td>
                    <strong>{s.symbol}</strong>
                    <div className="text-muted" style={{ fontSize: 12 }}>{s.stock_name}</div>
                  </td>
                  <td>
                    <span style={{
                      padding: '2px 6px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                      background: s.exchange === 'NSE' ? '#e3f2fd' : '#fce4ec',
                      color: s.exchange === 'NSE' ? '#1565c0' : '#c62828',
                    }}>{s.exchange}</span>
                  </td>
                  <td>{s.total_quantity}</td>
                  <td>{formatCurrency(s.avg_buy_price)}</td>
                  <td>{formatCurrency(s.total_invested)}</td>
                  <td>
                    {formatCurrency(s.current_price)}
                    {s.price_last_updated && <div className="text-muted" style={{ fontSize: 10 }}>{formatDate(s.price_last_updated)}</div>}
                  </td>
                  <td>{formatCurrency(s.current_value)}</td>
                  <td className={s.gain_loss >= 0 ? 'text-success' : 'text-danger'}>
                    {formatCurrency(s.gain_loss)}<br />
                    <span style={{ fontSize: 12 }}>{formatPercent(s.gain_loss_percent)}</span>
                  </td>
                  <td className={s.day_change >= 0 ? 'text-success' : 'text-danger'}>
                    {s.day_change >= 0 ? '+' : ''}{formatCurrency(s.day_change)}<br />
                    <span style={{ fontSize: 12 }}>{s.day_change_percent >= 0 ? '+' : ''}{s.day_change_percent.toFixed(2)}%</span>
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }} onClick={e => e.stopPropagation()}>
                      <button className="btn btn-sm btn-outline" onClick={() => setExpandedStock(expandedStock === s.id ? null : s.id)}>
                        {expandedStock === s.id ? 'Hide' : 'Txns'} ({s.transactions?.length || 0})
                      </button>
                      <button className="btn btn-sm btn-outline" onClick={() => openEdit(s)}>Edit</button>
                      <button className="btn btn-sm btn-outline" onClick={() => setShowTx(s.id)}>+ Txn</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(s.id)}>Del</button>
                    </div>
                  </td>
                </tr>
              ))}
              {stocks.map(s => expandedStock === s.id && s.transactions?.length > 0 && (
                <tr key={`${s.id}-txns`}>
                  <td colSpan={10} style={{ padding: 0 }}>
                    <div style={{ background: '#f8f9fa', padding: '12px 16px', borderTop: '1px solid #e0e0e0' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                        <strong style={{ fontSize: 14 }}>Transactions — {s.symbol}</strong>
                        <span className="text-muted" style={{ fontSize: 12 }}>{s.transactions.length} transactions</span>
                      </div>
                      <table style={{ fontSize: 13, background: '#fff', borderRadius: 6 }}>
                        <thead>
                          <tr>
                            <th>Date</th>
                            <th>Type</th>
                            <th>Quantity</th>
                            <th>Price/Share</th>
                            <th>Amount</th>
                          </tr>
                        </thead>
                        <tbody>
                          {[...s.transactions].sort((a, b) => b.date.localeCompare(a.date)).map(tx => (
                            <tr key={tx.transaction_id}>
                              <td>{formatDate(tx.date)}</td>
                              <td>
                                <span style={{
                                  padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                                  background: tx.type === 'sell' ? '#fde8e8' : '#e8f5e9',
                                  color: tx.type === 'sell' ? '#c62828' : '#2e7d32',
                                }}>
                                  {tx.type.toUpperCase()}
                                </span>
                              </td>
                              <td>{tx.quantity}</td>
                              <td>{formatCurrency(tx.price_per_share)}</td>
                              <td>{formatCurrency(tx.amount)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Add Stock Modal */}
      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Stock</h2>
            <form onSubmit={handleAdd}>
              <div className="form-group">
                <label>Company Name</label>
                <input required value={form.stock_name} onChange={e => setForm({ ...form, stock_name: e.target.value })} />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Symbol (e.g., RELIANCE, TCS)</label>
                  <input required value={form.symbol} onChange={e => setForm({ ...form, symbol: e.target.value.toUpperCase() })} />
                </div>
                <div className="form-group">
                  <label>Exchange</label>
                  <select value={form.exchange} onChange={e => setForm({ ...form, exchange: e.target.value as StockExchange })}>
                    {EXCHANGES.map(ex => <option key={ex} value={ex}>{ex}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea value={form.notes || ''} onChange={e => setForm({ ...form, notes: e.target.value })} rows={2} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Stock</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Stock Modal */}
      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Stock</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group">
                <label>Company Name</label>
                <input required value={editForm.stock_name || ''} onChange={e => setEditForm({ ...editForm, stock_name: e.target.value })} />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Symbol</label>
                  <input required value={editForm.symbol || ''} onChange={e => setEditForm({ ...editForm, symbol: e.target.value.toUpperCase() })} />
                </div>
                <div className="form-group">
                  <label>Exchange</label>
                  <select value={editForm.exchange || 'NSE'} onChange={e => setEditForm({ ...editForm, exchange: e.target.value as StockExchange })}>
                    {EXCHANGES.map(ex => <option key={ex} value={ex}>{ex}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} rows={2} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Changes</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Transaction Modal */}
      {showTx && (
        <div className="modal-overlay" onClick={() => setShowTx(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Transaction</h2>
            <form onSubmit={handleAddTx}>
              <div className="form-row">
                <div className="form-group">
                  <label>Date</label>
                  <input type="date" required value={txForm.date} onChange={e => setTxForm({ ...txForm, date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Type</label>
                  <select value={txForm.type} onChange={e => setTxForm({ ...txForm, type: e.target.value as 'buy' | 'sell' })}>
                    <option value="buy">Buy</option>
                    <option value="sell">Sell</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Quantity</label>
                  <input type="number" min="1" required value={txForm.quantity || ''} onChange={e => setTxForm({ ...txForm, quantity: parseInt(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Price per Share (₹)</label>
                  <input type="number" step="0.01" required value={txForm.price_per_share || ''} onChange={e => setTxForm({ ...txForm, price_per_share: parseFloat(e.target.value) })} />
                </div>
              </div>
              {txForm.quantity > 0 && txForm.price_per_share > 0 && (
                <div style={{ padding: '8px 12px', background: '#f5f5f5', borderRadius: 6, fontSize: 13, marginBottom: 12 }}>
                  Total: <strong>{formatCurrency(txForm.quantity * txForm.price_per_share)}</strong>
                </div>
              )}
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowTx(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Transaction</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
