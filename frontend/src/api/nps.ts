import api from './client';
import type { NPSAccount, CreateNPSAccountRequest, UpdateNPSAccountRequest, AddNPSContributionRequest } from '../types';

export const getNPSAccounts = () => api.get<NPSAccount[]>('/nps');
export const getNPSAccount = (id: string) => api.get<NPSAccount>(`/nps/${id}`);
export const createNPSAccount = (data: CreateNPSAccountRequest) => api.post<NPSAccount>('/nps', data);
export const updateNPSAccount = (id: string, data: UpdateNPSAccountRequest) => api.put<NPSAccount>(`/nps/${id}`, data);
export const deleteNPSAccount = (id: string) => api.delete(`/nps/${id}`);
export const addNPSContribution = (id: string, data: AddNPSContributionRequest) => api.post<NPSAccount>(`/nps/${id}/contributions`, data);
