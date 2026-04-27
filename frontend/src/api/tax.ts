import api from './client';
import type { TaxSummary, CapitalGainsSummary } from '../types';

export const getTaxSummary = (fy?: string) => api.get<TaxSummary>('/tax/summary', { params: { fy } });
export const getCapitalGains = (fy?: string) => api.get<CapitalGainsSummary>('/tax/capital-gains', { params: { fy } });
