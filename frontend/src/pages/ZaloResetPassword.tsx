import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { Lock, Eye, EyeOff, AlertCircle, Loader2, CheckCircle2, ArrowLeft, MessageCircle } from "lucide-react";
import { useConfirmZaloReset } from "@/hooks/api/useZaloReset";
import { PasswordStrengthIndicator } from "@/components/ui/password-strength-indicator";
import type { ApiError } from "@/services/api/client";

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
 * On success they're sent to `/login` with a success toast.
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
    const digit = val.replace(/\D/g, "").slice(-1);
    setCode((prev) => {
      const next = [...prev];
      next[idx] = digit;
      return next;
    });
    if (digit && idx < 5) {
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

  const apiError = mutation.error as ApiError | null;

  // --- success state ---
  if (success) {
    return (
      <div
        data-admin-ui=""
        data-theme="congtruong"
        className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-base-200 px-4 py-6 text-base-content"
      >
        <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100">
          <div className="ct-card-body gap-4 p-6 sm:p-8 text-center">
            <CheckCircle2 className="mx-auto h-14 w-14 text-success" />
            <h2 className="font-display text-2xl font-black leading-tight tracking-[-0.04em]">
              Đặt lại mật khẩu thành công
            </h2>
            <p className="text-sm text-base-content/70">Vui lòng đăng nhập bằng mật khẩu mới.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      data-admin-ui=""
      data-theme="congtruong"
      className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-base-200 px-4 py-6 text-base-content"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100">
        <div className="ct-card-body gap-0 p-6 sm:p-8">
          <Link to="/forgot-password" className="mb-4 inline-flex items-center gap-1.5 text-xs font-bold text-base-content/60 hover:text-primary">
            <ArrowLeft className="h-3.5 w-3.5" /> Quay lại
          </Link>

          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10">
              <MessageCircle className="h-6 w-6 text-primary" />
            </div>
            <div>
              <h2 className="font-display text-2xl font-black leading-none tracking-[-0.03em]">Đặt lại mật khẩu</h2>
              <p className="mt-1 text-xs text-base-content/60">
                {mobile
                  ? `Nếu số điện thoại ${mobile} tồn tại trong hệ thống, mã OTP đã được gửi qua Zalo.`
                  : "Nếu số điện thoại tồn tại trong hệ thống, mã OTP đã được gửi qua Zalo."}
              </p>
            </div>
          </div>

          {apiError && (
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
              <p className="text-sm text-destructive">
                {apiError.message || "Mã đặt lại không đúng hoặc đã hết hạn. Vui lòng yêu cầu mã mới."}
              </p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            {/* 6-digit OTP input */}
            <div>
              <label className="mb-2 block text-xs font-bold text-base-content/70">Mã OTP (6 số)</label>
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
                    className="h-12 w-full min-w-0 flex-1 rounded-lg border border-base-300 bg-base-100 text-center text-lg font-bold text-base-content outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 sm:h-14 sm:text-xl"
                    aria-label={`Số thứ ${idx + 1}`}
                  />
                ))}
              </div>
            </div>

            {/* New password */}
            <div>
              <label htmlFor="zalo-new-pwd" className="mb-2 block text-xs font-bold text-base-content/70">
                Mật khẩu mới
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
                <input
                  id="zalo-new-pwd"
                  type={showPassword ? "text" : "password"}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={mutation.isPending}
                  className="w-full rounded-lg border border-base-300 bg-base-100 py-2.5 pl-10 pr-10 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                  placeholder="Ít nhất 8 ký tự"
                  required
                  minLength={8}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content"
                  tabIndex={-1}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {newPassword && <PasswordStrengthIndicator password={newPassword} />}
            </div>

            {/* Confirm password */}
            <div>
              <label htmlFor="zalo-confirm-pwd" className="mb-2 block text-xs font-bold text-base-content/70">
                Xác nhận mật khẩu mới
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
                <input
                  id="zalo-confirm-pwd"
                  type={showPassword ? "text" : "password"}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={mutation.isPending}
                  className="w-full rounded-lg border border-base-300 bg-base-100 py-2.5 pl-10 pr-10 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                  placeholder="Nhập lại mật khẩu mới"
                  required
                  minLength={8}
                />
              </div>
              {matchError && <p className="mt-1.5 text-xs text-destructive">{matchError}</p>}
            </div>

            <button
              type="submit"
              disabled={mutation.isPending || code.some((d) => !d) || newPassword.length < 8}
              className="ct-btn-primary w-full"
            >
              {mutation.isPending ? (
                <><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Đang đặt lại...</>
              ) : (
                "Đặt lại mật khẩu"
              )}
            </button>
          </form>

          <p className="mt-4 text-center text-xs text-base-content/50">
            Không nhận được mã?{" "}
            <Link to="/forgot-password" className="font-bold text-primary hover:underline">
              Yêu cầu lại
            </Link>{" "}
            hoặc liên hệ quản trị viên.
          </p>
        </div>
      </div>
    </div>
  );
};

export default ZaloResetPassword;
