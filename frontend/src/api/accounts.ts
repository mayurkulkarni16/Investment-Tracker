import api from './client';

export type AccountType = 'savings' | 'current' | 'salary' | 'cash' | 'wallet' | 'other';
export type AccountStatus = 'active' | 'closed';

export interface BalancePoint {
  date: string;
  balance: number;
  source: 'manual' | 'sms' | string;
}

export interface BankAccount {
  id: string;
  user_id: string;
  name: string;
  institution: string;
  account_type: AccountType;
  last_four: string;
  balance: number;
  balance_as_of: string;
  balance_history: BalancePoint[] | null;
  interest_rate: number;
  is_emergency_fund: boolean;
  status: AccountStatus | string;
  notes: string;
  created_at: string;
  updated_at: string;
  month_inflow: number;
  month_outflow: number;
  days_stale: number;
}

export interface BankAccountRequest {
  name: string;
  institution?: string;
  account_type: AccountType;
  last_four?: string;
  balance: number;
  balance_as_of?: string;
  interest_rate?: number;
  is_emergency_fund?: boolean;
  status?: AccountStatus;
  notes?: string;
}

export interface UpdateBalanceRequest {
  balance: number;
  as_of?: string;
  source?: 'manual' | 'sms';
}

export const getAccounts = () => api.get<BankAccount[]>('/accounts');
export const getAccount = (id: string) => api.get<BankAccount>(`/accounts/${id}`);
export const createAccount = (data: BankAccountRequest) => api.post<BankAccount>('/accounts', data);
export const updateAccount = (id: string, data: BankAccountRequest) => api.put<BankAccount>(`/accounts/${id}`, data);
export const deleteAccount = (id: string) => api.delete(`/accounts/${id}`);
export const updateAccountBalance = (id: string, data: UpdateBalanceRequest) =>
  api.put<BankAccount>(`/accounts/${id}/balance`, data);
