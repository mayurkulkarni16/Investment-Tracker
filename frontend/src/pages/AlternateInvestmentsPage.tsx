import { useEffect, useState } from 'react';
import {
  getAlternateInvestments, createAlternateInvestment, updateAlternateInvestment, deleteAlternateInvestment,
  updateAltPayout, markDueAltPayoutsReceived, updateAltBuyback,
} from '../api/alternateInvestments';
import type { AlternateInvestment, AltPayout, AltAssetType, AltValuationMethod, CreateAlternateInvestmentRequest } from '../types';
import { formatCurrency, formatDate, formatPercent, toInputDate, statusLabel } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import InfoTooltip from '../components/InfoTooltip';
import { Bike, ChevronDown, Copy, Package, PawPrint, Pencil, Trash2, Plus, type LucideIcon } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

interface AssetTypeMeta {
  label: string;
  icon: LucideIcon;
  unit: string;
  units: string;
  nameLabel: string;
  namePlaceholder: string;
  providerPlaceholder: string;
  idLabel: string;
}

const ASSET_TYPES: Record<AltAssetType, AssetTypeMeta> = {
  vehicle: {
    label: 'Vehicle leasing', icon: Bike, unit: 'vehicle', units: 'vehicles',
    nameLabel: 'Vehicle Model', namePlaceholder: 'e.g. EV Scooty', providerPlaceholder: 'e.g. EZU Electric',
    idLabel: 'Vehicle Registration Numbers',
  },
  livestock: {
    label: 'Livestock', icon: PawPrint, unit: 'animal', units: 'animals',
    nameLabel: 'Breed', namePlaceholder: 'e.g. Gir', providerPlaceholder: 'e.g. Amrutha Farms',
    idLabel: 'Ear Tag / Animal IDs',
  },
  other: {
    label: 'Other', icon: Package, unit: 'unit', units: 'units',
    nameLabel: 'Asset', namePlaceholder: 'e.g. Solar panel', providerPlaceholder: 'Operator name',
    idLabel: 'Identifiers',
  },
};

const ASSET_TYPE_KEYS = Object.keys(ASSET_TYPES) as AltAssetType[];

const unitsLabel = (type: AltAssetType, n: number) => `${n} ${n === 1 ? ASSET_TYPES[type].unit : ASSET_TYPES[type].units}`;

// Defaults applied when a new investment's asset type is chosen.
const TYPE_DEFAULTS: Record<AltAssetType, Pick<CreateAlternateInvestmentRequest, 'gst_percent' | 'tenure_months' | 'buyback_percent' | 'valuation_method'>> = {
  vehicle: { gst_percent: 5, tenure_months: 48, buyback_percent: 10, valuation_method: 'straight_line' },
  livestock: { gst_percent: 0, tenure_months: 0, buyback_percent: 0, valuation_method: 'at_cost' },
  other: { gst_percent: 0, tenure_months: 12, buyback_percent: 0, valuation_method: 'straight_line' },
};

const emptyForm: CreateAlternateInvestmentRequest = {
  asset_type: 'vehicle', provider: '', asset_name: '', units: 1, unit_price: 0, other_charges: 0,
  purchase_date: '', payout_start_date: '', monthly_payout_per_unit: 0, manual_value: 0,
  daily_yield_litres: 0, price_per_litre: 0, identifiers: [], agreement_number: '', notes: '',
  ...TYPE_DEFAULTS.vehicle,
};

interface Preset {
  label: string;
  note: string;
  values: Partial<CreateAlternateInvestmentRequest> & { asset_type: AltAssetType };
}

