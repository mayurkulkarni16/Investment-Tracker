import api from './client';
import type { CorporateBond, CreateCorporateBondRequest } from '../types';

export const getCorporateBonds = () => api.get<CorporateBond[]>('/corporate-bonds');
export const getCorporateBond = (id: string) => api.get<CorporateBond>(`/corporate-bonds/${id}`);
export const createCorporateBond = (data: CreateCorporateBondRequest) => api.post<CorporateBond>('/corporate-bonds', data);
export const updateCorporateBond = (id: string, data: CreateCorporateBondRequest) => api.put<CorporateBond>(`/corporate-bonds/${id}`, data);
export const deleteCorporateBond = (id: string) => api.delete(`/corporate-bonds/${id}`);
export const markPayoutReceived = (bondId: string, payoutId: string) => api.put(`/corporate-bonds/${bondId}/payouts/${payoutId}`);
