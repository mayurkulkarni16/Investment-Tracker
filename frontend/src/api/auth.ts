import api from './client';

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export const authApi = {
  signup: (data: { name: string; email: string; password: string }) =>
    api.post<AuthResponse>('/auth/signup', data),

  login: (data: { email: string; password: string }) =>
    api.post<AuthResponse>('/auth/login', data),

  forgotPassword: (data: { email: string }) =>
    api.post('/auth/forgot-password', data),

  resetPassword: (data: { token: string; password: string }) =>
    api.post('/auth/reset-password', data),

  getMe: () => api.get<User>('/auth/me'),

  updateMe: (data: { name: string }) => api.put<User>('/auth/me', data),

  changePassword: (data: { current_password: string; new_password: string }) =>
    api.put('/auth/change-password', data),

  getUsers: () => api.get<User[]>('/auth/users'),
};
