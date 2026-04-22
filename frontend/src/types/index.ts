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
