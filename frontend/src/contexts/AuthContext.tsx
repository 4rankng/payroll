import React, { createContext, useContext, useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { authManager, type AppRole } from '@/lib/auth';
import { authService } from '@/services/api/auth.service';
import { generateAvatarUrl } from '@/utils/avatarHelpers';

interface User {
  id: string;
  email: string;
  name: string;
  username: string;
  role: AppRole;
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
  login: (user: User) => void;
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
      const userRole = authManager.getUserRole();

      if (userRole && authManager.hasSession()) {
        // Restore the session from the stored profile (display data only —
        // the credential is the server-managed HttpOnly cookie, PAYR-37B).
        const username = localStorage.getItem('userUsername') || '';
        const userData: User = {
          id: String(authManager.getUserId() ?? ''),
          email: localStorage.getItem('userEmail') || '',
          name: localStorage.getItem('userName') || getDisplayName(username, userRole),
          username,
          role: userRole,
          status: (localStorage.getItem('userStatus') as 'active' | 'inactive') || 'active',
          avatar: generateAvatarUrl(),
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

    // One-time migration (PAYR-37B): a session created by the pre-cookie
    // bundle still holds a legacy localStorage token. Exchange it for the
    // HttpOnly cookie via /auth/me, then re-evaluate the auth state. On any
    // failure the stale token is discarded and the user simply re-logs in.
    authService
      .bootstrapLegacySession()
      .then((converted) => {
        if (converted) initializeAuth();
      })
      .catch(() => {
        // Network error during bootstrap — keep whatever state we resolved.
      });

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

  // Session state (cookie + expiry metadata) is managed by auth.service via
  // authManager.setSessionMeta; the context only tracks the profile.
  const login = (userData: User) => {
    setUser(userData);
  };

  const logout = () => {
    authManager.clearSession();
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
    isAuthenticated: !!user,
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
const getDisplayName = (username: string, role: AppRole): string => {
  if (!username) {
    if (role === 'admin') return 'Quản trị viên';
    if (role === 'partner') return 'Quản lý';
    if (role === 'adv_partner') return 'Quản lý ứng lương';
    if (role === 'accountant') return 'Kế toán';
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
  if (role === 'accountant') return 'Kế toán';
  return 'Nhân viên';
};
