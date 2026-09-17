import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate, Link } from "react-router-dom";
import {
  Lock,
  User,
  Eye,
  EyeOff,
  AlertCircle,
  Loader2,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { authManager } from "@/lib/auth";
import { useAuth } from "@/contexts";
import { useLogin } from "@/hooks/api/useAuth";
import { apiClient } from "@/services/api/client";
import type { ApiError } from "@/services/api/client";

const Login = () => {
  const [emailOrUsername, setEmailOrUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [failedAttempts, setFailedAttempts] = useState(0);
  const [captchaImage, setCaptchaImage] = useState<string | null>(null);
  const [captchaId, setCaptchaId] = useState<string | null>(null);
  const [captchaCode, setCaptchaCode] = useState("");
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [loginSubmitting, setLoginSubmitting] = useState(false);
  const loginMutation = useLogin();
  const loginError = loginMutation.error;
  const resetLogin = loginMutation.reset;
  const navigate = useNavigate();
  const { isAuthenticated, isLoading } = useAuth();
  const needsCaptcha = failedAttempts >= 3;
  const loginSubmitInFlightRef = useRef(false);

  // Fetch a fresh CAPTCHA challenge. The endpoint is public (no auth token
  // required — see backend setupAuthRoutes), so it works pre-login. Goes
  // through apiClient so the configured baseURL/proxy + headers are honored
  // (a raw fetch("/api/v1/...") breaks when VITE_API_BASE_URL is a separate
  // host). Synchronously guards against overlapping requests via a ref.
  const captchaInFlightRef = useRef(false);
  const refreshCaptcha = useCallback(async () => {
    if (captchaInFlightRef.current) return;
    captchaInFlightRef.current = true;
    setCaptchaLoading(true);
    try {
      const res = await apiClient.get<{ captcha_id: string; image: string }>("/auth/captcha");
      const d = res?.data;
      if (d?.captcha_id && d?.image) {
        setCaptchaId(d.captcha_id);
        setCaptchaImage(d.image);
        setCaptchaCode("");
      }
    } catch {
      // Swallow: the form still submits without a captcha_id, and the server
      // returns a clear "captcha required" error the user can act on.
    } finally {
      captchaInFlightRef.current = false;
      setCaptchaLoading(false);
    }
  }, []);

  const syncCaptchaRequirement = useCallback(async (): Promise<boolean> => {
    const username = emailOrUsername.trim();
    if (!username) return false;

    try {
      const res = await apiClient.get<{ required: boolean }>("/auth/captcha/required", {
        params: { username },
      });
      const required = res?.data?.required === true;
      if (required) setFailedAttempts((attempts) => Math.max(attempts, 3));
      return required;
    } catch {
      // Do not make the status check a new login outage. The login endpoint
      // remains authoritative and the error handler below can still reveal
      // the CAPTCHA when the server requires it.
      return false;
    }
  }, [emailOrUsername]);

  // Redirect authenticated users away from /login
  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      const role = authManager.getUserRole();
      if (role === "admin") navigate("/admin", { replace: true });
      else if (role === "partner") navigate("/partner/dashboard", { replace: true });
      else if (role === "adv_partner") navigate("/adv-partner/advance-payments", { replace: true });
      else if (role === "accountant") navigate("/accountant", { replace: true });
      else if (role === "employee") navigate("/employee", { replace: true });
      else navigate("/", { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  useEffect(() => {
    if (!authManager.isTokenValid()) {
      authManager.removeToken();
    }
  }, []);

  // Track login failures + fetch CAPTCHA image when threshold is reached.
  useEffect(() => {
    if (loginMutation.isError) {
      const error = loginMutation.error;
      if (error?.message?.toLowerCase().includes("captcha")) {
        setFailedAttempts((attempts) => Math.max(attempts, 3));
      } else {
        setFailedAttempts((attempts) => attempts + 1);
      }
    }
    if (loginMutation.isSuccess) setFailedAttempts(0);
  }, [loginMutation.error, loginMutation.isError, loginMutation.isSuccess]);

  // Fetch the initial challenge when the threshold is first crossed, and
  // auto-refresh the image after each subsequent failed attempt (the prior
  // challenge is single-use server-side). Depends on failedAttempts so a
  // failed login with a captcha triggers a fresh image automatically.
  useEffect(() => {
    if (!needsCaptcha) return;
    refreshCaptcha();
  }, [needsCaptcha, failedAttempts, refreshCaptcha]);

  useEffect(() => {
    if (loginError) resetLogin();
  }, [emailOrUsername, password]); // eslint-disable-line react-hooks/exhaustive-deps

  const isDisabled = loginSubmitting || loginMutation.isPending;

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      // The CAPTCHA preflight awaits a network request before React Query marks
      // the login mutation pending. Guard that entire window synchronously so a
      // double click/Enter cannot create two OTP challenges and two emails.
      if (loginSubmitInFlightRef.current) return;
      loginSubmitInFlightRef.current = true;
      setLoginSubmitting(true);
      try {
        const serverRequiresCaptcha = await syncCaptchaRequirement();
        if (serverRequiresCaptcha && (!captchaId || !captchaCode)) {
          if (!captchaId) await refreshCaptcha();
          return;
        }
        await loginMutation.mutateAsync({
          username: emailOrUsername,
          password,
          ...((needsCaptcha || serverRequiresCaptcha) && captchaId
            ? { captcha_id: captchaId, captcha_code: captchaCode }
            : {}),
        });
      } catch {
        // The mutation owns error presentation; this only prevents an unhandled
        // rejection from mutateAsync while keeping the single-flight guard.
      } finally {
        loginSubmitInFlightRef.current = false;
        setLoginSubmitting(false);
      }
    },
    [emailOrUsername, password, loginMutation, needsCaptcha, captchaId, captchaCode, refreshCaptcha, syncCaptchaRequirement]
  );

  const togglePassword = useCallback(() => setShowPassword((v) => !v), []);

  if (isLoading) {
    return (
      <div
        data-admin-ui=""
        data-theme="congtruong"
        className="flex min-h-dvh items-center justify-center bg-base-200 text-base-content"
      >
        <span className="ct-loading ct-loading-spinner ct-loading-lg text-primary" aria-label="Đang tải" />
      </div>
    );
  }

  if (isAuthenticated) {
    return null;
  }

  return (
    <div
      id="main-content"
      data-admin-ui=""
      data-login-ui=""
      data-theme="congtruong"
      className="relative min-h-dvh w-full overflow-x-hidden bg-base-200 text-base-content"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_80%_8%,hsl(var(--info)/0.14),transparent_28%),radial-gradient(circle_at_8%_88%,hsl(var(--primary)/0.12),transparent_30%)]" />

      <main className="relative mx-auto grid min-h-dvh w-full max-w-[1560px] lg:grid-cols-[minmax(0,1.08fr)_minmax(440px,0.92fr)] lg:gap-3 lg:p-3 xl:gap-4 xl:p-5">
        <section
          aria-labelledby="login-brand-heading"
          className="ct-hero relative hidden min-h-[calc(100dvh-2.5rem)] overflow-hidden rounded-[2rem] border border-base-300 bg-base-100 shadow-[0_24px_70px_-50px_hsl(var(--neutral)/0.34)] lg:flex"
        >
          <img
            src="/login-employee-wallet-app-hero.webp"
            alt="Ứng dụng lương tuần bên cạnh ví và lịch trả lương"
            className="absolute bottom-0 right-0 h-full w-auto max-w-none object-contain object-bottom-right"
          />
          <div className="ct-hero-overlay absolute inset-0 !bg-base-100/5" />
          <div className="absolute inset-0 bg-gradient-to-r from-base-100 via-base-100/95 via-[46%] to-transparent to-[78%]" />
          <div className="absolute inset-0 bg-gradient-to-b from-base-100/55 via-transparent via-[46%] to-base-100/10" />

          <div className="ct-hero-content relative z-10 flex h-full w-full max-w-none items-start justify-start px-8 py-[clamp(3.5rem,8vh,7rem)] text-left text-base-content xl:px-12 2xl:px-14">
            <div className="max-w-[540px]">
              <p className="mb-4 flex items-center gap-3 text-xs font-extrabold uppercase tracking-[0.22em] text-primary">
                <span className="h-px w-8 bg-primary" aria-hidden="true" />
                Lương về đúng nhịp
              </p>
              <h1 id="login-brand-heading" className="font-display text-[clamp(2.65rem,3.8vw,4.1rem)] font-black leading-[0.96] tracking-[-0.05em] text-base-content">
                Ứng lương khi cần.
                <span className="mt-2 block text-primary">Trả lương mỗi tuần.</span>
              </h1>
              <p className="mt-6 max-w-[470px] text-base font-medium leading-7 text-base-content/65 xl:text-lg">
                Dòng tiền linh hoạt cho người lao động. Một chu kỳ lương gọn gàng, dễ kiểm soát cho doanh nghiệp.
              </p>
            </div>
          </div>
        </section>

        <section className="relative flex min-h-dvh flex-col bg-base-100 lg:min-h-[calc(100dvh-2.5rem)] lg:rounded-[2rem] lg:border lg:border-base-300/70">
          <div className="ct-hero relative h-44 min-h-44 overflow-hidden border-b border-base-300 bg-base-100 sm:h-52 sm:min-h-52 lg:hidden">
            <img
              src="/login-employee-wallet-app-hero.webp"
              alt="Ứng dụng lương tuần bên cạnh ví và lịch trả lương"
              className="absolute inset-y-0 right-0 h-full w-[56%] object-cover object-[54%_68%]"
            />
            <div className="ct-hero-overlay absolute inset-0 !bg-base-100/10" />
            <div className="absolute inset-y-0 left-0 w-[62%] bg-gradient-to-r from-base-100 via-base-100/95 to-transparent" />
            <div className="ct-hero-content relative z-10 flex h-full w-full max-w-none items-start justify-start px-5 py-5 text-base-content sm:px-8 sm:py-7">
              <div className="max-w-[56%]">
                <p className="text-xs font-extrabold uppercase tracking-[0.2em] text-primary">Lương về đúng nhịp</p>
                <p className="mt-1.5 max-w-[280px] font-display text-2xl font-black leading-[1.05] tracking-[-0.035em] sm:text-3xl">
                  Ứng lương khi cần.<br /><span className="text-primary">Trả lương mỗi tuần.</span>
                </p>
                <p className="mt-2.5 text-xs font-medium leading-[1.45] text-base-content/60 sm:text-xs">
                  Dòng tiền linh hoạt cho người lao động. Một chu kỳ lương gọn gàng, dễ kiểm soát cho doanh nghiệp.
                </p>
              </div>
            </div>
          </div>

          <div className="flex flex-1 items-center justify-center px-4 py-6 sm:px-8 sm:py-8 lg:px-10 xl:px-16">
            <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100 lg:border-0 lg:bg-transparent">
              <div className="ct-card-body gap-0 p-5 sm:p-8 lg:p-4">
                <div className="mb-7 flex items-center gap-3">
                  <img src="/logo-square.png" alt="TingTing logo" className="h-12 w-12 object-contain" />
                  <div>
                    <p className="font-display text-[1.35rem] font-black leading-none tracking-[-0.03em]">TingTing</p>
                  </div>
                </div>

                <div className="mb-7">
                  <h2 className="font-display text-3xl font-black leading-tight tracking-[-0.04em] sm:text-4xl">Chào mừng trở lại</h2>
                </div>

                {loginMutation.error && (
                  <div role="alert" className="ct-alert ct-alert-error mb-5 items-start rounded-xl text-sm shadow-none animate-fade-in">
                    <AlertCircle className="h-5 w-5 shrink-0" aria-hidden="true" />
                    <span className="font-semibold leading-5">
                      {(((loginMutation.error as unknown) as ApiError)?.http_status === 429
                        ? (((loginMutation.error as unknown) as ApiError)?.message || "Quá nhiều lần đăng nhập. Vui lòng thử lại sau ít phút.")
                        : "Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.")}
                    </span>
                  </div>
                )}

                <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                  <fieldset className="space-y-4">
                    <legend className="sr-only">Thông tin đăng nhập</legend>

                    <div className="space-y-2">
                      <label htmlFor="emailOrUsername" className="block text-xs font-bold text-base-content/65">Tên đăng nhập</label>
                      <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
                        <User className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
                        <input
                          id="emailOrUsername"
                          type="text"
                          placeholder="CCCD, số điện thoại hoặc tên đăng nhập"
                          value={emailOrUsername}
                          onChange={(e) => setEmailOrUsername(e.target.value)}
                          onBlur={() => void syncCaptchaRequirement()}
                          className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content/35"
                          required
                          autoComplete="username"
                          autoCapitalize="none"
                          disabled={isDisabled}
                        />
                      </label>
                    </div>

                    <div className="space-y-2">
                      <label htmlFor="password" className="block text-xs font-bold text-base-content/65">Mật khẩu</label>
                      <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
                        <Lock className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
                        <input
                          id="password"
                          type={showPassword ? "text" : "password"}
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Nhập mật khẩu của bạn"
                          value={password}
                          onChange={(e) => setPassword(e.target.value)}
                          className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content/35"
                          required
                          autoComplete="current-password"
                          disabled={isDisabled}
                        />
                        <button
                          type="button"
                          onClick={togglePassword}
                          aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
                          className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square -mr-2 min-h-9 h-9 w-9 text-base-content/40 hover:text-base-content"
                        >
                          {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </button>
                      </label>
                    </div>

                    <div className="flex justify-end">
                      <Link to="/forgot-password" className="text-xs font-bold text-primary hover:underline">
                        Quên mật khẩu?
                      </Link>
                    </div>

                    {needsCaptcha && (
                      <div className="space-y-2 animate-fade-in-up">
                        <label htmlFor="captchaCode" className="block text-xs font-bold text-base-content/65">Mã xác nhận</label>
                        <div className="flex items-center gap-2">
                          {captchaImage ? (
                            <button
                              type="button"
                              onClick={() => refreshCaptcha()}
                              className="h-12 overflow-hidden rounded-xl border border-base-300 bg-base-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                              aria-label="Tải hình mã xác nhận khác"
                            >
                              <img src={captchaImage} alt="Hình ảnh mã xác nhận 5 chữ số" className="h-full w-auto select-none" />
                            </button>
                          ) : (
                            <div className="flex h-12 min-w-24 items-center justify-center rounded-xl border border-base-300 bg-base-200" aria-hidden="true">
                              <span className="ct-loading ct-loading-spinner ct-loading-sm text-base-content/35" />
                            </div>
                          )}
                          <input
                            id="captchaCode"
                            type="text"
                            inputMode="numeric"
                            placeholder="Nhập mã"
                            value={captchaCode}
                            onChange={(e) => setCaptchaCode(e.target.value.replace(/\D/g, "").slice(0, 5))}
                            className="ct-input ct-input-bordered h-12 min-w-0 flex-1 rounded-xl border-base-300 bg-base-200/55 text-center text-sm tracking-[0.3em] focus:border-primary focus:bg-base-100 focus:outline-none"
                            autoComplete="off"
                            required={needsCaptcha}
                            disabled={isDisabled || captchaLoading}
                          />
                          <button
                            type="button"
                            onClick={() => refreshCaptcha()}
                            disabled={captchaLoading || isDisabled}
                            aria-label="Tải hình mã xác nhận khác"
                            className="ct-btn ct-btn-outline ct-btn-square h-12 min-h-12 w-12 rounded-xl border-base-300"
                          >
                            <RefreshCw className={`h-4 w-4 ${captchaLoading ? "animate-spin" : ""}`} />
                          </button>
                        </div>
                      </div>
                    )}
                  </fieldset>

                  <button
                    type="submit"
                    className="ct-btn ct-btn-primary ct-btn-lg mt-2 h-12 w-full rounded-xl text-sm font-extrabold normal-case"
                    disabled={isDisabled}
                  >
                    {loginMutation.isPending ? (
                      <>
                        <span className="ct-loading ct-loading-spinner ct-loading-sm" />
                        Đang đăng nhập...
                      </>
                    ) : (
                      "Đăng nhập"
                    )}
                  </button>
                </form>

                <div className="mt-6 flex items-center justify-center gap-2 text-xs text-base-content/45">
                  <ShieldCheck className="h-4 w-4 text-success" aria-hidden="true" />
                  <span>Phiên đăng nhập được bảo vệ và mã hóa</span>
                </div>
              </div>
            </div>
          </div>

          <footer className="px-5 pb-6 text-center text-xs text-base-content/40 lg:px-10">
            <p>© {new Date().getFullYear()} TingTing · Ứng lương nhanh · Trả lương tuần</p>
            <div className="mt-2 flex items-center justify-center gap-3">
              <span>Điều khoản</span>
              <span aria-hidden="true">·</span>
              <span>Quyền riêng tư</span>
              <span aria-hidden="true">·</span>
              <span>Hỗ trợ</span>
            </div>
          </footer>
        </section>
      </main>
    </div>
  );
};

export default Login;
