import api from './client';
import type {
  PersonalLoan,
  CreatePersonalLoanRequest,
  UpdatePersonalLoanRequest,
  AddEMIPaymentRequest,
  AddPrepaymentRequest,
  ChangeRateRequest,
  AmortizationEntry,
} from '../types';

export const getPersonalLoans = () => api.get<PersonalLoan[]>('/personal-loans');
export const getPersonalLoan = (id: string) => api.get<PersonalLoan>(`/personal-loans/${id}`);
export const createPersonalLoan = (data: CreatePersonalLoanRequest) => api.post<PersonalLoan>('/personal-loans', data);
export const updatePersonalLoan = (id: string, data: UpdatePersonalLoanRequest) => api.put<PersonalLoan>(`/personal-loans/${id}`, data);
export const deletePersonalLoan = (id: string) => api.delete(`/personal-loans/${id}`);
export const recordPersonalLoanEMI = (id: string, data: AddEMIPaymentRequest) => api.post<PersonalLoan>(`/personal-loans/${id}/emi`, data);
export const addPersonalLoanPrepayment = (id: string, data: AddPrepaymentRequest) => api.post<PersonalLoan>(`/personal-loans/${id}/prepayment`, data);
export const changePersonalLoanRate = (id: string, data: ChangeRateRequest) => api.post<PersonalLoan>(`/personal-loans/${id}/rate-change`, data);
export const getPersonalLoanAmortization = (id: string) => api.get<AmortizationEntry[]>(`/personal-loans/${id}/amortization`);
