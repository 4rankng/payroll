import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
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
import { PasswordStrengthIndicator } from '@/components/ui/password-strength-indicator';
import { Eye, EyeOff, LockKeyhole, X } from 'lucide-react';
import { useProfile } from '@/hooks/api/useProfile';

const changePasswordSchema = z.object({
  currentPassword: z.string().min(1, 'Vui lòng nhập mật khẩu hiện tại'),
  newPassword: z
    .string()
    .min(8, 'Mật khẩu phải có ít nhất 8 ký tự')
    .regex(/[A-Z]/, 'Mật khẩu phải chứa ít nhất một chữ hoa')
    .regex(/[a-z]/, 'Mật khẩu phải chứa ít nhất một chữ thường')
    .regex(/[0-9]/, 'Mật khẩu phải chứa ít nhất một số')
    .regex(/[^A-Za-z0-9]/, 'Mật khẩu phải chứa ít nhất một ký tự đặc biệt'),
});

type ChangePasswordFormData = z.infer<typeof changePasswordSchema>;

interface ChangePasswordModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ChangePasswordModal = ({ isOpen, onClose }: ChangePasswordModalProps) => {
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);

  const { changePasswordMutation } = useProfile();

  const form = useForm<ChangePasswordFormData>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      currentPassword: '',
      newPassword: '',
    },
  });

  const handleSubmit = async (data: ChangePasswordFormData) => {
    try {
      await changePasswordMutation.mutateAsync({
        current_password: data.currentPassword,
        new_password: data.newPassword,
      });

      form.reset();
      onClose();
    } catch (error) {
      // Error handling is done by the global error handler in React Query
      // The error will be displayed automatically via showErrorNotification
    }
  };

  const handleClose = () => {
    form.reset();
    onClose();
  };

  const newPassword = form.watch('newPassword');

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent
        title="Đổi mật khẩu"
        description="Cập nhật mật khẩu để giữ tài khoản an toàn"
        hideCloseButton
        contentPadding="none"
        data-theme="congtruong"
        className="max-h-[88dvh] gap-0 overflow-hidden border-0 bg-[var(--employee-page)] p-0 shadow-[0_-10px_32px_rgba(16,24,40,0.16)] sm:max-w-md sm:rounded-2xl"
      >
        <header className="relative shrink-0 bg-gradient-to-br from-employee-800 via-employee-700 to-employee-500 px-5 pb-5 pt-[max(1rem,env(safe-area-inset-top,0px))] text-white sm:pt-5">
          <div className="mb-3 h-1 w-9 rounded-full bg-white/25 sm:hidden" />
          <div className="flex items-start gap-3 pr-11">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white/15">
              <LockKeyhole className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <DialogTitle className="employee-type-section-title text-white">Đổi mật khẩu</DialogTitle>
              <p className="employee-type-body-sm mt-1 text-white/75">Cập nhật mật khẩu để giữ tài khoản an toàn.</p>
            </div>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="inline-flex items-center justify-center rounded-full absolute right-4 top-[max(0.75rem,env(safe-area-inset-top,0px))] h-11 min-h-11 w-11 border-0 bg-white/10 p-0 text-white transition-colors hover:bg-white/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/30 sm:top-4"
            aria-label="Đóng đổi mật khẩu"
            disabled={changePasswordMutation.isPending}
          >
            <X className="h-5 w-5" />
          </button>
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
                          autoComplete="current-password"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Nhập mật khẩu hiện tại"
                          className="ct-input ct-input-bordered h-12 rounded-xl border-[var(--employee-border-strong)] bg-white pr-12 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:ring-[var(--employee-accent-ring)]"
                          disabled={changePasswordMutation.isPending}
                        />
                        <button
                          type="button"
                          className="inline-flex items-center justify-center rounded-full absolute right-0 top-0 h-12 min-h-12 w-12 p-0 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
                          onClick={() => setShowCurrentPassword((visible) => !visible)}
                          disabled={changePasswordMutation.isPending}
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
                          autoComplete="new-password"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Tối thiểu 8 ký tự"
                          className="ct-input ct-input-bordered h-12 rounded-xl border-[var(--employee-border-strong)] bg-white pr-12 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:ring-[var(--employee-accent-ring)]"
                          disabled={changePasswordMutation.isPending}
                        />
                        <button
                          type="button"
                          className="inline-flex items-center justify-center rounded-full absolute right-0 top-0 h-12 min-h-12 w-12 p-0 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
                          onClick={() => setShowNewPassword((visible) => !visible)}
                          disabled={changePasswordMutation.isPending}
                          aria-label={showNewPassword ? 'Ẩn mật khẩu mới' : 'Hiện mật khẩu mới'}
                        >
                          {showNewPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                        </button>
                      </div>
                    </FormControl>
                    {newPassword && <PasswordStrengthIndicator password={newPassword} showRequirements={false} />}
                    <p className="ct-label px-0 text-[var(--employee-text-secondary)]">Gồm chữ hoa, chữ thường, số và ký tự đặc biệt.</p>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid shrink-0 grid-cols-2 gap-2 border-t border-[var(--employee-border)] bg-white px-5 py-3 pb-[max(0.75rem,calc(0.75rem+env(safe-area-inset-bottom)))]">
              <button
                type="button"
                onClick={handleClose}
                disabled={changePasswordMutation.isPending}
                className="inline-flex items-center justify-center rounded-md min-h-11 border border-[var(--employee-border)] bg-white text-sm font-semibold normal-case text-[var(--employee-text)] shadow-none transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-50"
              >
                Đóng
              </button>
              <button
                type="submit"
                disabled={changePasswordMutation.isPending}
                className="inline-flex items-center justify-center rounded-md min-h-11 border-0 bg-[var(--employee-accent)] px-4 text-sm font-semibold normal-case text-white shadow-[var(--employee-cta-shadow)] transition-colors hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-50"
              >
                {changePasswordMutation.isPending ? 'Đang xử lý...' : 'Đổi mật khẩu'}
              </button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
