import api from './client';
import type { ProjectionsResponse } from '../types';

export const getProjections = (years = 5) => api.get<ProjectionsResponse>(`/projections?years=${years}`);
