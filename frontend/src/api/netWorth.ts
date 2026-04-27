import api from './client';
import type { NetWorthCurrent, NetWorthSnapshot } from '../types';

export const getNetWorthCurrent = () => api.get<NetWorthCurrent>('/net-worth/current');
export const takeNetWorthSnapshot = () => api.post<NetWorthSnapshot>('/net-worth/snapshot');
export const getNetWorthHistory = () => api.get<NetWorthSnapshot[]>('/net-worth/history');
