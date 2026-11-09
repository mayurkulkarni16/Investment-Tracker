import api from './client';
import type { AlternateInvestment, CreateAlternateInvestmentRequest, UpdateAltPayoutRequest } from '../types';

export const getAlternateInvestments = () => api.get<AlternateInvestment[]>('/alternate-investments');
export const getAlternateInvestment = (id: string) => api.get<AlternateInvestment>(`/alternate-investments/${id}`);
export const createAlternateInvestment = (data: CreateAlternateInvestmentRequest) => api.post<AlternateInvestment>('/alternate-investments', data);
export const updateAlternateInvestment = (id: string, data: CreateAlternateInvestmentRequest) => api.put<AlternateInvestment>(`/alternate-investments/${id}`, data);
export const deleteAlternateInvestment = (id: string) => api.delete(`/alternate-investments/${id}`);
export const updateAltPayout = (id: string, payoutId: string, data: UpdateAltPayoutRequest) =>
  api.put<AlternateInvestment>(`/alternate-investments/${id}/payouts/${payoutId}`, data);
export const markDueAltPayoutsReceived = (id: string) => api.post<AlternateInvestment>(`/alternate-investments/${id}/mark-due-received`);
export const updateAltBuyback = (id: string, data: UpdateAltPayoutRequest) =>
  api.put<AlternateInvestment>(`/alternate-investments/${id}/buyback`, data);
