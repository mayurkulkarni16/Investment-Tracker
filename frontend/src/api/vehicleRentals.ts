import api from './client';
import type { VehicleRental, CreateVehicleRentalRequest, UpdateRentalPayoutRequest } from '../types';

export const getVehicleRentals = () => api.get<VehicleRental[]>('/vehicle-rentals');
export const getVehicleRental = (id: string) => api.get<VehicleRental>(`/vehicle-rentals/${id}`);
export const createVehicleRental = (data: CreateVehicleRentalRequest) => api.post<VehicleRental>('/vehicle-rentals', data);
export const updateVehicleRental = (id: string, data: CreateVehicleRentalRequest) => api.put<VehicleRental>(`/vehicle-rentals/${id}`, data);
export const deleteVehicleRental = (id: string) => api.delete(`/vehicle-rentals/${id}`);
export const updateRentalPayout = (id: string, payoutId: string, data: UpdateRentalPayoutRequest) =>
  api.put<VehicleRental>(`/vehicle-rentals/${id}/payouts/${payoutId}`, data);
export const markDueRentalPayoutsReceived = (id: string) => api.post<VehicleRental>(`/vehicle-rentals/${id}/mark-due-received`);
export const updateRentalBuyback = (id: string, data: UpdateRentalPayoutRequest) =>
  api.put<VehicleRental>(`/vehicle-rentals/${id}/buyback`, data);
