import api from './client';
import type { BudgetStatus } from './budgets';
import type { UpcomingPremium } from './insurance';

export interface FinanceOverview {
  net_worth: number;
  total_assets: number;
  total_liabilities: number;
  net_worth_change_month?: number;
  bank_balance: number;
  cash_in_hand: number;
  liquid_balance: number;
  emergency_fund: number;
  emergency_months: number;
  stale_balances: number;
  month_income: number;
  month_expense: number;
  month_savings_rate_pct: number;
  avg_monthly_expense: number;
  avg_savings_rate_pct: number;
  pending_review: number;
  budgets_over: number;
  budgets_warning: number;
  budgets: BudgetStatus[] | null;
  gold_value: number;
  real_estate_value: number;
  life_cover: number;
  health_cover: number;
  upcoming_premiums: UpcomingPremium[] | null;
  fi_number: number;
  fi_progress_pct: number;
}

export const getFinanceOverview = () => api.get<FinanceOverview>('/finance/overview');
