import api from './client';
import type { Stock, CreateStockRequest, UpdateStockRequest, AddStockTransactionRequest, AddStockDividendRequest } from '../types';

export const getStocks = () => api.get<Stock[]>('/stocks');
export const getStock = (id: string) => api.get<Stock>(`/stocks/${id}`);
export const createStock = (data: CreateStockRequest) => api.post<Stock>('/stocks', data);
export const updateStock = (id: string, data: UpdateStockRequest) => api.put<Stock>(`/stocks/${id}`, data);
export const deleteStock = (id: string) => api.delete(`/stocks/${id}`);
export const addStockTransaction = (id: string, data: AddStockTransactionRequest) => api.post<Stock>(`/stocks/${id}/transactions`, data);
export const deleteStockTransaction = (id: string, txnId: string) => api.delete<Stock>(`/stocks/${id}/transactions/${txnId}`);
export const updateStockTransaction = (id: string, txnId: string, data: AddStockTransactionRequest) => api.put<Stock>(`/stocks/${id}/transactions/${txnId}`, data);
export const addStockDividend = (id: string, data: AddStockDividendRequest) => api.post<Stock>(`/stocks/${id}/dividends`, data);
export const deleteStockDividend = (id: string, divId: string) => api.delete<Stock>(`/stocks/${id}/dividends/${divId}`);
export const refreshAllPrices = () => api.post<Stock[]>('/stocks/refresh-prices');
export const refreshStockPrice = (id: string) => api.post<Stock>(`/stocks/${id}/refresh-price`);
export const getMarketStatus = () => api.get<{ is_market_open: boolean }>('/stocks/market-status');
