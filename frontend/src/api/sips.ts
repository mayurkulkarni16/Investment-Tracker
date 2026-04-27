import api from './client';
import type { SIP, CreateSIPRequest, UpdateSIPRequest, RecordSIPInstallmentRequest } from '../types';

export const getSIPs = () => api.get<SIP[]>('/sips');
export const getSIP = (id: string) => api.get<SIP>(`/sips/${id}`);
export const createSIP = (data: CreateSIPRequest) => api.post<SIP>('/sips', data);
export const updateSIP = (id: string, data: UpdateSIPRequest) => api.put<SIP>(`/sips/${id}`, data);
export const deleteSIP = (id: string) => api.delete(`/sips/${id}`);
export const recordSIPInstallment = (id: string, data: RecordSIPInstallmentRequest) => api.post<SIP>(`/sips/${id}/installments`, data);
