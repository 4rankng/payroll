import { useState, useEffect, useCallback, useMemo } from "react";
import { Link } from "react-router-dom";
import { AlertCircle, ArrowLeft, CheckCircle, Eye, EyeOff, Lock01 } from "@untitledui/icons";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConfirmPasswordReset } from "@/hooks/api/usePasswordReset";
import { PasswordStrengthIndicator } from "@/components/ui/password-strength-indicator";
import { AuthBrandMark } from "@/components/auth/AuthBrandMark";
import { AuthCard } from "@/components/auth/AuthCard";
import { AuthSpinner } from "@/components/auth/AuthSpinner";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";
import { cx } from "@/utils/cx";

/**
 * ResetPassword — step 2 of the self-service password-reset flow.
 *
 * Reached via the magic link `/reset-password?token=...`. Red Team H3: the
 * token is read on mount then STRIPPED from the URL via replaceState so it
 * doesn't linger in browser history or leak via the Referer header on
 * sub-resource loads. A `Referrer-Policy: no-referrer` meta is also applied
 * for the page lifetime as defense in depth.
 */
const ResetPassword = () => {
  // Read the token exactly once from the initial URL.
  const initialToken = useMemo(() => new URLSearchParams(window.location.search).get("token"), []);
  const [token] = useState<string | null>(initialToken);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [matchError, setMatchError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const mutation = useConfirmPasswordReset();

  // H3: strip the token from the URL on mount so it doesn't persist in history
  // or leak via Referer. Keep it in component state for the confirm call.
  useEffect(() => {
    if (token) {
      window.history.replaceState({}, "", "/reset-password");
    }
  }, [token]);

  // H3: defense in depth — no-referrer for the page lifetime.
  useEffect(() => {
    const meta = document.createElement("meta");
    meta.name = "referrer";
    meta.content = "no-referrer";
    document.head.appendChild(meta);
    return () => {
      document.head.removeChild(meta);
    };
  }, []);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!token) return;
      if (newPassword !== confirmPassword) {
        setMatchError("Mật khẩu xác nhận không khớp.");
        return;
      }
      if (newPassword.length < 8) {
        setMatchError("Mật khẩu phải có ít nhất 8 ký tự.");
        return;
      }
      setMatchError(null);
      try {
        await mutation.mutateAsync({ token, new_password: newPassword });
        setSuccess(true);
      } catch {
        // mutation.error carries the VN message; rendered inline below.
      }
    },
    [token, newPassword, confirmPassword, mutation],
  );

  // No token at all → invalid link state.
  if (!token) {
    return (
      <ResetShell>
        <InvalidLinkState
          title="Liên kết không hợp lệ"
          message="Không tìm thấy mã đặt lại mật khẩu. Vui lòng yêu cầu liên kết mới."
        />
      </ResetShell>
    );
  }

  // Success → green confirmation + back-to-login.
  if (success) {
    return (
      <ResetShell>
        <div className="py-2" role="status">
          <div className="mb-5 flex items-start gap-3">
            <FeaturedIcon icon={CheckCircle} color="success" theme="light" size="md" />
            <div>
              <p className="text-sm font-bold text-fg-primary">Đặt lại mật khẩu thành công</p>
              <p className="mt-1 text-sm leading-5 text-fg-secondary">
                Vui lòng đăng nhập bằng mật khẩu mới.
              </p>
            </div>
          </div>
          <Link to="/login" className={cx(buttonVariants({ variant: "default" }), "w-full")}>
            <ArrowLeft className="size-4" aria-hidden="true" />
            Đi đến đăng nhập
          </Link>
        </div>
      </ResetShell>
    );
  }

  const apiError = mutation.error;
  const isTokenInvalid = (apiError?.message ?? "").toLowerCase().includes("không hợp lệ") || (apiError?.message ?? "").toLowerCase().includes("hết hạn");

  return (
    <ResetShell>
      <div className="mb-6">
        <h2 className="font-display text-display-md font-semibold leading-tight tracking-tight text-fg-primary">
          Đặt lại mật khẩu
        </h2>
        <p className="mt-2 text-sm text-fg-tertiary">Nhập mật khẩu mới cho tài khoản của bạn.</p>
      </div>

      {apiError && (
        <>
          {isTokenInvalid ? (
            <InvalidLinkState
              title="Liên kết đã hết hạn"
              message="Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn."
            />
          ) : (
            <div role="alert" className="mb-5 flex items-start gap-2.5 rounded-lg border border-utility-error-300 bg-utility-error-50 p-3 text-sm">
              <AlertCircle className="size-5 shrink-0 text-utility-error-600" aria-hidden="true" />
              <span className="font-semibold leading-5 text-fg-error-primary">{apiError.message || "Đã có lỗi xảy ra, vui lòng thử lại."}</span>
            </div>
          )}
        </>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <div className="space-y-1.5">
          <Label htmlFor="newPassword">Mật khẩu mới</Label>
          <div className="relative">
            <Lock01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
            <Input
              id="newPassword"
              type={showPassword ? "text" : "password"}
              placeholder="Nhập mật khẩu mới"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="pr-12 pl-9 sm:pr-10"
              required
              autoComplete="new-password"
              disabled={mutation.isPending}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
              className="absolute inset-y-0 right-0 flex h-11 w-11 items-center justify-center rounded-lg text-fg-quaternary outline-brand transition duration-100 ease-linear hover:text-fg-tertiary focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 sm:h-9 sm:w-9"
            >
              {showPassword ? <EyeOff className="size-5" aria-hidden="true" /> : <Eye className="size-5" aria-hidden="true" />}
            </button>
          </div>
          {newPassword && <PasswordStrengthIndicator password={newPassword} />}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="confirmPassword">Xác nhận mật khẩu</Label>
          <div className="relative">
            <Lock01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
            <Input
              id="confirmPassword"
              type={showPassword ? "text" : "password"}
              placeholder="Nhập lại mật khẩu mới"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="pl-9"
              required
              autoComplete="new-password"
              disabled={mutation.isPending}
            />
          </div>
          {matchError && <p className="text-xs font-semibold text-fg-error-primary">{matchError}</p>}
        </div>

        <Button type="submit" className="mt-2 w-full" disabled={mutation.isPending || !newPassword || !confirmPassword}>
          {mutation.isPending ? (
            <>
              <AuthSpinner className="size-4" />
              Đang đặt lại...
            </>
          ) : (
            "Đặt lại mật khẩu"
          )}
        </Button>
      </form>
    </ResetShell>
  );
};

// ResetShell wraps the page in the same card layout as ForgotPassword/Login.
const ResetShell: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div
    data-admin-ui=""
    data-theme="congtruong"
    className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-muted px-4 py-6 text-fg-primary"
    style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
  >
    <AuthCard>{children}</AuthCard>
  </div>
);

const InvalidLinkState: React.FC<{ title: string; message: string }> = ({ title, message }) => (
  <div className="py-2">
    <div className="mb-5 flex items-start gap-3">
      <FeaturedIcon icon={AlertCircle} color="error" theme="light" size="md" />
      <div>
        <p className="text-sm font-bold text-fg-primary">{title}</p>
        <p className="mt-1 text-sm leading-5 text-fg-secondary">{message}</p>
      </div>
    </div>
    <Link to="/forgot-password" className={cx(buttonVariants({ variant: "default" }), "w-full")}>
      Yêu cầu liên kết mới
    </Link>
  </div>
);

export default ResetPassword;



