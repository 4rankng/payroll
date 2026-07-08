import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
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
import { Button } from '@/components/ui/button';
import { useAuth } from '@/contexts';
import { authService } from '@/services/api/auth.service';
import { showSuccessNotification } from '@/utils/error-handler';

/**
 * EmailPromptGate
 *
 * Nudges authenticated admin/partner/adv_partner users who have no email on
 * file to provide one. OTP 2FA requires an email to deliver codes; this gate
 * runs while OTP is disabled so the population of no-email accounts is reduced
 * before OTP is turned on. (When OTP is enabled, a no-email admin/partner is
 * already blocked at login by ErrOTPRequiredMissingEmail and never reaches the
 * dashboard, so this gate is naturally inert for them.)
 *
 * The dialog is dismissable ("Nhắc tôi sau"): dismissal is recorded in
 * sessionStorage keyed by user id, so it survives reloads within the same
 * browser session and re-prompts on a new browser session. Keys off the
 * authenticated user state so it fires after direct login, Google login,
 * OTP-verify, and page reload alike.
 */
const emailPromptSchema = z.object({
  email: z.string().min(1, 'Email không được để trống').email('Email không hợp lệ'),
});

type EmailPromptFormData = z.infer<typeof emailPromptSchema>;

const dismissKey = (userId: string) => `emailPromptDismissed:${userId}`;

export const EmailPromptGate = () => {
  const { user, isAuthenticated, updateUser } = useAuth();
  const [dismissed, setDismissed] = useState(false);

  const userId = user?.id;
  const role = user?.role;
  const needsEmail =
    isAuthenticated &&
    !!user &&
    !user.email &&
    (role === 'admin' || role === 'partner' || role === 'adv_partner');

  // Sync dismissal from sessionStorage whenever the authenticated user changes.
  useEffect(() => {
    if (userId) {
      setDismissed(sessionStorage.getItem(dismissKey(userId)) === '1');
    } else {
      setDismissed(false);
    }
  }, [userId]);

  const updateEmailMutation = useMutation({
    mutationFn: (data: { email: string }) => authService.updateProfile(data),
    onSuccess: (response) => {
      showSuccessNotification(response?.message ?? 'Đã cập nhật email.');
    },
  });

  const form = useForm<EmailPromptFormData>({
    resolver: zodResolver(emailPromptSchema),
    defaultValues: { email: '' },
  });

  const handleDismiss = () => {
    if (userId) sessionStorage.setItem(dismissKey(userId), '1');
    setDismissed(true);
  };

  const handleSubmit = async (data: EmailPromptFormData) => {
    try {
      await updateEmailMutation.mutateAsync({ email: data.email });
      // Sync AuthContext so the gate closes immediately. authService.updateProfile
      // already persisted localStorage('userEmail').
      updateUser({ email: data.email });
      form.reset({ email: '' });
    } catch {
      // Error toast is shown by the global MutationCache onError; keep the
      // dialog open so the user can correct and retry.
    }
  };

  if (!needsEmail) return null;

  const isPending = updateEmailMutation.isPending;

  return (
    <Dialog open={!dismissed} onOpenChange={(open) => { if (!open) handleDismiss(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Cập nhật địa chỉ email</DialogTitle>
          <DialogDescription>
            Tài khoản của bạn chưa có email. Email cần thiết để bật xác thực hai bước (OTP) và nhận các thông báo quan trọng. Vui lòng cung cấp email để tiếp tục.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type="email"
                      placeholder="nhập email của bạn"
                      autoComplete="email"
                      disabled={isPending}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={handleDismiss}
                disabled={isPending}
              >
                Nhắc tôi sau
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? 'Đang lưu...' : 'Lưu'}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};

export default EmailPromptGate;
