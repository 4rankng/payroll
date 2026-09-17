import { useMutation } from '@tanstack/react-query';
import { authService } from '@/services/api/auth.service';

/**
 * useRequestZaloReset — request a Zalo OTP reset.
 *
 * `skipGlobalError` is set because the page handles the anti-enumeration UX
 * itself: the "code sent" state is shown regardless of outcome, and a global
 * toast would only add noise (the backend always returns 200 + a session id).
 */
export const useRequestZaloReset = () =>
  useMutation({
    mutationFn: (mobile: string) => authService.requestZaloReset(mobile),
    meta: { skipGlobalError: true },
  });

/**
 * useConfirmZaloReset — set a new password with the OTP session + code.
 *
 * `skipGlobalError` is set so the page can render a tailored wrong-code /
 * expired-session message inline rather than a generic global toast.
 */
export const useConfirmZaloReset = () =>
  useMutation({
    mutationFn: (payload: Parameters<typeof authService.confirmZaloReset>[0]) => authService.confirmZaloReset(payload),
    meta: { skipGlobalError: true },
  });
