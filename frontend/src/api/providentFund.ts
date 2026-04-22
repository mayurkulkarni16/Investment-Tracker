import api from './client';
import type { ProvidentFund, CreateProvidentFundRequest, AddMonthlyContributionRequest } from '../types';
import type { ParsedPFData } from '../utils/pfParser';

export const getProvidentFunds = () => api.get<ProvidentFund[]>('/provident-fund');
export const createProvidentFund = (data: CreateProvidentFundRequest) => api.post<ProvidentFund>('/provident-fund', data);
export const updateProvidentFund = (id: string, data: Partial<CreateProvidentFundRequest>) => api.put<ProvidentFund>(`/provident-fund/${id}`, data);
export const addContribution = (id: string, data: AddMonthlyContributionRequest) => api.post<ProvidentFund>(`/provident-fund/${id}/entries`, data);
export const getContributions = (id: string) => api.get(`/provident-fund/${id}/entries`);
export const deleteProvidentFund = (id: string) => api.delete(`/provident-fund/${id}`);
export const importPFFromPDF = (id: string, data: ParsedPFData) => api.post(`/provident-fund/${id}/import`, data);
