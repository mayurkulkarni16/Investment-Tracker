import api from './client';

export type PolicyType = 'term' | 'health' | 'life' | 'ulip' | 'vehicle' | 'home' | 'personal_accident' | 'other';
export type PremiumFrequency = 'monthly' | 'quarterly' | 'half_yearly' | 'yearly' | 'single';
export type PolicyStatus = 'active' | 'lapsed' | 'matured' | 'surrendered' | 'expired';

export interface InsurancePolicy {
  id: string;
  user_id: string;
  policy_type: PolicyType;
  insurer: string;
  policy_name: string;
  policy_number: string;
  insured_persons: string;
  sum_assured: number;
  premium_amount: number;
  premium_frequency: PremiumFrequency;
  start_date: string;
  end_date?: string;
  premium_end_date?: string;
  current_value: number;
  nominee: string;
  status: PolicyStatus;
  notes: string;
  created_at: string;
  updated_at: string;
  next_premium_date?: string;
  annual_premium: number;
  tax_section: '80C' | '80D' | '';
  days_to_expiry?: number;
}

export interface InsurancePolicyRequest {
  policy_type: PolicyType;
  insurer: string;
  policy_name?: string;
  policy_number?: string;
  insured_persons?: string;
  sum_assured: number;
  premium_amount: number;
  premium_frequency: PremiumFrequency;
  start_date: string;
  end_date?: string;
  premium_end_date?: string;
  current_value?: number;
  nominee?: string;
  status?: PolicyStatus;
  notes?: string;
}

export interface UpcomingPremium {
  policy_id: string;
  policy_name: string;
  insurer: string;
  date: string;
  amount: number;
  days_left: number;
}

export interface InsuranceSummary {
  total_life_cover: number;
  total_health_cover: number;
  annual_premium: number;
  premium_80c_this_fy: number;
  premium_80d_this_fy: number;
  cash_value: number;
  annual_income: number;
  annual_income_source: 'ledger' | 'query' | 'none' | string;
  outstanding_loans: number;
  recommended_life_cover: number;
  life_cover_gap: number;
  recommended_health_cover: number;
  health_cover_gap: number;
  upcoming_premiums: UpcomingPremium[] | null;
  suggestions: string[] | null;
}

export const getInsurancePolicies = () => api.get<InsurancePolicy[]>('/insurance');
export const getInsurancePolicy = (id: string) => api.get<InsurancePolicy>(`/insurance/${id}`);
export const getInsuranceSummary = (annualIncome?: number) =>
  api.get<InsuranceSummary>('/insurance/summary', {
    params: annualIncome && annualIncome > 0 ? { annual_income: annualIncome } : undefined,
  });
export const createInsurancePolicy = (data: InsurancePolicyRequest) => api.post<InsurancePolicy>('/insurance', data);
export const updateInsurancePolicy = (id: string, data: InsurancePolicyRequest) => api.put<InsurancePolicy>(`/insurance/${id}`, data);
export const deleteInsurancePolicy = (id: string) => api.delete(`/insurance/${id}`);
