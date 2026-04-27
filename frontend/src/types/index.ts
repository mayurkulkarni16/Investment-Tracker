// Mutual Fund types
export type TransactionType = 'purchase' | 'sip' | 'redemption' | 'switch_in' | 'switch_out';
export type FundType = 'Equity' | 'Debt' | 'Hybrid' | 'ELSS' | 'Index' | 'Liquid';

export interface MFTransaction {
  transaction_id: string;
  date: string;
  type: TransactionType;
  amount: number;
  nav_at_purchase: number;
  units: number;
  lock_in_end?: string;
}

export interface MutualFund {
  id: string;
  fund_name: string;
  amc: string;
  fund_type: FundType;
  scheme_code: string;
  folio_number: string;
  is_elss: boolean;
  lock_in_end_date?: string;
  transactions: MFTransaction[];
  total_units: number;
  total_invested: number;
  current_nav: number;
  current_value: number;
  nav_last_updated?: string;
  gain_loss: number;
  gain_loss_percent: number;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateMutualFundRequest {
  fund_name: string;
  amc: string;
  fund_type: FundType;
  scheme_code: string;
  folio_number?: string;
  is_elss: boolean;
  notes?: string;
}

export interface AddMFTransactionRequest {
  date: string;
  type: TransactionType;
  amount: number;
  nav_at_purchase: number;
  units?: number;
}

// Corporate Bond types
export type PayoutFrequency = 'monthly' | 'quarterly' | 'biannually' | 'annually';
export type MaturityType = 'bullet' | 'staggered';
export type PayoutStatus = 'pending' | 'received';
export type BondStatus = 'active' | 'matured' | 'partially_matured';

export interface PrincipalRepayment {
  repayment_id: string;
  scheduled_date: string;
  amount: number;
  status: PayoutStatus;
  received_date?: string;
}

export interface InterestPayout {
  payout_id: string;
  scheduled_date: string;
  principal_at_time: number;
  amount: number;
  status: PayoutStatus;
  received_date?: string;
}

export interface CorporateBond {
  id: string;
  bond_name: string;
  issuer: string;
  purchase_date: string;
  investment_amount: number;
  coupon_rate: number;
  interest_payout_frequency: PayoutFrequency;
  maturity_date: string;
  maturity_type: MaturityType;
  principal_repayments: PrincipalRepayment[];
  interest_payouts: InterestPayout[];
  remaining_principal: number;
  total_interest_earned: number;
  total_principal_returned: number;
  status: BondStatus;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface PrincipalRepaymentInput {
  scheduled_date: string;
  amount: number;
}

export interface CreateCorporateBondRequest {
  bond_name: string;
  issuer: string;
  purchase_date: string;
  investment_amount: number;
  coupon_rate: number;
  interest_payout_frequency: PayoutFrequency;
  maturity_date: string;
  maturity_type: MaturityType;
  principal_repayments?: PrincipalRepaymentInput[];
  notes?: string;
}

// Fixed Deposit types
export type InterestType = 'cumulative' | 'non_cumulative';
export type FDStatus = 'active' | 'matured' | 'premature_closed';

export interface FixedDeposit {
  id: string;
  bank_name: string;
  fd_number: string;
  principal_amount: number;
  interest_rate: number;
  start_date: string;
  maturity_date: string;
  tenure_months: number;
  interest_type: InterestType;
  payout_frequency?: PayoutFrequency;
  maturity_amount: number;
  interest_earned: number;
  is_auto_renewed: boolean;
  status: FDStatus;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateFixedDepositRequest {
  bank_name: string;
  fd_number?: string;
  principal_amount: number;
  interest_rate: number;
  start_date: string;
  maturity_date: string;
  tenure_months: number;
  interest_type: InterestType;
  payout_frequency?: PayoutFrequency;
  is_auto_renewed?: boolean;
  notes?: string;
}

// Provident Fund types
export type PFAccountType = 'EPF' | 'VPF' | 'PPF';

export interface MonthlyContribution {
  month: string;
  employee_contribution: number;
  employer_contribution: number;
  total: number;
}

export interface FinancialYearEntry {
  financial_year: string;
  monthly_contributions: MonthlyContribution[];
  opening_balance: number;
  interest_earned: number;
  closing_balance: number;
}

export interface ProvidentFund {
  id: string;
  account_type: PFAccountType;
  account_number: string;
  employer_name: string;
  interest_rate: number;
  financial_year_entries: FinancialYearEntry[];
  current_balance: number;
  total_employee_contribution: number;
  total_employer_contribution: number;
  total_interest_earned: number;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProvidentFundRequest {
  account_type: PFAccountType;
  account_number: string;
  employer_name?: string;
  interest_rate: number;
  notes?: string;
}

export interface AddMonthlyContributionRequest {
  financial_year: string;
  month: string;
  employee_contribution: number;
  employer_contribution?: number;
}

// Dashboard types
export interface InvestmentSummary {
  total_invested: number;
  current_value: number;
  count: number;
}

export interface UpcomingPayout {
  type: string;
  name: string;
  date: string;
  amount: number;
  payout_type: string;
}

export interface DashboardSummary {
  total_invested: number;
  current_value: number;
  total_gains: number;
  overall_return_percent: number;
  elss_tax_saving: number;
  asset_allocation: Record<string, number>;
  upcoming_payouts: UpcomingPayout[];
  mutual_fund_summary: InvestmentSummary;
  corporate_bond_summary: InvestmentSummary;
  fixed_deposit_summary: InvestmentSummary;
  provident_fund_summary: InvestmentSummary;
  stock_summary: InvestmentSummary;
  home_loan_summary: LoanSummary;
  personal_loan_summary: LoanSummary;
}

export interface LoanSummary {
  total_disbursed: number;
  total_outstanding: number;
  total_interest_paid: number;
  total_prepayments: number;
  monthly_emi: number;
  count: number;
}

// CAS Import types
export interface ImportCASTransaction {
  date: string;
  type: string;
  amount: number;
  nav: number;
  units: number;
}

export interface ImportCASFund {
  fund_name: string;
  category: string;
  folio_number: string;
  transactions: ImportCASTransaction[];
}

export interface ImportCASRequest {
  funds: ImportCASFund[];
}

export interface ImportResult {
  funds_created: number;
  funds_updated: number;
  transactions_added: number;
  errors?: string[];
}

// Update types
export interface UpdateMutualFundRequest {
  fund_name: string;
  amc: string;
  fund_type: FundType;
  scheme_code: string;
  folio_number?: string;
  is_elss: boolean;
  notes?: string;
}

// Stock types
export type StockExchange = 'NSE' | 'BSE';
export type StockTransactionType = 'buy' | 'sell';

export interface StockTransaction {
  transaction_id: string;
  date: string;
  type: StockTransactionType;
  quantity: number;
  price_per_share: number;
  amount: number;
}

export interface Stock {
  id: string;
  stock_name: string;
  symbol: string;
  exchange: StockExchange;
  transactions: StockTransaction[];
  total_quantity: number;
  total_invested: number;
  avg_buy_price: number;
  current_price: number;
  current_value: number;
  day_change: number;
  day_change_percent: number;
  gain_loss: number;
  gain_loss_percent: number;
  price_last_updated?: string;
  is_market_open: boolean;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateStockRequest {
  stock_name: string;
  symbol: string;
  exchange: StockExchange;
  notes?: string;
}

export interface UpdateStockRequest {
  stock_name?: string;
  symbol?: string;
  exchange?: StockExchange;
  notes?: string;
}

export interface AddStockTransactionRequest {
  date: string;
  type: StockTransactionType;
  quantity: number;
  price_per_share: number;
}

// Home Loan types
export type LoanRateType = 'fixed' | 'floating';
export type LoanStatus = 'active' | 'closed' | 'foreclosed';
export type PrepaymentType = 'part_payment' | 'foreclosure';

export interface RateChangeEntry {
  effective_date: string;
  old_rate: number;
  new_rate: number;
  new_emi: number;
}

export interface DisbursementEntry {
  disbursement_id: string;
  date: string;
  amount: number;
  tranche: number;
  pre_emi_amount: number;
  pre_emi_days: number;
  days_in_month: number;
  notes: string;
}

export interface PreEMIEntry {
  pre_emi_id: string;
  month: string;
  disbursed_at_time: number;
  tranche_amount: number;
  interest_rate: number;
  interest_amount: number;
  days_charged: number;
  days_in_month: number;
  paid: boolean;
  paid_date?: string;
}

export interface EMIEntry {
  emi_id: string;
  month: string;
  due_date: string;
  emi_amount: number;
  principal_portion: number;
  interest_portion: number;
  outstanding_after: number;
  paid: boolean;
  paid_date?: string;
}

export interface Prepayment {
  prepayment_id: string;
  date: string;
  amount: number;
  type: PrepaymentType;
  new_emi?: number;
  new_tenure?: number;
  notes: string;
}

export interface HomeLoan {
  id: string;
  bank_name: string;
  loan_account_number: string;
  property_address: string;
  loan_purpose: string;
  sanctioned_amount: number;
  disbursed_amount: number;
  interest_rate: number;
  rate_type: LoanRateType;
  tenure_months: number;
  emi_amount: number;
  emi_start_date: string;
  disbursement_date: string;
  co_borrower: string;
  is_under_construction: boolean;
  full_emi_started: boolean;
  disbursements: DisbursementEntry[];
  pre_emis_paid: PreEMIEntry[];
  total_pre_emi_paid: number;
  rate_change_history: RateChangeEntry[];
  emis_paid: EMIEntry[];
  prepayments: Prepayment[];
  outstanding_principal: number;
  total_principal_paid: number;
  total_interest_paid: number;
  total_amount_paid: number;
  total_prepayments: number;
  remaining_tenure_months: number;
  loan_end_date: string;
  interest_paid_this_fy: number;
  principal_paid_this_fy: number;
  pre_emi_paid_this_fy: number;
  status: LoanStatus;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateHomeLoanRequest {
  bank_name: string;
  loan_account_number?: string;
  property_address?: string;
  loan_purpose?: string;
  sanctioned_amount: number;
  disbursed_amount: number;
  interest_rate: number;
  rate_type: LoanRateType;
  tenure_months: number;
  emi_start_date: string;
  disbursement_date: string;
  is_under_construction?: boolean;
  co_borrower?: string;
  notes?: string;
}

export interface UpdateHomeLoanRequest {
  bank_name?: string;
  loan_account_number?: string;
  property_address?: string;
  loan_purpose?: string;
  co_borrower?: string;
  notes?: string;
}

export interface AddEMIPaymentRequest {
  month: string;
  paid_date: string;
}

export interface AddPrepaymentRequest {
  date: string;
  amount: number;
  type: PrepaymentType;
  notes?: string;
}

export interface ChangeRateRequest {
  effective_date: string;
  new_rate: number;
}

export interface AddDisbursementRequest {
  date: string;
  amount: number;
  notes?: string;
}

export interface MarkConstructionCompleteRequest {
  completion_date?: string;
}

export interface AmortizationEntry {
  month: number;
  emi: number;
  principal_portion: number;
  interest_portion: number;
  outstanding_after: number;
}

// Personal Loan types
export interface PersonalLoan {
  id: string;
  lender_name: string;
  loan_account_number: string;
  loan_purpose: string;
  principal_amount: number;
  disbursed_amount: number;
  interest_rate: number;
  rate_type: LoanRateType;
  tenure_months: number;
  emi_amount: number;
  emi_start_date: string;
  disbursement_date: string;
  processing_fee: number;
  foreclosure_charges: number;
  rate_change_history: RateChangeEntry[];
  emis_paid: EMIEntry[];
  prepayments: Prepayment[];
  outstanding_principal: number;
  total_principal_paid: number;
  total_interest_paid: number;
  total_amount_paid: number;
  total_prepayments: number;
  remaining_tenure_months: number;
  loan_end_date: string;
  status: LoanStatus;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePersonalLoanRequest {
  lender_name: string;
  loan_account_number?: string;
  loan_purpose?: string;
  principal_amount: number;
  disbursed_amount: number;
  interest_rate: number;
  rate_type: LoanRateType;
  tenure_months: number;
  emi_start_date: string;
  disbursement_date: string;
  processing_fee?: number;
  foreclosure_charges?: number;
  notes?: string;
}

export interface UpdatePersonalLoanRequest {
  lender_name?: string;
  loan_account_number?: string;
  loan_purpose?: string;
  notes?: string;
}

// Projection types
export interface ProjectionPoint {
  year: number;
  month: number;
  value: number;
}

export interface InvestmentProjection {
  name: string;
  category: string;
  current_value: number;
  assumed_rate_pct: number;
  projections: ProjectionPoint[];
}

export interface ProjectionsResponse {
  investments: InvestmentProjection[];
  aggregate_monthly: ProjectionPoint[];
  total_current: number;
  total_projected_1y: number;
  total_projected_3y: number;
  total_projected_5y: number;
}

// NPS types
export type NPSAccountType = 'tier_1' | 'tier_2';

export interface NPSContribution {
  contribution_id: string;
  date: string;
  amount: number;
  type: string;
  fy: string;
}

export interface NPSAccount {
  id: string;
  account_holder_name: string;
  pran: string;
  account_type: NPSAccountType;
  fund_manager: string;
  date_of_joining: string;
  equity_pct: number;
  corporate_bond_pct: number;
  govt_sec_pct: number;
  alternate_pct: number;
  contributions: NPSContribution[];
  total_self_contribution: number;
  total_employer_contribution: number;
  total_contribution: number;
  current_value: number;
  section_80ccd1: number;
  section_80ccd1b: number;
  section_80ccd2: number;
  status: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateNPSAccountRequest {
  account_holder_name: string;
  pran: string;
  account_type: NPSAccountType;
  fund_manager: string;
  date_of_joining: string;
  equity_pct: number;
  corporate_bond_pct: number;
  govt_sec_pct: number;
  alternate_pct: number;
  current_value: number;
  notes?: string;
}

export interface UpdateNPSAccountRequest {
  account_holder_name?: string;
  fund_manager?: string;
  equity_pct?: number;
  corporate_bond_pct?: number;
  govt_sec_pct?: number;
  alternate_pct?: number;
  current_value?: number;
  notes?: string;
}

export interface AddNPSContributionRequest {
  date: string;
  amount: number;
  type: string;
}

// Credit Card types
export interface CardTransaction {
  transaction_id: string;
  date: string;
  description: string;
  amount: number;
  category: string;
  is_emi: boolean;
  emi_reference: string;
}

export interface CardStatement {
  statement_id: string;
  month: string;
  statement_date: string;
  due_date: string;
  total_amount: number;
  minimum_due: number;
  amount_paid: number;
  paid_date?: string;
  transactions: CardTransaction[];
  is_paid: boolean;
  paid_full: boolean;
}

export interface CardEMI {
  emi_id: string;
  description: string;
  merchant_name: string;
  original_amount: number;
  emi_amount: number;
  tenure_months: number;
  remaining_months: number;
  interest_rate: number;
  processing_fee: number;
  start_date: string;
  total_paid: number;
  status: string;
}

export interface CreditScoreEntry {
  score_id: string;
  date: string;
  score: number;
  bureau: string;
  notes: string;
}

export interface CreditCard {
  id: string;
  card_name: string;
  bank_name: string;
  card_network: string;
  last_four_digits: string;
  card_holder_name: string;
  credit_limit: number;
  billing_date: number;
  due_date_offset: number;
  annual_fee: number;
  joining_date: string;
  current_outstanding: number;
  available_credit: number;
  utilization_pct: number;
  reward_points: number;
  reward_point_value: number;
  card_emis: CardEMI[];
  statements: CardStatement[];
  credit_scores: CreditScoreEntry[];
  total_spent_this_month: number;
  spend_by_category: Record<string, number>;
  credit_score_tips: string[];
  status: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCreditCardRequest {
  card_name: string;
  bank_name: string;
  card_network: string;
  last_four_digits: string;
  card_holder_name: string;
  credit_limit: number;
  billing_date: number;
  due_date_offset: number;
  annual_fee?: number;
  joining_date: string;
  reward_point_value?: number;
  notes?: string;
}

export interface UpdateCreditCardRequest {
  card_name?: string;
  credit_limit?: number;
  billing_date?: number;
  due_date_offset?: number;
  annual_fee?: number;
  reward_points?: number;
  current_outstanding?: number;
  notes?: string;
}

export interface AddCardStatementRequest {
  month: string;
  statement_date: string;
  due_date: string;
  total_amount: number;
  minimum_due: number;
}

export interface PayStatementRequest {
  statement_id: string;
  amount_paid: number;
  paid_date: string;
}

export interface AddCardTransactionRequest {
  statement_id: string;
  date: string;
  description: string;
  amount: number;
  category: string;
}

export interface AddCardEMIRequest {
  description: string;
  merchant_name: string;
  original_amount: number;
  tenure_months: number;
  interest_rate?: number;
  processing_fee?: number;
  start_date: string;
}

export interface AddCreditScoreRequest {
  date: string;
  score: number;
  bureau: string;
  notes?: string;
}

// Goal types
export interface LinkedInvestment {
  investment_type: string;
  investment_id: string;
  investment_name: string;
  allocated_pct: number;
}

export interface Goal {
  id: string;
  name: string;
  category: string;
  icon: string;
  priority: string;
  target_amount: number;
  target_date: string;
  assumed_return_rate: number;
  linked_investments: LinkedInvestment[];
  current_value: number;
  progress_pct: number;
  monthly_needed: number;
  months_remaining: number;
  on_track: boolean;
  status: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateGoalRequest {
  name: string;
  category: string;
  icon?: string;
  priority?: string;
  target_amount: number;
  target_date: string;
  assumed_return_rate: number;
  notes?: string;
}

export interface UpdateGoalRequest {
  name?: string;
  category?: string;
  icon?: string;
  priority?: string;
  target_amount?: number;
  target_date?: string;
  assumed_return_rate?: number;
  status?: string;
  notes?: string;
}

export interface LinkInvestmentRequest {
  investment_type: string;
  investment_id: string;
  investment_name: string;
  allocated_pct: number;
}

// Net Worth types
export interface NetWorthSnapshot {
  id: string;
  date: string;
  month: string;
  mutual_funds: number;
  stocks: number;
  fixed_deposits: number;
  provident_fund: number;
  nps: number;
  corporate_bonds: number;
  other_assets: number;
  total_assets: number;
  home_loans: number;
  personal_loans: number;
  credit_card_outstanding: number;
  other_liabilities: number;
  total_liabilities: number;
  net_worth: number;
  created_at: string;
}

export interface NetWorthCurrent {
  assets: Record<string, number>;
  liabilities: Record<string, number>;
  total_assets: number;
  total_liabilities: number;
  net_worth: number;
}

// SIP types
export interface SIPInstallment {
  installment_id: string;
  date: string;
  amount: number;
  nav: number;
  units: number;
  status: string;
}

export interface SIP {
  id: string;
  fund_name: string;
  fund_id: string;
  amc_code: string;
  scheme_code: number;
  amount: number;
  frequency: string;
  sip_date: number;
  start_date: string;
  end_date?: string;
  total_installments: number;
  completed_installments: number;
  missed_installments: number;
  total_invested: number;
  installments: SIPInstallment[];
  next_sip_date: string;
  months_active: number;
  status: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSIPRequest {
  fund_name: string;
  fund_id?: string;
  amc_code?: string;
  scheme_code?: number;
  amount: number;
  frequency: string;
  sip_date: number;
  start_date: string;
  end_date?: string;
  notes?: string;
}

export interface UpdateSIPRequest {
  amount?: number;
  sip_date?: number;
  end_date?: string;
  status?: string;
  notes?: string;
}

export interface RecordSIPInstallmentRequest {
  date: string;
  amount: number;
  nav?: number;
  units?: number;
  status: string;
}

// Notification types
export interface Notification {
  id: string;
  type: string;
  title: string;
  message: string;
  date: string;
  reference_type: string;
  reference_id: string;
  is_read: boolean;
  created_at: string;
}

// Profile types
export interface Profile {
  id: string;
  name: string;
  relationship: string;
  color: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateProfileRequest {
  name: string;
  relationship: string;
  color: string;
  is_default?: boolean;
}

export interface UpdateProfileRequest {
  name?: string;
  relationship?: string;
  color?: string;
}

// Tax types
export interface Section80CBreakdown {
  epf_contribution: number;
  ppf_contribution: number;
  elss_investment: number;
  home_loan_principal: number;
  life_insurance: number;
  total: number;
  limit: number;
  deduction: number;
}

export interface Section80CCDBreakdown {
  nps_80ccd1: number;
  nps_80ccd1b: number;
  nps_80ccd2: number;
}

export interface Section24bBreakdown {
  home_loan_interest: number;
  pre_emi_interest: number;
  total: number;
  limit: number;
  deduction: number;
}

export interface CapitalGainEntry {
  investment_type: string;
  investment_name: string;
  buy_date: string;
  sell_date: string;
  buy_amount: number;
  sell_amount: number;
  gain: number;
  holding_days: number;
  is_long_term: boolean;
  tax_rate: number;
  tax_liability: number;
}

export interface CapitalGainsSummary {
  stcg: number;
  ltcg: number;
  stcg_tax: number;
  ltcg_tax: number;
  total_tax: number;
  entries: CapitalGainEntry[];
  ltcg_exemption: number;
}

export interface TaxSummary {
  fy: string;
  section_80c: Section80CBreakdown;
  section_80ccd: Section80CCDBreakdown;
  section_24b: Section24bBreakdown;
  capital_gains: CapitalGainsSummary;
  interest_income: number;
  dividend_income: number;
  total_deductions: number;
  total_taxable_gains: number;
}

// Benchmark types
export interface BenchmarkPoint {
  date: string;
  value: number;
}

export interface BenchmarkData {
  index_name: string;
  symbol: string;
  points: BenchmarkPoint[];
  current: number;
  change_1y: number;
  change_3y: number;
  change_5y: number;
}

export interface BenchmarkResponse {
  indices: BenchmarkData[];
}
