import client from './client';

export interface Insight {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'warning' | 'info' | 'positive';
  category: string;
  action?: string;
}

export interface InsightsResponse {
  insights: Insight[];
  critical_count: number;
  warning_count: number;
  info_count: number;
  positive_count: number;
}

export const getInsights = async (): Promise<InsightsResponse> => {
  const res = await client.get<InsightsResponse>('/insights');
  return res.data;
};
