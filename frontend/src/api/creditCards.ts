import api from './client';
import type {
  CreditCard, CreateCreditCardRequest, UpdateCreditCardRequest,
  AddCardStatementRequest, PayStatementRequest, AddCardTransactionRequest,
  AddCardEMIRequest, AddCreditScoreRequest
} from '../types';

export const getCreditCards = () => api.get<CreditCard[]>('/credit-cards');
export const getCreditCard = (id: string) => api.get<CreditCard>(`/credit-cards/${id}`);
export const createCreditCard = (data: CreateCreditCardRequest) => api.post<CreditCard>('/credit-cards', data);
export const updateCreditCard = (id: string, data: UpdateCreditCardRequest) => api.put<CreditCard>(`/credit-cards/${id}`, data);
export const deleteCreditCard = (id: string) => api.delete(`/credit-cards/${id}`);
export const addCardStatement = (id: string, data: AddCardStatementRequest) => api.post<CreditCard>(`/credit-cards/${id}/statements`, data);
export const payStatement = (id: string, data: PayStatementRequest) => api.post<CreditCard>(`/credit-cards/${id}/pay-statement`, data);
export const addCardTransaction = (id: string, data: AddCardTransactionRequest) => api.post<CreditCard>(`/credit-cards/${id}/transactions`, data);
export const addCardEMI = (id: string, data: AddCardEMIRequest) => api.post<CreditCard>(`/credit-cards/${id}/emis`, data);
export const addCreditScore = (id: string, data: AddCreditScoreRequest) => api.post<CreditCard>(`/credit-cards/${id}/credit-score`, data);
