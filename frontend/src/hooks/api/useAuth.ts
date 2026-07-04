import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { authService } from '@/services/api/auth.service';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';
import { useAuth } from '@/contexts';
import { generateAvatarUrl } from '@/utils/avatarHelpers';
import type { LoginCredentials, ChangePasswordData } from '@/types/api/auth.types';

export const useLogin = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { login } = useAuth();

  return useMutation({
    mutationFn: (credentials: LoginCredentials) => authService.login(credentials),
    meta: { skipGlobalError: true },
    onSuccess: (response) => {
      const data = response.data!;

      // RT-C3: when the email-OTP second factor is required, do NOT call
      // login() or navigate to the dashboard. The pending session id is
      // already stored under a distinct sessionStorage key by authService.login
      // (NOT auth_token). Route to the OTP entry screen.
      if (data.otp_required) {
        navigate('/login/otp');
        return;
      }

      // Clear stale query cache from previous sessions before starting new one
      queryClient.clear();
      sessionStorage.removeItem('payroll-query-cache');

      // Store token and user data in AuthContext
      login(data.access_token, {
        id: String(data.user.id),
        email: data.user.email,
        name: data.user.fullname,
        username: data.user.username,
        role: data.user.role,
        avatar: generateAvatarUrl(data.user.username)
      });

      // Navigate based on role
      if (data.user.role === 'admin') {

        navigate('/admin');
      } else if (data.user.role === 'partner') {

        navigate('/partner/dashboard');
      } else if (data.user.role === 'employee') {

        navigate('/employee');
      } else {

        navigate('/');
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

export const useGoogleLogin = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { login } = useAuth();

  return useMutation({
    mutationFn: (idToken: string) => authService.loginWithGoogle(idToken),
    meta: { skipGlobalError: true },
    onSuccess: (response) => {
      const data = response.data!;

      // RT-C3: same OTP gate as password login (RT-H5 — Google login can also
      // require the second factor for admin/partner).
      if (data.otp_required) {
        navigate('/login/otp');
        return;
      }

      // Clear stale query cache from previous sessions before starting new one
      queryClient.clear();
      sessionStorage.removeItem('payroll-query-cache');

      // Store token and user data in AuthContext
      login(data.access_token, {
        id: String(data.user.id),
        email: data.user.email,
        name: data.user.fullname,
        username: data.user.username,
        role: data.user.role,
        avatar: generateAvatarUrl(data.user.username)
      });

      // Navigate based on role
      if (data.user.role === 'admin') {
        navigate('/admin');
      } else if (data.user.role === 'partner') {
        navigate('/partner/dashboard');
      } else if (data.user.role === 'employee') {
        navigate('/employee');
      } else {
        navigate('/');
      }
    },
  });
};

export const useLogout = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { logout } = useAuth();

  return useMutation({
    mutationFn: () => authService.logout(),
    onSuccess: (response) => {
      // Clear all cached data
      queryClient.clear();

      // Clear AuthContext
      logout();

      if (response.message) {
        showSuccessNotification(response.message);
      }

      navigate('/login');
    },
    onError: (error: unknown) => {
      // Even if logout fails, clear local data and redirect
      queryClient.clear();
      logout();
      navigate('/login');
      // Don't show error notification for logout failures
    },
  });
};

export const useCurrentUser = () => {
  return useQuery({
    queryKey: ['currentUser'],
    queryFn: () => authService.getCurrentUser(),
    enabled: authService.isAuthenticated(),
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useChangePassword = () => {

  return useMutation({
    mutationFn: (data: ChangePasswordData) => authService.changePassword(data),
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Check if user is authenticated
export const useIsAuthenticated = () => {
  return authService.isAuthenticated();
};

// Get user role
export const useUserRole = () => {
  return authService.getUserRole();
};

// Check if user has specific role
export const useHasRole = (role: 'admin' | 'partner') => {
  return authService.hasRole(role);
};

// Check if user has unknown of the roles
export const useHasAnyRole = (roles: ('admin' | 'partner')[]) => {
  return authService.hasAnyRole(roles);
};
