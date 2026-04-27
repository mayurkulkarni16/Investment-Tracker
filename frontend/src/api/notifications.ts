import api from './client';
import type { Notification } from '../types';

export const getNotifications = () => api.get<Notification[]>('/notifications');
export const getUnreadNotifications = () => api.get<Notification[]>('/notifications/unread');
export const generateNotifications = () => api.post<Notification[]>('/notifications/generate');
export const markNotificationRead = (id: string) => api.put(`/notifications/${id}/read`);
export const markAllNotificationsRead = () => api.put('/notifications/read-all');
