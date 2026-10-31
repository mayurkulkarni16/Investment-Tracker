import React, { useCallback, useEffect, useState } from 'react';
import {
  getGoldHoldings, createGoldHolding, updateGoldHolding, deleteGoldHolding, setGoldRate, refreshGoldPrices,
  GOLD_FORM_LABELS, isGramForm,
} from '../api/gold';
import type { GoldHolding, GoldHoldingRequest, GoldForm } from '../api/gold';
import { formatCurrency, formatDate, formatPercent, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { apiErrorMessage } from '../utils/errors';
import { Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

const today = () => new Date().toISOString().split('T')[0];

const emptyForm = (): GoldHoldingRequest => ({
  name: '', form: 'physical', quantity: 0, purity: 22, purchase_price: 0,
  purchase_date: today(), current_price: 0, symbol: '', interest_rate: 2.5, maturity_date: '', notes: '',
});

const formBadgeStyle: Record<GoldForm, React.CSSProperties> = {
  physical: { background: 'var(--warning-soft)', color: 'var(--warning-text)' },
  digital: { background: 'var(--primary-soft)', color: '#4f46e5' },
  sgb: { background: 'var(--success-soft)', color: 'var(--success-text)' },
  etf: { background: 'var(--purple-soft)', color: 'var(--purple-text)' },
  mutual_fund: { background: 'var(--danger-soft)', color: 'var(--danger-text)' },
};

const fmtQty = (h: GoldHolding) =>
  isGramForm(h.form) ? `${h.quantity.toLocaleString('en-IN', { maximumFractionDigits: 3 })} g` : `${h.quantity.toLocaleString('en-IN', { maximumFractionDigits: 4 })} units`;

export default function GoldPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [holdings, setHoldings] = useState<GoldHolding[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<GoldHolding | null>(null);
  const [form, setForm] = useState<GoldHoldingRequest>(emptyForm);
  const [rate, setRate] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    getGoldHoldings()
      .then(r => setHoldings(r.data || []))
      .catch(err => toast(apiErrorMessage(err, 'Failed to load gold holdings'), 'error'))
      .finally(() => setLoading(false));
  }, [toast]);

  useEffect(() => { load(); }, [load, viewAsUserId]);

  const openAdd = () => { setEditing(null); setForm(emptyForm()); setShowModal(true); };

  const openEdit = (h: GoldHolding) => {
    setEditing(h);
    setForm({
      name: h.name, form: h.form, quantity: h.quantity, purity: h.purity || 24,
      purchase_price: h.purchase_price, purchase_date: toInputDate(h.purchase_date),
      current_price: h.current_price, symbol: h.symbol || '', interest_rate: h.interest_rate,
      maturity_date: toInputDate(h.maturity_date), notes: h.notes || '',
    });
    setShowModal(true);
  };

  const changeForm = (f: GoldForm) => {
    setForm(prev => ({
      ...prev,
      form: f,
      purity: f === 'physical' ? (prev.purity || 22) : 24,
      interest_rate: f === 'sgb' ? (prev.interest_rate || 2.5) : prev.interest_rate,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload: GoldHoldingRequest = {
      ...form,
      purity: form.form === 'physical' ? form.purity : 24,
      symbol: form.form === 'etf' ? form.symbol.trim().toUpperCase() : '',
      interest_rate: form.form === 'sgb' ? form.interest_rate : 0,
      maturity_date: form.form === 'sgb' ? form.maturity_date : '',
    };
    try {
      if (editing) { await updateGoldHolding(editing.id, payload); toast('Gold holding updated'); }
      else { await createGoldHolding(payload); toast('Gold holding added'); }
      setShowModal(false);
      load();
    } catch (err) {
      toast(apiErrorMessage(err, editing ? 'Failed to update holding' : 'Failed to add holding'), 'error');
    }
  };

  const handleDelete = async (h: GoldHolding) => {
    if (!await confirm({ title: 'Delete holding', message: `Delete "${h.name}"?`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteGoldHolding(h.id); toast('Gold holding deleted'); load(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete holding'), 'error'); }
  };

  const handleSetRate = async (e: React.FormEvent) => {
    e.preventDefault();
    const r = parseFloat(rate);
    if (!(r > 0)) { toast('Enter a valid 24K rate per gram', 'error'); return; }
    setBusy(true);
    try {
      const res = await setGoldRate(r);
      setHoldings(res.data || []);
      toast('Gold rate updated');
      setRate('');
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to set gold rate'), 'error');
    } finally { setBusy(false); }
  };

  const handleRefresh = async () => {
    setBusy(true);
    try {
      const res = await refreshGoldPrices();
      setHoldings(res.data || []);
      toast('ETF prices refreshed');
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to refresh prices'), 'error');
    } finally { setBusy(false); }
  };

  if (loading) return <PageSkeleton label="Loading" />;

  const invested = holdings.reduce((s, h) => s + h.purchase_price, 0);
  const currentValue = holdings.reduce((s, h) => s + h.current_value, 0);
  const gain = currentValue - invested;
  const gainPct = invested > 0 ? (gain / invested) * 100 : 0;
  const totalGrams = holdings.filter(h => isGramForm(h.form)).reduce((s, h) => s + h.quantity, 0);
  const interestTotal = holdings.reduce((s, h) => s + (h.interest_earned || 0), 0);
  const hasEtf = holdings.some(h => h.form === 'etf' && h.symbol);
  const gramForm = isGramForm(form.form);

  return (
    <div>
      <div className="page-header">
        <h1>Gold</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Holding</button>}
      </div>

      {holdings.length > 0 && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card">
            <div className="label">Invested</div>
            <div className="value">{formatCurrency(invested)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Current Value</div>
            <div className="value">{formatCurrency(currentValue)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Gain / Loss</div>
            <div className={`value ${gain >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(gain)}</div>
            <div className={`${gain >= 0 ? 'text-success' : 'text-danger'} text-md`}>{formatPercent(gainPct)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Total Gold (Physical + Digital + SGB)</div>
            <div className="value">{totalGrams.toLocaleString('en-IN', { maximumFractionDigits: 3 })} g</div>
            {interestTotal > 0 && <div className="text-muted text-md">SGB interest: {formatCurrency(interestTotal)}</div>}
          </div>
        </div>
      )}

      {!isViewOnly && (
        <div className="card flex gap-3 items-end flex-wrap">
          <form className="flex gap-2 items-end flex-wrap" onSubmit={handleSetRate}>
            <div className="form-group mb-0">
              <label>Set today's 24K rate (₹/g)</label>
              <input type="number" step="0.01" min="0" placeholder="e.g. 7250" value={rate} onChange={e => setRate(e.target.value)} />
            </div>
            <button type="submit" className="btn btn-primary" disabled={busy || !rate}>Update Rate</button>
          </form>
          <button type="button" className="btn btn-outline" onClick={handleRefresh} disabled={busy || !hasEtf} title={hasEtf ? 'Fetch latest NSE prices for ETFs' : 'Add an ETF with a symbol to enable'}>
            Refresh ETF prices
          </button>
          <span className="text-muted text-sm">Physical, digital and SGB holdings are revalued pro rata by purity.</span>
        </div>
      )}

      {holdings.length === 0 ? (
        <div className="empty-state">
          <h3>No gold holdings yet</h3>
          <p>Track physical gold, digital gold, Sovereign Gold Bonds, ETFs and gold funds.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Holding</button>}
        </div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Form</th>
                <th>Quantity</th>
                <th>Purity</th>
                <th className="num">Cost</th>
                <th>Current Price</th>
                <th className="num">Value</th>
                <th className="num">Gain</th>
                <th className="num">CAGR</th>
                <th className="num">SGB Interest</th>
                <th>Maturity</th>
                {!isViewOnly && <th>Actions</th>}
              </tr>
            </thead>
            <tbody>
              {holdings.map(h => (
                <tr key={h.id}>
                  <td>
                    <div>{h.name}</div>
                    {h.symbol && <div className="text-muted text-sm">{h.symbol}</div>}
                  </td>
                  <td><span className="badge" style={formBadgeStyle[h.form]}>{GOLD_FORM_LABELS[h.form] || h.form}</span></td>
                  <td>{fmtQty(h)}</td>
                  <td>{h.form === 'physical' ? `${h.purity}K` : '-'}</td>
                  <td className="num">{formatCurrency(h.purchase_price)}</td>
                  <td>
                    <div>{formatCurrency(h.current_price)}{isGramForm(h.form) ? '/g' : '/unit'}</div>
                    {h.price_updated_at && <div className="text-muted text-sm">{formatDate(h.price_updated_at)}</div>}
                  </td>
                  <td className="num">{formatCurrency(h.current_value)}</td>
                  <td className={`${h.gain_loss >= 0 ? 'text-success' : 'text-danger'} num`}>
                    <div>{formatCurrency(h.gain_loss)}</div>
                    <div className="text-sm">{formatPercent(h.gain_loss_pct)}</div>
                  </td>
                  <td className={`${h.cagr >= 0 ? 'text-success' : 'text-danger'} num`}>{formatPercent(h.cagr)}</td>
                  <td className="num">{h.form === 'sgb' ? formatCurrency(h.interest_earned) : '-'}</td>
                  <td>{h.form === 'sgb' && h.maturity_date ? formatDate(h.maturity_date) : '-'}</td>
                  {!isViewOnly && (
                    <td className="cell-actions">
                      <RowActions
                        inline={1}
                        label={`Actions for ${h.name}`}
                        actions={[
                          { label: 'Edit', icon: Pencil, onClick: () => openEdit(h) },
                          { label: 'Delete', icon: Trash2, onClick: () => handleDelete(h), danger: true },
                        ]}
                      />
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing ? 'Edit Gold Holding' : 'Add Gold Holding'}</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Name</label>
                  <input required value={form.name} placeholder="e.g. Wedding necklace, SGB 2023-24 Series III" onChange={e => setForm({ ...form, name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Form</label>
                  <select value={form.form} onChange={e => changeForm(e.target.value as GoldForm)}>
                    {(Object.keys(GOLD_FORM_LABELS) as GoldForm[]).map(f => (
                      <option key={f} value={f}>{GOLD_FORM_LABELS[f]}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>{gramForm ? 'Quantity (grams)' : 'Quantity (units)'}</label>
                  <input type="number" step="any" min="0" required value={form.quantity || ''} onChange={e => setForm({ ...form, quantity: parseFloat(e.target.value) || 0 })} />
                </div>
                {form.form === 'physical' && (
                  <div className="form-group">
                    <label>Purity</label>
                    <select value={form.purity} onChange={e => setForm({ ...form, purity: parseFloat(e.target.value) })}>
                      <option value={24}>24K</option>
                      <option value={22}>22K</option>
                      <option value={18}>18K</option>
                    </select>
                  </div>
                )}
                {form.form === 'etf' && (
                  <div className="form-group">
                    <label>NSE Symbol</label>
                    <input value={form.symbol} placeholder="e.g. GOLDBEES" onChange={e => setForm({ ...form, symbol: e.target.value })} />
                  </div>
                )}
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Total Cost (₹)</label>
                  <input type="number" step="0.01" min="0" required value={form.purchase_price || ''} onChange={e => setForm({ ...form, purchase_price: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Purchase Date</label>
                  <input type="date" required value={form.purchase_date} onChange={e => setForm({ ...form, purchase_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>{gramForm ? 'Current Price (₹/g)' : 'Current Price (₹/unit)'}</label>
                  <input type="number" step="0.01" min="0" value={form.current_price || ''} onChange={e => setForm({ ...form, current_price: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              {form.form === 'sgb' && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Interest Rate (% p.a.)</label>
                    <input type="number" step="0.01" min="0" value={form.interest_rate} onChange={e => setForm({ ...form, interest_rate: parseFloat(e.target.value) || 0 })} />
                  </div>
                  <div className="form-group">
                    <label>Maturity Date</label>
                    <input type="date" value={form.maturity_date} onChange={e => setForm({ ...form, maturity_date: e.target.value })} />
                  </div>
                </div>
              )}
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">{editing ? 'Save Changes' : 'Add Holding'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
