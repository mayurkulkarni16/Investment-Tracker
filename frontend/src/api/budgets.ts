import api from './client';

export interface Budget {
  id: string;
  user_id: string;
  category: string;
  monthly_limit: number;
  created_at: string;
  updated_at: string;
}

export interface BudgetRequest {
  category: string;
  monthly_limit: number;
}

export interface LedgerMonth {
  month: string; // YYYY-MM
  income: number;
  expense: number;
  bills: number;
  transfers: number;
  net_savings: number;
  savings_rate_pct: number;
  by_category: Record<string, number> | null;
  income_by_category: Record<string, number> | null;
  entry_count: number;
}

export type BudgetState = 'ok' | 'warning' | 'over';

export interface BudgetStatus {
  category: string;
  limit: number;
  spent: number;
  remaining: number;
  used_pct: number;
  status: BudgetState | string;
}

export interface CategorySpend {
  category: string;
  amount: number;
  pct: number;
}

export interface LedgerSummary {
  months: LedgerMonth[] | null; // oldest first
  current_month: LedgerMonth;
  avg_monthly_income: number;
  avg_monthly_expense: number;
  avg_savings_rate_pct: number;
  top_categories: CategorySpend[] | null;
  budgets: BudgetStatus[] | null;
  pending_review_count: number;
}

export const getBudgets = () => api.get<Budget[]>('/budgets');
/** Upserts by category (case-insensitive). */
export const saveBudget = (data: BudgetRequest) => api.post<Budget>('/budgets', data);
export const deleteBudget = (id: string) => api.delete(`/budgets/${id}`);
export const getLedgerSummary = (months = 12) =>
  api.get<LedgerSummary>('/ledger/summary', { params: { months } });
