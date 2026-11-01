import type { LucideIcon } from 'lucide-react';
import {
  ArrowLeftRight, Bike, Briefcase, Building2, Calculator, ChartCandlestick, ChartColumn, ChartPie, CreditCard,
  FileText, Flame, Gem, HandCoins, House, Landmark, LayoutDashboard, Lightbulb, PiggyBank, Receipt, ScrollText,
  Settings, ShieldCheck, Target, Telescope, TrendingUp, Users, Wallet,
} from 'lucide-react';

export interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  keywords?: string;
}

export interface NavSection {
  title: string;
  items: NavItem[];
}

export const NAV_SECTIONS: NavSection[] = [
  {
    title: 'Overview',
    items: [
      { to: '/', label: 'Dashboard', icon: LayoutDashboard, keywords: 'home summary' },
      { to: '/net-worth', label: 'Net Worth', icon: TrendingUp, keywords: 'assets liabilities' },
      { to: '/insights', label: 'Insights', icon: Lightbulb, keywords: 'alerts rebalance suggestions' },
    ],
  },
  {
    title: 'Investments',
    items: [
      { to: '/mutual-funds', label: 'Mutual Funds', icon: ChartPie, keywords: 'mf sip nav elss cas' },
      { to: '/stocks', label: 'Stocks', icon: ChartCandlestick, keywords: 'shares equity nse bse' },
      { to: '/fixed-deposits', label: 'Fixed Deposits', icon: Landmark, keywords: 'fd' },
      { to: '/corporate-bonds', label: 'Corporate Bonds', icon: ScrollText, keywords: 'ncd debt coupon' },
      { to: '/provident-fund', label: 'Provident Fund', icon: PiggyBank, keywords: 'pf epf ppf' },
      { to: '/nps', label: 'NPS', icon: Briefcase, keywords: 'pension' },
      { to: '/gold', label: 'Gold', icon: Gem, keywords: 'sgb etf' },
      { to: '/real-estate', label: 'Real Estate', icon: Building2, keywords: 'property land' },
      { to: '/vehicle-rentals', label: 'Vehicle Rentals', icon: Bike, keywords: 'rental bike' },
    ],
  },
  {
    title: 'Money',
    items: [
      { to: '/accounts', label: 'Bank & Cash', icon: Wallet, keywords: 'accounts balance savings' },
      { to: '/ledger', label: 'Income & Expenses', icon: Receipt, keywords: 'transactions ledger spending sms' },
      { to: '/budgets', label: 'Budgets', icon: ChartColumn, keywords: 'limits categories' },
      { to: '/cashflow', label: 'Cashflow', icon: ArrowLeftRight, keywords: 'inflow outflow' },
    ],
  },
  {
    title: 'Loans & Cards',
    items: [
      { to: '/home-loans', label: 'Home Loans', icon: House, keywords: 'emi mortgage' },
      { to: '/personal-loans', label: 'Personal Loans', icon: HandCoins, keywords: 'emi' },
      { to: '/credit-cards', label: 'Credit Cards', icon: CreditCard, keywords: 'cc bill rewards' },
    ],
  },
  {
    title: 'Planning',
    items: [
      { to: '/goals', label: 'Goals', icon: Target },
      { to: '/fire', label: 'FIRE Planner', icon: Flame, keywords: 'retirement' },
      { to: '/projections', label: 'Projections', icon: Telescope, keywords: 'forecast future' },
      { to: '/tax-center', label: 'Tax Center', icon: Calculator, keywords: '80c capital gains regime old new family income salary where to invest' },
      { to: '/insurance', label: 'Insurance', icon: ShieldCheck, keywords: 'term health life policy premium' },
    ],
  },
  {
    title: 'Account',
    items: [
      { to: '/family', label: 'Family', icon: Users, keywords: 'members profiles occupation salary income joint ownership' },
      { to: '/documents', label: 'Documents', icon: FileText, keywords: 'files vault' },
      { to: '/settings', label: 'Settings', icon: Settings, keywords: 'profile backup export password' },
    ],
  },
];
