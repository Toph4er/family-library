import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { api, initCSRF } from '../services/api';
import type { SessionUser } from '../types/user';

interface AuthContextValue {
  user: SessionUser | null;
  isAdmin: boolean;
  loading: boolean;
  refreshUser: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  isAdmin: false,
  loading: true,
  refreshUser: async () => {},
  logout: async () => {},
});

export const useAuth = () => useContext(AuthContext);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshUser = useCallback(async () => {
    try {
      const response = await api.me();
      setUser(response.data.user);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      // Ignore errors — cookie will be cleared server-side regardless
    }
    setUser(null);
  }, []);

  useEffect(() => {
    // Initialize CSRF token before any authenticated requests.
    initCSRF().then(() => {
      refreshUser();
    });
  }, [refreshUser]);

  const isAdmin = user?.role === 'admin';

  return (
    <AuthContext.Provider value={{ user, isAdmin, loading, refreshUser, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
