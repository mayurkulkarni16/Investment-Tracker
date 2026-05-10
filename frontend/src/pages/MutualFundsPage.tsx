import { useEffect, useState } from 'react';
import { getMutualFunds, createMutualFund, deleteMutualFund, addMFTransaction, deleteMFTransaction, refreshNAV, updateMutualFund, importFromCAS } from '../api/mutualFunds';
import { getSIPs, createSIP, updateSIP, deleteSIP, recordSIPInstallment } from '../api/sips';
import type { MutualFund, CreateMutualFundRequest, AddMFTransactionRequest, FundType, UpdateMutualFundRequest, SIP, CreateSIPRequest, UpdateSIPRequest, RecordSIPInstallmentRequest, TransactionType } from '../types';
import { formatCurrency, formatPercent, formatDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';
import { extractTextFromPDF, parseCASText } from '../utils/casParser';
import type { ParsedCASFund } from '../utils/casParser';

const FUND_TYPES: FundType[] = ['Equity', 'Debt', 'Hybrid', 'ELSS', 'Index', 'Liquid'];

export default function MutualFundsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [funds, setFunds] = useState<MutualFund[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showTx, setShowTx] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<MutualFund | null>(null);
  const [showImport, setShowImport] = useState(false);
  const [expandedFund, setExpandedFund] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'elss'>('all');
  const [search, setSearch] = useState('');
  const [tab, setTab] = useState<'funds' | 'sips'>('funds');
  const [submitting, setSubmitting] = useState(false);
  const [sortCol, setSortCol] = useState<string>('');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const { toast } = useToast();
  const { confirm } = useConfirm();

  // SIP state
  const [sips, setSips] = useState<SIP[]>([]);
  const [showAddSIP, setShowAddSIP] = useState(false);
  const [showEditSIP, setShowEditSIP] = useState<SIP | null>(null);
  const [showRecordSIP, setShowRecordSIP] = useState<string | null>(null);
  const [expandedSIP, setExpandedSIP] = useState<string | null>(null);
  const [sipForm, setSipForm] = useState<CreateSIPRequest>({ fund_name: '', amount: 0, frequency: 'monthly', sip_date: 1, start_date: '' });
  const [editSIPForm, setEditSIPForm] = useState<UpdateSIPRequest>({});
  const [recordForm, setRecordForm] = useState<RecordSIPInstallmentRequest>({ date: '', amount: 0, status: 'success' });
  const [showSIPImport, setShowSIPImport] = useState(false);
  const [sipImportFile, setSipImportFile] = useState<File | null>(null);
  const [sipImportPassword, setSipImportPassword] = useState('');
  const [sipImportLoading, setSipImportLoading] = useState(false);
  const [sipImportMessage, setSipImportMessage] = useState('');
  const [sipImportStep, setSipImportStep] = useState<'upload' | 'preview'>('upload');
  const [parsedSIPs, setParsedSIPs] = useState<{ fund_name: string; folio: string; amount: number; sip_date: number; count: number; first_date: string; last_date: string; installments: { date: string; amount: number; nav: number; units: number }[] }[]>([]);

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
    Promise.all([
      getMutualFunds().then(r => setFunds(r.data || [])),
      getSIPs().then(r => setSips(r.data || [])),
    ]).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [viewAsUserId]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await createMutualFund(form);
      setShowAdd(false);
      setForm({ fund_name: '', amc: '', fund_type: 'Equity', scheme_code: '', is_elss: false });
      toast('Mutual fund added successfully');
      load();
    } catch { toast('Failed to add mutual fund', 'error'); }
    finally { setSubmitting(false); }
  };

  const handleAddTx = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showTx) return;
    setSubmitting(true);
    try {
      await addMFTransaction(showTx, txForm);
      setShowTx(null);
      setTxForm({ date: '', type: 'purchase', amount: 0, nav_at_purchase: 0 });
      toast('Transaction added');
      load();
    } catch { toast('Failed to add transaction', 'error'); }
    finally { setSubmitting(false); }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    setSubmitting(true);
    try {
      await updateMutualFund(showEdit.id, editForm);
      setShowEdit(null);
      toast('Fund updated');
      load();
    } catch { toast('Failed to update fund', 'error'); }
    finally { setSubmitting(false); }
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
    if (!await confirm({ message: 'Delete this mutual fund?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteMutualFund(id); toast('Fund deleted'); load(); }
    catch { toast('Failed to delete fund', 'error'); }
  };

  const handleDeleteTxn = async (fundId: string, txnId: string) => {
    if (!await confirm({ message: 'Delete this transaction? Fund totals will be recalculated.', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteMFTransaction(fundId, txnId); toast('Transaction deleted'); load(); }
    catch { toast('Failed to delete transaction', 'error'); }
  };

  const handleRefresh = async () => {
    setSubmitting(true);
    try { await refreshNAV(); toast('NAV refreshed'); load(); }
    catch { toast('Failed to refresh NAV', 'error'); }
    finally { setSubmitting(false); }
  };

  // SIP handlers
  const handleAddSIP = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const fund = funds.find(f => f.id === sipForm.fund_id);
      await createSIP({ ...sipForm, fund_name: fund?.fund_name || sipForm.fund_name });
      setShowAddSIP(false);
      setSipForm({ fund_name: '', amount: 0, frequency: 'monthly', sip_date: 1, start_date: '' });
      toast('SIP added');
      load();
    } catch { toast('Failed to add SIP', 'error'); }
    finally { setSubmitting(false); }
  };

  const handleRecordSIP = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showRecordSIP) return;
    setSubmitting(true);
    try {
      await recordSIPInstallment(showRecordSIP, recordForm);
      setShowRecordSIP(null);
      setRecordForm({ date: '', amount: 0, status: 'success' });
      toast('Installment recorded');
      load();
    } catch { toast('Failed to record installment', 'error'); }
    finally { setSubmitting(false); }
  };

  const handleEditSIP = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEditSIP) return;
    setSubmitting(true);
    try {
      await updateSIP(showEditSIP.id, editSIPForm);
      setShowEditSIP(null);
      toast('SIP updated');
      load();
    } catch { toast('Failed to update SIP', 'error'); }
    finally { setSubmitting(false); }
  };

  const handleToggleSIP = async (sip: SIP) => {
    const newStatus = sip.status === 'active' ? 'paused' : 'active';
    try { await updateSIP(sip.id, { status: newStatus }); toast(`SIP ${newStatus}`); load(); }
    catch { toast('Failed to update SIP', 'error'); }
  };

  const handleDeleteSIP = async (id: string) => {
    if (!await confirm({ message: 'Delete this SIP?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteSIP(id); toast('SIP deleted'); load(); }
    catch { toast('Failed to delete SIP', 'error'); }
  };

  // SIP CAS Import
  const handleParseSIPCAS = async () => {
    if (!sipImportFile) return;
    setSipImportLoading(true);
    setSipImportMessage('');
    try {
      const text = await extractTextFromPDF(sipImportFile, sipImportPassword);
      const casFunds = parseCASText(text);
      const sipTxns = casFunds.map(f => {
        const sipOnly = f.transactions.filter(t => t.type.includes('SIP'));
        if (sipOnly.length === 0) return null;
        const amounts = sipOnly.map(t => t.amount);
        const mostCommon = amounts.sort((a, b) => amounts.filter(v => v === a).length - amounts.filter(v => v === b).length).pop() || 0;
        const dates = sipOnly.map(t => new Date(t.date).getDate());
        const mostCommonDate = dates.sort((a, b) => dates.filter(v => v === a).length - dates.filter(v => v === b).length).pop() || 1;
        return {
          fund_name: f.fund_name, folio: f.folio_number, amount: mostCommon, sip_date: mostCommonDate,
          count: sipOnly.length, first_date: sipOnly[0].date, last_date: sipOnly[sipOnly.length - 1].date,
          installments: sipOnly.map(t => ({ date: t.date, amount: t.amount, nav: t.nav, units: t.units })),
        };
      }).filter(Boolean) as typeof parsedSIPs;
      if (sipTxns.length === 0) {
        setSipImportMessage('No SIP transactions found in the CAS.');
      } else {
        setParsedSIPs(sipTxns);
        setSipImportStep('preview');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to parse PDF';
      setSipImportMessage(msg.includes('password') ? 'Incorrect password.' : `Error: ${msg}`);
    } finally {
      setSipImportLoading(false);
    }
  };

  const handleImportSIPs = async () => {
    setSipImportLoading(true);
    let created = 0, recorded = 0;
    try {
      for (const ps of parsedSIPs) {
        const matchedFund = funds.find(f => f.folio_number === ps.folio || f.fund_name === ps.fund_name);
        const res = await createSIP({ fund_name: ps.fund_name, fund_id: matchedFund?.id, amount: ps.amount, frequency: 'monthly', sip_date: ps.sip_date, start_date: ps.first_date });
        created++;
        const sipId = res.data?.id;
        if (sipId) {
          for (const inst of ps.installments) {
            await recordSIPInstallment(sipId, { date: inst.date, amount: inst.amount, nav: inst.nav, units: inst.units, status: 'success' });
            recorded++;
          }
        }
      }
      setSipImportMessage(`Imported ${created} SIPs with ${recorded} installments.`);
      setSipImportStep('upload');
      setParsedSIPs([]);
      load();
    } catch {
      setSipImportMessage('Import failed. Some SIPs may have been partially imported.');
    } finally {
      setSipImportLoading(false);
    }
  };

  const closeSIPImport = () => {
    setShowSIPImport(false);
    setSipImportStep('upload');
    setParsedSIPs([]);
    setSipImportFile(null);
    setSipImportPassword('');
    setSipImportMessage('');
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

  const filtered = (filter === 'elss' ? funds.filter(f => f.is_elss) : funds)
    .filter(f => !search || f.fund_name.toLowerCase().includes(search.toLowerCase()) || f.amc?.toLowerCase().includes(search.toLowerCase()) || f.folio_number?.toLowerCase().includes(search.toLowerCase()));

  const toggleSort = (col: string) => { if (sortCol === col) setSortDir(d => d === 'asc' ? 'desc' : 'asc'); else { setSortCol(col); setSortDir('asc'); } };
  const sorted = [...filtered].sort((a, b) => {
    if (!sortCol) return 0;
    const dir = sortDir === 'asc' ? 1 : -1;
    const av = (a as any)[sortCol], bv = (b as any)[sortCol];
    if (typeof av === 'string') return av.localeCompare(bv) * dir;
    return ((av ?? 0) - (bv ?? 0)) * dir;
  });

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Mutual Funds</h1>
        <div style={{ display: 'flex', gap: 8 }}>
          {tab === 'funds' && (
            <>
              <button className="btn btn-outline" onClick={() => setFilter(filter === 'all' ? 'elss' : 'all')}>
                {filter === 'all' ? 'Show ELSS Only' : 'Show All'}
              </button>
              {!isViewOnly && <button className="btn btn-outline" disabled={submitting} onClick={handleRefresh}>{submitting ? 'Refreshing...' : 'Refresh NAV'}</button>}
              {!isViewOnly && <button className="btn btn-outline" onClick={() => setShowImport(true)}>Import CAS</button>}
              {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Fund</button>}
            </>
          )}
          {tab === 'sips' && (
            <>
              {!isViewOnly && <button className="btn btn-outline" onClick={() => setShowSIPImport(true)}>Import from CAS</button>}
              {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAddSIP(true)}>+ Add SIP</button>}
            </>
          )}
        </div>
      </div>

      <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
        <button className={`btn ${tab === 'funds' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setTab('funds')}>
          Funds ({funds.length})
        </button>
        <button className={`btn ${tab === 'sips' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setTab('sips')}>
          SIPs ({sips.length})
        </button>
      </div>

      {tab === 'funds' && (<>
      <div className="search-bar">
        <input type="search" placeholder="Search funds by name, AMC, or folio..." value={search} onChange={e => setSearch(e.target.value)} />
      </div>
      {filtered.length > 0 && (() => {
        const totalInvested = filtered.reduce((s, f) => s + f.total_invested, 0);
        const totalCurrent = filtered.reduce((s, f) => s + f.current_value, 0);
        const totalGain = totalCurrent - totalInvested;
        const gainPct = totalInvested > 0 ? (totalGain / totalInvested) * 100 : 0;
        return (
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
            <div className="stat-card">
              <div className="label">Total Funds</div>
              <div className="value">{filtered.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Invested</div>
              <div className="value">{formatCurrency(totalInvested)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Current Value</div>
              <div className="value">{formatCurrency(totalCurrent)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Gain/Loss</div>
              <div className={`value ${totalGain >= 0 ? 'positive' : 'negative'}`}>
                {formatCurrency(totalGain)} ({formatPercent(gainPct)})
              </div>
            </div>
            <div className="stat-card">
              <div className="label">Avg XIRR</div>
              <div className={`value ${(() => { const w = filtered.filter(f => f.total_invested > 0); const t = w.reduce((s, f) => s + f.total_invested, 0); const x = t > 0 ? w.reduce((s, f) => s + f.xirr * f.total_invested, 0) / t : 0; return x >= 0 ? 'positive' : 'negative'; })()}`}>
                {formatPercent((() => { const w = filtered.filter(f => f.total_invested > 0); const t = w.reduce((s, f) => s + f.total_invested, 0); return t > 0 ? w.reduce((s, f) => s + f.xirr * f.total_invested, 0) / t : 0; })())}
              </div>
            </div>
          </div>
        );
      })()}
      {filtered.length === 0 ? (
        <div className="empty-state">
          <h3>No mutual funds yet</h3>
          <p>Add your first mutual fund investment or import from CAS statement.</p>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
            {!isViewOnly && <button className="btn btn-outline" onClick={() => setShowImport(true)}>Import CAS</button>}
            {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add Fund</button>}
          </div>
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('fund_name')}>Fund Name {sortCol === 'fund_name' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('fund_type')}>Type {sortCol === 'fund_type' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('total_invested')}>Invested {sortCol === 'total_invested' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('current_value')}>Current Value {sortCol === 'current_value' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('current_nav')}>NAV {sortCol === 'current_nav' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('total_units')}>Units {sortCol === 'total_units' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('gain_loss')}>Gain/Loss {sortCol === 'gain_loss' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => toggleSort('xirr')}>XIRR {sortCol === 'xirr' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map(f => (
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
                  <td className={f.xirr >= 0 ? 'text-success' : 'text-danger'}>
                    {formatPercent(f.xirr)}
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }} onClick={e => e.stopPropagation()}>
                      <button className="btn btn-sm btn-outline" onClick={() => setExpandedFund(expandedFund === f.id ? null : f.id)}>
                        {expandedFund === f.id ? 'Hide' : 'Txns'} ({f.transactions?.length || 0})
                      </button>
                      {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => openEdit(f)}>Edit</button>}
                      {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowTx(f.id)}>+ Txn</button>}
                      {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDelete(f.id)}>Del</button>}
                    </div>
                  </td>
                </tr>
              ))}
              {sorted.map(f => expandedFund === f.id && f.transactions?.length > 0 && (
                <tr key={`${f.id}-txns`}>
                  <td colSpan={9} style={{ padding: 0 }}>
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
                            <th>Actions</th>
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
                              {!isViewOnly && <td><button className="btn btn-sm btn-danger" onClick={e => { e.stopPropagation(); handleDeleteTxn(f.id, tx.transaction_id); }}>Del</button></td>}
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
      {filtered.length > 0 && <p className="text-muted" style={{ fontSize: 11, marginTop: 6 }}>NAV data via mfapi.in • Prices may be delayed by 1 business day</p>}
      </>)}

      {tab === 'sips' && (
        <>
          {sips.length === 0 ? (
            <div className="empty-state">
              <h3>No SIPs set up</h3>
              <p>Add a Systematic Investment Plan to automate your mutual fund investing.</p>
              {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAddSIP(true)}>+ Add SIP</button>}
            </div>
          ) : (
            <>
              <div className="card-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', marginBottom: 20 }}>
                <div className="stat-card">
                  <div className="label">Active SIPs</div>
                  <div className="value">{sips.filter(s => s.status === 'active').length}</div>
                </div>
                <div className="stat-card">
                  <div className="label">Monthly SIP Amount</div>
                  <div className="value">{formatCurrency(sips.filter(s => s.status === 'active').reduce((s, p) => s + p.amount, 0))}</div>
                </div>
                <div className="stat-card">
                  <div className="label">Total SIP Invested</div>
                  <div className="value positive">{formatCurrency(sips.reduce((s, p) => s + p.total_invested, 0))}</div>
                </div>
              </div>

              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>Fund</th>
                      <th>Amount</th>
                      <th>Frequency</th>
                      <th>SIP Date</th>
                      <th>Next SIP</th>
                      <th>Installments</th>
                      <th>Total Invested</th>
                      <th>Status</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sips.map(sip => (
                      <tr key={sip.id} style={{ cursor: 'pointer' }} onClick={() => setExpandedSIP(expandedSIP === sip.id ? null : sip.id)}>
                        <td>
                          {sip.fund_name}
                          <div className="text-muted" style={{ fontSize: 12 }}>Since {formatDate(sip.start_date)} | {sip.months_active} months</div>
                        </td>
                        <td>{formatCurrency(sip.amount)}</td>
                        <td style={{ textTransform: 'capitalize' }}>{sip.frequency}</td>
                        <td>{sip.sip_date}<sup>{sip.sip_date === 1 ? 'st' : sip.sip_date === 2 ? 'nd' : sip.sip_date === 3 ? 'rd' : 'th'}</sup></td>
                        <td>{sip.status === 'active' ? formatDate(sip.next_sip_date) : '-'}</td>
                        <td>{sip.completed_installments}/{sip.total_installments}{sip.missed_installments > 0 && <span className="text-danger"> ({sip.missed_installments} missed)</span>}</td>
                        <td>{formatCurrency(sip.total_invested)}</td>
                        <td><span className={`badge ${sip.status === 'active' ? 'badge-active' : sip.status === 'paused' ? 'badge-pending' : 'badge-matured'}`}>{sip.status}</span></td>
                        <td>
                          <div style={{ display: 'flex', gap: 4 }} onClick={e => e.stopPropagation()}>
                            {!isViewOnly && <button className="btn btn-sm btn-success" onClick={() => { setShowRecordSIP(sip.id); setRecordForm({ date: '', amount: sip.amount, status: 'success' }); }}>Record</button>}
                            {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => { setShowEditSIP(sip); setEditSIPForm({ amount: sip.amount, sip_date: sip.sip_date, end_date: sip.end_date, notes: sip.notes }); }}>Edit</button>}
                            {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => handleToggleSIP(sip)}>{sip.status === 'active' ? 'Pause' : 'Resume'}</button>}
                            {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDeleteSIP(sip.id)}>Del</button>}
                          </div>
                        </td>
                      </tr>
                    ))}
                    {sips.map(sip => expandedSIP === sip.id && sip.installments?.length > 0 && (
                      <tr key={`${sip.id}-inst`}>
                        <td colSpan={9} style={{ padding: 0 }}>
                          <div style={{ background: '#f8f9fa', padding: '12px 16px', borderTop: '1px solid #e0e0e0' }}>
                            <strong style={{ fontSize: 14 }}>Installment History — {sip.fund_name}</strong>
                            <table style={{ fontSize: 13, background: '#fff', borderRadius: 6, marginTop: 8 }}>
                              <thead><tr><th>Date</th><th>Amount</th><th>NAV</th><th>Units</th><th>Status</th></tr></thead>
                              <tbody>
                                {[...sip.installments].reverse().map(inst => (
                                  <tr key={inst.installment_id}>
                                    <td>{formatDate(inst.date)}</td>
                                    <td>{formatCurrency(inst.amount)}</td>
                                    <td>{inst.nav ? `₹${inst.nav.toFixed(4)}` : '-'}</td>
                                    <td>{inst.units ? inst.units.toFixed(3) : '-'}</td>
                                    <td><span className={`badge ${inst.status === 'success' ? 'badge-active' : inst.status === 'skipped' ? 'badge-pending' : 'badge-matured'}`}>{inst.status}</span></td>
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
            </>
          )}
        </>
      )}

      {/* Add SIP Modal */}
      {showAddSIP && (
        <div className="modal-overlay" onClick={() => setShowAddSIP(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 480 }}>
            <h2>Add SIP</h2>
            <form onSubmit={handleAddSIP}>
              <div className="form-group">
                <label>Mutual Fund</label>
                {funds.length > 0 ? (
                  <select value={sipForm.fund_id || ''} onChange={e => { const f = funds.find(f => f.id === e.target.value); setSipForm({ ...sipForm, fund_id: e.target.value, fund_name: f?.fund_name || '', scheme_code: f?.scheme_code ? parseInt(f.scheme_code) : undefined }); }}>
                    <option value="">-- Select a fund --</option>
                    {funds.map(f => <option key={f.id} value={f.id}>{f.fund_name} ({f.amc})</option>)}
                  </select>
                ) : (
                  <input required value={sipForm.fund_name} onChange={e => setSipForm({ ...sipForm, fund_name: e.target.value })} placeholder="Fund name (add funds first for dropdown)" />
                )}
              </div>
              <div className="form-row">
                <div className="form-group"><label>SIP Amount (₹)</label><input type="number" required value={sipForm.amount || ''} onChange={e => setSipForm({ ...sipForm, amount: +e.target.value })} /></div>
                <div className="form-group">
                  <label>Frequency</label>
                  <select value={sipForm.frequency} onChange={e => setSipForm({ ...sipForm, frequency: e.target.value })}>
                    <option value="monthly">Monthly</option><option value="weekly">Weekly</option><option value="quarterly">Quarterly</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>SIP Date (day of month)</label><input type="number" min={1} max={28} required value={sipForm.sip_date} onChange={e => setSipForm({ ...sipForm, sip_date: +e.target.value })} /></div>
                <div className="form-group"><label>Start Date</label><input type="date" required value={sipForm.start_date} onChange={e => setSipForm({ ...sipForm, start_date: e.target.value })} /></div>
              </div>
              <div className="form-group"><label>End Date (optional)</label><input type="date" value={sipForm.end_date || ''} onChange={e => setSipForm({ ...sipForm, end_date: e.target.value })} /></div>
              <div className="form-group"><label>Notes</label><input value={sipForm.notes || ''} onChange={e => setSipForm({ ...sipForm, notes: e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAddSIP(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={submitting || (!sipForm.fund_name && !sipForm.fund_id)}>{submitting ? 'Adding...' : 'Add SIP'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Record SIP Installment Modal */}
      {showRecordSIP && (
        <div className="modal-overlay" onClick={() => setShowRecordSIP(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 420 }}>
            <h2>Record Installment</h2>
            <form onSubmit={handleRecordSIP}>
              <div className="form-group"><label>Date</label><input type="date" required value={recordForm.date} onChange={e => setRecordForm({ ...recordForm, date: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Amount (₹)</label><input type="number" required value={recordForm.amount || ''} onChange={e => setRecordForm({ ...recordForm, amount: +e.target.value })} /></div>
                <div className="form-group">
                  <label>Status</label>
                  <select value={recordForm.status} onChange={e => setRecordForm({ ...recordForm, status: e.target.value })}>
                    <option value="success">Success</option><option value="failed">Failed</option><option value="skipped">Skipped</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>NAV (optional)</label><input type="number" step="0.0001" value={recordForm.nav || ''} onChange={e => setRecordForm({ ...recordForm, nav: +e.target.value })} /></div>
                <div className="form-group"><label>Units (optional)</label><input type="number" step="0.001" value={recordForm.units || ''} onChange={e => setRecordForm({ ...recordForm, units: +e.target.value })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowRecordSIP(null)}>Cancel</button>
                <button type="submit" className="btn btn-success" disabled={submitting}>{submitting ? 'Recording...' : 'Record'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* SIP CAS Import Modal */}
      {showSIPImport && (
        <div className="modal-overlay" onClick={closeSIPImport}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: sipImportStep === 'preview' ? 800 : 500 }}>
            <h2>Import SIPs from CAS</h2>
            {sipImportStep === 'upload' && (
              <div>
                <p className="text-muted" style={{ marginBottom: 16 }}>Upload your CAS PDF to auto-detect SIP transactions and create SIP trackers with full installment history.</p>
                <div className="form-group"><label>CAS PDF File</label><input type="file" accept=".pdf" onChange={e => setSipImportFile(e.target.files?.[0] || null)} /></div>
                <div className="form-group"><label>PDF Password (usually your PAN)</label><input type="password" value={sipImportPassword} onChange={e => setSipImportPassword(e.target.value)} placeholder="e.g. ABCDE1234F" /></div>
                {sipImportMessage && <div className={sipImportMessage.includes('Error') || sipImportMessage.includes('Incorrect') || sipImportMessage.includes('No SIP') ? 'text-danger' : 'text-success'} style={{ marginBottom: 12 }}>{sipImportMessage}</div>}
                <div className="modal-actions">
                  <button type="button" className="btn btn-outline" onClick={closeSIPImport}>Cancel</button>
                  <button className="btn btn-primary" onClick={handleParseSIPCAS} disabled={!sipImportFile || sipImportLoading}>{sipImportLoading ? 'Parsing...' : 'Parse PDF'}</button>
                </div>
              </div>
            )}
            {sipImportStep === 'preview' && (
              <div>
                <p style={{ marginBottom: 12 }}>Found <strong>{parsedSIPs.length}</strong> SIPs with <strong>{parsedSIPs.reduce((s, p) => s + p.count, 0)}</strong> total installments:</p>
                <div style={{ maxHeight: 400, overflow: 'auto', marginBottom: 16 }}>
                  <table style={{ fontSize: 13 }}>
                    <thead><tr><th>Fund Name</th><th>SIP Amount</th><th>Date</th><th>Installments</th><th>Period</th></tr></thead>
                    <tbody>
                      {parsedSIPs.map((s, i) => (
                        <tr key={i}>
                          <td>{s.fund_name}<div className="text-muted" style={{ fontSize: 11 }}>Folio: {s.folio}</div></td>
                          <td>{formatCurrency(s.amount)}</td>
                          <td>{s.sip_date}<sup>{s.sip_date === 1 ? 'st' : s.sip_date === 2 ? 'nd' : s.sip_date === 3 ? 'rd' : 'th'}</sup></td>
                          <td>{s.count}</td>
                          <td className="text-muted" style={{ fontSize: 12 }}>{s.first_date} → {s.last_date}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {sipImportMessage && <div className={sipImportMessage.includes('failed') ? 'text-danger' : 'text-success'} style={{ marginBottom: 12 }}>{sipImportMessage}</div>}
                <div className="modal-actions">
                  <button className="btn btn-outline" onClick={() => { setSipImportStep('upload'); setParsedSIPs([]); }}>Back</button>
                  <button className="btn btn-primary" onClick={handleImportSIPs} disabled={sipImportLoading}>{sipImportLoading ? 'Importing...' : `Import ${parsedSIPs.length} SIPs`}</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Edit SIP Modal */}
      {showEditSIP && (
        <div className="modal-overlay" onClick={() => setShowEditSIP(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 420 }}>
            <h2>Edit SIP — {showEditSIP.fund_name}</h2>
            <form onSubmit={handleEditSIP}>
              <div className="form-row">
                <div className="form-group"><label>SIP Amount (₹)</label><input type="number" required value={editSIPForm.amount || ''} onChange={e => setEditSIPForm({ ...editSIPForm, amount: +e.target.value })} /></div>
                <div className="form-group"><label>SIP Date (day)</label><input type="number" min={1} max={28} value={editSIPForm.sip_date || ''} onChange={e => setEditSIPForm({ ...editSIPForm, sip_date: +e.target.value })} /></div>
              </div>
              <div className="form-group"><label>End Date (optional)</label><input type="date" value={editSIPForm.end_date || ''} onChange={e => setEditSIPForm({ ...editSIPForm, end_date: e.target.value })} /></div>
              <div className="form-group"><label>Notes</label><textarea value={editSIPForm.notes || ''} onChange={e => setEditSIPForm({ ...editSIPForm, notes: e.target.value })} rows={2} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEditSIP(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={submitting}>{submitting ? 'Saving...' : 'Save Changes'}</button>
              </div>
            </form>
          </div>
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
                <button type="submit" className="btn btn-primary" disabled={submitting}>{submitting ? 'Adding...' : 'Add Fund'}</button>
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
                <button type="submit" className="btn btn-primary" disabled={submitting}>{submitting ? 'Saving...' : 'Save Changes'}</button>
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
                  <select value={txForm.type} onChange={e => setTxForm({ ...txForm, type: e.target.value as TransactionType })}>
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
                <button type="submit" className="btn btn-primary" disabled={submitting}>{submitting ? 'Adding...' : 'Add Transaction'}</button>
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
