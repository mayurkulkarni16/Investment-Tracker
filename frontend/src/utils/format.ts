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

// Short Indian-unit amounts for chart axes and compact labels: ₹850, ₹85K, ₹14.5L, ₹1.2Cr.
export const formatCompactINR = (value: number): string => {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  const trim = (n: number, digits: number) => n.toFixed(digits).replace(/\.0+$/, '');
  if (abs >= 1e7) return `${sign}₹${trim(abs / 1e7, abs >= 1e9 ? 0 : 2)}Cr`;
  if (abs >= 1e5) return `${sign}₹${trim(abs / 1e5, 1)}L`;
  if (abs >= 1e3) return `${sign}₹${trim(abs / 1e3, 0)}K`;
  return `${sign}₹${trim(abs, 0)}`;
};

export const toInputDate = (dateStr?: string): string => {
  if (!dateStr) return '';
  return new Date(dateStr).toISOString().split('T')[0];
};

// "partially_matured" -> "Partially matured"
export const statusLabel = (status: string): string => {
  const s = (status || '').replace(/_/g, ' ').trim();
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : '-';
};
