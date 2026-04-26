import api from './client';
import type {
  HomeLoan,
  CreateHomeLoanRequest,
  UpdateHomeLoanRequest,
  AddEMIPaymentRequest,
  AddPrepaymentRequest,
  ChangeRateRequest,
  AddDisbursementRequest,
  MarkConstructionCompleteRequest,
  AmortizationEntry,
} from '../types';

export const getHomeLoans = () => api.get<HomeLoan[]>('/home-loans');
export const getHomeLoan = (id: string) => api.get<HomeLoan>(`/home-loans/${id}`);
export const createHomeLoan = (data: CreateHomeLoanRequest) => api.post<HomeLoan>('/home-loans', data);
export const updateHomeLoan = (id: string, data: UpdateHomeLoanRequest) => api.put<HomeLoan>(`/home-loans/${id}`, data);
export const deleteHomeLoan = (id: string) => api.delete(`/home-loans/${id}`);
export const recordEMI = (id: string, data: AddEMIPaymentRequest) => api.post<HomeLoan>(`/home-loans/${id}/emi`, data);
export const addPrepayment = (id: string, data: AddPrepaymentRequest) => api.post<HomeLoan>(`/home-loans/${id}/prepayment`, data);
export const changeRate = (id: string, data: ChangeRateRequest) => api.post<HomeLoan>(`/home-loans/${id}/rate-change`, data);
export const getAmortization = (id: string) => api.get<AmortizationEntry[]>(`/home-loans/${id}/amortization`);
export const addDisbursement = (id: string, data: AddDisbursementRequest) => api.post<HomeLoan>(`/home-loans/${id}/disbursement`, data);
export const markConstructionComplete = (id: string, data?: MarkConstructionCompleteRequest) => api.post<HomeLoan>(`/home-loans/${id}/mark-complete`, data || {});
export const recalculateLoan = (id: string) => api.post<HomeLoan>(`/home-loans/${id}/recalculate`);
