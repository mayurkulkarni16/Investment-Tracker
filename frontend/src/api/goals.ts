import api from './client';
import type { Goal, CreateGoalRequest, UpdateGoalRequest, LinkInvestmentRequest } from '../types';

export const getGoals = () => api.get<Goal[]>('/goals');
export const getGoal = (id: string) => api.get<Goal>(`/goals/${id}`);
export const createGoal = (data: CreateGoalRequest) => api.post<Goal>('/goals', data);
export const updateGoal = (id: string, data: UpdateGoalRequest) => api.put<Goal>(`/goals/${id}`, data);
export const deleteGoal = (id: string) => api.delete(`/goals/${id}`);
export const linkInvestment = (id: string, data: LinkInvestmentRequest) => api.post<Goal>(`/goals/${id}/link`, data);
export const batchLinkInvestments = (id: string, investments: LinkInvestmentRequest[]) => api.post<Goal>(`/goals/${id}/link-batch`, { investments });
export const unlinkInvestment = (id: string, investmentId: string) => api.delete(`/goals/${id}/link/${investmentId}`);
