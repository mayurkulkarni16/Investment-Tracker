import type { NetWorthSnapshot } from '../../types';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

// "2026-10" -> "Oct 2026" (or "Oct '26" when short)
export function monthLabel(month: string, short = false): string {
  const [y, m] = month.split('-').map(Number);
  if (!y || !m) return month;
  return short ? `${MONTHS[m - 1]} '${String(y).slice(-2)}` : `${MONTHS[m - 1]} ${y}`;
}

export const SNAPSHOT_INVESTMENT_KEYS = [
  'mutual_funds', 'stocks', 'fixed_deposits', 'corporate_bonds', 'provident_fund', 'nps', 'vehicle_rentals',
] as const;

export type SnapshotInvestmentKey = typeof SNAPSHOT_INVESTMENT_KEYS[number];

export function snapshotValue(s: NetWorthSnapshot, key: SnapshotInvestmentKey): number {
  return Number((s as unknown as Record<string, number | undefined>)[key] || 0);
}
