import api from './client';
import type { Stock, CreateStockRequest, UpdateStockRequest, AddStockTransactionRequest } from '../types';

export const getStocks = () => api.get<Stock[]>('/stocks');
export const getStock = (id: string) => api.get<Stock>(`/stocks/${id}`);
export const createStock = (data: CreateStockRequest) => api.post<Stock>('/stocks', data);
export const updateStock = (id: string, data: UpdateStockRequest) => api.put<Stock>(`/stocks/${id}`, data);
export const deleteStock = (id: string) => api.delete(`/stocks/${id}`);
export const addStockTransaction = (id: string, data: AddStockTransactionRequest) => api.post<Stock>(`/stocks/${id}/transactions`, data);
export const refreshAllPrices = () => api.post<Stock[]>('/stocks/refresh-prices');
export const refreshStockPrice = (id: string) => api.post<Stock>(`/stocks/${id}/refresh-price`);
export const getMarketStatus = () => api.get<{ is_market_open: boolean }>('/stocks/market-status');
