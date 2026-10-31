import { createContext, useContext } from 'react';
import type { User } from '../api/auth';

export interface AuthContextType {
  user: User | null;
  token: string | null;
  isAdmin: boolean;
  isViewOnly: boolean;
  viewAsUserId: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  signup: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  setViewAsUser: (userId: string | null) => void;
  updateUser: (user: User) => void;
}

export const AuthContext = createContext<AuthContextType | null>(null);

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
