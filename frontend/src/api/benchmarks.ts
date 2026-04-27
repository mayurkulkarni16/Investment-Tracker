import api from './client';
import type { BenchmarkResponse } from '../types';

export const getBenchmarks = (period?: string) => api.get<BenchmarkResponse>('/benchmarks', { params: { period } });
