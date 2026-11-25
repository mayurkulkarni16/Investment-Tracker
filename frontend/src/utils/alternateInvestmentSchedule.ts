import type { CreateAlternateInvestmentRequest } from '../types';

const DAY = 86_400_000;

function calendarDate(year: number, month: number, day: number) {
  const date = new Date(0);
  date.setUTCFullYear(year, month, day);
  return date;
}

function addMonthsClamped(date: Date, months: number) {
  const year = date.getUTCFullYear();
  const month = date.getUTCMonth() + months;
  const lastDay = calendarDate(year, month + 1, 0).getUTCDate();
  return calendarDate(year, month, Math.min(date.getUTCDate(), lastDay));
}

export const roundPayout = (amount: number) => Math.round(amount * 100) / 100;

export interface PreviewPayout { date: Date; amount: number }

export function previewAltSchedule(terms: Pick<CreateAlternateInvestmentRequest,
  'asset_type' | 'purchase_date' | 'tenure_months' | 'units' | 'monthly_payout_per_unit'>) {
  const purchase = new Date(`${terms.purchase_date}T00:00:00Z`);
  if (!Number.isFinite(purchase.getTime())) return null;
  const monthly = terms.units * terms.monthly_payout_per_unit;
  const livestock = terms.asset_type === 'livestock';
  const start = livestock ? new Date(purchase.getTime() + 15 * DAY) : purchase;
  const fixed = terms.tenure_months > 0;
  const months = Math.min(terms.tenure_months, 240);
  const billingEnd = fixed && livestock ? addMonthsClamped(start, months) : null;
  const yearEnd = addMonthsClamped(purchase, 12);
  const payouts: PreviewPayout[] = [];
  for (let i = 0; ; i++) {
    const year = start.getUTCFullYear();
    const month = start.getUTCMonth() + i;
    const billingMonth = calendarDate(year, month, 1);
    const nextMonth = calendarDate(year, month + 1, 1);
    const date = calendarDate(year, month + 1, 5);
    if (fixed ? (billingEnd ? billingMonth >= billingEnd : i >= months) : (i > 0 && date > yearEnd)) break;
    const coveredStart = Math.max(billingMonth.getTime(), start.getTime());
    const coveredEnd = Math.min(nextMonth.getTime(), billingEnd?.getTime() ?? Infinity);
    const days = (coveredEnd - coveredStart) / DAY;
    const monthDays = (nextMonth.getTime() - billingMonth.getTime()) / DAY;
    payouts.push({ date, amount: roundPayout(livestock ? monthly * days / monthDays : monthly) });
  }
  return { purchase, billingStart: livestock ? start : null, payouts, yearEnd };
}

export function impliedAnnualReturn(cost: number, purchase: Date, payouts: PreviewPayout[], terminal: PreviewPayout) {
  if (cost <= 0 || !payouts.some(p => p.amount > 0)) return null;
  const inflows = [...payouts, terminal].filter(p => p.amount > 0).map(p => ({
    amount: p.amount, years: (p.date.getTime() - purchase.getTime()) / (365 * DAY),
  }));
  if (inflows.some(p => p.years <= 0)) return null;
  const pv = (rate: number) => inflows.reduce((sum, p) => sum + p.amount / Math.pow(1 + rate, p.years), 0);
  let lo = -0.999999, hi = 1;
  while (pv(hi) > cost && hi < 1e6) hi = 2 * hi + 1;
  if (pv(lo) < cost || pv(hi) > cost) return null;
  for (let i = 0; i < 80; i++) {
    const mid = (lo + hi) / 2;
    if (pv(mid) > cost) lo = mid; else hi = mid;
  }
  return (lo + hi) / 2 * 100;
}
