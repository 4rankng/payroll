import { useMutation, useQuery } from '@tanstack/react-query';
import { authService } from '@/services/api/auth.service';
import { ChangePasswordRequest, CompleteUserProfile, PasswordStrengthResponse } from '@/types/api/auth.types';
import { showSuccessNotification } from '@/utils/error-handler';
import { useLogout } from '@/hooks/api/useAuth';

export const useProfile = () => {
  const logoutMutation = useLogout();

  const getCompleteUserQuery = useQuery({
    queryKey: ['profile', 'complete-user'],
    queryFn: () => authService.getCompleteUserProfile(),
  });

  const updateProfileMutation = useMutation({
    mutationFn: (data: { fullname?: string; email?: string; cccd?: string; mobile?: string }) => authService.updateProfile(data),
    onSuccess: (response) => {
      if (response?.message) {
        showSuccessNotification(response.message);
      }
      getCompleteUserQuery.refetch();
    },
  });

  const changePasswordMutation = useMutation({
    mutationFn: (data: ChangePasswordRequest) => authService.changePassword(data),
    onSuccess: (response) => {
      if (response?.message) {
        showSuccessNotification(response.message);
      }
      // Wait 3 seconds before logging out and redirecting to login
      setTimeout(() => {
        logoutMutation.mutate();
      }, 3000);
    },
  });

  const passwordStrengthMutation = useMutation({
    mutationFn: (password: string) => authService.checkPasswordStrength(password),
  });

  return {
    completeUser: getCompleteUserQuery.data,
    isLoadingCompleteUser: getCompleteUserQuery.isLoading,
    completeUserError: getCompleteUserQuery.error,

    updateProfileMutation,
    changePasswordMutation,
    passwordStrengthMutation,

    refetchProfile: getCompleteUserQuery.refetch,
  };
};