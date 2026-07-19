import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  Lock,
  User,
  Eye,
  EyeOff,
  AlertCircle,
  Loader2,
  ChevronRight,
  RefreshCw,
  CalendarDays,
  ShieldCheck,
  WalletCards,
} from "lucide-react";
import { authManager } from "@/lib/auth";
import { useAuth } from "@/contexts";
import { useLogin, useGoogleLogin } from "@/hooks/api/useAuth";
import { apiClient } from "@/services/api/client";
import type { ApiError } from "@/services/api/client";

const GoogleIcon = () => (
  <svg className="h-[18px] w-[18px] shrink-0" viewBox="0 0 24 24" aria-hidden="true">
    <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
    <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
    <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
    <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
  </svg>
);

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
  const googleLoginMutation = useGoogleLogin();
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
      const error = loginMutation.error as ApiError;
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

  // Handle Google OAuth redirect callback — Google returns id_token in the URL hash
  useEffect(() => {
    const hash = window.location.hash;
    if (!hash) return;
    const params = new URLSearchParams(hash.slice(1)); // strip leading '#'
    const idToken = params.get("id_token");
    if (idToken) {
      // Remove token from URL before sending to backend
      window.history.replaceState(null, "", window.location.pathname);
      googleLoginMutation.mutate(idToken);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Redirect to Google OIDC — returns id_token in fragment, no popup, no FedCM needed
  const handleGoogleLogin = useCallback(() => {
    const clientId = import.meta.env.VITE_GOOGLE_CLIENT_ID;
    if (!clientId) return;
    const nonce = crypto.getRandomValues(new Uint8Array(16))
      .reduce((acc, b) => acc + b.toString(16).padStart(2, "0"), "");
    const params = new URLSearchParams({
      client_id: clientId,
      redirect_uri: window.location.origin + "/login",
      response_type: "id_token",
      scope: "openid email profile",
      nonce,
      prompt: "select_account",
    });
    window.location.href = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
  }, []);

  const isDisabled = loginSubmitting || loginMutation.isPending || googleLoginMutation.isPending;

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
      <div className="pointer-events-none absolute inset-0 opacity-[0.42] [background-image:linear-gradient(to_right,hsl(var(--border)/0.45)_1px,transparent_1px),linear-gradient(to_bottom,hsl(var(--border)/0.45)_1px,transparent_1px)] [background-size:32px_32px]" />

      <main className="relative mx-auto grid min-h-dvh w-full max-w-[1600px] lg:grid-cols-[minmax(0,1.08fr)_minmax(440px,0.92fr)] lg:p-4 xl:p-6">
        <section
          aria-labelledby="login-brand-heading"
          className="relative hidden min-h-[calc(100dvh-3rem)] overflow-hidden rounded-[2rem] border border-base-300 bg-base-100 lg:block"
        >
          <img
            src="/login-payroll-hero.webp"
            alt="Ví lương, lịch tuần và bảng công minh họa cho ứng lương nhanh"
            className="absolute inset-0 h-full w-full object-cover object-center"
          />
          <div className="absolute inset-0 bg-gradient-to-b from-base-100/15 via-transparent to-neutral/10" />

          <div className="absolute left-8 top-8 flex items-center gap-3 rounded-2xl border border-base-300/80 bg-base-100/90 px-4 py-3 shadow-sm backdrop-blur-md xl:left-10 xl:top-10">
            <img src="/logo-square.png" alt="" className="h-10 w-10 object-contain" />
            <div>
              <p className="font-display text-xl font-extrabold leading-none tracking-tight text-base-content">TingTing</p>
              <p className="mt-1 text-[10px] font-bold uppercase tracking-[0.22em] text-base-content/55">Nhịp lương thông minh</p>
            </div>
          </div>

          <div className="absolute right-8 top-10 max-w-[390px] text-right xl:right-12 xl:top-14 xl:max-w-[470px]">
            <p className="mb-3 text-xs font-bold uppercase tracking-[0.24em] text-base-content/55">Dòng tiền chủ động</p>
            <h1 id="login-brand-heading" className="font-display text-4xl font-black leading-[0.98] tracking-[-0.045em] text-neutral xl:text-6xl">
              Ứng lương nhanh.
              <span className="mt-2 block text-primary">Trả lương tuần.</span>
            </h1>
            <p className="ml-auto mt-5 max-w-sm text-sm font-medium leading-6 text-base-content/65 xl:text-base">
              Một nhịp lương rõ ràng cho người lao động chủ động và doanh nghiệp vận hành nhẹ nhàng hơn.
            </p>
          </div>

          <div className="absolute bottom-7 left-7 right-7 grid grid-cols-3 overflow-hidden rounded-2xl border border-base-300/80 bg-base-100/88 shadow-lg backdrop-blur-xl xl:bottom-9 xl:left-9 xl:right-9">
            {[
              { icon: WalletCards, label: "Ứng lương", value: "Chủ động" },
              { icon: CalendarDays, label: "Lương tuần", value: "Đúng nhịp" },
              { icon: ShieldCheck, label: "Dữ liệu", value: "Bảo mật" },
            ].map(({ icon: Icon, label, value }, index) => (
              <div key={label} className={`flex items-center gap-3 px-4 py-4 xl:px-6 ${index > 0 ? "border-l border-base-300/80" : ""}`}>
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <Icon className="h-5 w-5" aria-hidden="true" />
                </span>
                <span className="min-w-0">
                  <span className="block text-[10px] font-bold uppercase tracking-[0.16em] text-base-content/45">{label}</span>
                  <span className="mt-0.5 block truncate text-sm font-bold text-base-content">{value}</span>
                </span>
              </div>
            ))}
          </div>
        </section>

        <section className="relative flex min-h-dvh flex-col bg-base-100 lg:min-h-[calc(100dvh-3rem)] lg:rounded-[2rem] lg:bg-base-100/96">
          <div className="relative h-56 overflow-hidden border-b border-base-300 lg:hidden">
            <img
              src="/login-payroll-hero.webp"
              alt="Ví lương và lịch trả lương tuần"
              className="h-full w-full object-cover object-[38%_68%]"
            />
            <div className="absolute inset-0 bg-gradient-to-b from-base-100/30 via-base-100/5 to-neutral/45" />
            <div className="absolute left-5 top-5 flex items-center gap-2.5 rounded-2xl border border-base-300/80 bg-base-100/92 px-3 py-2 shadow-sm backdrop-blur-md">
              <img src="/logo-square.png" alt="TingTing logo" className="h-9 w-9 object-contain" />
              <span className="font-display text-lg font-extrabold tracking-tight">TingTing</span>
            </div>
            <div className="absolute bottom-5 left-5 right-5 text-primary-content">
              <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-primary-content/75">Lương về đúng nhịp</p>
              <p className="mt-1 font-display text-2xl font-black leading-tight tracking-[-0.03em]">Ứng lương nhanh. Trả lương tuần.</p>
            </div>
          </div>

          <div className="flex flex-1 items-center justify-center px-4 py-8 sm:px-8 lg:px-10 xl:px-16">
            <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100 shadow-[0_24px_70px_-42px_rgba(13,63,39,0.48)] lg:border-0 lg:bg-transparent lg:shadow-none">
              <div className="ct-card-body gap-0 p-6 sm:p-8 lg:p-4">
                <div className="mb-7 hidden items-center gap-3 lg:flex">
                  <img src="/logo-square.png" alt="TingTing logo" className="h-11 w-11 object-contain" />
                  <div>
                    <p className="font-display text-xl font-extrabold leading-none tracking-tight">TingTing</p>
                    <p className="mt-1 text-[10px] font-bold uppercase tracking-[0.2em] text-base-content/45">Cổng lương thông minh</p>
                  </div>
                </div>

                <div className="mb-7">
                  <div className="mb-3 flex items-center gap-2 text-xs font-bold uppercase tracking-[0.18em] text-primary">
                    <span className="h-2 w-2 rounded-full bg-success shadow-[0_0_0_5px_hsl(var(--su)/0.12)]" />
                    Hệ thống đang hoạt động
                  </div>
                  <h2 className="font-display text-3xl font-black leading-tight tracking-[-0.035em] sm:text-4xl">Chào mừng trở lại</h2>
                  <p className="mt-2 text-sm leading-6 text-base-content/55">Đăng nhập để quản lý ứng lương và chu kỳ trả lương của bạn.</p>
                </div>

                {(loginMutation.error || googleLoginMutation.error) && (
                  <div role="alert" className="ct-alert ct-alert-error mb-5 items-start rounded-xl text-sm shadow-none animate-fade-in">
                    <AlertCircle className="h-5 w-5 shrink-0" aria-hidden="true" />
                    <span className="font-semibold leading-5">
                      {loginMutation.error
                        ? (((loginMutation.error as unknown) as ApiError)?.http_status === 429
                          ? (((loginMutation.error as unknown) as ApiError)?.message || "Quá nhiều lần đăng nhập. Vui lòng thử lại sau ít phút.")
                          : "Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.")
                        : (((googleLoginMutation.error as unknown) as ApiError)?.message || "Đăng nhập Google thất bại. Vui lòng thử lại.")}
                    </span>
                  </div>
                )}

                <button
                  type="button"
                  onClick={handleGoogleLogin}
                  disabled={isDisabled}
                  className="ct-btn ct-btn-outline ct-btn-lg group h-12 w-full justify-start rounded-xl border-base-300 bg-base-100 px-3 text-sm normal-case shadow-sm hover:border-base-content/20 hover:bg-base-200"
                >
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-base-300 bg-base-100">
                    {googleLoginMutation.isPending
                      ? <span className="ct-loading ct-loading-spinner ct-loading-sm text-base-content/45" />
                      : <GoogleIcon />}
                  </span>
                  <span className="flex-1 text-left font-bold">
                    {googleLoginMutation.isPending ? "Đang đăng nhập..." : "Tiếp tục bằng Google"}
                  </span>
                  <ChevronRight className="h-4 w-4 text-base-content/35 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
                </button>

                <div className="ct-divider my-5 text-[10px] font-bold uppercase tracking-[0.2em] text-base-content/35">hoặc đăng nhập bằng tài khoản</div>

                <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                  <fieldset className="space-y-4">
                    <legend className="sr-only">Thông tin đăng nhập</legend>

                    <div className="space-y-2">
                      <label htmlFor="emailOrUsername" className="block text-xs font-bold text-base-content/65">Tên đăng nhập</label>
                      <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:bg-base-100 focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
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
                      <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:bg-base-100 focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
                        <Lock className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
                        <input
                          id="password"
                          type={showPassword ? "text" : "password"}
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
                    className="ct-btn ct-btn-primary ct-btn-lg mt-2 h-12 w-full rounded-xl text-sm font-extrabold normal-case shadow-[0_12px_28px_-14px_rgba(8,120,62,0.8)]"
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

          <footer className="px-5 pb-6 text-center text-[11px] text-base-content/40 lg:px-10">
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
