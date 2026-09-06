import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { PasswordStrengthIndicator } from '@/components/ui/password-strength-indicator';
import { Eye, EyeOff } from 'lucide-react';
import { useProfile } from '@/hooks/api/useProfile';

// The special-character clause must match the BACKEND validator's enumerated
// set (internal/pkg/password/validator.go: "!@#$%^&*()_+-=[]{}|;:,.<>?").
// A broader rule such as [^A-Za-z0-9] lets characters like ~ pass the UI and
// zod, then get rejected by the server after submit.
const SPECIAL_CHAR_REGEX = /[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/;

const changePasswordSchema = z.object({
  currentPassword: z.string().min(1, 'Vui lòng nhập mật khẩu hiện tại'),
  newPassword: z
    .string()
    .min(8, 'Mật khẩu phải có ít nhất 8 ký tự')
    .regex(/[A-Z]/, 'Mật khẩu phải chứa ít nhất một chữ hoa')
    .regex(/[a-z]/, 'Mật khẩu phải chứa ít nhất một chữ thường')
    .regex(/[0-9]/, 'Mật khẩu phải chứa ít nhất một số')
    .regex(SPECIAL_CHAR_REGEX, 'Mật khẩu phải chứa ít nhất một ký tự đặc biệt (!@#$%^&*()_+-=[]{}|;:,.<>?)')
    .max(72, 'Mật khẩu không được vượt quá 72 ký tự'),
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

  const isPending = changePasswordMutation.isPending;
  const newPassword = form.watch('newPassword');

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      {/* Standard admin-surface dialog: emerald header + shadcn primitives.
        The employee-portal theme (gradient header, employee-type/ct-* classes)
        must not be used here — daisyUI tokens are scoped to [data-admin-ui]
        and are undefined inside this Radix portal, so those classes render
        with fallback values (see components/ui/alert-dialog.tsx). */}
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Đổi mật khẩu</DialogTitle>
          <DialogDescription>Cập nhật mật khẩu để giữ tài khoản an toàn.</DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form id="change-password-form" onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="currentPassword"
              render={({ field }) => (
                <FormItem className="space-y-2">
                  <FormLabel>Mật khẩu hiện tại</FormLabel>
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
                        className="pr-10"
                        disabled={isPending}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                        onClick={() => setShowCurrentPassword((visible) => !visible)}
                        disabled={isPending}
                        aria-label={showCurrentPassword ? 'Ẩn mật khẩu hiện tại' : 'Hiện mật khẩu hiện tại'}
                      >
                        {showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </Button>
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
                <FormItem className="space-y-2">
                  <FormLabel>Mật khẩu mới</FormLabel>
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
                        className="pr-10"
                        disabled={isPending}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                        onClick={() => setShowNewPassword((visible) => !visible)}
                        disabled={isPending}
                        aria-label={showNewPassword ? 'Ẩn mật khẩu mới' : 'Hiện mật khẩu mới'}
                      >
                        {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </Button>
                    </div>
                  </FormControl>
                  {newPassword ? (
                    <PasswordStrengthIndicator password={newPassword} />
                  ) : (
                    <p className="text-sm text-muted-foreground">Gồm chữ hoa, chữ thường, số và ký tự đặc biệt.</p>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose} disabled={isPending}>
            Đóng
          </Button>
          <Button type="submit" form="change-password-form" disabled={isPending}>
            {isPending ? 'Đang xử lý...' : 'Đổi mật khẩu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ChangePasswordModal;
