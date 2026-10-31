import api from './client';

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
}
export interface LedgerEntry extends LedgerDraft {
  id: string;
  created_at: string;
  updated_at: string;
}
export const ledgerApi = {
  list: () => api.get<LedgerEntry[]>('/ledger'),
  create: (draft: LedgerDraft) => api.post<LedgerEntry>('/ledger', draft),
  update: (id: string, draft: LedgerDraft) => api.put<LedgerEntry>(`/ledger/${id}`, draft),
  export: () => api.get<Blob>('/ledger/export', { responseType: 'blob' }),
};
