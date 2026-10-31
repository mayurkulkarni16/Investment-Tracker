// Shared chart palette. Use these for SVG chart props (fill/stroke), where CSS
// variables don't resolve reliably; use the CSS theme variables everywhere else.
export const CHART_COLORS = ['#6366f1', '#10b981', '#f59e0b', '#06b6d4', '#ec4899', '#8b5cf6', '#f97316', '#64748b', '#14b8a6', '#ef4444'];

export const SERIES = {
  primary: '#6366f1',
  positive: '#10b981',
  negative: '#ef4444',
  warning: '#f59e0b',
  info: '#06b6d4',
  purple: '#8b5cf6',
  orange: '#f97316',
  neutral: '#64748b',
} as const;

export const truncate = (text: string, max: number) => (text.length > max ? `${text.slice(0, max - 1)}…` : text);
