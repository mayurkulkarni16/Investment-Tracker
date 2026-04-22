export const formatCurrency = (amount: number): string => {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
};

export const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
};

export const formatPercent = (val: number): string => {
  return `${val >= 0 ? '+' : ''}${val.toFixed(2)}%`;
};

export const toInputDate = (dateStr?: string): string => {
  if (!dateStr) return '';
  return new Date(dateStr).toISOString().split('T')[0];
};
