import api from './client';
import type { FixedDeposit, CreateFixedDepositRequest } from '../types';

export const getFixedDeposits = () => api.get<FixedDeposit[]>('/fixed-deposits');
export const getFixedDeposit = (id: string) => api.get<FixedDeposit>(`/fixed-deposits/${id}`);
export const createFixedDeposit = (data: CreateFixedDepositRequest) => api.post<FixedDeposit>('/fixed-deposits', data);
export const updateFixedDeposit = (id: string, data: Partial<CreateFixedDepositRequest>) => api.put<FixedDeposit>(`/fixed-deposits/${id}`, data);
export const deleteFixedDeposit = (id: string) => api.delete(`/fixed-deposits/${id}`);
