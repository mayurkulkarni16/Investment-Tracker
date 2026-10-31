import api from './client';
import type { ProjectionsResponse } from '../types';

// inflation > 0 returns all values in today's money.
export const getProjections = (years = 5, scenario = 'base', inflation = 0) =>
  api.get<ProjectionsResponse>('/projections', { params: { years, scenario, inflation: inflation > 0 ? inflation : undefined } });
