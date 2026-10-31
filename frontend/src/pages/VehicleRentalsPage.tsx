import { useEffect, useState } from 'react';
import {
  getVehicleRentals, createVehicleRental, updateVehicleRental, deleteVehicleRental,
  updateRentalPayout, markDueRentalPayoutsReceived, updateRentalBuyback,
} from '../api/vehicleRentals';
import type { VehicleRental, RentalPayout, CreateVehicleRentalRequest } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate, statusLabel } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import InfoTooltip from '../components/InfoTooltip';
import { ChevronDown, Pencil, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

const emptyForm: CreateVehicleRentalRequest = {
  provider: '', vehicle_model: '', units: 1, unit_price: 0, gst_percent: 5, other_charges: 0,
  purchase_date: '', payout_start_date: '', tenure_months: 48, monthly_payout_per_unit: 0,
  buyback_percent: 10, registration_numbers: [], agreement_number: '', notes: '',
};

// Local (not UTC) date as YYYY-MM-DD, so "today" is correct in IST.
const todayInput = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};

// Annualised IRR of: pay `cost` now, receive `monthly` for `months` months, then `buyback`.
function impliedAnnualReturn(cost: number, monthly: number, months: number, buyback: number): number | null {
  if (cost <= 0 || monthly <= 0 || months <= 0) return null;
  const pv = (r: number) => {
    let s = 0;
    for (let i = 1; i <= months; i++) s += monthly / Math.pow(1 + r, i);
    return s + buyback / Math.pow(1 + r, months);
  };
  let lo = -0.5, hi = 1;
  for (let k = 0; k < 200; k++) {
    const mid = (lo + hi) / 2;
    if (pv(mid) > cost) lo = mid; else hi = mid;
  }
  return (Math.pow(1 + lo, 12) - 1) * 100;
}

type PayoutState = 'received' | 'overdue' | 'due' | 'upcoming';

const payoutState = (p: RentalPayout, today: string): PayoutState => {
  if (p.status === 'received') return 'received';
  if (p.is_overdue) return 'overdue';
  return toInputDate(p.scheduled_date) <= today ? 'due' : 'upcoming';
};

const stateBadge: Record<PayoutState, string> = {
  received: 'badge-received', overdue: 'badge-overdue', due: 'badge-pending', upcoming: 'badge-pending',
};

interface ReceiptDraft {
  rental: VehicleRental;
  payout: RentalPayout | null; // null = buyback
  date: string;
  amount: number;
}

