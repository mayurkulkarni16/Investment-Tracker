import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { authApi, type User } from '../api/auth';

interface AuthContextType {
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

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('auth_token'));
  const [viewAsUserId, setViewAsUserId] = useState<string | null>(localStorage.getItem('view_as_user'));
  const [loading, setLoading] = useState(true);

  const isAdmin = user?.role === 'admin';
  const isViewOnly = !!viewAsUserId;

  useEffect(() => {
    if (token) {
      authApi.getMe()
        .then(res => setUser(res.data))
        .catch(() => {
          localStorage.removeItem('auth_token');
          setToken(null);
          setUser(null);
        })
        .finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [token]);

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login({ email, password });
    localStorage.setItem('auth_token', res.data.token);
    setToken(res.data.token);
    setUser(res.data.user);
  }, []);

  const signup = useCallback(async (name: string, email: string, password: string) => {
    const res = await authApi.signup({ name, email, password });
    localStorage.setItem('auth_token', res.data.token);
    setToken(res.data.token);
    setUser(res.data.user);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('view_as_user');
    setToken(null);
    setUser(null);
    setViewAsUserId(null);
  }, []);

  const setViewAsUser = useCallback((userId: string | null) => {
    if (userId) {
      localStorage.setItem('view_as_user', userId);
    } else {
      localStorage.removeItem('view_as_user');
    }
    setViewAsUserId(userId);
  }, []);

  const updateUser = useCallback((u: User) => setUser(u), []);

  return (
    <AuthContext.Provider value={{
      user, token, isAdmin, isViewOnly, viewAsUserId, loading,
      login, signup, logout, setViewAsUser, updateUser,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
