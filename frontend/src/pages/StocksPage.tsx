import { Fragment, useEffect, useState, useRef, useCallback } from 'react';
import { ChevronRight, Coins, List, Pencil, Plus, RefreshCw, Trash2, Upload } from 'lucide-react';
import { getStocks, createStock, deleteStock, addStockTransaction, deleteStockTransaction, addStockDividend, deleteStockDividend, refreshAllPrices, updateStock } from '../api/stocks';
import type { Stock, CreateStockRequest, AddStockTransactionRequest, StockExchange, UpdateStockRequest, AddStockDividendRequest } from '../types';
import { formatCurrency, formatPercent, formatDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';
import BrokerImportModal from '../components/BrokerImportModal';
import RowActions from '../components/RowActions';
import { PageSkeleton, SortableTh } from '../components/ui';
import { useSort } from '../utils/useSort';

const EXCHANGES: StockExchange[] = ['NSE', 'BSE'];
const REFRESH_INTERVAL_MS = 60_000; // Auto-refresh every 60s during market hours

const STOCK_SORT: Record<string, (s: Stock) => string | number> = {
  symbol: s => s.symbol,
  total_quantity: s => s.total_quantity,
  avg_buy_price: s => s.avg_buy_price,
  current_price: s => s.current_price,
  total_invested: s => s.total_invested,
  current_value: s => s.current_value,
  gain_loss: s => s.gain_loss,
  day_change_percent: s => s.day_change_percent ?? 0,
  xirr: s => s.xirr,
};

export default function StocksPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [showTx, setShowTx] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<Stock | null>(null);
  const [expandedStock, setExpandedStock] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [search, setSearch] = useState('');
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();

  const [form, setForm] = useState<CreateStockRequest>({ stock_name: '', symbol: '', exchange: 'NSE' });
  const [txForm, setTxForm] = useState<AddStockTransactionRequest>({ date: '', type: 'buy', quantity: 0, price_per_share: 0 });
  const [editForm, setEditForm] = useState<UpdateStockRequest>({});
  const [showDiv, setShowDiv] = useState<string | null>(null);
  const [divForm, setDivForm] = useState<AddStockDividendRequest>({ date: '', amount_per_share: 0 });

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

  // Reload when the viewed user changes
  useEffect(() => {
    load();
  }, [load, viewAsUserId]);

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
    if (!await confirm({ message: 'Delete this stock?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteStock(id); toast('Stock deleted'); load(); }
    catch { toast('Failed to delete stock', 'error'); }
  };

  const handleDeleteTxn = async (stockId: string, txnId: string) => {
    if (!await confirm({ message: 'Delete this transaction? Stock totals will be recalculated.', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteStockTransaction(stockId, txnId); toast('Transaction deleted'); load(); }
    catch { toast('Failed to delete transaction', 'error'); }
  };

  const handleAddDiv = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showDiv) return;
    try { await addStockDividend(showDiv, divForm); toast('Dividend recorded'); setShowDiv(null); setDivForm({ date: '', amount_per_share: 0 }); load(); }
    catch { toast('Failed to add dividend', 'error'); }
  };

  const handleDeleteDiv = async (stockId: string, divId: string) => {
    if (!await confirm({ message: 'Delete this dividend record?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteStockDividend(stockId, divId); toast('Dividend deleted'); load(); }
    catch { toast('Failed to delete dividend', 'error'); }
  };

  // Totals
  const totalInvested = stocks.reduce((s, st) => s + st.total_invested, 0);
  const totalValue = stocks.reduce((s, st) => s + st.current_value, 0);
  const totalGain = totalValue - totalInvested;
  const totalGainPct = totalInvested > 0 ? (totalGain / totalInvested) * 100 : 0;
  const totalDayChange = stocks.reduce((s, st) => s + (st.day_change * st.total_quantity), 0);
  const xirrBase = stocks.filter(s => s.total_invested > 0);
  const xirrWeight = xirrBase.reduce((s, x) => s + x.total_invested, 0);
  const avgXirr = xirrWeight > 0 ? xirrBase.reduce((s, x) => s + x.xirr * x.total_invested, 0) / xirrWeight : 0;

  const filteredStocks = stocks.filter(s => !search || s.symbol.toLowerCase().includes(search.toLowerCase()) || s.stock_name.toLowerCase().includes(search.toLowerCase()));
  const { sorted, sortCol, sortDir, onSort } = useSort(filteredStocks, STOCK_SORT);
  const sortProps = { sortCol, sortDir, onSort };

  if (loading) return <PageSkeleton label="Loading stocks" />;

  return (
    <div>
      <div className="page-header">
        <div className="flex items-center gap-3 flex-wrap">
          <h1>Stocks</h1>
          <span className={`market-status${isMarketOpen ? ' open' : ''}`}>
            <i aria-hidden="true" />{isMarketOpen ? 'Market open' : 'Market closed'}
          </span>
          {isMarketOpen && (
            <label className="inline-check">
              <input type="checkbox" checked={autoRefresh} onChange={e => setAutoRefresh(e.target.checked)} />
              Auto-refresh
            </label>
          )}
        </div>
        {!isViewOnly && (
          <div className="header-actions">
            <button className="btn btn-outline" onClick={handleRefresh} disabled={refreshing} title="Fetch the latest prices">
              <RefreshCw size={15} className={refreshing ? 'spin' : undefined} aria-hidden="true" />{refreshing ? 'Refreshing…' : 'Refresh prices'}
            </button>
            <button className="btn btn-outline" onClick={() => setShowImport(true)}><Upload size={15} aria-hidden="true" />Import from broker</button>
            <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add stock</button>
          </div>
        )}
      </div>

      {stocks.length === 0 ? (
        <div className="empty-state">
          <h3>No stocks yet</h3>
          <p>Add a stock by hand, or import your holdings from a broker statement.</p>
          {!isViewOnly && (
            <div className="header-actions justify-center">
              <button className="btn btn-outline" onClick={() => setShowImport(true)}><Upload size={15} aria-hidden="true" />Import from broker</button>
              <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add stock</button>
            </div>
          )}
        </div>
      ) : (
        <>
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card">
            <div className="label">Invested</div>
            <div className="value">{formatCurrency(totalInvested)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Current value</div>
            <div className="value">{formatCurrency(totalValue)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Total P&amp;L</div>
            <div className={`value ${totalGain >= 0 ? 'positive' : 'negative'}`}>{totalGain >= 0 ? '+' : ''}{formatCurrency(totalGain)}</div>
            <span className={`pill ${totalGain >= 0 ? 'pill-up' : 'pill-down'} mt-1`}>{formatPercent(totalGainPct)}</span>
          </div>
          <div className="stat-card">
            <div className="label">Today</div>
            <div className={`value ${totalDayChange >= 0 ? 'positive' : 'negative'}`}>{totalDayChange >= 0 ? '+' : ''}{formatCurrency(totalDayChange)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Avg XIRR</div>
            <div className={`value ${avgXirr >= 0 ? 'positive' : 'negative'}`}>{formatPercent(avgXirr)}</div>
          </div>
        </div>

        <div className="toolbar">
          <div className="search-bar">
            <input type="search" placeholder="Search by name or symbol" aria-label="Search stocks" value={search} onChange={e => setSearch(e.target.value)} />
          </div>
        </div>

        {sorted.length === 0 ? (
          <div className="empty-state">
            <h3>No stocks match</h3>
            <p>Nothing matches “{search}”.</p>
            <button className="btn btn-outline" onClick={() => setSearch('')}>Clear search</button>
          </div>
        ) : (
        <div className="table-container table-stack">
          <table>
            <thead>
              <tr>
                <SortableTh col="symbol" {...sortProps}>Stock</SortableTh>
                <SortableTh col="total_quantity" align="right" {...sortProps}>Qty</SortableTh>
               <SortableTh col="current_price" align="right" {...sortProps}>LTP</SortableTh>
                <SortableTh col="total_invested" align="right" {...sortProps}>Invested</SortableTh>
                <SortableTh col="current_value" align="right" {...sortProps}>Value</SortableTh>
                <SortableTh col="gain_loss" align="right" {...sortProps}>P&amp;L</SortableTh>
                <SortableTh col="day_change_percent" align="right" {...sortProps}>Today</SortableTh>
                <SortableTh col="xirr" align="right" {...sortProps}>XIRR</SortableTh>
                <th><span className="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody>
              {sorted.map(s => {
                const open = expandedStock === s.id;
                const txCount = s.transactions?.length || 0;
                const divCount = s.dividends?.length || 0;
                const canExpand = txCount + divCount > 0;
                const toggle = () => { if (canExpand) setExpandedStock(open ? null : s.id); };
                const up = s.gain_loss >= 0;
                const dayUp = s.day_change >= 0;
                return (
                  <Fragment key={s.id}>
                    <tr className={open ? 'is-expanded' : undefined} style={{ cursor: canExpand ? 'pointer' : undefined }} onClick={toggle}>
                      <td className="cell-primary cell-primary-narrow">
                        <div className="cell-title">
                          {canExpand && <ChevronRight size={15} className={`row-chevron${open ? ' open' : ''}`} aria-hidden="true" />}
                          <span className="cell-main fw-semibold">{s.symbol}</span>
                          <span className={`badge ${s.exchange === 'NSE' ? 'badge-completed' : 'badge-neutral'}`}>{s.exchange}</span>
                        </div>
                        <span className="cell-sub">{s.stock_name}</span>
                      </td>
                      <td className="num" data-label="Qty">{s.total_quantity}</td>
                     <td className="num" data-label="LTP">
                        {formatCurrency(s.current_price)}
                        <span className="cell-sub" title={s.price_last_updated ? `Updated ${formatDate(s.price_last_updated)}` : undefined}>Avg {formatCurrency(s.avg_buy_price)}</span>
                      </td>
                      <td className="num" data-label="Invested">{formatCurrency(s.total_invested)}</td>
                      <td className="num cell-main" data-label="Current value">{formatCurrency(s.current_value)}</td>
                      <td className="num" data-label="P&L">
                        <span className={`${up ? 'text-success' : 'text-danger'} fw-semibold`}>{up ? '+' : ''}{formatCurrency(s.gain_loss)}</span>
                        <span className="cell-sub"><span className={`pill pill-sm ${up ? 'pill-up' : 'pill-down'}`}>{formatPercent(s.gain_loss_percent)}</span></span>
                      </td>
                      <td className={`num ${dayUp ? 'text-success' : 'text-danger'}`} data-label="Today">
                        {dayUp ? '+' : ''}{formatCurrency(s.day_change)}
                        <span className={`cell-sub ${dayUp ? 'text-success' : 'text-danger'}`}>{dayUp ? '+' : ''}{(s.day_change_percent ?? 0).toFixed(2)}%</span>
                      </td>
                      <td className={`num ${s.xirr >= 0 ? 'text-success' : 'text-danger'}`} data-label="XIRR">{formatPercent(s.xirr)}</td>
                      <td className="cell-actions">
                        <RowActions
                          inline={isViewOnly ? 0 : 1}
                          label={`Actions for ${s.symbol}`}
                          actions={[
                            { label: 'Add transaction', inlineLabel: 'Add txn', icon: Plus, onClick: () => setShowTx(s.id), hidden: isViewOnly },
                            { label: open ? 'Hide history' : `View history (${txCount + divCount})`, icon: List, onClick: toggle, hidden: !canExpand },
                            { label: 'Record dividend', icon: Coins, onClick: () => { setShowDiv(s.id); setDivForm({ date: '', amount_per_share: 0 }); }, hidden: isViewOnly },
                            { label: 'Edit stock', icon: Pencil, onClick: () => openEdit(s), hidden: isViewOnly },
                            { label: 'Delete stock', icon: Trash2, onClick: () => handleDelete(s.id), danger: true, hidden: isViewOnly },
                          ]}
                        />
                      </td>
                    </tr>
                    {open && canExpand && (
                      <tr className="row-detail">
                        <td className="p-0" colSpan={9}>
                          <div className="row-detail-panel">
                            {txCount > 0 && (
                              <>
                                <div className="row-detail-head">
                                  <strong className="text-base">Transactions</strong>
                                  <span className="text-muted text-sm">{txCount} transaction{txCount === 1 ? '' : 's'}</span>
                                </div>
                                <table>
                                  <thead>
                                    <tr>
                                      <th>Date</th>
                                      <th>Type</th>
                                      <th className="num">Quantity</th>
                                      <th className="num">Price / share</th>
                                      <th className="num">Amount</th>
                                      {!isViewOnly && <th><span className="sr-only">Actions</span></th>}
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {[...s.transactions].sort((a, b) => b.date.localeCompare(a.date)).map(tx => (
                                      <tr key={tx.transaction_id}>
                                        <td>{formatDate(tx.date)}</td>
                                        <td><span className={`badge ${tx.type === 'sell' ? 'badge-matured' : 'badge-active'} capitalize`}>{tx.type}</span></td>
                                        <td className="num">{tx.quantity}</td>
                                        <td className="num">{formatCurrency(tx.price_per_share)}</td>
                                        <td className="num">{formatCurrency(tx.amount)}</td>
                                        {!isViewOnly && (
                                          <td className="cell-actions">
                                            <button className="btn btn-sm btn-danger" title="Delete transaction" aria-label={`Delete ${tx.type} on ${formatDate(tx.date)}`} onClick={e => { e.stopPropagation(); handleDeleteTxn(s.id, tx.transaction_id); }}>
                                              <Trash2 size={14} aria-hidden="true" />
                                            </button>
                                          </td>
                                        )}
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </>
                            )}
                            {divCount > 0 && (
                              <>
                                <div className="row-detail-head" style={{ marginTop: txCount > 0 ? 16 : 0 }}>
                                  <strong className="text-base">Dividends</strong>
                                  <span className="text-muted text-sm">Total {formatCurrency(s.total_dividends)}</span>
                                </div>
                                <table>
                                  <thead>
                                    <tr>
                                      <th>Date</th>
                                      <th className="num">₹ / share</th>
                                      <th className="num">Total</th>
                                      {!isViewOnly && <th><span className="sr-only">Actions</span></th>}
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {[...s.dividends].sort((a, b) => b.date.localeCompare(a.date)).map(d => (
                                      <tr key={d.dividend_id}>
                                        <td>{formatDate(d.date)}</td>
                                        <td className="num">₹{d.amount_per_share.toFixed(2)}</td>
                                        <td className="num">{formatCurrency(d.total_amount)}</td>
                                        {!isViewOnly && (
                                          <td className="cell-actions">
                                            <button className="btn btn-sm btn-danger" title="Delete dividend" aria-label={`Delete dividend on ${formatDate(d.date)}`} onClick={e => { e.stopPropagation(); handleDeleteDiv(s.id, d.dividend_id); }}>
                                              <Trash2 size={14} aria-hidden="true" />
                                            </button>
                                          </td>
                                        )}
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
            {sorted.length > 1 && (
              <tfoot>
                <tr>
                  <td className="cell-primary">Total · {sorted.length} stocks</td>
                  <td /><td />
                  <td className="num">{formatCurrency(sorted.reduce((t, s) => t + s.total_invested, 0))}</td>
                  <td className="num">{formatCurrency(sorted.reduce((t, s) => t + s.current_value, 0))}</td>
                  <td className="num">
                    {(() => {
                      const g = sorted.reduce((t, s) => t + s.gain_loss, 0);
                      return <span className={g >= 0 ? 'text-success' : 'text-danger'}>{g >= 0 ? '+' : ''}{formatCurrency(g)}</span>;
                    })()}
                  </td>
                  <td className="num">
                    {(() => {
                      const d = sorted.reduce((t, s) => t + s.day_change * s.total_quantity, 0);
                      return <span className={d >= 0 ? 'text-success' : 'text-danger'}>{d >= 0 ? '+' : ''}{formatCurrency(d)}</span>;
                    })()}
                  </td>
                  <td /><td className="cell-actions" />
                </tr>
              </tfoot>
            )}
          </table>
        </div>
        )}
        <p className="table-note">Prices via Yahoo Finance · may be delayed by 15 minutes</p>
        </>
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
                <div className="text-md mb-3" style={{ padding: '8px 12px', background: 'var(--surface-muted)', borderRadius: 6 }}>
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

      {showDiv && (
        <div className="modal-overlay" onClick={() => setShowDiv(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Record Dividend</h2>
            <form onSubmit={handleAddDiv}>
              <div className="form-group">
                <label>Date</label>
                <input type="date" required value={divForm.date} onChange={e => setDivForm({ ...divForm, date: e.target.value })} />
              </div>
              <div className="form-group">
                <label>Amount per Share (₹)</label>
                <input type="number" step="0.01" min="0.01" required value={divForm.amount_per_share || ''} onChange={e => setDivForm({ ...divForm, amount_per_share: parseFloat(e.target.value) })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowDiv(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Record Dividend</button>
              </div>
            </form>
          </div>
        </div>
      )}
      <BrokerImportModal open={showImport} onClose={() => setShowImport(false)} onImported={load} />
    </div>
  );
}
