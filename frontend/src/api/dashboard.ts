import api from './client';
import type { DashboardSummary } from '../types';

export const getDashboard = () => api.get<DashboardSummary>('/dashboard');
