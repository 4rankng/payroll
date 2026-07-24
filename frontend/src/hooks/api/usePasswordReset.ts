import { useMutation } from '@tanstack/react-query';
import { authService } from '@/services/api/auth.service';

export interface PasswordResetConfirmPayload {
  token: string;
  new_password: string;
}

/**
 * useRequestPasswordReset — request a magic-link reset email.
 *
 * `skipGlobalError` is set because the page handles the anti-enumeration UX
 * itself: the success state is shown regardless of outcome, and the global
 * toast would only add noise (the backend always returns 200).
 */
export const useRequestPasswordReset = () =>
  useMutation({
    mutationFn: (email: string) => authService.requestPasswordReset(email),
    meta: { skipGlobalError: true },
  });

/**
 * useConfirmPasswordReset — set a new password with a single-use token.
 *
 * `skipGlobalError` is set so the page can render a tailored expired-link /
 * invalid-token message inline rather than a generic global toast.
 */
export const useConfirmPasswordReset = () =>
  useMutation({
    mutationFn: (payload: PasswordResetConfirmPayload) => authService.confirmPasswordReset(payload),
    meta: { skipGlobalError: true },
  });
