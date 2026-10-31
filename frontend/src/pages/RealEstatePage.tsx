import React, { useCallback, useEffect, useState } from 'react';
import { getProperties, createProperty, updateProperty, deleteProperty, PROPERTY_TYPE_LABELS } from '../api/properties';
import type { Property, PropertyRequest, PropertyType } from '../api/properties';
import { getHomeLoans } from '../api/homeLoans';
import type { HomeLoan } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { apiErrorMessage } from '../utils/errors';

const today = () => new Date().toISOString().split('T')[0];

const emptyForm = (): PropertyRequest => ({
  name: '', property_type: 'residential', location: '', purchase_price: 0, purchase_date: '',
  current_value: 0, value_updated_at: today(), ownership_pct: 100, home_loan_id: '',
  is_self_occupied: true, monthly_rent: 0, annual_costs: 0, notes: '',
});

const loanLabel = (l: HomeLoan) =>
  `${l.bank_name}${l.loan_account_number ? ` (${l.loan_account_number})` : ''}${l.property_address ? ` – ${l.property_address}` : ''}`;

export default function RealEstatePage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [properties, setProperties] = useState<Property[]>([]);
  const [loans, setLoans] = useState<HomeLoan[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Property | null>(null);
  const [form, setForm] = useState<PropertyRequest>(emptyForm);

  const load = useCallback(() => {
    getProperties()
      .then(r => setProperties(r.data || []))
      .catch(err => toast(apiErrorMessage(err, 'Failed to load properties'), 'error'))
      .finally(() => setLoading(false));
    getHomeLoans().then(r => setLoans(r.data || [])).catch(() => {});
  }, [toast]);

  useEffect(() => { load(); }, [load, viewAsUserId]);

  const openAdd = () => { setEditing(null); setForm(emptyForm()); setShowModal(true); };

  const openEdit = (p: Property) => {
    setEditing(p);
    setForm({
      name: p.name, property_type: p.property_type, location: p.location || '',
      purchase_price: p.purchase_price, purchase_date: toInputDate(p.purchase_date),
      current_value: p.current_value, value_updated_at: toInputDate(p.value_updated_at) || today(),
      ownership_pct: p.ownership_pct || 100, home_loan_id: p.home_loan_id || '',
      is_self_occupied: p.is_self_occupied, monthly_rent: p.monthly_rent, annual_costs: p.annual_costs,
      notes: p.notes || '',
    });
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload: PropertyRequest = {
      ...form,
      ownership_pct: form.ownership_pct > 0 ? form.ownership_pct : 100,
      monthly_rent: form.is_self_occupied ? 0 : form.monthly_rent,
    };
    try {
      if (editing) { await updateProperty(editing.id, payload); toast('Property updated'); }
      else { await createProperty(payload); toast('Property added'); }
      setShowModal(false);
      load();
    } catch (err) {
      toast(apiErrorMessage(err, editing ? 'Failed to update property' : 'Failed to add property'), 'error');
    }
  };

  const handleDelete = async (p: Property) => {
    if (!await confirm({ title: 'Delete property', message: `Delete "${p.name}"? The linked home loan is not affected.`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteProperty(p.id); toast('Property deleted'); load(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete property'), 'error'); }
  };

  if (loading) return <div className="loading">Loading...</div>;

  const ownedValue = properties.reduce((s, p) => s + p.owned_value, 0);
  const totalGain = properties.reduce((s, p) => s + p.gain_loss, 0);
  const totalEquity = properties.reduce((s, p) => s + p.equity, 0);
  const totalLoan = properties.reduce((s, p) => s + p.loan_outstanding, 0);
  const annualRent = properties
    .filter(p => !p.is_self_occupied)
    .reduce((s, p) => s + p.monthly_rent * 12 * ((p.ownership_pct || 100) / 100), 0);
  const loanById = new Map(loans.map(l => [l.id, l]));

  return (
    <div>
      <div className="page-header">
        <h1>Real Estate</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}>+ Add Property</button>}
      </div>

      {properties.length > 0 && (
        <div className="card-grid" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', marginBottom: 20 }}>
          <div className="stat-card">
            <div className="label">Owned Value</div>
            <div className="value">{formatCurrency(ownedValue)}</div>
            <div className="text-muted" style={{ fontSize: 13 }}>{properties.length} {properties.length === 1 ? 'property' : 'properties'}</div>
          </div>
          <div className="stat-card">
            <div className="label">Total Gain</div>
            <div className={`value ${totalGain >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(totalGain)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Equity After Loans</div>
            <div className="value">{formatCurrency(totalEquity)}</div>
            {totalLoan > 0 && <div className="text-muted" style={{ fontSize: 13 }}>Loans: {formatCurrency(totalLoan)}</div>}
          </div>
          <div className="stat-card">
            <div className="label">Annual Rent</div>
            <div className="value positive">{formatCurrency(annualRent)}</div>
          </div>
        </div>
      )}

      {properties.length === 0 ? (
        <div className="empty-state">
          <h3>No properties yet</h3>
          <p>Track homes, land and commercial property, and link them to your home loans to see equity.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}>+ Add Property</button>}
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Property</th>
                <th>Type</th>
                <th>Purchase</th>
                <th>Current Value</th>
                <th>Ownership</th>
                <th>Owned Value</th>
                <th>Gain</th>
                <th>CAGR</th>
                <th>Net Rental Yield</th>
                <th>Loan Outstanding</th>
                <th>Equity</th>
                {!isViewOnly && <th>Actions</th>}
              </tr>
            </thead>
            <tbody>
              {properties.map(p => {
                const loan = p.home_loan_id ? loanById.get(p.home_loan_id) : undefined;
                return (
                  <tr key={p.id}>
                    <td>
                      <div>{p.name}</div>
                      {p.location && <div className="text-muted" style={{ fontSize: 12 }}>{p.location}</div>}
                    </td>
                    <td>
                      <span className="badge badge-completed">{PROPERTY_TYPE_LABELS[p.property_type] || p.property_type}</span>
                      <div className="text-muted" style={{ fontSize: 12 }}>{p.is_self_occupied ? 'Self-occupied' : 'Let out'}</div>
                    </td>
                    <td>
                      <div>{formatCurrency(p.purchase_price)}</div>
                      <div className="text-muted" style={{ fontSize: 12 }}>{formatDate(p.purchase_date)}</div>
                    </td>
                    <td>
                      <div>{formatCurrency(p.current_value)}</div>
                      <div className="text-muted" style={{ fontSize: 12 }}>Value updated {formatDate(p.value_updated_at)}</div>
                    </td>
                    <td>{p.ownership_pct}%</td>
                    <td>{formatCurrency(p.owned_value)}</td>
                    <td className={p.gain_loss >= 0 ? 'text-success' : 'text-danger'}>{formatCurrency(p.gain_loss)}</td>
                    <td className={p.cagr >= 0 ? 'text-success' : 'text-danger'}>{formatPercent(p.cagr)}</td>
                    <td>
                      {p.is_self_occupied ? '-' : (
                        <>
                          <div>{p.net_rental_yield.toFixed(2)}%</div>
                          <div className="text-muted" style={{ fontSize: 12 }}>{formatCurrency(p.monthly_rent)}/mo</div>
                        </>
                      )}
                    </td>
                    <td>
                      {p.home_loan_id ? (
                        <>
                          <div>{formatCurrency(p.loan_outstanding)}</div>
                          {loan && <div className="text-muted" style={{ fontSize: 12 }}>{loan.bank_name}</div>}
                        </>
                      ) : '-'}
                    </td>
                    <td><strong>{formatCurrency(p.equity)}</strong></td>
                    {!isViewOnly && (
                      <td>
                        <div style={{ display: 'flex', gap: 4 }}>
                          <button className="btn btn-sm btn-outline" onClick={() => openEdit(p)}>Edit</button>
                          <button className="btn btn-sm btn-danger" onClick={() => handleDelete(p)}>Del</button>
                        </div>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing ? 'Edit Property' : 'Add Property'}</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Name</label>
                  <input required value={form.name} placeholder="e.g. 2BHK Whitefield" onChange={e => setForm({ ...form, name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Type</label>
                  <select value={form.property_type} onChange={e => setForm({ ...form, property_type: e.target.value as PropertyType })}>
                    {(Object.keys(PROPERTY_TYPE_LABELS) as PropertyType[]).map(t => (
                      <option key={t} value={t}>{PROPERTY_TYPE_LABELS[t]}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>Location</label>
                <input value={form.location} onChange={e => setForm({ ...form, location: e.target.value })} />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Purchase Price (₹, incl. stamp duty)</label>
                  <input type="number" step="0.01" min="0" required value={form.purchase_price || ''} onChange={e => setForm({ ...form, purchase_price: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Purchase Date</label>
                  <input type="date" required value={form.purchase_date} onChange={e => setForm({ ...form, purchase_date: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Current Value (₹, full)</label>
                  <input type="number" step="0.01" min="0" value={form.current_value || ''} onChange={e => setForm({ ...form, current_value: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Value As Of</label>
                  <input type="date" value={form.value_updated_at} onChange={e => setForm({ ...form, value_updated_at: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Ownership (%)</label>
                  <input type="number" step="0.01" min="0" max="100" value={form.ownership_pct || ''} onChange={e => setForm({ ...form, ownership_pct: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-group">
                <label>Linked Home Loan</label>
                <select value={form.home_loan_id} onChange={e => setForm({ ...form, home_loan_id: e.target.value })}>
                  <option value="">None</option>
                  {loans.map(l => <option key={l.id} value={l.id}>{loanLabel(l)}</option>)}
                  {form.home_loan_id && !loanById.has(form.home_loan_id) && (
                    <option value={form.home_loan_id}>Unknown loan ({form.home_loan_id})</option>
                  )}
                </select>
              </div>
              <div className="form-group">
                <label>
                  <input type="checkbox" checked={form.is_self_occupied} onChange={e => setForm({ ...form, is_self_occupied: e.target.checked })} style={{ width: 'auto', marginRight: 6 }} />
                  Self-occupied
                </label>
              </div>
              <div className="form-row">
                {!form.is_self_occupied && (
                  <div className="form-group">
                    <label>Monthly Rent (₹)</label>
                    <input type="number" step="0.01" min="0" value={form.monthly_rent || ''} onChange={e => setForm({ ...form, monthly_rent: parseFloat(e.target.value) || 0 })} />
                  </div>
                )}
                <div className="form-group">
                  <label>Annual Costs (₹, maintenance + tax)</label>
                  <input type="number" step="0.01" min="0" value={form.annual_costs || ''} onChange={e => setForm({ ...form, annual_costs: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">{editing ? 'Save Changes' : 'Add Property'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
