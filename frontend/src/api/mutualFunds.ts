import api from './client';
import type { MutualFund, CreateMutualFundRequest, AddMFTransactionRequest, ImportCASRequest, ImportResult, UpdateMutualFundRequest } from '../types';

export const getMutualFunds = () => api.get<MutualFund[]>('/mutual-funds');
export const getMutualFund = (id: string) => api.get<MutualFund>(`/mutual-funds/${id}`);
export const createMutualFund = (data: CreateMutualFundRequest) => api.post<MutualFund>('/mutual-funds', data);
export const updateMutualFund = (id: string, data: UpdateMutualFundRequest) => api.put<MutualFund>(`/mutual-funds/${id}`, data);
export const deleteMutualFund = (id: string) => api.delete(`/mutual-funds/${id}`);
export const addMFTransaction = (id: string, data: AddMFTransactionRequest) => api.post<MutualFund>(`/mutual-funds/${id}/transactions`, data);
export const deleteMFTransaction = (id: string, txnId: string) => api.delete<MutualFund>(`/mutual-funds/${id}/transactions/${txnId}`);
export const updateMFTransaction = (id: string, txnId: string, data: AddMFTransactionRequest) => api.put<MutualFund>(`/mutual-funds/${id}/transactions/${txnId}`, data);
export const refreshNAV = () => api.post('/mutual-funds/refresh-nav');
export const importFromCAS = (data: ImportCASRequest) => api.post<ImportResult>('/mutual-funds/import', data);
