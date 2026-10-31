import type { LucideIcon } from 'lucide-react';
import { Bike, Briefcase, ChartCandlestick, ChartPie, Landmark, PiggyBank, ScrollText, Wallet } from 'lucide-react';

export interface CategoryMeta {
  key: string;
  label: string;
  color: string;
  icon: LucideIcon;
  to: string;
}

const CATEGORIES: CategoryMeta[] = [
  { key: 'mutual_funds', label: 'Mutual Funds', color: '#6366f1', icon: ChartPie, to: '/mutual-funds' },
  { key: 'stocks', label: 'Stocks', color: '#10b981', icon: ChartCandlestick, to: '/stocks' },
  { key: 'fixed_deposits', label: 'Fixed Deposits', color: '#f59e0b', icon: Landmark, to: '/fixed-deposits' },
  { key: 'corporate_bonds', label: 'Corporate Bonds', color: '#06b6d4', icon: ScrollText, to: '/corporate-bonds' },
  { key: 'provident_fund', label: 'Provident Fund', color: '#ec4899', icon: PiggyBank, to: '/provident-fund' },
  { key: 'nps', label: 'NPS', color: '#8b5cf6', icon: Briefcase, to: '/nps' },
  { key: 'vehicle_rentals', label: 'Vehicle Rentals', color: '#f97316', icon: Bike, to: '/vehicle-rentals' },
];

const BY_KEY = new Map(CATEGORIES.map(c => [c.key, c]));

// The dashboard API keys asset allocation by display name ("Mutual Funds");
// net worth snapshots use snake_case ("mutual_funds"). Both resolve to the same entry.
export const normalizeCategoryKey = (key: string) => key.trim().toLowerCase().replace(/[\s&-]+/g, '_');

export function categoryMeta(key: string): CategoryMeta {
  return BY_KEY.get(normalizeCategoryKey(key)) || { key, label: key, color: '#64748b', icon: Wallet, to: '/' };
}

export const INVESTMENT_CATEGORIES = CATEGORIES;
