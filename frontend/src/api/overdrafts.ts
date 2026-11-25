import api from './client';
import type { Overdraft, CreateOverdraftRequest, UpdateOverdraftRequest, AddOverdraftTransactionRequest } from '../types';

export const getOverdrafts = () => api.get<Overdraft[]>('/overdrafts');
export const createOverdraft = (data: CreateOverdraftRequest) => api.post<Overdraft>('/overdrafts', data);
export const updateOverdraft = (id: string, data: UpdateOverdraftRequest) => api.put<Overdraft>(`/overdrafts/${id}`, data);
export const deleteOverdraft = (id: string) => api.delete(`/overdrafts/${id}`);
export const addOverdraftTransaction = (id: string, data: AddOverdraftTransactionRequest) => api.post<Overdraft>(`/overdrafts/${id}/transactions`, data);
export const deleteOverdraftTransaction = (id: string, txnId: string) => api.delete<Overdraft>(`/overdrafts/${id}/transactions/${txnId}`);
