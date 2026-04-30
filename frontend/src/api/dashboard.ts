import api from './client';
import type { DashboardSummary } from '../types';

export const getDashboard = () => api.get<DashboardSummary>('/dashboard');

export interface RebalanceSuggestion {
  category: string;
  current_pct: number;
  target_pct: number;
  diff_pct: number;
  current_value: number;
  target_value: number;
  adjustment_amount: number;
  action: string;
}

export interface RebalanceResponse {
  total_value: number;
  suggestions: RebalanceSuggestion[];
}

export const getRebalanceSuggestions = (targets: Record<string, number>) =>
  api.post<RebalanceResponse>('/dashboard/rebalance', { targets });
