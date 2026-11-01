import api from './client';
import type { TaxSummary, CapitalGainsSummary } from '../types';

export const getTaxSummary = (fy?: string, profileId?: string) =>
  api.get<TaxSummary>('/tax/summary', { params: { fy, profile_id: profileId || undefined } });
export const getCapitalGains = (fy?: string, profileId?: string) =>
  api.get<CapitalGainsSummary>('/tax/capital-gains', { params: { fy, profile_id: profileId || undefined } });

// ---------- Family tax: income records, regime comparison, placement ----------

export type Regime = 'old' | 'new';

export interface Payslip {
  month: string; // YYYY-MM
  projected: boolean;
  employer?: string;
  basic: number;
  da: number;
  hra: number;
  special_allowance: number;
  lta: number;
  other_allowances: number;
  perquisites: number;
  bonus: number;
  incentives: number;
  esop_perquisite: number;
  other_variable: number;
  employee_pf: number;
  employer_nps: number;
  professional_tax: number;
  tds: number;
  rent_paid: number;
}

export interface OtherIncome {
  pension: number;
  family_pension: number;
  business_scheme: '' | 'actual' | '44AD' | '44ADA';
  business_receipts: number;
  business_profit: number;
  savings_interest: number;
  other_interest: number;
  dividends: number;
  rental_income: number;
  municipal_tax: number;
  other_income: number;
  stcg_equity: number;
  ltcg_equity: number;
  other_gains_slab: number;
}

export interface ManualDeductions {
  sec_80c_other: number;
  sec_80d_self: number;
  sec_80d_parents: number;
  parents_senior: boolean;
  sec_80e: number;
  sec_80g: number;
  nps_other: number;
  lta_exempt: number;
  home_loan_interest: number;
  other_tds: number;
  advance_tax: number;
}

export interface MemberIncome {
  id?: string;
  profile_id: string;
  fy: string;
  payslips: Payslip[];
  project_remaining: boolean;
  other: OtherIncome;
  deductions: ManualDeductions;
  notes?: string;
}

export type MemberIncomeRequest = Pick<MemberIncome, 'payslips' | 'project_remaining' | 'other' | 'deductions' | 'notes'>;

export interface DeductionLine {
  code: string;
  new_section?: string;
  label: string;
  claimed: number;
  allowed: number;
  limit?: number;
}

export interface SlabLine { from: number; to: number; rate: number; amount: number; tax: number }

export interface RegimeResult {
  regime: Regime;
  gross_salary: number;
  standard_deduction: number;
  exemptions: number;
  professional_tax: number;
  salary_income: number;
  house_property: number;
  business_income: number;
  other_sources: number;
  special_income: number;
  gross_total_income: number;
  deductions: DeductionLine[];
  total_deductions: number;
  normal_income: number;
  total_income: number;
  slab_tax: number;
  special_tax: number;
  rebate_87a: number;
  surcharge: number;
  cess: number;
  total_tax: number;
  effective_rate: number;
  marginal_rate: number;
  slabs: SlabLine[];
}

export interface Headroom {
  sec_80c: number;
  sec_80ccd1b: number;
  sec_80d_self: number;
  sec_80d_parents: number;
  ltcg_exemption: number;
  rebate_new_room: number;
}

export interface RegimeComparison {
  fy: string;
  act: string;
  old: RegimeResult;
  new: RegimeResult;
  recommended: Regime;
  savings: number;
  break_even_deductions: number;
  old_tax_with_headroom: number;
  headroom: Headroom;
}

export interface IncomeBreakdown {
  salary_fixed: number;
  salary_variable: number;
  employer_nps: number;
  projected_months: number;
  actual_months: number;
  pension: number;
  business: number;
  interest: number;
  dividends: number;
  rent: number;
  other_rental: number;
  other_income: number;
  stcg_equity: number;
  ltcg_equity: number;
  gains_at_slab: number;
  clubbed_income: number;
  total: number;
}

