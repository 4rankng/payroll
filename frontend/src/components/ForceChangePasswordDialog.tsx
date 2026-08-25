import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { CheckCircle2, Eye, EyeOff, ShieldAlert, XCircle } from 'lucide-react';
import { useAuth } from '@/contexts';
import { useLogout } from '@/hooks/api/useAuth';
import { authService } from '@/services/api/auth.service';
import type { ChangePasswordRequest, User } from '@/types/api/auth.types';
import { showSuccessNotification } from '@/utils/error-handler';

/**
 * ForceChangePasswordDialog
 *
 * Blocks employee accounts whose current password still matches the shared
 * default (checked live via GET /auth/me — must_change_password) behind a
 * non-dismissible change-password form. Runs for any authenticated employee
 * session, not just a fresh login, so an account that was already signed in
 * before this gate shipped still gets caught on its next page load. Unlike
 * ChangePasswordModal this dialog has no close button and ignores Escape /
 * outside-click — the only way out is a successful password change, which
 * logs the user out so they re-authenticate with the new credentials.
 */
const forceChangePasswordSchema = z.object({
  currentPassword: z.string().min(1, 'Vui lòng nhập mật khẩu hiện tại'),
  newPassword: z
    .string()
    .min(8, 'Mật khẩu phải có ít nhất 8 ký tự')
    .regex(/[A-Z]/, 'Mật khẩu phải chứa ít nhất một chữ hoa')
    .regex(/[a-z]/, 'Mật khẩu phải chứa ít nhất một chữ thường')
    .regex(/[0-9]/, 'Mật khẩu phải chứa ít nhất một số')
    .regex(/[^A-Za-z0-9]/, 'Mật khẩu phải chứa ít nhất một ký tự đặc biệt'),
});

type ForceChangePasswordFormData = z.infer<typeof forceChangePasswordSchema>;

