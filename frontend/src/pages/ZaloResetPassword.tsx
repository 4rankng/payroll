import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { AlertCircle, ArrowLeft, CheckCircle, Eye, EyeOff, Lock01, MessageChatCircle } from "@untitledui/icons";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConfirmZaloReset } from "@/hooks/api/useZaloReset";
import { PasswordStrengthIndicator } from "@/components/ui/password-strength-indicator";
import { AuthCard } from "@/components/auth/AuthCard";
import { AuthSpinner } from "@/components/auth/AuthSpinner";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";

/**
 * ZaloResetPassword — step 2 of the Zalo-OTP password-reset flow.
 *
 * Reached from `/forgot-password` with router state `{ otp_session_id, mobile }`
 * (NOT a query string — the session id is kept out of the URL so it doesn't
 * leak via Referer or browser history, same hardening as ResetPassword.tsx).
 * If the state is missing (user navigated here directly / refreshed), redirect
 * back to `/forgot-password`.
 *
 * The user enters the 6-digit code they received via ZNS + a new password.
 * On success they're sent to /login with a success toast.
 */
const ZaloResetPassword = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const state = location.state as { otp_session_id?: string; mobile?: string } | null;

  const sessionId = state?.otp_session_id ?? "";
  const mobile = state?.mobile ?? "";

  const [code, setCode] = useState(["", "", "", "", "", ""]);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [matchError, setMatchError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const mutation = useConfirmZaloReset();
  const inputsRef = useRef<(HTMLInputElement | null)[]>([]);

  // Redirect to forgot-password if state is missing (direct nav / refresh).
  useEffect(() => {
    if (!sessionId) {
      navigate("/forgot-password", { replace: true });
    }
  }, [sessionId, navigate]);

  // no-referrer for the page lifetime (defense in depth).
  useEffect(() => {
    const meta = document.createElement("meta");
    meta.name = "referrer";
    meta.content = "no-referrer";
    document.head.appendChild(meta);
    return () => {
      document.head.removeChild(meta);
    };
  }, []);

  const handleCodeChange = useCallback((idx: number, val: string) => {
    const digits = val.replace(/\D/g, "");
    // Mobile "autofill from message" drops the whole code into whichever box
    // is focused as one multi-digit value (no `paste` event fires) — split it
    // across the remaining boxes the same way handleCodePaste does.
    if (digits.length > 1) {
      setCode((prev) => {
        const next = [...prev];
        for (let i = 0; i < digits.length && idx + i < 6; i++) {
          next[idx + i] = digits[i];
        }
        return next;
      });
      inputsRef.current[Math.min(idx + digits.length, 5)]?.focus();
      return;
    }
    setCode((prev) => {
      const next = [...prev];
      next[idx] = digits;
      return next;
    });
    if (digits && idx < 5) {
      inputsRef.current[idx + 1]?.focus();
    }
  }, []);

  const handleCodeKeyDown = useCallback((idx: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace" && !code[idx] && idx > 0) {
      inputsRef.current[idx - 1]?.focus();
    }
  }, [code]);

  const handleCodePaste = useCallback((e: React.ClipboardEvent) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (pasted) {
      const next = ["", "", "", "", "", ""];
      for (let i = 0; i < pasted.length; i++) next[i] = pasted[i];
      setCode(next);
      inputsRef.current[Math.min(pasted.length, 5)]?.focus();
    }
  }, []);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!sessionId) return;
      if (newPassword !== confirmPassword) {
        setMatchError("Mật khẩu xác nhận không khớp");
        return;
      }
      setMatchError(null);
      try {
        await mutation.mutateAsync({
          otp_session_id: sessionId,
          code: code.join(""),
          new_password: newPassword,
        });
        setSuccess(true);
        setTimeout(() => navigate("/login", { state: { resetSuccess: true }, replace: true }), 1500);
      } catch {
        // Error rendered inline from mutation.error — no action needed.
      }
    },
    [sessionId, code, newPassword, confirmPassword, mutation, navigate],
  );

  const apiError = mutation.error;

  // --- success state ---
  if (success) {
    return (
      <div
        data-admin-ui=""
        data-theme="congtruong"
        className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-muted px-4 py-6 text-fg-primary"
      >
        <AuthCard className="text-center">
          <div className="flex flex-col items-center gap-4">
            <FeaturedIcon icon={CheckCircle} color="success" theme="light" size="xl" />
            <h2 className="font-display text-display-sm font-semibold leading-tight tracking-tight text-fg-primary">
              Đặt lại mật khẩu thành công
            </h2>
            <p className="text-sm text-fg-tertiary">Vui lòng đăng nhập bằng mật khẩu mới.</p>
          </div>
        </AuthCard>
      </div>
    );
  }

  return (
    <div
      data-admin-ui=""
      data-theme="congtruong"
      className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-muted px-4 py-6 text-fg-primary"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <AuthCard>
        <Link
          to="/forgot-password"
          className="-mx-1 inline-flex h-9 items-center gap-1 rounded-md px-1 text-xs font-semibold text-fg-tertiary outline-brand transition duration-100 ease-linear hover:text-tertiary_hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 sm:h-8"
        >
          <ArrowLeft className="size-4" aria-hidden="true" /> Quay lại
        </Link>

        <div className="mb-6 mt-4 flex items-center gap-3">
          <FeaturedIcon icon={MessageChatCircle} color="brand" theme="light" size="md" />
          <div>
            <h2 className="font-display text-display-sm font-semibold leading-none tracking-tight text-fg-primary">Đặt lại mật khẩu</h2>
            <p className="mt-1 text-xs text-fg-tertiary">
              {mobile
                ? `Nếu số điện thoại ${mobile} tồn tại trong hệ thống, mã OTP đã được gửi qua Zalo.`
                : "Nếu số điện thoại tồn tại trong hệ thống, mã OTP đã được gửi qua Zalo."}
            </p>
          </div>
        </div>

        {apiError && (
          <div role="alert" className="mb-4 flex items-start gap-2 rounded-lg border border-utility-error-300 bg-utility-error-50 p-3 text-sm">
            <AlertCircle className="size-4 shrink-0 text-utility-error-600" aria-hidden="true" />
            <p className="text-sm leading-5 text-fg-error-primary">
              {apiError.message || "Mã đặt lại không đúng hoặc đã hết hạn. Vui lòng yêu cầu mã mới."}
            </p>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          {/* 6-digit OTP input */}
          <div>
            <Label className="mb-2 block">Mã OTP (6 số)</Label>
            <div className="flex justify-between gap-1.5 sm:gap-2" onPaste={handleCodePaste}>
              {code.map((digit, idx) => (
                <input
                  key={idx}
                  ref={(el) => { inputsRef.current[idx] = el; }}
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]"
                  maxLength={1}
                  value={digit}
                  onChange={(e) => handleCodeChange(idx, e.target.value)}
                  onKeyDown={(e) => handleCodeKeyDown(idx, e)}
                  disabled={mutation.isPending}
                  className="h-12 w-full min-w-0 flex-1 rounded-lg border border-input bg-card text-center text-lg font-semibold text-fg-primary shadow-xs outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 sm:h-14"
                  aria-label={`Số thứ ${idx + 1}`}
                />
              ))}
            </div>
          </div>

          {/* New password */}
          <div>
            <Label htmlFor="zalo-new-pwd">Mật khẩu mới</Label>
            <div className="relative mt-1.5">
              <Lock01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
              <Input
                id="zalo-new-pwd"
                type={showPassword ? "text" : "password"}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                disabled={mutation.isPending}
                className="pr-12 pl-9 sm:pr-10"
                placeholder="Ít nhất 8 ký tự"
                required
                minLength={8}
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
            {newPassword && <div className="mt-2"><PasswordStrengthIndicator password={newPassword} /></div>}
          </div>

          {/* Confirm password */}
          <div>
            <Label htmlFor="zalo-confirm-pwd">Xác nhận mật khẩu mới</Label>
            <div className="relative mt-1.5">
              <Lock01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
              <Input
                id="zalo-confirm-pwd"
                type={showPassword ? "text" : "password"}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                disabled={mutation.isPending}
                className="pl-9"
                placeholder="Nhập lại mật khẩu mới"
                required
                minLength={8}
              />
            </div>
            {matchError && <p className="mt-1.5 text-xs font-semibold text-fg-error-primary">{matchError}</p>}
          </div>

          <Button
            type="submit"
            disabled={mutation.isPending || code.some((d) => !d) || newPassword.length < 8}
            className="w-full"
          >
            {mutation.isPending ? (
              <>
                <AuthSpinner className="size-4" /> Đang đặt lại...
              </>
            ) : (
              "Đặt lại mật khẩu"
            )}
          </Button>
        </form>

        <p className="mt-4 text-center text-xs text-fg-tertiary">
          Không nhận được mã?{" "}
          <Link to="/forgot-password" className="font-semibold text-brand-secondary underline-offset-2 outline-brand transition duration-100 ease-linear hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 rounded-sm">
            Yêu cầu lại
          </Link>{" "}
          hoặc liên hệ quản trị viên.
        </p>
      </AuthCard>
    </div>
  );
};

export default ZaloResetPassword;
