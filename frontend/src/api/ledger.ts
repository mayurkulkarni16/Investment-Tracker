import api from './client';

export type LedgerAccountType = '' | 'bank_account' | 'credit_card';

export interface LedgerDraft {
  source_key: string;
  source: 'manual' | 'sms';
  status: 'pending' | 'confirmed' | 'rejected';
  kind: 'income' | 'expense' | 'refund' | 'transfer' | 'bill';
  amount_paise: number;
  date: string;
  description: string;
  category: string;
  bank: string;
  account: string;
  due_date: string;
  profile_id: string;
  linked_account_type: LedgerAccountType;
  linked_account_id: string;
  sms_received_at: string;
}
export interface LedgerEntry extends LedgerDraft {
  id: string;
  created_at: string;
  updated_at: string;
}

export interface LedgerMember {
  id: string;
  name: string;
}

export interface LedgerAccount {
  type: Exclude<LedgerAccountType, ''>;
  id: string;
  name: string;
  bank: string;
  last_four: string;
  status: string;
  member_ids: string[];
}

export interface LedgerContext {
  members: LedgerMember[];
  accounts: LedgerAccount[];
}

export const ledgerApi = {
  list: () => api.get<LedgerEntry[]>('/ledger'),
  context: () => api.get<LedgerContext>('/ledger/context'),
  create: (draft: LedgerDraft) => api.post<LedgerEntry>('/ledger', draft),
  update: (id: string, draft: LedgerDraft) => api.put<LedgerEntry>(`/ledger/${id}`, draft),
  export: () => api.get<Blob>('/ledger/export', { responseType: 'blob' }),
};
