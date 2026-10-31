import api from './client';

// ---------- Family ----------

export interface AssetItem {
  asset_type: string;
  asset_id: string;
  name: string;
  category: string;
  value: number;
  is_liability: boolean;
  profile_id: string; // "" = unassigned
}

export interface MemberSummary {
  profile_id: string; // "" for unassigned
  name: string;
  relationship: string;
  color: string;
  total_assets: number;
  total_liabilities: number;
  net_worth: number;
  assets: Record<string, number> | null;
  liabilities: Record<string, number> | null;
  item_count: number;
}

export interface FamilySummary {
  members: MemberSummary[] | null;
  items: AssetItem[] | null;
}

export interface AssignProfileRequest {
  asset_type: string;
  asset_id: string;
  profile_id: string;
}

export const familyApi = {
  summary: () => api.get<FamilySummary>('/family/summary'),
  assign: (data: AssignProfileRequest) => api.put<{ message: string }>('/family/assign', data),
};

// ---------- FIRE planner ----------

export interface FIREDefaults {
  monthly_expenses: number;
  monthly_expenses_source: string; // ledger, none
  monthly_savings: number;
  monthly_savings_source: string; // ledger, sips, none
  current_corpus: number;
  pre_retirement_return: number;
  post_retirement_return: number;
  inflation: number;
}

export interface FIREPlanRequest {
  current_age: number;
  retirement_age: number;
  life_expectancy: number;
  monthly_expenses: number;
  current_corpus: number;
  monthly_savings: number;
  annual_step_up_pct: number;
  pre_retirement_return: number;
  post_retirement_return: number;
  inflation: number;
  retirement_expense_pct: number;
}

export interface FIREYear {
  age: number;
  year: number;
  phase: 'accumulation' | 'retirement' | string;
  contribution: number;
  withdrawal: number;
  end_corpus: number;
  required_corpus: number;
}

export interface FIREPlan {
  input: FIREPlanRequest;
  years_to_retirement: number;
  annual_expense_at_retirement: number;
  target_corpus: number;
  swr_corpus: number;
  projected_corpus: number;
  gap: number;
  on_track: boolean;
  required_monthly_savings: number;
  earliest_retirement_age: number;
  corpus_depletes_at_age: number;
  success_probability_pct: number;
  fi_number: number;
  progress_pct: number;
  years: FIREYear[] | null;
}

export const fireApi = {
  defaults: () => api.get<FIREDefaults>('/fire/defaults'),
  plan: (data: FIREPlanRequest) => api.post<FIREPlan>('/fire/plan', data),
};

// ---------- Documents ----------

export type DocumentCategory = 'agreement' | 'receipt' | 'policy' | 'statement' | 'tax' | 'id_proof' | 'other';

export const DOCUMENT_CATEGORIES: { value: DocumentCategory; label: string }[] = [
  { value: 'agreement', label: 'Agreement' },
  { value: 'receipt', label: 'Receipt' },
  { value: 'policy', label: 'Policy' },
  { value: 'statement', label: 'Statement' },
  { value: 'tax', label: 'Tax' },
  { value: 'id_proof', label: 'ID Proof' },
  { value: 'other', label: 'Other' },
];

export const MAX_DOCUMENT_BYTES = 10 * 1024 * 1024;

export interface VaultDocument {
  id: string;
  user_id: string;
  title: string;
  file_name: string;
  content_type: string;
  size: number;
  category: DocumentCategory | string;
  asset_type: string;
  asset_id: string;
  expiry_date?: string;
  notes: string;
  created_at: string;
}

export interface UploadDocumentInput {
  file: File;
  title: string;
  category: string;
  asset_type?: string;
  asset_id?: string;
  expiry_date?: string; // YYYY-MM-DD
  notes?: string;
}

// The client defaults to JSON; a multipart Content-Type makes axios send FormData as-is
// (the browser then fills in the boundary).
const multipart = { headers: { 'Content-Type': 'multipart/form-data' } };

export const documentsApi = {
  list: (filter?: { asset_type?: string; asset_id?: string }) =>
    api.get<VaultDocument[]>('/documents', {
      params: filter?.asset_type ? { asset_type: filter.asset_type, asset_id: filter.asset_id || undefined } : undefined,
    }),
  upload: (input: UploadDocumentInput) => {
    const fd = new FormData();
    fd.append('file', input.file);
    fd.append('title', input.title);
    fd.append('category', input.category);
    if (input.asset_type) fd.append('asset_type', input.asset_type);
    if (input.asset_id) fd.append('asset_id', input.asset_id);
    if (input.expiry_date) fd.append('expiry_date', input.expiry_date);
    if (input.notes) fd.append('notes', input.notes);
    return api.post<VaultDocument>('/documents', fd, multipart);
  },
  download: (id: string) => api.get<Blob>(`/documents/${id}/download`, { responseType: 'blob' }),
  remove: (id: string) => api.delete<{ message: string }>(`/documents/${id}`),
};

// ---------- Broker import ----------

export type BrokerName = 'auto' | 'zerodha' | 'groww';
export type BrokerImportAction = 'create' | 'add_trade' | 'skip_duplicate' | 'skip_exists' | 'conflict' | 'error';

export interface BrokerImportRow {
  symbol: string;
  name: string;
  action: BrokerImportAction | string;
  type?: string;
  date?: string;
  quantity: number;
  price: number;
  message?: string;
}

export interface BrokerImportResult {
  broker: string;
  kind: 'holdings' | 'tradebook' | string;
  dry_run: boolean;
  rows: number;
  stocks_created: number;
  trades_added: number;
  skipped: number;
  items: BrokerImportRow[] | null;
  errors?: string[] | null;
}

export const brokerImportApi = {
  importCSV: (file: File, broker: BrokerName, dryRun: boolean, asOf?: string) => {
    const fd = new FormData();
    fd.append('file', file);
    fd.append('broker', broker);
    fd.append('dry_run', dryRun ? 'true' : 'false');
    if (asOf) fd.append('as_of', asOf);
    return api.post<BrokerImportResult>('/stocks/import-broker', fd, multipart);
  },
};

// ---------- Notification settings ----------

export interface NotificationSettings {
  id: string;
  user_id: string;
  email_enabled: boolean;
  email: string;
  telegram_enabled: boolean;
  telegram_chat_id: string;
  last_dispatch_at?: string;
  updated_at: string;
  email_available: boolean;
  telegram_available: boolean;
}

export interface NotificationSettingsRequest {
  email_enabled: boolean;
  email: string;
  telegram_enabled: boolean;
  telegram_chat_id: string;
}

export const notificationSettingsApi = {
  get: () => api.get<NotificationSettings>('/notifications/settings'),
  update: (data: NotificationSettingsRequest) => api.put<NotificationSettings>('/notifications/settings', data),
  test: () => api.post<{ message: string; sent: string[] | null }>('/notifications/test'),
};