export const ForceChangePasswordDialog = () => {
  const { user, isAuthenticated } = useAuth();
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  // Hides the dialog the instant the password change succeeds, instead of
  // keeping it up for the ~3s the mutation waits before logging the user out.
  const [justChanged, setJustChanged] = useState(false);

  const isEmployee = isAuthenticated && user?.role === 'employee';

  const queryClient = useQueryClient();
  // Once /auth/me has answered "no change needed" this session, idle the
  // query: every employee /auth/me runs an argon2id compare (~64MB alloc,
  // tens of ms CPU), so refetching on every window focus would tax the auth
  // path forever. A per-app-load check is all the "catch sessions that
  // predate this gate" goal needs.
  const cachedUser = queryClient.getQueryData<User>(['currentUser']);
  const cleared = cachedUser ? !cachedUser.must_change_password : false;

  const currentUserQuery = useQuery({
    queryKey: ['currentUser'],
    queryFn: () => authService.getCurrentUser(),
    enabled: isEmployee && !cleared,
    staleTime: 0,
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  });

  // Dedicated mutation instead of useProfile()'s shared one — mounting
  // useProfile here would light up its complete-user-profile query for every
  // admin/partner session too, not just employees inside this gate.
  const logoutMutation = useLogout();
  const changePasswordMutation = useMutation({
    mutationFn: (data: ChangePasswordRequest) => authService.changePassword(data),
    onSuccess: (response) => {
      if (response?.message) {
        showSuccessNotification(response.message);
      }
      // Wait 3 seconds before logging out and redirecting to login
      setTimeout(() => logoutMutation.mutate(), 3000);
    },
  });

  const form = useForm<ForceChangePasswordFormData>({
    resolver: zodResolver(forceChangePasswordSchema),
    defaultValues: { currentPassword: '', newPassword: '' },
  });

  const mustChangePassword = isEmployee && !!currentUserQuery.data?.must_change_password && !justChanged;

  const handleSubmit = async (data: ForceChangePasswordFormData) => {
    try {
      await changePasswordMutation.mutateAsync({
        current_password: data.currentPassword,
        new_password: data.newPassword,
      });
      setJustChanged(true);
      form.reset();
    } catch {
      // Global error handler shows the toast; keep the dialog open to retry.
    }
  };

  if (!mustChangePassword) return null;

  const isPending = changePasswordMutation.isPending;
  const newPassword = form.watch('newPassword');
  // Same rules the resolver enforces, so the live indicator can never disagree
  // with what submit actually rejects.
  const newPasswordValid = forceChangePasswordSchema.shape.newPassword.safeParse(newPassword).success;

  return (
    <Dialog open>
      <DialogContent
        hideCloseButton
        contentPadding="none"
        data-theme="congtruong"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
        className="max-h-[88dvh] gap-0 overflow-hidden border-0 bg-[var(--employee-page)] p-0 shadow-[0_-10px_32px_rgba(16,24,40,0.16)] sm:max-w-md sm:rounded-2xl"
      >
        <header className="relative shrink-0 bg-gradient-to-br from-employee-800 via-employee-700 to-employee-500 px-5 pb-5 pt-[max(1rem,env(safe-area-inset-top,0px))] text-white sm:pt-5">
          <div className="mb-3 h-1 w-9 rounded-full bg-white/25 sm:hidden" />
          <div className="flex items-start gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white/15">
              <ShieldAlert className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <DialogTitle className="employee-type-section-title text-white">Đổi mật khẩu bắt buộc</DialogTitle>
              <p className="employee-type-body-sm mt-1 text-white/75">
                Tài khoản của bạn đang dùng mật khẩu mặc định. Vui lòng đặt mật khẩu mới để tiếp tục sử dụng.
              </p>
            </div>
          </div>
        </header>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 space-y-4 overflow-y-auto bg-white px-5 py-5">
              <FormField
                control={form.control}
                name="currentPassword"
                render={({ field }) => (
                  <FormItem className="ct-fieldset gap-1">
                    <FormLabel className="ct-fieldset-legend text-[var(--employee-text)]">Mật khẩu hiện tại</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          {...field}
                          type={showCurrentPassword ? 'text' : 'password'}
                          placeholder="Nhập mật khẩu hiện tại"
                          className="ct-input ct-input-bordered h-12 rounded-xl border-[var(--employee-border-strong)] bg-white pr-12 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:ring-[var(--employee-accent-ring)]"
                          disabled={isPending}
                        />
                        <button
                          type="button"
                          className="inline-flex items-center justify-center rounded-full absolute right-0 top-0 h-12 min-h-12 w-12 p-0 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
                          onClick={() => setShowCurrentPassword((visible) => !visible)}
                          disabled={isPending}
                          aria-label={showCurrentPassword ? 'Ẩn mật khẩu hiện tại' : 'Hiện mật khẩu hiện tại'}
                        >
                          {showCurrentPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                        </button>
                      </div>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="newPassword"
                render={({ field }) => (
                  <FormItem className="ct-fieldset gap-1">
                    <FormLabel className="ct-fieldset-legend text-[var(--employee-text)]">Mật khẩu mới</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          {...field}
                          type={showNewPassword ? 'text' : 'password'}
                          placeholder="Tối thiểu 8 ký tự"
                          className="ct-input ct-input-bordered h-12 rounded-xl border-[var(--employee-border-strong)] bg-white pr-12 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:ring-[var(--employee-accent-ring)]"
                          disabled={isPending}
                        />
                        <button
                          type="button"
                          className="inline-flex items-center justify-center rounded-full absolute right-0 top-0 h-12 min-h-12 w-12 p-0 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
                          onClick={() => setShowNewPassword((visible) => !visible)}
                          disabled={isPending}
                          aria-label={showNewPassword ? 'Ẩn mật khẩu mới' : 'Hiện mật khẩu mới'}
                        >
                          {showNewPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                        </button>
                      </div>
                    </FormControl>
                    {newPassword && (
                      <p
                        className={`ct-label flex items-center gap-1.5 px-0 ${
                          newPasswordValid ? 'text-[#067647]' : 'text-red-600'
                        }`}
                      >
                        {newPasswordValid ? (
                          <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
                        ) : (
                          <XCircle className="h-4 w-4" aria-hidden="true" />
                        )}
                        {newPasswordValid ? 'Mật khẩu hợp lệ' : 'Mật khẩu chưa hợp lệ'}
                      </p>
                    )}
                    <p className="ct-label px-0 text-[var(--employee-text-secondary)]">Gồm chữ hoa, chữ thường, số và ký tự đặc biệt.</p>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="shrink-0 border-t border-[var(--employee-border)] bg-white px-5 py-3 pb-[max(0.75rem,calc(0.75rem+env(safe-area-inset-bottom)))]">
              <button
                type="submit"
                disabled={isPending}
                className="inline-flex w-full items-center justify-center rounded-md min-h-11 border-0 bg-[var(--employee-accent)] px-4 text-sm font-semibold normal-case text-white shadow-[var(--employee-cta-shadow)] transition-colors hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-50"
              >
                {isPending ? 'Đang xử lý...' : 'Đổi mật khẩu'}
              </button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};

export default ForceChangePasswordDialog;
