import { useEffect, useState } from 'react';
import { getMutualFunds, createMutualFund, deleteMutualFund, addMFTransaction, refreshNAV, updateMutualFund, importFromCAS } from '../api/mutualFunds';
import type { MutualFund, CreateMutualFundRequest, AddMFTransactionRequest, FundType, UpdateMutualFundRequest } from '../types';
import { formatCurrency, formatPercent, formatDate, toInputDate } from '../utils/format';
import { extractTextFromPDF, parseCASText } from '../utils/casParser';
import type { ParsedCASFund } from '../utils/casParser';

const FUND_TYPES: FundType[] = ['Equity', 'Debt', 'Hybrid', 'ELSS', 'Index', 'Liquid'];

export default function MutualFundsPage() {
  const [funds, setFunds] = useState<MutualFund[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showTx, setShowTx] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<MutualFund | null>(null);
  const [showImport, setShowImport] = useState(false);
  const [expandedFund, setExpandedFund] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'elss'>('all');
  const [form, setForm] = useState<CreateMutualFundRequest>({ fund_name: '', amc: '', fund_type: 'Equity', scheme_code: '', is_elss: false });
  const [txForm, setTxForm] = useState<AddMFTransactionRequest>({ date: '', type: 'purchase', amount: 0, nav_at_purchase: 0 });
  const [editForm, setEditForm] = useState<UpdateMutualFundRequest>({ fund_name: '', amc: '', fund_type: 'Equity', scheme_code: '', is_elss: false });

  // CAS Import state
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importPassword, setImportPassword] = useState('');
  const [parsedFunds, setParsedFunds] = useState<ParsedCASFund[]>([]);
  const [importLoading, setImportLoading] = useState(false);
  const [importStep, setImportStep] = useState<'upload' | 'preview'>('upload');
  const [importMessage, setImportMessage] = useState('');

  const load = () => {
    getMutualFunds().then(r => setFunds(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    await createMutualFund(form);
    setShowAdd(false);
    setForm({ fund_name: '', amc: '', fund_type: 'Equity', scheme_code: '', is_elss: false });
    load();
  };

  const handleAddTx = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showTx) return;
    await addMFTransaction(showTx, txForm);
    setShowTx(null);
    setTxForm({ date: '', type: 'purchase', amount: 0, nav_at_purchase: 0 });
    load();
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    await updateMutualFund(showEdit.id, editForm);
    setShowEdit(null);
    load();
  };

  const openEdit = (f: MutualFund) => {
    setEditForm({
      fund_name: f.fund_name,
      amc: f.amc,
      fund_type: f.fund_type,
      scheme_code: f.scheme_code,
      folio_number: f.folio_number,
      is_elss: f.is_elss,
      notes: f.notes,
    });
    setShowEdit(f);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this mutual fund?')) return;
    await deleteMutualFund(id);
    load();
  };

  const handleRefresh = async () => {
    await refreshNAV();
    load();
  };

  // CAS Import handlers
  const handleParsePDF = async () => {
    if (!importFile) return;
    setImportLoading(true);
    setImportMessage('');
    try {
      const text = await extractTextFromPDF(importFile, importPassword);
      const funds = parseCASText(text);
      if (funds.length === 0) {
        setImportMessage('No transactions found in the PDF. Please check the file and password.');
      } else {
        setParsedFunds(funds);
        setImportStep('preview');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to parse PDF';
      setImportMessage(msg.includes('password') ? 'Incorrect password. Please try again.' : `Error: ${msg}`);
    } finally {
      setImportLoading(false);
    }
  };

  const handleImport = async () => {
    setImportLoading(true);
    try {
      const res = await importFromCAS({ funds: parsedFunds });
      const r = res.data;
      setImportMessage(`Imported: ${r.funds_created} new funds, ${r.funds_updated} updated, ${r.transactions_added} transactions added.${r.errors?.length ? ' Errors: ' + r.errors.join('; ') : ''}`);
      setImportStep('upload');
      setParsedFunds([]);
      load();
    } catch {
      setImportMessage('Failed to import. Check backend is running.');
    } finally {
      setImportLoading(false);
    }
  };

  const closeImport = () => {
    setShowImport(false);
    setImportStep('upload');
    setParsedFunds([]);
    setImportFile(null);
    setImportPassword('');
    setImportMessage('');
  };

  const filtered = filter === 'elss' ? funds.filter(f => f.is_elss) : funds;

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Mutual Funds</h1>
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="btn btn-outline" onClick={() => setFilter(filter === 'all' ? 'elss' : 'all')}>
            {filter === 'all' ? 'Show ELSS Only' : 'Show All'}
          </button>
          <button className="btn btn-outline" onClick={handleRefresh}>Refresh NAV</button>
          <button className="btn btn-outline" onClick={() => setShowImport(true)}>Import CAS</button>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Fund</button>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="empty-state">
          <h3>No mutual funds yet</h3>
          <p>Add your first mutual fund investment or import from CAS statement.</p>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
            <button className="btn btn-outline" onClick={() => setShowImport(true)}>Import CAS</button>
            <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Fund</button>
          </div>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Fund Name</th>
                <th>Type</th>
                <th>Invested</th>
                <th>Current Value</th>
                <th>NAV</th>
                <th>Units</th>
                <th>Gain/Loss</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(f => (
                <tr key={f.id} style={{ cursor: 'pointer' }} onClick={() => setExpandedFund(expandedFund === f.id ? null : f.id)}>
                  <td>
                    {f.fund_name}
                    {f.is_elss && <span className="badge badge-elss" style={{ marginLeft: 6 }}>ELSS</span>}
                    <div className="text-muted" style={{ fontSize: 12 }}>{f.amc} | Folio: {f.folio_number || '-'}</div>
                  </td>
                  <td>{f.fund_type}</td>
                  <td>{formatCurrency(f.total_invested)}</td>
                  <td>{formatCurrency(f.current_value)}</td>
                  <td>
                    {f.current_nav.toFixed(2)}
                    {f.nav_last_updated && <div className="text-muted" style={{ fontSize: 11 }}>{formatDate(f.nav_last_updated)}</div>}
                  </td>
                  <td>{f.total_units.toFixed(3)}</td>
                  <td className={f.gain_loss >= 0 ? 'text-success' : 'text-danger'}>
                    {formatCurrency(f.gain_loss)}<br />
                    <span style={{ fontSize: 12 }}>{formatPercent(f.gain_loss_percent)}</span>
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }} onClick={e => e.stopPropagation()}>
                      <button className="btn btn-sm btn-outline" onClick={() => setExpandedFund(expandedFund === f.id ? null : f.id)}>
                        {expandedFund === f.id ? 'Hide' : 'Txns'} ({f.transactions?.length || 0})
                      </button>
                      <button className="btn btn-sm btn-outline" onClick={() => openEdit(f)}>Edit</button>
                      <button className="btn btn-sm btn-outline" onClick={() => setShowTx(f.id)}>+ Txn</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(f.id)}>Del</button>
                    </div>
                  </td>
                </tr>
              ))}
              {filtered.map(f => expandedFund === f.id && f.transactions?.length > 0 && (
                <tr key={`${f.id}-txns`}>
                  <td colSpan={8} style={{ padding: 0 }}>
                    <div style={{ background: '#f8f9fa', padding: '12px 16px', borderTop: '1px solid #e0e0e0' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                        <strong style={{ fontSize: 14 }}>Transactions — {f.fund_name}</strong>
                        <span className="text-muted" style={{ fontSize: 12 }}>{f.transactions.length} transactions</span>
                      </div>
                      <table style={{ fontSize: 13, background: '#fff', borderRadius: 6 }}>
                        <thead>
                          <tr>
                            <th>Date</th>
                            <th>Type</th>
                            <th>Amount</th>
                            <th>NAV</th>
                            <th>Units</th>
                            {f.is_elss && <th>Lock-in Ends</th>}
                          </tr>
                        </thead>
                        <tbody>
                          {[...f.transactions].sort((a, b) => b.date.localeCompare(a.date)).map(tx => (
                            <tr key={tx.transaction_id}>
                              <td>{formatDate(tx.date)}</td>
                              <td>
                                <span style={{
                                  padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                                  background: tx.type === 'redemption' || tx.type === 'switch_out' ? '#fde8e8' : '#e8f5e9',
                                  color: tx.type === 'redemption' || tx.type === 'switch_out' ? '#c62828' : '#2e7d32',
                                }}>
                                  {tx.type.toUpperCase()}
                                </span>
                              </td>
                              <td>{formatCurrency(tx.amount)}</td>
                              <td>₹{tx.nav_at_purchase.toFixed(4)}</td>
                              <td>{tx.units.toFixed(3)}</td>
                              {f.is_elss && <td>{tx.lock_in_end ? formatDate(tx.lock_in_end) : '-'}</td>}
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

      {/* Add Fund Modal */}
      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Mutual Fund</h2>
            <form onSubmit={handleAdd}>
              <div className="form-group">
                <label>Fund Name</label>
                <input required value={form.fund_name} onChange={e => setForm({ ...form, fund_name: e.target.value })} />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>AMC</label>
                  <input required value={form.amc} onChange={e => setForm({ ...form, amc: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Fund Type</label>
                  <select value={form.fund_type} onChange={e => setForm({ ...form, fund_type: e.target.value as FundType, is_elss: e.target.value === 'ELSS' })}>
                    {FUND_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>AMFI Scheme Code</label>
                  <input required value={form.scheme_code} onChange={e => setForm({ ...form, scheme_code: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Folio Number</label>
                  <input value={form.folio_number || ''} onChange={e => setForm({ ...form, folio_number: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>
                  <input type="checkbox" checked={form.is_elss} onChange={e => setForm({ ...form, is_elss: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                  This is an ELSS fund (3-year lock-in)
                </label>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Fund</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Fund Modal */}
      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Mutual Fund</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group">
                <label>Fund Name</label>
                <input required value={editForm.fund_name} onChange={e => setEditForm({ ...editForm, fund_name: e.target.value })} />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>AMC</label>
                  <input required value={editForm.amc} onChange={e => setEditForm({ ...editForm, amc: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Fund Type</label>
                  <select value={editForm.fund_type} onChange={e => setEditForm({ ...editForm, fund_type: e.target.value as FundType, is_elss: e.target.value === 'ELSS' })}>
                    {FUND_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>AMFI Scheme Code</label>
                  <input required value={editForm.scheme_code} onChange={e => setEditForm({ ...editForm, scheme_code: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Folio Number</label>
                  <input value={editForm.folio_number || ''} onChange={e => setEditForm({ ...editForm, folio_number: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>
                  <input type="checkbox" checked={editForm.is_elss} onChange={e => setEditForm({ ...editForm, is_elss: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                  ELSS fund
                </label>
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
                  <select value={txForm.type} onChange={e => setTxForm({ ...txForm, type: e.target.value as any })}>
                    <option value="purchase">Purchase</option>
                    <option value="sip">SIP</option>
                    <option value="redemption">Redemption</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Amount (₹)</label>
                  <input type="number" step="0.01" required value={txForm.amount || ''} onChange={e => setTxForm({ ...txForm, amount: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>NAV at Purchase</label>
                  <input type="number" step="0.0001" required value={txForm.nav_at_purchase || ''} onChange={e => setTxForm({ ...txForm, nav_at_purchase: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="form-group">
                <label>Units (optional, auto-calculated if empty)</label>
                <input type="number" step="0.001" value={txForm.units || ''} onChange={e => setTxForm({ ...txForm, units: parseFloat(e.target.value) })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowTx(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Transaction</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Import CAS Modal */}
      {showImport && (
        <div className="modal-overlay" onClick={closeImport}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: importStep === 'preview' ? 800 : 500 }}>
            <h2>Import from CAS Statement</h2>

            {importStep === 'upload' && (
              <div>
                <p className="text-muted" style={{ marginBottom: 16 }}>Upload your CAMS/KFintech/MFCentral CAS PDF to import mutual fund transactions.</p>
                <div className="form-group">
                  <label>CAS PDF File</label>
                  <input type="file" accept=".pdf" onChange={e => setImportFile(e.target.files?.[0] || null)} />
                </div>
                <div className="form-group">
                  <label>PDF Password (usually your PAN)</label>
                  <input type="password" value={importPassword} onChange={e => setImportPassword(e.target.value)} placeholder="e.g. ABCDE1234F" />
                </div>
                {importMessage && <div className={importMessage.includes('Error') || importMessage.includes('Incorrect') || importMessage.includes('No transactions') ? 'text-danger' : 'text-success'} style={{ marginBottom: 12 }}>{importMessage}</div>}
                <div className="modal-actions">
                  <button type="button" className="btn btn-outline" onClick={closeImport}>Cancel</button>
                  <button className="btn btn-primary" onClick={handleParsePDF} disabled={!importFile || importLoading}>
                    {importLoading ? 'Parsing...' : 'Parse PDF'}
                  </button>
                </div>
              </div>
            )}

            {importStep === 'preview' && (
              <div>
                <p style={{ marginBottom: 12 }}>Found <strong>{parsedFunds.length}</strong> funds with <strong>{parsedFunds.reduce((s, f) => s + f.transactions.length, 0)}</strong> transactions:</p>
                <div style={{ maxHeight: 400, overflow: 'auto', marginBottom: 16 }}>
                  {parsedFunds.map((f, fi) => (
                    <div key={fi} style={{ marginBottom: 16 }}>
                      <h4 style={{ margin: '0 0 4px' }}>{f.fund_name}</h4>
                      <div className="text-muted" style={{ fontSize: 12, marginBottom: 8 }}>Folio: {f.folio_number} | {f.category} | {f.transactions.length} transactions</div>
                      <table style={{ fontSize: 12 }}>
                        <thead>
                          <tr><th>Date</th><th>Type</th><th>Units</th><th>NAV</th><th>Amount</th></tr>
                        </thead>
                        <tbody>
                          {f.transactions.map((tx, ti) => (
                            <tr key={ti}>
                              <td>{tx.date}</td>
                              <td>{tx.type}</td>
                              <td>{tx.units.toFixed(3)}</td>
                              <td>₹{tx.nav.toFixed(4)}</td>
                              <td>{formatCurrency(tx.amount)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ))}
                </div>
                {importMessage && <div className="text-success" style={{ marginBottom: 12 }}>{importMessage}</div>}
                <div className="modal-actions">
                  <button className="btn btn-outline" onClick={() => { setImportStep('upload'); setParsedFunds([]); }}>Back</button>
                  <button className="btn btn-primary" onClick={handleImport} disabled={importLoading}>
                    {importLoading ? 'Importing...' : `Import ${parsedFunds.reduce((s, f) => s + f.transactions.length, 0)} Transactions`}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
