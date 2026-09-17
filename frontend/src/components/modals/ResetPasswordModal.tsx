import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordStrengthIndicator } from "@/components/ui/password-strength-indicator";
import { Eye, EyeOff } from "lucide-react";

export const modalConfig = {
  id: 'reset-password',
};

type TargetType = 'user' | 'employee';

interface TargetInfo {
  id: number;
  fullname: string;
}

interface ResetPasswordModalProps {
  open: boolean;
  onClose: () => void;
  onResetPassword: (id: number, password: string) => void;
  target: TargetInfo | null;
  targetType?: TargetType;
  loading?: boolean;
}

const TARGET_LABELS: Record<TargetType, { noun: string; footer: string }> = {
  user: { noun: 'người dùng', footer: 'Người dùng sẽ phải sử dụng mật khẩu mới này để đăng nhập.' },
  employee: { noun: 'nhân viên', footer: 'Nhân viên sẽ phải sử dụng mật khẩu mới này để đăng nhập.' },
};

export function ResetPasswordModal({
  open,
  onClose,
  onResetPassword,
  target,
  targetType = 'user',
  loading = false
}: ResetPasswordModalProps) {
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!password.trim()) {
      newErrors.password = "Mật khẩu là bắt buộc";
    } else {
      const passwordErrors = [];

      if (password.length < 8) {
        passwordErrors.push("ít nhất 8 ký tự");
      }
      if (!/(?=.*[a-z])/.test(password)) {
        passwordErrors.push("ít nhất 1 chữ thường");
      }
      if (!/(?=.*[A-Z])/.test(password)) {
        passwordErrors.push("ít nhất 1 chữ hoa");
      }
      if (!/(?=.*\d)/.test(password)) {
        passwordErrors.push("ít nhất 1 số");
      }
      if (!/(?=.*[!@#$%^&*(),.?":{}|<>])/.test(password)) {
        passwordErrors.push("ít nhất 1 ký tự đặc biệt");
      }

      if (passwordErrors.length > 0) {
        newErrors.password = `Mật khẩu phải có ${passwordErrors.join(", ")}`;
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (!validateForm() || !target) return;

    onResetPassword(target.id, password);
    handleClose();
  };

  const handleClose = () => {
    setPassword("");
    setShowPassword(false);
    setErrors({});
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Đặt lại mật khẩu </DialogTitle>
          <DialogDescription>
            Đặt lại mật khẩu cho {TARGET_LABELS[targetType].noun} <strong>{target?.fullname}</strong>
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="new-password">Mật khẩu mới *</Label>
            <div className="relative">
              <Input
                id="new-password"
                type={showPassword ? "text" : "password"}
                placeholder="Nhập mật khẩu mới"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={errors.password ? "border-destructive pr-10" : "pr-10"}
                disabled={loading}
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                onClick={() => setShowPassword(!showPassword)}
                disabled={loading}
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
            {errors.password && <p className="typography-body-medium text-destructive">{errors.password}</p>}
            <p className="text-sm text-muted-foreground">
              Mật khẩu là bắt buộc
            </p>
          </div>

          <div className="bg-muted/50 rounded-xl p-3 space-y-2">
            <h4 className="typography-body-medium">Yêu cầu mật khẩu</h4>
            {password ? (
              <PasswordStrengthIndicator password={password} showRequirements={true} />
            ) : (
              <div className="typography-body-medium text-muted-foreground space-y-1">
                <p>Mật khẩu phải có:</p>
                <ul className="list-disc list-inside space-y-0.5 ml-2">
                  <li>Ít nhất 8 ký tự</li>
                  <li>Ít nhất 1 chữ thường (a-z)</li>
                  <li>Ít nhất 1 chữ hoa (A-Z)</li>
                  <li>Ít nhất 1 số (0-9)</li>
                  <li>Ít nhất 1 ký tự đặc biệt (!@#$%^&*)</li>
                </ul>
              </div>
            )}
            <p className="typography-body-medium text-muted-foreground mt-2">
              {TARGET_LABELS[targetType].footer}
            </p>
          </div>
        </div>

        <DialogFooter>
          <button
            onClick={handleClose}
            disabled={loading}
            className="inline-flex items-center gap-1.5 h-11 px-3 sm:h-8 rounded border border-border bg-background text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            Đóng
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            className="inline-flex items-center gap-1.5 h-11 px-3 sm:h-8 rounded bg-amber-500 text-white text-sm font-medium whitespace-nowrap hover:bg-amber-600 transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {loading ? "Đang reset..." : "Đổi mật khẩu"}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
