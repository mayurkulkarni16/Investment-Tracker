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
