import api from './client';
import type { Profile, CreateProfileRequest, UpdateProfileRequest } from '../types';

export const getProfiles = () => api.get<Profile[]>('/profiles');
export const createProfile = (data: CreateProfileRequest) => api.post<Profile>('/profiles', data);
export const updateProfile = (id: string, data: UpdateProfileRequest) => api.put<Profile>(`/profiles/${id}`, data);
export const deleteProfile = (id: string) => api.delete(`/profiles/${id}`);