// Brochure figures, used only to pre-fill the form. Operators change the rate
// offered to new investors, so each saved investment keeps its own terms.
const EZU_NOTE = 'EZU Electric brochure (Nov 2026), price incl. GST. Tenure and buyback are not stated in the brochure; take them from your agreement.';
const AMRUTHA_NOTE = 'Amrutha Farms brochure (Nov 2026). No fixed term is stated, so this is open-ended and held at cost.';
const PRESETS: Preset[] = [
  { label: 'EZU Electric · 2 vehicles', note: EZU_NOTE, values: { asset_type: 'vehicle', provider: 'EZU Electric', asset_name: 'EV Scooter', units: 2, unit_price: 68000, gst_percent: 0, monthly_payout_per_unit: 4000 } },
  { label: 'EZU Electric · 5 vehicles', note: EZU_NOTE, values: { asset_type: 'vehicle', provider: 'EZU Electric', asset_name: 'EV Scooter', units: 5, unit_price: 67000, gst_percent: 0, monthly_payout_per_unit: 4400 } },
  { label: 'EZU Electric · 10 vehicles', note: EZU_NOTE, values: { asset_type: 'vehicle', provider: 'EZU Electric', asset_name: 'EV Scooter', units: 10, unit_price: 66000, gst_percent: 0, monthly_payout_per_unit: 4700 } },
  { label: 'EZU Electric · 20 vehicles', note: EZU_NOTE, values: { asset_type: 'vehicle', provider: 'EZU Electric', asset_name: 'EV Scooter', units: 20, unit_price: 66000, gst_percent: 0, monthly_payout_per_unit: 4700 } },
  {
    label: 'Amrutha Farms · Gir cow', note: AMRUTHA_NOTE,
    values: { asset_type: 'livestock', provider: 'Amrutha Farms', asset_name: 'Gir', units: 1, unit_price: 200000, gst_percent: 0, monthly_payout_per_unit: 20000, tenure_months: 0, buyback_percent: 0, valuation_method: 'at_cost', daily_yield_litres: 15, price_per_litre: 100 },
  },
  {
    label: 'Amrutha Farms · Jersey cow', note: AMRUTHA_NOTE,
    values: { asset_type: 'livestock', provider: 'Amrutha Farms', asset_name: 'Jersey', units: 1, unit_price: 300000, gst_percent: 0, monthly_payout_per_unit: 35000, tenure_months: 0, buyback_percent: 0, valuation_method: 'at_cost', daily_yield_litres: 30, price_per_litre: 60 },
  },
];

const VALUATION_LABELS: Record<AltValuationMethod, string> = {
  straight_line: 'Straight-line down to the buyback value',
  at_cost: 'Hold at purchase cost',
  manual: 'Enter current value manually',
};

// Local (not UTC) date as YYYY-MM-DD, so "today" is correct in IST.
const todayInput = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};

// Annualised IRR of: pay `cost` now, receive `monthly` for `months` months, then `terminal`.
function impliedAnnualReturn(cost: number, monthly: number, months: number, terminal: number): number | null {
  if (cost <= 0 || monthly <= 0 || months <= 0) return null;
  const pv = (r: number) => {
    let s = 0;
    for (let i = 1; i <= months; i++) s += monthly / Math.pow(1 + r, i);
    return s + terminal / Math.pow(1 + r, months);
  };
  let lo = -0.5, hi = 1;
  for (let k = 0; k < 200; k++) {
    const mid = (lo + hi) / 2;
    if (pv(mid) > cost) lo = mid; else hi = mid;
  }
  return (Math.pow(1 + lo, 12) - 1) * 100;
}

type PayoutState = 'received' | 'overdue' | 'due' | 'upcoming';

const payoutState = (p: AltPayout, today: string): PayoutState => {
  if (p.status === 'received') return 'received';
  if (p.is_overdue) return 'overdue';
  return toInputDate(p.scheduled_date) <= today ? 'due' : 'upcoming';
};

const stateBadge: Record<PayoutState, string> = {
  received: 'badge-received', overdue: 'badge-overdue', due: 'badge-pending', upcoming: 'badge-pending',
};

interface ReceiptDraft {
  investment: AlternateInvestment;
  payout: AltPayout | null; // null = buyback
  date: string;
  amount: number;
}

