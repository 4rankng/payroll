import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AlertCircle, Loader2, ArrowLeft, ShieldCheck } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { authManager } from "@/lib/auth";
import { useAuth } from "@/contexts";
import { authService, getPendingOtpSessionId, clearPendingOtp } from "@/services/api/auth.service";
import { showErrorNotification } from "@/utils/error-handler";
import { generateAvatarUrl } from "@/utils/avatarHelpers";

const RESEND_COOLDOWN_SECONDS = 30;

/**
 * OTPLogin — the second step of the email-OTP login flow.
 *
 * Shown when /auth/login returns otp_required. Reads the pending session id
 * from sessionStorage (NOT auth_token — that key is reserved for the real JWT,
 * see auth.service.ts). The user enters the 6-digit code emailed to them;
 * on success the real token is stored and the user is navigated by role.
 *
 * RT-C3: this screen is reached ONLY via the useAuth.onSuccess fork — never
 * by storing the otp_session_id in auth_token.
 */
const OTPLogin = () => {
  const [code, setCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [resending, setResending] = useState(false);
  // The initial OTP was just sent by /auth/login. Start the same cooldown used
  // after an explicit resend so the page cannot immediately emit a duplicate.
  const [cooldown, setCooldown] = useState(RESEND_COOLDOWN_SECONDS);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const resendInFlightRef = useRef(false);

  const navigate = useNavigate();
  const { login } = useAuth();

  // Read the pending session id once. If absent (e.g. user navigated here
  // directly / refreshed after tab close), bounce back to /login.
  const [sessionId] = useState<string | null>(() => getPendingOtpSessionId());
  useEffect(() => {
    if (!sessionId) {
      navigate("/login", { replace: true });
      return;
    }
    inputRef.current?.focus();
  }, [sessionId, navigate]);

  // Resend cooldown countdown.
  useEffect(() => {
    if (cooldown <= 0) return;
    const t = setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => clearTimeout(t);
  }, [cooldown]);

  const handleVerify = useCallback(async () => {
    if (!sessionId) return;
    const trimmed = code.trim();
    if (trimmed.length !== 6 || !/^\d{6}$/.test(trimmed)) {
      setError("Mã xác thực phải gồm 6 chữ số.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = await authService.verifyLoginOtp({
        otp_session_id: sessionId,
        code: trimmed,
      });
      const data = response.data;
      if (data?.access_token && data?.user) {
        login(data.access_token, {
          id: String(data.user.id),
          email: data.user.email,
          name: data.user.fullname,
          username: data.user.username,
          role: data.user.role,
          avatar: generateAvatarUrl(),
        });
        // Navigate by role (mirrors useLogin).
        if (data.user.role === "admin") navigate("/admin", { replace: true });
        else if (data.user.role === "partner") navigate("/partner/dashboard", { replace: true });
        else if (data.user.role === "employee") navigate("/employee", { replace: true });
        else navigate("/", { replace: true });
      } else {
        setError("Không nhận được token. Vui lòng thử lại.");
      }
    } catch (err) {
      // The backend returns a generic 401 for invalid code/session/lockout so
      // as not to leak which failed.
      setError("Mã xác thực không đúng hoặc đã hết hạn.");
    } finally {
      setSubmitting(false);
    }
  }, [sessionId, code, login, navigate]);

  const handleResend = useCallback(async () => {
    if (!sessionId || cooldown > 0 || resendInFlightRef.current) return;
    resendInFlightRef.current = true;
    setResending(true);
    setError(null);
    try {
      await authService.resendOtp({ otp_session_id: sessionId });
      setCooldown(RESEND_COOLDOWN_SECONDS);
    } catch (err) {
      showErrorNotification(err as Error);
    } finally {
      resendInFlightRef.current = false;
      setResending(false);
    }
  }, [sessionId, cooldown]);

  const handleCancel = useCallback(() => {
    clearPendingOtp();
    navigate("/login", { replace: true });
  }, [navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50 px-4">
      <div className="w-full max-w-md space-y-6 rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="space-y-2 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
            <ShieldCheck className="h-6 w-6 text-slate-600" aria-hidden="true" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Xác thực hai bước</h1>
          <p className="text-sm text-slate-500">
            Chúng tôi đã gửi mã xác thực 6 chữ số đến email của bạn. Vui lòng nhập mã để tiếp tục.
          </p>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" aria-hidden="true" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-2">
          <Label htmlFor="otp-code">Mã xác thực</Label>
          <Input
            id="otp-code"
            ref={inputRef}
            type="text"
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9]*"
            maxLength={6}
            value={code}
            onChange={(e) => {
              const v = e.target.value.replace(/\D/g, "").slice(0, 6);
              setCode(v);
              if (error) setError(null);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" && code.length === 6 && !submitting) {
                handleVerify();
              }
            }}
            placeholder="••••••"
            className="text-center text-2xl tracking-[0.5em]"
            disabled={submitting}
          />
        </div>

        <Button
          onClick={handleVerify}
          className="w-full"
          disabled={submitting || code.length !== 6}
        >
          {submitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
              Đang xác thực...
            </>
          ) : (
            "Xác thực"
          )}
        </Button>

        <div className="flex items-center justify-between text-sm">
          <button
            type="button"
            onClick={handleCancel}
            className="inline-flex items-center gap-1 text-slate-500 hover:text-slate-700"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Quay lại đăng nhập
          </button>
          <button
            type="button"
            onClick={handleResend}
            disabled={cooldown > 0 || resending}
            className="font-medium text-slate-600 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {resending ? "Đang gửi..." : cooldown > 0 ? `Gửi lại (${cooldown}s)` : "Gửi lại mã"}
          </button>
        </div>
      </div>
    </div>
  );
};

export default OTPLogin;
