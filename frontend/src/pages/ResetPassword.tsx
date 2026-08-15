import { useState, useEffect, useCallback, useMemo } from "react";
import { Link } from "react-router-dom";
import { Lock, Eye, EyeOff, AlertCircle, Loader2, CheckCircle2, ArrowLeft } from "lucide-react";
import { useConfirmPasswordReset } from "@/hooks/api/usePasswordReset";
import { PasswordStrengthIndicator } from "@/components/ui/password-strength-indicator";
import type { ApiError } from "@/services/api/client";

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
          <div className="mb-5 flex items-start gap-3 rounded-xl border border-success/30 bg-success/10 p-4">
            <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-success" aria-hidden="true" />
            <div>
              <p className="text-sm font-bold text-base-content">Đặt lại mật khẩu thành công</p>
              <p className="mt-1 text-sm leading-5 text-base-content/70">
                Vui lòng đăng nhập bằng mật khẩu mới.
              </p>
            </div>
          </div>
          <Link
            to="/login"
            className="ct-btn ct-btn-primary ct-btn-lg flex h-12 w-full items-center justify-center gap-2 rounded-xl text-sm font-extrabold normal-case"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Đi đến đăng nhập
          </Link>
        </div>
      </ResetShell>
    );
  }

  const apiError = mutation.error as ApiError | null;
  const isTokenInvalid = (apiError?.message ?? "").toLowerCase().includes("không hợp lệ") || (apiError?.message ?? "").toLowerCase().includes("hết hạn");

  return (
    <ResetShell>
      <div className="mb-6">
        <h2 className="font-display text-3xl font-black leading-tight tracking-[-0.04em] sm:text-4xl">
          Đặt lại mật khẩu
        </h2>
        <p className="mt-2 text-sm text-base-content/60">Nhập mật khẩu mới cho tài khoản của bạn.</p>
      </div>

      {apiError && (
        <>
          {isTokenInvalid ? (
            <InvalidLinkState
              title="Liên kết đã hết hạn"
              message="Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn."
            />
          ) : (
            <div role="alert" className="ct-alert ct-alert-error mb-5 items-start rounded-xl text-sm shadow-none">
              <AlertCircle className="h-5 w-5 shrink-0" aria-hidden="true" />
              <span className="font-semibold leading-5">{apiError.message || "Đã có lỗi xảy ra, vui lòng thử lại."}</span>
            </div>
          )}
        </>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <div className="space-y-2">
          <label htmlFor="newPassword" className="block text-xs font-bold text-base-content/65">
            Mật khẩu mới
          </label>
          <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
            <Lock className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
            <input
              id="newPassword"
              type={showPassword ? "text" : "password"}
              placeholder="Nhập mật khẩu mới"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content/35"
              required
              autoComplete="new-password"
              disabled={mutation.isPending}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
              className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square -mr-2 min-h-9 h-9 w-9 text-base-content/40 hover:text-base-content"
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </label>
          {newPassword && <PasswordStrengthIndicator password={newPassword} />}
        </div>

        <div className="space-y-2">
          <label htmlFor="confirmPassword" className="block text-xs font-bold text-base-content/65">
            Xác nhận mật khẩu
          </label>
          <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
            <Lock className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
            <input
              id="confirmPassword"
              type={showPassword ? "text" : "password"}
              placeholder="Nhập lại mật khẩu mới"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content/35"
              required
              autoComplete="new-password"
              disabled={mutation.isPending}
            />
          </label>
          {matchError && <p className="text-xs font-semibold text-destructive">{matchError}</p>}
        </div>

        <button
          type="submit"
          className="ct-btn ct-btn-primary ct-btn-lg mt-2 h-12 w-full rounded-xl text-sm font-extrabold normal-case"
          disabled={mutation.isPending || !newPassword || !confirmPassword}
        >
          {mutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Đang đặt lại...
            </>
          ) : (
            "Đặt lại mật khẩu"
          )}
        </button>
      </form>
    </ResetShell>
  );
};

// ResetShell wraps the page in the same card layout as ForgotPassword/Login.
const ResetShell: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div
    data-admin-ui=""
    data-theme="congtruong"
    className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-base-200 px-4 py-6 text-base-content"
    style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
  >
    <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100">
      <div className="ct-card-body gap-0 p-6 sm:p-8">
        <div className="mb-6 flex items-center gap-3">
          <img src="/logo-square.png" alt="TingTing logo" className="h-12 w-12 object-contain" />
          <p className="font-display text-[1.35rem] font-black leading-none tracking-[-0.03em]">TingTing</p>
        </div>
        {children}
      </div>
    </div>
  </div>
);

const InvalidLinkState: React.FC<{ title: string; message: string }> = ({ title, message }) => (
  <div className="py-2">
    <div className="mb-5 flex items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/10 p-4">
      <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-destructive" aria-hidden="true" />
      <div>
        <p className="text-sm font-bold text-base-content">{title}</p>
        <p className="mt-1 text-sm leading-5 text-base-content/70">{message}</p>
      </div>
    </div>
    <Link
      to="/forgot-password"
      className="ct-btn ct-btn-primary ct-btn-lg flex h-12 w-full items-center justify-center gap-2 rounded-xl text-sm font-extrabold normal-case"
    >
      Yêu cầu liên kết mới
    </Link>
  </div>
);

export default ResetPassword;
