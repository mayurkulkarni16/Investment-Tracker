import client from './client';

export interface MonthlyCashflow {
  month: string;
  inflow: number;
  outflow: number;
  net: number;
}

export interface CashflowDetail {
  date: string;
  category: string;
  name: string;
  type: string;
  amount: number;
  direction: string;
}

export const getMonthlyCashflows = async (months: number = 12): Promise<MonthlyCashflow[]> => {
  const res = await client.get<MonthlyCashflow[]>(`/cashflow?months=${months}`);
  return res.data;
};

export const getMonthDetail = async (month: string): Promise<CashflowDetail[]> => {
  const res = await client.get<CashflowDetail[]>(`/cashflow/${month}`);
  return res.data;
};
