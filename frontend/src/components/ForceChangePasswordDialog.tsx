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
import { CheckCircle2, Circle, Eye, EyeOff, ShieldAlert } from 'lucide-react';
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
// The special-character clause must match the BACKEND validator's enumerated
// set (internal/pkg/password/validator.go: "!@#$%^&*()_+-=[]{}|;:,.<>?").
// A broader rule such as [^A-Za-z0-9] lets characters like ~ pass the UI and
// zod, then get rejected by the server after submit.
const SPECIAL_CHAR_REGEX = /[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/;

const forceChangePasswordSchema = z.object({
  currentPassword: z.string().min(1, 'Vui lòng nhập mật khẩu hiện tại'),
  newPassword: z
    .string()
    .min(8, 'Mật khẩu phải có ít nhất 8 ký tự')
    .regex(/[A-Z]/, 'Mật khẩu phải chứa ít nhất một chữ hoa')
    .regex(/[a-z]/, 'Mật khẩu phải chứa ít nhất một chữ thường')
    .regex(/[0-9]/, 'Mật khẩu phải chứa ít nhất một số')
    .regex(SPECIAL_CHAR_REGEX, 'Mật khẩu phải chứa ít nhất một ký tự đặc biệt (!@#$%^&*()_+-=[]{}|;:,.<>?)'),
});

type ForceChangePasswordFormData = z.infer<typeof forceChangePasswordSchema>;

/**
 * Live requirement checklist shown under the new-password field. Each rule
 * mirrors one clause of forceChangePasswordSchema so the ticks can never
 * disagree with what submit rejects — and unlike a single valid/invalid line
 * it tells the user which clause is still missing.
 */
const PASSWORD_RULES: ReadonlyArray<{ label: string; test: (value: string) => boolean }> = [
  { label: '8 ký tự trở lên', test: (value) => value.length >= 8 },
  { label: 'Chữ hoa', test: (value) => /[A-Z]/.test(value) },
  { label: 'Chữ thường', test: (value) => /[a-z]/.test(value) },
  { label: 'Chữ số', test: (value) => /[0-9]/.test(value) },
  { label: 'Ký tự đặc biệt', test: (value) => SPECIAL_CHAR_REGEX.test(value) },
];

const passwordFieldClass =
  'h-12 w-full rounded-xl border border-[var(--employee-border-strong)] bg-white px-3.5 pr-12 text-[0.9375rem] leading-normal text-[var(--employee-text)] transition-colors placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[var(--employee-accent-ring)] disabled:cursor-not-allowed disabled:opacity-60';

const revealButtonClass =
  'absolute inset-y-1 right-1 inline-flex w-10 items-center justify-center rounded-lg text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-surface-muted)] hover:text-[var(--employee-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:opacity-60';

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
  // skipApiCall: /auth/change-password blacklists the session token server-side
  // on success, so the follow-up POST /auth/logout would deterministically 401.
  const logoutMutation = useLogout({ skipApiCall: true });
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
  const newPassword = form.watch('newPassword') ?? '';
  // Rules stay unmet-but-neutral until the user has actually tried to submit;
  // turning the whole list red while someone is still typing reads as failure.
  const showUnmetAsError = !!form.formState.errors.newPassword;

  return (
    <Dialog open>
      <DialogContent
        hideCloseButton
        contentPadding="none"
        data-theme="congtruong"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
        className="max-h-[88dvh] gap-0 overflow-hidden border-0 bg-white p-0 shadow-[0_16px_48px_rgba(16,24,40,0.20)] sm:max-w-[26rem] sm:rounded-2xl"
      >
        <header className="shrink-0 bg-gradient-to-b from-employee-800 to-employee-700 px-5 pb-5 pt-[max(1.25rem,env(safe-area-inset-top,0px))] text-white">
          <span className="mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-white/12 ring-1 ring-inset ring-white/20">
            <ShieldAlert className="h-[1.125rem] w-[1.125rem]" aria-hidden="true" />
          </span>
          <DialogTitle className="employee-type-strong text-white">Đổi mật khẩu bắt buộc</DialogTitle>
          <p className="employee-type-body-sm mt-1.5 text-white/70">
            Tài khoản của bạn đang dùng mật khẩu mặc định. Vui lòng đặt mật khẩu mới để tiếp tục sử dụng.
          </p>
        </header>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-5 py-5">
              <FormField
                control={form.control}
                name="currentPassword"
                render={({ field }) => (
                  <FormItem className="space-y-1.5">
                    <FormLabel className="employee-type-label text-[var(--employee-text)]">Mật khẩu hiện tại</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          {...field}
                          type={showCurrentPassword ? 'text' : 'password'}
                          autoComplete="current-password"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Nhập mật khẩu hiện tại"
                          className={passwordFieldClass}
                          disabled={isPending}
                        />
                        <button
                          type="button"
                          className={revealButtonClass}
                          onClick={() => setShowCurrentPassword((visible) => !visible)}
                          disabled={isPending}
                          aria-label={showCurrentPassword ? 'Ẩn mật khẩu hiện tại' : 'Hiện mật khẩu hiện tại'}
                        >
                          {showCurrentPassword ? <EyeOff className="h-[1.125rem] w-[1.125rem]" /> : <Eye className="h-[1.125rem] w-[1.125rem]" />}
                        </button>
                      </div>
                    </FormControl>
                    <FormMessage className="employee-type-body-sm" />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="newPassword"
                render={({ field }) => (
                  <FormItem className="space-y-1.5">
                    <FormLabel className="employee-type-label text-[var(--employee-text)]">Mật khẩu mới</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          {...field}
                          type={showNewPassword ? 'text' : 'password'}
                          autoComplete="new-password"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Nhập mật khẩu mới"
                          className={passwordFieldClass}
                          disabled={isPending}
                        />
                        <button
                          type="button"
                          className={revealButtonClass}
                          onClick={() => setShowNewPassword((visible) => !visible)}
                          disabled={isPending}
                          aria-label={showNewPassword ? 'Ẩn mật khẩu mới' : 'Hiện mật khẩu mới'}
                        >
                          {showNewPassword ? <EyeOff className="h-[1.125rem] w-[1.125rem]" /> : <Eye className="h-[1.125rem] w-[1.125rem]" />}
                        </button>
                      </div>
                    </FormControl>
                    <ul className="grid grid-cols-2 gap-x-3 gap-y-1.5 pt-1.5">
                      {PASSWORD_RULES.map((rule) => {
                        const met = rule.test(newPassword);
                        return (
                          <li
                            key={rule.label}
                            className={`employee-type-body-sm flex items-center gap-1.5 ${
                              met
                                ? 'text-[var(--employee-accent)]'
                                : showUnmetAsError
                                  ? 'text-[var(--employee-error)]'
                                  : 'text-[var(--employee-text-secondary)]'
                            }`}
                          >
                            {met ? (
                              <CheckCircle2 className="h-4 w-4 shrink-0" aria-hidden="true" />
                            ) : (
                              <Circle className="h-4 w-4 shrink-0 opacity-60" aria-hidden="true" />
                            )}
                            <span className="min-w-0 truncate">{rule.label}</span>
                          </li>
                        );
                      })}
                    </ul>
                  </FormItem>
                )}
              />
            </div>

            <div className="shrink-0 border-t border-[var(--employee-border)] px-5 py-4 pb-[max(1rem,calc(1rem+env(safe-area-inset-bottom)))]">
              <button
                type="submit"
                disabled={isPending}
                className="employee-type-action inline-flex h-12 w-full items-center justify-center rounded-xl bg-[var(--employee-accent)] px-4 text-white shadow-[var(--employee-cta-shadow)] transition-colors hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-60"
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
