import React, { createContext, useContext, useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { authManager } from '@/lib/auth';
import { generateAvatarUrl } from '@/utils/avatarHelpers';

interface User {
  id: string;
  email: string;
  name: string;
  username: string;
  role: 'admin' | 'partner' | 'employee' | 'adv_partner';
  status?: 'active' | 'inactive';
  avatar?: string;
  created_at?: string;
  updated_at?: string;
  last_login?: string;
}

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  updateUser: (updates: Partial<User>) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

interface AuthProviderProps {
  children: React.ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const initializeAuth = () => {
      const token = authManager.getToken();
      const payload = authManager.getTokenPayload();

      if (token && payload && authManager.isTokenValid()) {
        // Extract user info from JWT payload and stored data
        const userData: User = {
          id: String(payload.user_id),
          email: localStorage.getItem('userEmail') || '',
          name: localStorage.getItem('userName') || getDisplayName(payload.username, payload.role),
          username: payload.username,
          role: payload.role,
          status: (localStorage.getItem('userStatus') as 'active' | 'inactive') || 'active',
          avatar: generateAvatarUrl(payload.username),
          created_at: localStorage.getItem('userCreatedAt') || undefined,
          updated_at: localStorage.getItem('userUpdatedAt') || undefined,
          last_login: localStorage.getItem('userLastLogin') || undefined,
        };

        setUser(userData);
        authManager.startSessionMonitoring();
      } else {
        setUser(null);
        queryClient.clear();
        sessionStorage.removeItem('payroll-query-cache');
      }

      setIsLoading(false);
    };

    initializeAuth();

    const handleAuthChanged = () => {
      initializeAuth();
    };

    window.addEventListener('auth-changed', handleAuthChanged);

    // BUG-012 fix: Listen for auth changes from other tabs/windows
    // sessionStorage doesn't trigger StorageEvent, so use a custom event via BroadcastChannel
    const channel = 'BroadcastChannel' in window ? new BroadcastChannel('auth-channel') : null;
    if (channel) {
      channel.onmessage = handleAuthChanged;
    }

    return () => {
      window.removeEventListener('auth-changed', handleAuthChanged);
      channel?.close();
    };
  }, [queryClient]);

  const login = (token: string, userData: User) => {
    authManager.setToken(token);
    setUser(userData);
  };

  const logout = () => {
    authManager.removeToken();
    setUser(null);
    queryClient.clear();
    sessionStorage.removeItem('payroll-query-cache');
    // Navigation is handled by ProtectedRoute's useEffect via React Router's navigate(),
    // avoiding the blank white screen that window.location.href causes between
    // setUser(null) re-render and the browser completing the hard redirect.
  };

  const updateUser = (updates: Partial<User>) => {
    if (user) {
      setUser({ ...user, ...updates });
    }
  };

  const value: AuthContextValue = {
    user,
    isAuthenticated: !!user && authManager.isTokenValid(),
    isLoading,
    login,
    logout,
    updateUser
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextValue => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

// Helper functions
const getDisplayName = (username: string, role: 'admin' | 'partner' | 'employee' | 'adv_partner'): string => {
  if (!username) {
    if (role === 'admin') return 'Quản trị viên';
    if (role === 'partner') return 'Quản lý';
    if (role === 'adv_partner') return 'Quản lý ứng lương';
    return 'Nhân viên';
  }

  const capitalizedName = username
    .split(/[._-]/)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');

  if (capitalizedName) return capitalizedName;

  if (role === 'admin') return 'Quản trị viên';
  if (role === 'partner') return 'Quản lý';
  if (role === 'adv_partner') return 'Quản lý ứng lương';
  return 'Nhân viên';
};
