import { useState, useEffect, useCallback, type ReactNode } from 'react';
import { authApi, type User } from '../api/auth';
import { AuthContext } from './AuthContext';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('auth_token'));
  const [viewAsUserId, setViewAsUserId] = useState<string | null>(() => localStorage.getItem('view_as_user'));
  // Only the initial token needs verifying; logins/signups set the user directly.
  const [loading, setLoading] = useState(() => !!localStorage.getItem('auth_token'));

  const isAdmin = user?.role === 'admin';
  const isViewOnly = !!viewAsUserId;

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    authApi.getMe()
      .then(res => { if (!cancelled) setUser(res.data); })
      .catch(() => {
        if (cancelled) return;
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
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