export interface TaxLine {
  asset_type: string;
  asset_id: string;
  name: string;
  kind: string;
  amount: number;
  senior?: boolean;
  profile_id?: string;
  owner_id?: string;
  clubbed?: boolean;
  clubbing_ignored?: boolean;
  share_pct?: number;
}

export interface MemberTaxReport {
  profile_id: string;
  name: string;
  relationship: string;
  color: string;
  occupation: string;
  age: number;
  resident: boolean;
  has_income_record: boolean;
  preferred_regime: Regime | '';
  income: IncomeBreakdown;
  comparison: RegimeComparison;
  chosen_regime: Regime;
  chosen_tax: number;
  extra_tax_vs_best: number;
  tax_paid: number;
  balance_payable: number;
  lines: TaxLine[];
  warnings: string[];
}

export interface TaxSuggestion {
  id: string;
  profile_id: string;
  member_name: string;
  category: 'regime' | 'deduction' | 'placement' | 'clubbing' | 'data' | 'capital_gains';
  title: string;
  detail: string;
  est_saving: number;
}

export interface TaxSettings {
  ignore_spouse_clubbing: boolean;
  ignore_minor_clubbing: boolean;
  updated_at?: string;
}

export interface FamilyTaxReport {
  fy: string;
  settings: TaxSettings;
  clubbing_ignored: number;
  act: string;
  supported_fys: string[];
  members: MemberTaxReport[];
  total_tax_best: number;
  total_tax_preferred: number;
  regime_savings: number;
  unassigned_items: number;
  default_member_name: string;
  suggestions: TaxSuggestion[];
  disclaimer: string;
}

export interface TaxMeta {
  supported_fys: string[];
  current_fy: string;
  occupations: Record<string, string>;
  instruments: Record<string, string>;
}

export interface PlacementRequest {
  fy?: string;
  amount: number;
  instrument: string;
  expected_return_pct: number;
  funded_by?: string;
}

export interface PlacementOption {
  profile_id: string;
  name: string;
  eligible: boolean;
  taxpayer_id: string;
  taxpayer_name: string;
  annual_income: number;
  incremental_tax: number;
  deduction_saving: number;
  net_tax_impact: number;
  post_tax_income: number;
  post_tax_yield_pct: number;
  notes: string[];
}

export interface PlacementResult {
  fy: string;
  request: PlacementRequest;
  options: PlacementOption[];
  best_profile_id: string;
  notes: string[];
}

export const familyTaxApi = {
  meta: () => api.get<TaxMeta>('/tax/meta'),
  report: (fy?: string) => api.get<FamilyTaxReport>('/tax/family', { params: { fy } }),
  settings: () => api.get<TaxSettings>('/tax/settings'),
  saveSettings: (data: Pick<TaxSettings, 'ignore_spouse_clubbing' | 'ignore_minor_clubbing'>) => api.put<TaxSettings>('/tax/settings', data),
  incomes: (fy?: string) => api.get<MemberIncome[]>('/tax/incomes', { params: { fy } }),
  getIncome: (profileId: string, fy?: string) => api.get<MemberIncome>(`/tax/incomes/${profileId}`, { params: { fy } }),
  saveIncome: (profileId: string, fy: string, data: MemberIncomeRequest) =>
    api.put<MemberIncome>(`/tax/incomes/${profileId}`, data, { params: { fy } }),
  deleteIncome: (profileId: string, fy: string) => api.delete(`/tax/incomes/${profileId}`, { params: { fy } }),
  placement: (req: PlacementRequest) => api.get<PlacementResult>('/tax/placement', { params: { ...req, funded_by: req.funded_by || undefined } }),
};

// Old-Act section with the Income-tax Act, 2025 number when applicable: "80C · s.123".
export const sectionLabel = (code: string, newSection?: string) =>
  newSection ? `${code} · ${newSection === 'Schedule' ? 'Schedule' : `s.${newSection}`}` : code;

export const fyOptions = (supported: string[] | undefined, current?: string) => {
  const set = new Set([...(supported || []), ...(current ? [current] : [])]);
  return [...set].sort();
};
