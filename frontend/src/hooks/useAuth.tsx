import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { api, type User } from '../api';

type AuthState = {
  user: User | null;
  couponCount: number;
  loading: boolean;
  isAuthenticated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

const USER_KEY = 'user';

function readStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(readStoredUser);
  const [couponCount, setCouponCount] = useState(0);
  const [loading, setLoading] = useState(Boolean(localStorage.getItem('token')));

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    localStorage.removeItem(USER_KEY);
    setUser(null);
    setCouponCount(0);
  }, []);

  const refresh = useCallback(async () => {
    const token = localStorage.getItem('token');
    if (!token) {
      logout();
      return;
    }
    try {
      const profile = await api.getMe();
      setUser(profile.user);
      setCouponCount(profile.coupon_count);
      localStorage.setItem(USER_KEY, JSON.stringify(profile.user));
    } catch {
      logout();
    }
  }, [logout]);

  useEffect(() => {
    if (!localStorage.getItem('token')) {
      setLoading(false);
      return;
    }
    refresh().finally(() => setLoading(false));
  }, [refresh]);

  const login = useCallback((token: string, nextUser: User) => {
    localStorage.setItem('token', token);
    localStorage.setItem(USER_KEY, JSON.stringify(nextUser));
    setUser(nextUser);
    api.getMe().then((profile) => {
      setUser(profile.user);
      setCouponCount(profile.coupon_count);
      localStorage.setItem(USER_KEY, JSON.stringify(profile.user));
    }).catch(() => {});
  }, []);

  const value = useMemo(
    () => ({
      user,
      couponCount,
      loading,
      isAuthenticated: Boolean(user && localStorage.getItem('token')),
      login,
      logout,
      refresh,
    }),
    [user, couponCount, loading, login, logout, refresh]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return ctx;
}

export function userInitials(user: User) {
  const source = user.display_name?.trim() || user.email;
  const parts = source.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return source.slice(0, 2).toUpperCase();
}

export function userLabel(user: User) {
  return user.display_name?.trim() || user.email.split('@')[0];
}
