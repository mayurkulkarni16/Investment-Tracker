import { useEffect, useState } from 'react';
import { getProvidentFunds, createProvidentFund, addContribution, updateProvidentFund, deleteProvidentFund, importPFFromPDF } from '../api/providentFund';
import type { ProvidentFund, CreateProvidentFundRequest, AddMonthlyContributionRequest, PFAccountType } from '../types';
import { formatCurrency, formatPercent } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { extractPFFromPDF } from '../utils/pfParser';
import type { ParsedPFData } from '../utils/pfParser';

export default function ProvidentFundPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [funds, setFunds] = useState<ProvidentFund[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showContrib, setShowContrib] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<ProvidentFund | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [form, setForm] = useState<CreateProvidentFundRequest>({ account_type: 'EPF', account_number: '', interest_rate: 8.25 });
  const [editForm, setEditForm] = useState<CreateProvidentFundRequest>({ account_type: 'EPF', account_number: '', interest_rate: 8.25 });
  const [contribForm, setContribForm] = useState<AddMonthlyContributionRequest>({ financial_year: '2025-2026', month: '', employee_contribution: 0, employer_contribution: 0 });

  // Import state
  const [showImport, setShowImport] = useState<string | null>(null);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importProgress, setImportProgress] = useState('');
  const [importStep, setImportStep] = useState<'upload' | 'preview'>('upload');
  const [parsedPF, setParsedPF] = useState<ParsedPFData | null>(null);
  const [importMessage, setImportMessage] = useState('');

  const load = () => {
    getProvidentFunds().then(r => setFunds(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [viewAsUserId]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await createProvidentFund(form); setShowAdd(false); setForm({ account_type: 'EPF', account_number: '', interest_rate: 8.25 }); toast('PF account added'); load(); }
    catch { toast('Failed to add PF account', 'error'); }
  };

  const handleContrib = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showContrib) return;
    try { await addContribution(showContrib, contribForm); setShowContrib(null); setContribForm({ financial_year: '2025-2026', month: '', employee_contribution: 0, employer_contribution: 0 }); toast('Contribution added'); load(); }
    catch { toast('Failed to add contribution', 'error'); }
  };

  const openEdit = (pf: ProvidentFund) => {
    setEditForm({
      account_type: pf.account_type,
      account_number: pf.account_number,
      employer_name: pf.employer_name,
      interest_rate: pf.interest_rate,
      notes: pf.notes,
    });
    setShowEdit(pf);
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showEdit) return;
    try { await updateProvidentFund(showEdit.id, editForm); setShowEdit(null); toast('PF account updated'); load(); }
    catch { toast('Failed to update PF account', 'error'); }
  };

  const handleDelete = async (id: string) => {
    if (!await confirm({ message: 'Delete this PF account?', danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteProvidentFund(id); toast('PF account deleted'); load(); }
    catch { toast('Failed to delete PF account', 'error'); }
  };

  const handleParsePF = async () => {
    if (!importFile) return;
    setImportLoading(true);
    setImportMessage('');
    setImportProgress('');
    try {
      const data = await extractPFFromPDF(importFile, (msg) => setImportProgress(msg));
      const totalContribs = data.years.reduce((s, y) => s + y.contributions.length, 0);
      if (data.years.length === 0 && totalContribs === 0) {
        setImportMessage('No contributions found in the PDF. OCR may have failed to read the document.');
      } else {
        setParsedPF(data);
        setImportStep('preview');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to process PDF';
      setImportMessage(`Error: ${msg}`);
    } finally {
      setImportLoading(false);
      setImportProgress('');
    }
  };

  const handleImportPF = async () => {
    if (!showImport || !parsedPF) return;
    setImportLoading(true);
    try {
      const res = await importPFFromPDF(showImport, parsedPF);
      const r = res.data as { years_added: number; contributions_added: number; errors?: string[] };
      setImportMessage(`Imported: ${r.years_added} years, ${r.contributions_added} contributions.${r.errors?.length ? ' Errors: ' + r.errors.join('; ') : ''}`);
      setImportStep('upload');
      setParsedPF(null);
      load();
    } catch {
      setImportMessage('Failed to import. Check backend is running.');
    } finally {
      setImportLoading(false);
    }
  };

  const closeImport = () => {
    setShowImport(null);
    setImportStep('upload');
    setParsedPF(null);
    setImportFile(null);
    setImportMessage('');
    setImportProgress('');
  };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Provident Fund</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add PF Account</button>}
      </div>

      {funds.length > 0 && (() => {
        const totalBalance = funds.reduce((s, f) => s + f.current_balance, 0);
        const totalEmployee = funds.reduce((s, f) => s + f.total_employee_contribution, 0);
        const totalEmployer = funds.reduce((s, f) => s + f.total_employer_contribution, 0);
        const totalInterest = funds.reduce((s, f) => s + f.total_interest_earned, 0);
        return (
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: 20 }}>
            <div className="stat-card">
              <div className="label">Total Balance</div>
              <div className="value">{formatCurrency(totalBalance)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Employee Contribution</div>
              <div className="value">{formatCurrency(totalEmployee)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Employer Contribution</div>
              <div className="value">{formatCurrency(totalEmployer)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Interest Earned</div>
              <div className="value positive">{formatCurrency(totalInterest)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Avg XIRR</div>
              <div className={`value ${(() => { const w = funds.filter(p => p.current_balance > 0); const t = w.reduce((s, p) => s + p.current_balance, 0); const x = t > 0 ? w.reduce((s, p) => s + p.xirr * p.current_balance, 0) / t : 0; return x >= 0 ? 'positive' : 'negative'; })()}`}>
                {formatPercent((() => { const w = funds.filter(p => p.current_balance > 0); const t = w.reduce((s, p) => s + p.current_balance, 0); return t > 0 ? w.reduce((s, p) => s + p.xirr * p.current_balance, 0) / t : 0; })())}
              </div>
            </div>
          </div>
        );
      })()}

      {funds.length === 0 ? (
        <div className="empty-state">
          <h3>No PF accounts yet</h3>
          <p>Add your EPF, VPF, or PPF account to start tracking.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Add PF Account</button>}
        </div>
      ) : (
        <>
        <div className="search-bar" style={{ marginBottom: 16 }}>
          <input type="search" placeholder="Search by employer name or account number..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        {funds.filter(pf => !search || pf.employer_name?.toLowerCase().includes(search.toLowerCase()) || pf.account_number.toLowerCase().includes(search.toLowerCase())).map(pf => (
          <div key={pf.id} className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <h3 style={{ fontSize: 16 }}>
                  <span className="badge badge-active" style={{ marginRight: 8 }}>{pf.account_type}</span>
                  {pf.account_number}
                </h3>
                {pf.employer_name && <div className="text-muted" style={{ fontSize: 13 }}>{pf.employer_name}</div>}
              </div>
              <div style={{ display: 'flex', gap: 4 }}>
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => openEdit(pf)}>Edit</button>}
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowImport(pf.id)}>Import PDF</button>}
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowContrib(pf.id)}>+ Contribution</button>}
                <button className="btn btn-sm btn-outline" onClick={() => setExpanded(expanded === pf.id ? null : pf.id)}>
                  {expanded === pf.id ? 'Hide' : 'Details'}
                </button>
                {!isViewOnly && <button className="btn btn-sm btn-danger" onClick={() => handleDelete(pf.id)}>Del</button>}
              </div>
            </div>

            <div className="card-grid" style={{ marginTop: 12 }}>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Current Balance</span><div style={{ fontWeight: 700, fontSize: 20 }}>{formatCurrency(pf.current_balance)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Interest Rate</span><div style={{ fontWeight: 600 }}>{pf.interest_rate}%</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Employee Contribution</span><div style={{ fontWeight: 600 }}>{formatCurrency(pf.total_employee_contribution)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Employer Contribution</span><div style={{ fontWeight: 600 }}>{formatCurrency(pf.total_employer_contribution)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>Interest Earned</span><div style={{ fontWeight: 600, color: 'var(--success)' }}>{formatCurrency(pf.total_interest_earned)}</div></div>
              <div><span className="text-muted" style={{ fontSize: 12 }}>XIRR</span><div style={{ fontWeight: 600, color: pf.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatPercent(pf.xirr)}</div></div>
            </div>

            {expanded === pf.id && pf.financial_year_entries && pf.financial_year_entries.length > 0 && (
              <div style={{ marginTop: 16 }}>
                {pf.financial_year_entries.map(fy => (
                  <div key={fy.financial_year} style={{ marginBottom: 16 }}>
                    <h4 style={{ marginBottom: 8 }}>FY {fy.financial_year}</h4>
                    <div style={{ fontSize: 13, marginBottom: 8 }}>
                      Opening: {formatCurrency(fy.opening_balance)} | Interest: {formatCurrency(fy.interest_earned)} | Closing: {formatCurrency(fy.closing_balance)}
                    </div>
                    {fy.monthly_contributions && fy.monthly_contributions.length > 0 && (
                      <div className="table-container">
                        <table>
                          <thead><tr><th>Month</th><th>Employee</th><th>Employer</th><th>Total</th></tr></thead>
                          <tbody>
                            {fy.monthly_contributions.map(mc => (
                              <tr key={mc.month}>
                                <td>{mc.month}</td>
                                <td>{formatCurrency(mc.employee_contribution)}</td>
                                <td>{formatCurrency(mc.employer_contribution)}</td>
                                <td>{formatCurrency(mc.total)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
        </>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add PF Account</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group">
                  <label>Account Type</label>
                  <select value={form.account_type} onChange={e => setForm({ ...form, account_type: e.target.value as PFAccountType })}>
                    <option value="EPF">EPF</option>
                    <option value="VPF">VPF</option>
                    <option value="PPF">PPF</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Account Number</label>
                  <input required value={form.account_number} onChange={e => setForm({ ...form, account_number: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Employer Name</label>
                  <input value={form.employer_name || ''} onChange={e => setForm({ ...form, employer_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Interest Rate (%)</label>
                  <input type="number" step="0.01" required value={form.interest_rate || ''} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Account</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showContrib && (
        <div className="modal-overlay" onClick={() => setShowContrib(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Monthly Contribution</h2>
            <form onSubmit={handleContrib}>
              <div className="form-row">
                <div className="form-group">
                  <label>Financial Year</label>
                  <input required value={contribForm.financial_year} onChange={e => setContribForm({ ...contribForm, financial_year: e.target.value })} placeholder="2025-2026" />
                </div>
                <div className="form-group">
                  <label>Month</label>
                  <input required value={contribForm.month} onChange={e => setContribForm({ ...contribForm, month: e.target.value })} placeholder="2025-04" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Employee Contribution (₹)</label>
                  <input type="number" step="0.01" required value={contribForm.employee_contribution || ''} onChange={e => setContribForm({ ...contribForm, employee_contribution: parseFloat(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>Employer Contribution (₹)</label>
                  <input type="number" step="0.01" value={contribForm.employer_contribution || ''} onChange={e => setContribForm({ ...contribForm, employer_contribution: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowContrib(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add Contribution</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit PF Account</h2>
            <form onSubmit={handleEdit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Account Type</label>
                  <select value={editForm.account_type} onChange={e => setEditForm({ ...editForm, account_type: e.target.value as PFAccountType })}>
                    <option value="EPF">EPF</option>
                    <option value="VPF">VPF</option>
                    <option value="PPF">PPF</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Account Number</label>
                  <input required value={editForm.account_number} onChange={e => setEditForm({ ...editForm, account_number: e.target.value })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Employer Name</label>
                  <input value={editForm.employer_name || ''} onChange={e => setEditForm({ ...editForm, employer_name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Interest Rate (%)</label>
                  <input type="number" step="0.01" required value={editForm.interest_rate || ''} onChange={e => setEditForm({ ...editForm, interest_rate: parseFloat(e.target.value) })} />
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={editForm.notes || ''} onChange={e => setEditForm({ ...editForm, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Changes</button>
              </div>
            </form>
          </div>
        </div>
      )}
      {/* Import PF PDF Modal */}
      {showImport && (
        <div className="modal-overlay" onClick={closeImport}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: importStep === 'preview' ? 700 : 500 }}>
            <h2>Import PF Passbook PDF</h2>

            {importStep === 'upload' && (
              <div>
                <p className="text-muted" style={{ marginBottom: 16 }}>Upload your EPF passbook PDF. OCR will be used to extract contribution data (this may take a minute).</p>
                <div className="form-group">
                  <label>PDF File</label>
                  <input type="file" accept=".pdf" onChange={e => setImportFile(e.target.files?.[0] || null)} />
                </div>
                {importProgress && <div className="text-muted" style={{ marginBottom: 12, fontSize: 13 }}>{importProgress}</div>}
                {importMessage && <div className={importMessage.includes('Error') || importMessage.includes('No contributions') ? 'text-danger' : 'text-success'} style={{ marginBottom: 12 }}>{importMessage}</div>}
                <div className="modal-actions">
                  <button type="button" className="btn btn-outline" onClick={closeImport}>Cancel</button>
                  <button className="btn btn-primary" onClick={handleParsePF} disabled={!importFile || importLoading}>
                    {importLoading ? 'Processing...' : 'Process PDF'}
                  </button>
                </div>
              </div>
            )}

            {importStep === 'preview' && parsedPF && (
              <div>
                {parsedPF.account_number && <p style={{ marginBottom: 4 }}><strong>Account:</strong> {parsedPF.account_number}</p>}
                {parsedPF.employer_name && <p style={{ marginBottom: 12 }}><strong>Employer:</strong> {parsedPF.employer_name}</p>}
                <p style={{ marginBottom: 12 }}>Found <strong>{parsedPF.years.length}</strong> financial years with <strong>{parsedPF.years.reduce((s, y) => s + y.contributions.length, 0)}</strong> monthly contributions:</p>
                <div style={{ maxHeight: 400, overflow: 'auto', marginBottom: 16 }}>
                  {parsedPF.years.map((yr, yi) => (
                    <div key={yi} style={{ marginBottom: 16 }}>
                      <h4 style={{ margin: '0 0 4px' }}>FY {yr.financial_year}</h4>
                      <div className="text-muted" style={{ fontSize: 12, marginBottom: 8 }}>
                        Opening: {formatCurrency(yr.opening_balance)} | Interest: {formatCurrency(yr.interest_earned)} | Closing: {formatCurrency(yr.closing_balance)}
                      </div>
                      {yr.contributions.length > 0 && (
                        <table style={{ fontSize: 12 }}>
                          <thead>
                            <tr><th>Month</th><th>Employee</th><th>Employer</th><th>Total</th></tr>
                          </thead>
                          <tbody>
                            {yr.contributions.map((c, ci) => (
                              <tr key={ci}>
                                <td>{c.month}</td>
                                <td>{formatCurrency(c.employee_contribution)}</td>
                                <td>{formatCurrency(c.employer_contribution)}</td>
                                <td>{formatCurrency(c.employee_contribution + c.employer_contribution)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  ))}
                </div>
                {importMessage && <div className="text-success" style={{ marginBottom: 12 }}>{importMessage}</div>}
                <div className="modal-actions">
                  <button className="btn btn-outline" onClick={() => { setImportStep('upload'); setParsedPF(null); }}>Back</button>
                  <button className="btn btn-primary" onClick={handleImportPF} disabled={importLoading}>
                    {importLoading ? 'Importing...' : `Import ${parsedPF.years.reduce((s, y) => s + y.contributions.length, 0)} Contributions`}
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
