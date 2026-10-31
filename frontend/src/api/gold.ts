import api from './client';

export type GoldForm = 'physical' | 'digital' | 'sgb' | 'etf' | 'mutual_fund';

export interface GoldHolding {
  id: string;
  user_id: string;
  name: string;
  form: GoldForm;
  /** grams (physical/digital/sgb) or units (etf/mutual_fund) */
  quantity: number;
  /** karat: 24, 22, 18 */
  purity: number;
  /** total cost incl. making charges */
  purchase_price: number;
  purchase_date: string;
  /** per gram (of this purity) or per unit */
  current_price: number;
  price_updated_at?: string;
  symbol: string;
  interest_rate: number;
  maturity_date?: string;
  notes: string;
  created_at: string;
  updated_at: string;
  current_value: number;
  gain_loss: number;
  gain_loss_pct: number;
  cagr: number;
  interest_earned: number;
}

export interface GoldHoldingRequest {
  name: string;
  form: GoldForm;
  quantity: number;
  purity: number;
  purchase_price: number;
  purchase_date: string;
  current_price: number;
  symbol: string;
  interest_rate: number;
  maturity_date: string;
  notes: string;
}

export const GOLD_FORM_LABELS: Record<GoldForm, string> = {
  physical: 'Physical',
  digital: 'Digital',
  sgb: 'SGB',
  etf: 'ETF',
  mutual_fund: 'Mutual Fund',
};

/** Forms whose quantity is measured in grams (others are units). */
export const isGramForm = (form: GoldForm) => form === 'physical' || form === 'digital' || form === 'sgb';

export const getGoldHoldings = () => api.get<GoldHolding[]>('/gold');
export const getGoldHolding = (id: string) => api.get<GoldHolding>(`/gold/${id}`);
export const createGoldHolding = (data: GoldHoldingRequest) => api.post<GoldHolding>('/gold', data);
export const updateGoldHolding = (id: string, data: GoldHoldingRequest) => api.put<GoldHolding>(`/gold/${id}`, data);
export const deleteGoldHolding = (id: string) => api.delete(`/gold/${id}`);
export const setGoldRate = (rate_24k_per_gram: number) => api.post<GoldHolding[]>('/gold/rate', { rate_24k_per_gram });
export const refreshGoldPrices = () => api.post<GoldHolding[]>('/gold/refresh-prices');