export default function VehicleRentalsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [items, setItems] = useState<VehicleRental[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<VehicleRental | 'new' | null>(null);
  const [form, setForm] = useState<CreateVehicleRentalRequest>(emptyForm);
  const [regText, setRegText] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<ReceiptDraft | null>(null);
  const today = todayInput();

  const load = () => {
    getVehicleRentals().then(r => setItems(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [viewAsUserId]);

  const replaceItem = (updated: VehicleRental) => setItems(prev => prev.map(i => (i.id === updated.id ? updated : i)));

  const openAdd = () => { setForm(emptyForm); setRegText(''); setEditing('new'); };

  const openEdit = (v: VehicleRental) => {
    setForm({
      provider: v.provider, vehicle_model: v.vehicle_model, units: v.units, unit_price: v.unit_price,
      gst_percent: v.gst_percent, other_charges: v.other_charges,
      purchase_date: toInputDate(v.purchase_date), payout_start_date: toInputDate(v.payout_start_date),
      tenure_months: v.tenure_months, monthly_payout_per_unit: v.monthly_payout_per_unit,
      buyback_percent: v.buyback_percent, registration_numbers: v.registration_numbers || [],
      agreement_number: v.agreement_number, notes: v.notes,
    });
    setRegText((v.registration_numbers || []).join(', '));
    setEditing(v);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = { ...form, registration_numbers: regText.split(/[,\n]/).map(s => s.trim()).filter(Boolean) };
    try {
      if (editing === 'new') {
        await createVehicleRental(data);
        toast('Vehicle rental added');
      } else if (editing) {
        await updateVehicleRental(editing.id, data);
        toast('Vehicle rental updated');
      }
      setEditing(null);
      load();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      toast(msg ? `Failed to save: ${msg}` : 'Failed to save', 'error');
    }
  };

  const handleDelete = async (v: VehicleRental) => {
    if (!await confirm({ message: `Delete ${v.provider} (${v.units} vehicle${v.units > 1 ? 's' : ''}) and its payout history?`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteVehicleRental(v.id); toast('Deleted'); load(); }
    catch { toast('Failed to delete', 'error'); }
  };

  const openReceipt = (rental: VehicleRental, payout: RentalPayout | null) => {
    const scheduled = payout ? toInputDate(payout.scheduled_date) : toInputDate(rental.agreement_end_date);
    setReceipt({
      rental, payout,
      date: scheduled <= today ? scheduled : today,
      amount: payout ? payout.amount : rental.buyback_amount,
    });
  };

  const saveReceipt = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!receipt) return;
    const body = { status: 'received' as const, received_date: receipt.date, amount: receipt.amount };
    try {
      const res = receipt.payout
        ? await updateRentalPayout(receipt.rental.id, receipt.payout.payout_id, body)
        : await updateRentalBuyback(receipt.rental.id, body);
      replaceItem(res.data);
      setReceipt(null);
      toast(receipt.payout ? 'Payout marked received' : 'Buyback marked received');
    } catch { toast('Failed to save receipt', 'error'); }
  };

  const undoReceipt = async (rental: VehicleRental, payout: RentalPayout | null) => {
    try {
      const res = payout
        ? await updateRentalPayout(rental.id, payout.payout_id, { status: 'pending' })
        : await updateRentalBuyback(rental.id, { status: 'pending' });
      replaceItem(res.data);
      toast('Marked as not received');
    } catch { toast('Failed to update', 'error'); }
  };

  const markAllDue = async (rental: VehicleRental) => {
    const due = rental.payouts.filter(p => p.status === 'pending' && toInputDate(p.scheduled_date) <= today);
    if (due.length === 0) { toast('No due payouts to mark'); return; }
    const total = due.reduce((s, p) => s + p.amount, 0);
    if (!await confirm({ message: `Confirm you received all ${due.length} due payout(s) totalling ${formatCurrency(total)} in full, on their scheduled dates?`, confirmLabel: 'Mark received' })) return;
    try {
      const res = await markDueRentalPayoutsReceived(rental.id);
      replaceItem(res.data);
      toast(`${due.length} payout(s) marked received`);
    } catch { toast('Failed to update', 'error'); }
  };

  if (loading) return <PageSkeleton label="Loading" />;

  const active = items.filter(i => i.status === 'active');
  const totalInvested = items.reduce((s, i) => s + i.investment_amount, 0);
  const totalReceived = items.reduce((s, i) => s + i.total_payouts_received + i.buyback_received, 0);
  const bookValue = items.reduce((s, i) => s + i.current_value, 0);
  const netReturn = items.reduce((s, i) => s + i.total_return, 0);
  const overdueCount = items.reduce((s, i) => s + i.overdue_count, 0);
  const overdueAmount = items.reduce((s, i) => s + i.overdue_amount, 0);
  const monthlyIncome = active.reduce((s, i) => s + i.monthly_payout, 0);
  const avgXirr = totalInvested > 0 ? items.reduce((s, i) => s + i.xirr * i.investment_amount, 0) / totalInvested : 0;

  // Live preview for the add/edit form
  const vehicleCost = form.units * form.unit_price;
  const previewInvestment = vehicleCost * (1 + form.gst_percent / 100) + form.other_charges;
  const previewMonthly = form.units * form.monthly_payout_per_unit;
  const previewPayouts = previewMonthly * form.tenure_months;
  const previewBuyback = vehicleCost * form.buyback_percent / 100;
  const previewIrr = impliedAnnualReturn(previewInvestment, previewMonthly, form.tenure_months, previewBuyback);

  return (
    <div>
      <div className="page-header">
        <h1>Vehicle Rentals</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Investment</button>}
      </div>

      {overdueCount > 0 && (
        <div className="card mb-4" style={{ borderLeft: '4px solid var(--danger)' }}>
          <strong className="text-danger">
            {overdueCount} rental payout{overdueCount > 1 ? 's' : ''} overdue ({formatCurrency(overdueAmount)})
          </strong>
          <div className="text-muted text-md mt-1">
            Payouts more than 5 days past their scheduled date. Follow up with the provider, and keep the written agreement and payment records.
          </div>
        </div>
      )}

      {items.length > 0 && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card">
            <div className="label">Vehicles (active)</div>
            <div className="value">{active.reduce((s, i) => s + i.units, 0)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Total Invested</div>
            <div className="value">{formatCurrency(totalInvested)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Expected Monthly Income</div>
            <div className="value">{formatCurrency(monthlyIncome)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Received So Far</div>
            <div className="value positive">{formatCurrency(totalReceived)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Capital Recovered<InfoTooltip tip="Payouts and buyback received, as a share of the amount invested." /></div>
            <div className="value">{totalInvested > 0 ? (totalReceived / totalInvested * 100).toFixed(1) : '0.0'}%</div>
          </div>
          <div className="stat-card">
            <div className="label">Book Value<InfoTooltip tip="Straight-line from purchase cost down to the buyback value over the agreement period." /></div>
            <div className="value">{formatCurrency(bookValue)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Net Return<InfoTooltip tip="Received + book value − invested." /></div>
            <div className={`value ${netReturn >= 0 ? 'positive' : 'negative'}`}>{formatCurrency(netReturn)}</div>
          </div>
          <div className="stat-card">
            <div className="label">Overdue</div>
            <div className={`value ${overdueCount > 0 ? 'negative' : ''}`}>{overdueCount > 0 ? `${overdueCount} · ${formatCurrency(overdueAmount)}` : 'None'}</div>
          </div>
          <div className="stat-card">
            <div className="label">Avg XIRR<InfoTooltip tip="Based on payouts actually received plus current book value." /></div>
            <div className={`value ${avgXirr >= 0 ? 'positive' : 'negative'}`}>{formatPercent(avgXirr)}</div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <div className="empty-state">
          <h3>No vehicle rental investments yet</h3>
          <p>Track EV fleet leasing schemes: monthly rental payouts, overdue payments, buyback and actual returns.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Investment</button>}
        </div>
      ) : items.map(v => {
        const dueCount = v.payouts.filter(p => p.status === 'pending' && toInputDate(p.scheduled_date) <= today).length;
        return (
          <div key={v.id} className="card" style={v.overdue_count > 0 ? { borderLeft: '4px solid var(--danger)' } : undefined}>
            <div className="flex justify-between items-start">
              <div>
                <h3 className="text-lg">
                  {v.provider}{v.vehicle_model ? ` · ${v.vehicle_model}` : ''}
                  <span className={`badge badge-${v.status} ml-2`}>{statusLabel(v.status)}</span>
                  {v.overdue_count > 0 && <span className="badge badge-overdue ml-2">{v.overdue_count} overdue</span>}
                </h3>
                <div className="text-muted text-md">
                  {v.units} vehicle{v.units > 1 ? 's' : ''} · {formatDate(v.purchase_date)} → {formatDate(v.agreement_end_date)}
                  {v.agreement_number ? ` · Agreement ${v.agreement_number}` : ''}
                </div>
              </div>
              <div className="flex gap-1 items-center">
                <button className="btn btn-sm btn-ghost" aria-expanded={expanded === v.id} onClick={() => setExpanded(expanded === v.id ? null : v.id)}>
                  <ChevronDown size={14} className={`row-chevron${expanded === v.id ? ' flip' : ''}`} aria-hidden="true" />{expanded === v.id ? 'Hide payouts' : 'Payouts'}
                </button>
                {!isViewOnly && (
                  <RowActions
                    label={`More actions for ${v.provider}`}
                    actions={[
                      { label: 'Edit rental', icon: Pencil, onClick: () => openEdit(v) },
                      { label: 'Delete rental', icon: Trash2, onClick: () => handleDelete(v), danger: true },
                    ]}
                  />
                )}
              </div>
            </div>

            <div className="card-grid mt-3">
              <div><span className="text-muted text-sm">Invested</span><div className="fw-semibold">{formatCurrency(v.investment_amount)}</div></div>
              <div><span className="text-muted text-sm">Monthly Payout</span><div className="fw-semibold">{formatCurrency(v.monthly_payout)} <span className="text-muted text-sm">({formatCurrency(v.monthly_payout_per_unit)}/unit)</span></div></div>
              <div><span className="text-muted text-sm">Payouts Received</span><div className="fw-semibold">{v.payouts_received_count} / {v.payouts.length} · {formatCurrency(v.total_payouts_received)}</div></div>
              <div><span className="text-muted text-sm">Next Payout</span><div className="fw-semibold">{v.next_payout_date ? formatDate(v.next_payout_date) : '-'}</div></div>
              <div><span className="text-muted text-sm">Capital Recovered</span><div className="fw-semibold">{(v.investment_amount > 0 ? (v.total_payouts_received + v.buyback_received) / v.investment_amount * 100 : 0).toFixed(1)}%</div></div>
              <div><span className="text-muted text-sm">Book Value</span><div className="fw-semibold">{formatCurrency(v.current_value)}</div></div>
              <div><span className="text-muted text-sm">Net Return</span><div className="fw-semibold" style={{ color: v.total_return >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatCurrency(v.total_return)}</div></div>
              <div><span className="text-muted text-sm">XIRR</span><div className="fw-semibold" style={{ color: v.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatPercent(v.xirr)}</div></div>
            </div>
            {v.registration_numbers?.length > 0 && (
              <div className="text-muted text-sm mt-2">Registration: {v.registration_numbers.join(', ')}</div>
            )}

            {expanded === v.id && (
              <div className="mt-4">
                <div className="flex justify-between items-center mb-2">
                  <h4>Monthly Payouts</h4>
                  {!isViewOnly && dueCount > 0 && (
                    <button className="btn btn-sm btn-success" onClick={() => markAllDue(v)}>Mark all due as received ({dueCount})</button>
                  )}
                </div>
                <div className="table-container mb-4" style={{ maxHeight: 420, overflowY: 'auto' }}>
                  <table>
                    <thead><tr><th>#</th><th>Scheduled</th><th className="num">Amount</th><th>Status</th><th>Received</th><th>Action</th></tr></thead>
                    <tbody>
                      {v.payouts.map((p, idx) => {
                        const st = payoutState(p, today);
                        const short = p.status === 'received' && p.received_amount < p.amount;
                        return (
                          <tr key={p.payout_id}>
                            <td>{idx + 1}</td>
                            <td>{formatDate(p.scheduled_date)}</td>
                            <td className="num">{formatCurrency(p.amount)}</td>
                            <td><span className={`badge ${stateBadge[st]}`}>{st}</span></td>
                            <td>
                              {p.status === 'received' ? (
                                <>
                                  {formatCurrency(p.received_amount)} on {formatDate(p.received_date || p.scheduled_date)}
                                  {short && <span className="badge badge-overdue ml-2">short {formatCurrency(p.amount - p.received_amount)}</span>}
                                </>
                              ) : '-'}
                            </td>
                            <td>
                              {!isViewOnly && (p.status === 'received'
                                ? <button className="btn btn-sm btn-outline" onClick={() => undoReceipt(v, p)}>Undo</button>
                                : st !== 'upcoming' && <button className="btn btn-sm btn-success" onClick={() => openReceipt(v, p)}>Mark Received</button>)}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>

                {v.buyback_amount > 0 && (
                  <>
                    <h4 className="mb-2">Buyback ({v.buyback_percent}% of vehicle cost)</h4>
                    <div className="table-container">
                      <table>
                        <thead><tr><th>Due</th><th className="num">Amount</th><th>Status</th><th>Received</th><th>Action</th></tr></thead>
                        <tbody>
                          <tr>
                            <td>{formatDate(v.agreement_end_date)}</td>
                            <td className="num">{formatCurrency(v.buyback_amount)}</td>
                            <td><span className={`badge badge-${v.buyback_status}`}>{v.buyback_status}</span></td>
                            <td>{v.buyback_status === 'received' ? `${formatCurrency(v.buyback_received)} on ${formatDate(v.buyback_received_date || '')}` : '-'}</td>
                            <td>
                              {!isViewOnly && (v.buyback_status === 'received'
                                ? <button className="btn btn-sm btn-outline" onClick={() => undoReceipt(v, null)}>Undo</button>
                                : <button className="btn btn-sm btn-success" onClick={() => openReceipt(v, null)}>Mark Received</button>)}
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </>
                )}
                {v.notes && <div className="text-muted text-md mt-3">{v.notes}</div>}
              </div>
            )}
          </div>
        );
      })}

      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>{editing === 'new' ? 'Add Vehicle Rental Investment' : 'Edit Vehicle Rental Investment'}</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Provider</label>
                  <input required placeholder="e.g. Ridoji" value={form.provider} onChange={e => setForm({ ...form, provider: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Vehicle Model</label>
                  <input placeholder="e.g. EV Scooty" value={form.vehicle_model} onChange={e => setForm({ ...form, vehicle_model: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Number of Vehicles</label>
                  <input type="number" min={1} step={1} required value={form.units || ''} onChange={e => setForm({ ...form, units: parseInt(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Price per Vehicle, ex-GST (₹)</label>
                  <input type="number" min={0} step="0.01" required value={form.unit_price || ''} onChange={e => setForm({ ...form, unit_price: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>GST (%)</label>
                  <input type="number" min={0} max={100} step="0.01" value={form.gst_percent} onChange={e => setForm({ ...form, gst_percent: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Other Charges (₹)</label>
                  <input type="number" min={0} step="0.01" value={form.other_charges || ''} onChange={e => setForm({ ...form, other_charges: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Purchase Date</label>
                  <input type="date" required value={form.purchase_date} onChange={e => setForm({ ...form, purchase_date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>First Payout Date</label>
                  <input type="date" required min={form.purchase_date || undefined} value={form.payout_start_date} onChange={e => setForm({ ...form, payout_start_date: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Monthly Payout per Vehicle (₹)</label>
                  <input type="number" min={0} step="0.01" required value={form.monthly_payout_per_unit || ''} onChange={e => setForm({ ...form, monthly_payout_per_unit: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Agreement Tenure (months)</label>
                  <input type="number" min={1} max={240} step={1} required value={form.tenure_months || ''} onChange={e => setForm({ ...form, tenure_months: parseInt(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Buyback (% of vehicle cost)</label>
                  <input type="number" min={0} max={100} step="0.01" value={form.buyback_percent} onChange={e => setForm({ ...form, buyback_percent: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Vehicle Registration Numbers</label>
                  <input placeholder="Comma separated" value={regText} onChange={e => setRegText(e.target.value)} />
                </div>
                <div className="form-group">
                  <label>Agreement Number</label>
                  <input value={form.agreement_number} onChange={e => setForm({ ...form, agreement_number: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>

              {previewInvestment > 0 && previewMonthly > 0 && form.tenure_months > 0 && (
                <div className="card p-3 mb-4 text-md">
                  <div className="form-row gap-2">
                    <div>Total investment: <strong>{formatCurrency(previewInvestment)}</strong></div>
                    <div>Monthly payout: <strong>{formatCurrency(previewMonthly)}</strong></div>
                    <div>Total payouts ({form.tenure_months} mo): <strong>{formatCurrency(previewPayouts)}</strong></div>
                    <div>Buyback: <strong>{formatCurrency(previewBuyback)}</strong></div>
                    <div>Money back in: <strong>{(previewInvestment / previewMonthly).toFixed(1)} months</strong></div>
                    <div>Promised return: <strong>{previewIrr === null ? '-' : `${previewIrr.toFixed(0)}% / year`}</strong></div>
                  </div>
                  {previewIrr !== null && previewIrr > 25 && (
                    <div className="text-danger mt-2">
                      This promises about {previewIrr.toFixed(0)}% a year. Fixed returns this high are far above what regulated investments pay, so make sure you've verified the provider.
                    </div>
                  )}
                </div>
              )}

              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setEditing(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">{editing === 'new' ? 'Add' : 'Save Changes'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {receipt && (
        <div className="modal-overlay" onClick={() => setReceipt(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>{receipt.payout ? 'Record Payout Received' : 'Record Buyback Received'}</h2>
            <p className="text-muted text-md mb-3">
              {receipt.payout
                ? `Scheduled ${formatDate(receipt.payout.scheduled_date)} · ${formatCurrency(receipt.payout.amount)}`
                : `Buyback ${formatCurrency(receipt.rental.buyback_amount)}`}
            </p>
            <form onSubmit={saveReceipt}>
              <div className="form-row">
                <div className="form-group">
                  <label>Date Received</label>
                  <input type="date" required max={today} value={receipt.date} onChange={e => setReceipt({ ...receipt, date: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Amount Received (₹)</label>
                  <input type="number" min={0} step="0.01" required value={receipt.amount || ''} onChange={e => setReceipt({ ...receipt, amount: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setReceipt(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