const formFromInvestment = (v: AlternateInvestment): CreateAlternateInvestmentRequest => ({
  asset_type: v.asset_type || 'vehicle', provider: v.provider, asset_name: v.asset_name, units: v.units,
  unit_price: v.unit_price, gst_percent: v.gst_percent, other_charges: v.other_charges,
  purchase_date: toInputDate(v.purchase_date), payout_start_date: toInputDate(v.payout_start_date),
  tenure_months: v.tenure_months, monthly_payout_per_unit: v.monthly_payout_per_unit,
  buyback_percent: v.buyback_percent, valuation_method: v.valuation_method || 'straight_line',
  manual_value: v.manual_value || 0, daily_yield_litres: v.daily_yield_litres || 0, price_per_litre: v.price_per_litre || 0,
  identifiers: v.identifiers || [], agreement_number: v.agreement_number, notes: v.notes,
});

export default function AlternateInvestmentsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [items, setItems] = useState<AlternateInvestment[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<AlternateInvestment | 'new' | null>(null);
  const [form, setForm] = useState<CreateAlternateInvestmentRequest>(emptyForm);
  const [idText, setIdText] = useState('');
  const [presetNote, setPresetNote] = useState('');
  const [typeFilter, setTypeFilter] = useState<AltAssetType | 'all'>('all');
  const [expanded, setExpanded] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<ReceiptDraft | null>(null);
  const today = todayInput();

  const load = () => {
    getAlternateInvestments().then(r => setItems(r.data || [])).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [viewAsUserId]);

  const replaceItem = (updated: AlternateInvestment) => setItems(prev => prev.map(i => (i.id === updated.id ? updated : i)));

  const openAdd = () => {
    const assetType = typeFilter === 'all' ? 'vehicle' : typeFilter;
    setForm({ ...emptyForm, asset_type: assetType, ...TYPE_DEFAULTS[assetType] });
    setIdText(''); setPresetNote(''); setEditing('new');
  };

  const openEdit = (v: AlternateInvestment) => {
    setForm(formFromInvestment(v));
    setIdText((v.identifiers || []).join(', '));
    setPresetNote('');
    setEditing(v);
  };

  // Starts a new investment with the same terms. Dates, identifiers and the
  // agreement number are cleared, since they belong to the original agreement.
  const openDuplicate = (v: AlternateInvestment) => {
    setForm({ ...formFromInvestment(v), purchase_date: '', payout_start_date: '', identifiers: [], agreement_number: '' });
    setIdText('');
    setPresetNote('Copied from an existing investment. Check the rate and dates against the new agreement.');
    setEditing('new');
  };

  const changeAssetType = (assetType: AltAssetType) => {
    setForm({ ...form, asset_type: assetType, ...(editing === 'new' ? TYPE_DEFAULTS[assetType] : {}) });
  };

  const applyPreset = (index: string) => {
    const preset = PRESETS[Number(index)];
    if (!preset) return;
    setForm({ ...form, ...TYPE_DEFAULTS[preset.values.asset_type], ...preset.values });
    setPresetNote(preset.note);
  };

  const setOpenEnded = (openEnded: boolean) => {
    if (openEnded) {
      setForm({
        ...form, tenure_months: 0, buyback_percent: 0,
        valuation_method: form.valuation_method === 'straight_line' ? 'at_cost' : form.valuation_method,
      });
    } else {
      setForm({ ...form, tenure_months: 12 });
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = { ...form, identifiers: idText.split(/[,\n]/).map(s => s.trim()).filter(Boolean) };
    try {
      if (editing === 'new') {
        await createAlternateInvestment(data);
        toast('Investment added');
      } else if (editing) {
        await updateAlternateInvestment(editing.id, data);
        toast('Investment updated');
      }
      setEditing(null);
      load();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      toast(msg ? `Failed to save: ${msg}` : 'Failed to save', 'error');
    }
  };

  const handleDelete = async (v: AlternateInvestment) => {
    if (!await confirm({ message: `Delete ${v.provider} (${unitsLabel(v.asset_type, v.units)}) and its payout history?`, danger: true, confirmLabel: 'Delete' })) return;
    try { await deleteAlternateInvestment(v.id); toast('Deleted'); load(); }
    catch { toast('Failed to delete', 'error'); }
  };

  const openReceipt = (investment: AlternateInvestment, payout: AltPayout | null) => {
    const scheduled = payout ? toInputDate(payout.scheduled_date) : toInputDate(investment.agreement_end_date);
    setReceipt({
      investment, payout,
      date: scheduled <= today ? scheduled : today,
      amount: payout ? payout.amount : investment.buyback_amount,
    });
  };

  const saveReceipt = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!receipt) return;
    const body = { status: 'received' as const, received_date: receipt.date, amount: receipt.amount };
    try {
      const res = receipt.payout
        ? await updateAltPayout(receipt.investment.id, receipt.payout.payout_id, body)
        : await updateAltBuyback(receipt.investment.id, body);
      replaceItem(res.data);
      setReceipt(null);
      toast(receipt.payout ? 'Payout marked received' : 'Buyback marked received');
    } catch { toast('Failed to save receipt', 'error'); }
  };

  const undoReceipt = async (investment: AlternateInvestment, payout: AltPayout | null) => {
    try {
      const res = payout
        ? await updateAltPayout(investment.id, payout.payout_id, { status: 'pending' })
        : await updateAltBuyback(investment.id, { status: 'pending' });
      replaceItem(res.data);
      toast('Marked as not received');
    } catch { toast('Failed to update', 'error'); }
  };

  const markAllDue = async (investment: AlternateInvestment) => {
    const due = investment.payouts.filter(p => p.status === 'pending' && toInputDate(p.scheduled_date) <= today);
    if (due.length === 0) { toast('No due payouts to mark'); return; }
    const total = due.reduce((s, p) => s + p.amount, 0);
    if (!await confirm({ message: `Confirm you received all ${due.length} due payout(s) totalling ${formatCurrency(total)} in full, on their scheduled dates?`, confirmLabel: 'Mark received' })) return;
    try {
      const res = await markDueAltPayoutsReceived(investment.id);
      replaceItem(res.data);
      toast(`${due.length} payout(s) marked received`);
    } catch { toast('Failed to update', 'error'); }
  };

  if (loading) return <PageSkeleton label="Loading" />;

  const presentTypes = ASSET_TYPE_KEYS.filter(t => items.some(i => (i.asset_type || 'vehicle') === t));
  const shown = typeFilter === 'all' ? items : items.filter(i => (i.asset_type || 'vehicle') === typeFilter);
  const active = shown.filter(i => i.status === 'active');
  const totalInvested = shown.reduce((s, i) => s + i.investment_amount, 0);
  const totalReceived = shown.reduce((s, i) => s + i.total_payouts_received + i.buyback_received, 0);
  const bookValue = shown.reduce((s, i) => s + i.current_value, 0);
  const netReturn = shown.reduce((s, i) => s + i.total_return, 0);
  const overdueCount = shown.reduce((s, i) => s + i.overdue_count, 0);
  const overdueAmount = shown.reduce((s, i) => s + i.overdue_amount, 0);
  const monthlyIncome = active.reduce((s, i) => s + i.monthly_payout, 0);
  const avgXirr = totalInvested > 0 ? shown.reduce((s, i) => s + i.xirr * i.investment_amount, 0) / totalInvested : 0;

  // Live preview for the add/edit form
  const meta = ASSET_TYPES[form.asset_type];
  const openEnded = form.tenure_months === 0;
  const assetCost = form.units * form.unit_price;
  const previewInvestment = assetCost * (1 + form.gst_percent / 100) + form.other_charges;
  const previewMonthly = form.units * form.monthly_payout_per_unit;
  const previewPayouts = previewMonthly * form.tenure_months;
  const previewBuyback = assetCost * form.buyback_percent / 100;
  // Open-ended: one year of payouts, ending at the year-end book value.
  const previewIrr = openEnded
    ? impliedAnnualReturn(previewInvestment, previewMonthly, 12, form.valuation_method === 'manual' ? form.manual_value : previewInvestment)
    : impliedAnnualReturn(previewInvestment, previewMonthly, form.tenure_months, previewBuyback);
  const previewMilkRevenue = form.asset_type === 'livestock' ? form.units * form.daily_yield_litres * form.price_per_litre * 30 : 0;

  return (
    <div>
      <div className="page-header">
        <h1>Alternate Investments</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Investment</button>}
      </div>

      {presentTypes.length > 1 && (
        <div className="flex gap-2 mb-4" role="group" aria-label="Filter by asset type">
          {(['all', ...presentTypes] as const).map(t => (
            <button key={t} className={`btn btn-sm ${typeFilter === t ? 'btn-primary' : 'btn-outline'}`} aria-pressed={typeFilter === t} onClick={() => setTypeFilter(t)}>
              {t === 'all' ? 'All' : ASSET_TYPES[t].label}
            </button>
          ))}
        </div>
      )}

      {overdueCount > 0 && (
        <div className="card mb-4" style={{ borderLeft: '4px solid var(--danger)' }}>
          <strong className="text-danger">
            {overdueCount} payout{overdueCount > 1 ? 's' : ''} overdue ({formatCurrency(overdueAmount)})
          </strong>
          <div className="text-muted text-md mt-1">
            Payouts more than 5 days past their scheduled date. Follow up with the provider, and keep the written agreement and payment records.
          </div>
        </div>
      )}

      {shown.length > 0 && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card">
            <div className="label">Units (active)</div>
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
            <div className="label">Book Value<InfoTooltip tip="Per investment: straight-line from cost down to the buyback value, held at cost, or the value you entered." /></div>
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
          <h3>No alternate investments yet</h3>
          <p>Track operator-managed schemes such as EV fleet leasing or cow ownership: monthly payouts, overdue payments, buyback and actual returns.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}><Plus size={16} aria-hidden="true" />Add Investment</button>}
        </div>
      ) : shown.map(v => {
        const type = v.asset_type || 'vehicle';
        const TypeIcon = ASSET_TYPES[type].icon;
        const dueCount = v.payouts.filter(p => p.status === 'pending' && toInputDate(p.scheduled_date) <= today).length;
        return (
          <div key={v.id} className="card" style={v.overdue_count > 0 ? { borderLeft: '4px solid var(--danger)' } : undefined}>
            <div className="flex justify-between items-start">
              <div>
                <h3 className="text-lg">
                  <TypeIcon size={18} aria-label={ASSET_TYPES[type].label} className="mr-2" style={{ verticalAlign: 'text-bottom' }} />
                  {v.provider}{v.asset_name ? ` · ${v.asset_name}` : ''}
                  <span className={`badge badge-${v.status} ml-2`}>{statusLabel(v.status)}</span>
                  {v.overdue_count > 0 && <span className="badge badge-overdue ml-2">{v.overdue_count} overdue</span>}
                </h3>
                <div className="text-muted text-md">
                  {unitsLabel(type, v.units)} · {formatDate(v.purchase_date)} → {v.open_ended ? 'open-ended' : formatDate(v.agreement_end_date)}
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
                      { label: 'Edit investment', icon: Pencil, onClick: () => openEdit(v) },
                      { label: 'Duplicate as new', icon: Copy, onClick: () => openDuplicate(v) },
                      { label: 'Delete investment', icon: Trash2, onClick: () => handleDelete(v), danger: true },
                    ]}
                  />
                )}
              </div>
            </div>

            <div className="card-grid mt-3">
              <div><span className="text-muted text-sm">Invested</span><div className="fw-semibold">{formatCurrency(v.investment_amount)}</div></div>
              <div><span className="text-muted text-sm">Monthly Payout</span><div className="fw-semibold">{formatCurrency(v.monthly_payout)} <span className="text-muted text-sm">({formatCurrency(v.monthly_payout_per_unit)}/{ASSET_TYPES[type].unit})</span></div></div>
              <div><span className="text-muted text-sm">Payouts Received</span><div className="fw-semibold">{v.open_ended ? v.payouts_received_count : `${v.payouts_received_count} / ${v.payouts.length}`} · {formatCurrency(v.total_payouts_received)}</div></div>
              <div><span className="text-muted text-sm">Next Payout</span><div className="fw-semibold">{v.next_payout_date ? formatDate(v.next_payout_date) : '-'}</div></div>
              <div><span className="text-muted text-sm">Capital Recovered</span><div className="fw-semibold">{(v.capital_recovered_pct || 0).toFixed(1)}%</div></div>
              <div><span className="text-muted text-sm">Break-even<InfoTooltip tip="Months of payouts needed to get the amount invested back." /></span><div className="fw-semibold">{v.break_even_months > 0 ? `${v.break_even_months.toFixed(1)} months` : '-'}</div></div>
              <div><span className="text-muted text-sm">Book Value</span><div className="fw-semibold">{formatCurrency(v.current_value)}</div></div>
              <div><span className="text-muted text-sm">Net Return</span><div className="fw-semibold" style={{ color: v.total_return >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatCurrency(v.total_return)}</div></div>
              <div><span className="text-muted text-sm">XIRR</span><div className="fw-semibold" style={{ color: v.xirr >= 0 ? 'var(--success)' : 'var(--danger)' }}>{formatPercent(v.xirr)}</div></div>
            </div>
            {v.gross_monthly_revenue > 0 && (
              <div className="text-muted text-sm mt-2">
                Stated milk sales: {formatCurrency(v.gross_monthly_revenue)}/month · your payout is {(v.monthly_payout / v.gross_monthly_revenue * 100).toFixed(0)}% of it
              </div>
            )}
            {v.identifiers?.length > 0 && (
              <div className="text-muted text-sm mt-2">{ASSET_TYPES[type].idLabel}: {v.identifiers.join(', ')}</div>
            )}

            {expanded === v.id && (
              <div className="mt-4">
                <div className="flex justify-between items-center mb-2">
                  <h4>Monthly Payouts</h4>
                  {!isViewOnly && dueCount > 0 && (
                    <button className="btn btn-sm btn-success" onClick={() => markAllDue(v)}>Mark all due as received ({dueCount})</button>
                  )}
                </div>
                {v.open_ended && <p className="text-muted text-sm mb-2">Open-ended: the schedule always runs 12 months ahead.</p>}
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
                    <h4 className="mb-2">Buyback ({v.buyback_percent}% of cost)</h4>
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
            <h2>{editing === 'new' ? 'Add Alternate Investment' : 'Edit Alternate Investment'}</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label htmlFor="alt-asset-type">Asset Type</label>
                  <select id="alt-asset-type" value={form.asset_type} onChange={e => changeAssetType(e.target.value as AltAssetType)}>
                    {ASSET_TYPE_KEYS.map(t => <option key={t} value={t}>{ASSET_TYPES[t].label}</option>)}
                  </select>
                </div>
                {editing === 'new' && (
                  <div className="form-group">
                    <label htmlFor="alt-preset">Quick Fill (optional)</label>
                    <select id="alt-preset" value="" onChange={e => applyPreset(e.target.value)}>
                      <option value="">Choose a known plan…</option>
                      {PRESETS.map((p, i) => <option key={p.label} value={i}>{p.label}</option>)}
                    </select>
                  </div>
                )}
              </div>
              {presetNote && <p className="text-muted text-sm mb-3">{presetNote}</p>}
              <div className="form-row">
                <div className="form-group">
                  <label>Provider</label>
                  <input required placeholder={meta.providerPlaceholder} value={form.provider} onChange={e => setForm({ ...form, provider: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>{meta.nameLabel}</label>
                  <input placeholder={meta.namePlaceholder} value={form.asset_name} onChange={e => setForm({ ...form, asset_name: e.target.value })} />
                </div>
              </div>
              <div className="form-row-3">
                <div className="form-group">
                  <label>Number of {meta.units}</label>
                  <input type="number" min={1} step={1} required value={form.units || ''} onChange={e => setForm({ ...form, units: parseInt(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Price per {meta.unit}, ex-GST (₹)</label>
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
                  <label>Monthly Payout per {meta.unit} (₹)</label>
                  <input type="number" min={0} step="0.01" required value={form.monthly_payout_per_unit || ''} onChange={e => setForm({ ...form, monthly_payout_per_unit: parseFloat(e.target.value) || 0 })} />
                </div>
                <div className="form-group">
                  <label>Agreement Tenure (months)</label>
                  <input type="number" min={1} max={240} step={1} required={!openEnded} disabled={openEnded} value={openEnded ? '' : form.tenure_months || ''} placeholder={openEnded ? 'Open-ended' : undefined} onChange={e => setForm({ ...form, tenure_months: parseInt(e.target.value) || 0 })} />
                  <label className="text-sm mt-1">
                    <input className="w-auto mr-2" type="checkbox" checked={openEnded} onChange={e => setOpenEnded(e.target.checked)} />Open-ended (no fixed term)
                  </label>
                </div>
                <div className="form-group">
                  <label>Buyback (% of cost)</label>
                  <input type="number" min={0} max={100} step="0.01" disabled={openEnded} value={form.buyback_percent} onChange={e => setForm({ ...form, buyback_percent: parseFloat(e.target.value) || 0 })} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label htmlFor="alt-valuation">Book Value Method</label>
                  <select id="alt-valuation" value={form.valuation_method} onChange={e => setForm({ ...form, valuation_method: e.target.value as AltValuationMethod })}>
                    {(Object.keys(VALUATION_LABELS) as AltValuationMethod[]).map(m => (
                      <option key={m} value={m} disabled={m === 'straight_line' && openEnded}>{VALUATION_LABELS[m]}</option>
                    ))}
                  </select>
                </div>
                {form.valuation_method === 'manual' && (
                  <div className="form-group">
                    <label>Current Value (₹)</label>
                    <input type="number" min={0} step="0.01" required value={form.manual_value || ''} onChange={e => setForm({ ...form, manual_value: parseFloat(e.target.value) || 0 })} />
                  </div>
                )}
              </div>
              {form.asset_type === 'livestock' && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Stated Milk Yield per {meta.unit} (litres/day)</label>
                    <input type="number" min={0} step="0.1" value={form.daily_yield_litres || ''} onChange={e => setForm({ ...form, daily_yield_litres: parseFloat(e.target.value) || 0 })} />
                  </div>
                  <div className="form-group">
                    <label>Stated Milk Price (₹/litre)</label>
                    <input type="number" min={0} step="0.01" value={form.price_per_litre || ''} onChange={e => setForm({ ...form, price_per_litre: parseFloat(e.target.value) || 0 })} />
                  </div>
                </div>
              )}
              <div className="form-row">
                <div className="form-group">
                  <label>{meta.idLabel}</label>
                  <input placeholder="Comma separated" value={idText} onChange={e => setIdText(e.target.value)} />
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

              {previewInvestment > 0 && previewMonthly > 0 && (openEnded || form.tenure_months > 0) && (
                <div className="card p-3 mb-4 text-md">
                  <div className="form-row gap-2">
                    <div>Total investment: <strong>{formatCurrency(previewInvestment)}</strong></div>
                    <div>Monthly payout: <strong>{formatCurrency(previewMonthly)}</strong></div>
                    {openEnded
                      ? <div>Payouts per year: <strong>{formatCurrency(previewMonthly * 12)}</strong></div>
                      : <div>Total payouts ({form.tenure_months} mo): <strong>{formatCurrency(previewPayouts)}</strong></div>}
                    {!openEnded && <div>Buyback: <strong>{formatCurrency(previewBuyback)}</strong></div>}
                    <div>Money back in: <strong>{(previewInvestment / previewMonthly).toFixed(1)} months</strong></div>
                    <div>Promised return: <strong>{previewIrr === null ? '-' : `${previewIrr.toFixed(0)}% / year`}</strong></div>
                  </div>
                  {previewMilkRevenue > 0 && (
                    <div className="text-muted mt-2">
                      Stated milk sales: {formatCurrency(previewMilkRevenue)}/month. The payout is {(previewMonthly / previewMilkRevenue * 100).toFixed(0)}% of that, before feed, care and labour costs.
                    </div>
                  )}
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
                : `Buyback ${formatCurrency(receipt.investment.buyback_amount)}`}
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
