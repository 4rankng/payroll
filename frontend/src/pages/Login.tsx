import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Lock, User, Eye, EyeOff, AlertCircle, Loader2, ChevronRight, RefreshCw } from "lucide-react";
import { authManager } from "@/lib/auth";
import { useAuth } from "@/contexts";
import { useLogin, useGoogleLogin } from "@/hooks/api/useAuth";
import { Alert, AlertDescription } from "@/components/ui/alert";
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
  const loginMutation = useLogin();
  const loginError = loginMutation.error;
  const resetLogin = loginMutation.reset;
  const googleLoginMutation = useGoogleLogin();
  const navigate = useNavigate();
  const { isAuthenticated, isLoading } = useAuth();
  const needsCaptcha = failedAttempts >= 3;

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

  const isDisabled = loginMutation.isPending || googleLoginMutation.isPending;

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const serverRequiresCaptcha = await syncCaptchaRequirement();
      if (serverRequiresCaptcha && (!captchaId || !captchaCode)) {
        if (!captchaId) await refreshCaptcha();
        return;
      }
      loginMutation.mutate({
        username: emailOrUsername,
        password,
        ...((needsCaptcha || serverRequiresCaptcha) && captchaId
          ? { captcha_id: captchaId, captcha_code: captchaCode }
          : {}),
      });
    },
    [emailOrUsername, password, loginMutation, needsCaptcha, captchaId, captchaCode, refreshCaptcha, syncCaptchaRequirement]
  );

  const togglePassword = useCallback(() => setShowPassword((v) => !v), []);

  if (isLoading) {
    return <div className="min-h-screen flex items-center justify-center bg-white"><Loader2 className="h-8 w-8 animate-spin text-primary" /></div>;
  }

  if (isAuthenticated) {
    return null;
  }

  return (
    <div
      id="main-content"
      className="min-h-[100dvh] w-full flex items-center justify-center relative overflow-hidden"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      {/* Background */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat"
        style={{ backgroundImage: "url('/login-bg-employee.png')" }}
      />
      <div className="absolute inset-0 bg-gradient-to-br from-white/5 via-white/20 to-emerald-50/20" />

      {/* Brand Logo — desktop: top left */}
      <div className="hidden md:absolute md:z-20 md:top-6 md:left-6 md:flex md:items-center md:gap-3 md:animate-fade-in-up md:delay-100">
        <img src="/logo-square.png" alt="TingTing logo" className="h-10 w-10 rounded-lg object-contain" />
        <h1 className="font-display font-extrabold text-xl text-emerald-800 tracking-tight">TingTing</h1>
      </div>

      <div className="relative z-10 w-full max-w-[420px] flex flex-col items-center px-4 py-6 md:py-0">

        {/* Brand Logo — mobile: above card */}
        <div className="flex md:hidden items-center gap-3 mb-7 animate-fade-in-up delay-100">
          <img src="/logo-square.png" alt="TingTing logo" className="h-11 w-11 rounded-xl object-contain shadow-lg" />
          <h1 className="font-display font-extrabold text-2xl text-emerald-800 tracking-tight">TingTing</h1>
        </div>

        {/* Login Card */}
        <div className="w-full animate-hero-reveal delay-200 rounded-2xl overflow-hidden shadow-[0_24px_64px_rgba(0,120,70,0.16)] bg-white">

          {/* Card header */}
          <div className="bg-white px-8 pt-8 pb-6">
            <h2 className="font-display font-bold text-[22px] text-gray-900 tracking-tight mb-1">
              Chào mừng trở lại
            </h2>
            <p className="text-sm text-gray-400 font-normal">
              Đăng nhập để truy cập bảng lương của bạn
            </p>
          </div>

          {/* Error alerts */}
          {(loginMutation.error || googleLoginMutation.error) && (
            <div className="bg-white px-8 pb-1">
              <Alert variant="destructive" className="bg-red-50 border-red-200 text-red-700 animate-fade-in">
                <AlertCircle className="h-4 w-4 text-red-500" />
                <AlertDescription className="font-medium text-sm">
                  {loginMutation.error
                    ? (((loginMutation.error as unknown) as ApiError)?.http_status === 429
                      ? (((loginMutation.error as unknown) as ApiError)?.message || "Quá nhiều lần đăng nhập. Vui lòng thử lại sau ít phút.")
                      : "Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.")
                    : (((googleLoginMutation.error as unknown) as ApiError)?.message || "Đăng nhập Google thất bại. Vui lòng thử lại.")}
                </AlertDescription>
              </Alert>
            </div>
          )}

          {/* Google button — PRIMARY action */}
          <div className="bg-white px-8 pb-6">
            <button
              type="button"
              onClick={handleGoogleLogin}
              disabled={isDisabled}
              className="group relative w-full h-[52px] flex items-center gap-3.5 px-4 rounded-xl
                bg-white border border-gray-200
                shadow-[0_1px_4px_rgba(0,0,0,0.07),0_2px_8px_rgba(0,0,0,0.04)]
                hover:shadow-[0_4px_20px_rgba(0,0,0,0.12)]
                hover:border-gray-300
                hover:-translate-y-[1px]
                transition-all duration-150 ease-out
                disabled:pointer-events-none disabled:opacity-40
                cursor-pointer select-none"
            >
              {/* Icon badge */}
              <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-gray-50 border border-gray-100 shrink-0">
                {googleLoginMutation.isPending
                  ? <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
                  : <GoogleIcon />}
              </div>
              <span className="flex-1 text-left text-sm font-semibold text-gray-700">
                {googleLoginMutation.isPending ? "Đang đăng nhập..." : "Tiếp tục bằng Google"}
              </span>
              <ChevronRight className="h-4 w-4 text-gray-300 group-hover:text-gray-400 group-hover:translate-x-px transition-all duration-150 shrink-0" />
            </button>
          </div>

          {/* Divider */}
          <div className="flex items-center bg-white px-8 mb-5">
            <div className="h-px flex-1 bg-gray-100" />
            <span className="px-4 text-xs font-medium text-gray-400 tracking-wide">hoặc</span>
            <div className="h-px flex-1 bg-gray-100" />
          </div>

          {/* Username / password form */}
          <div className="bg-white px-8 pb-8">
            <form onSubmit={handleSubmit} className="space-y-4" noValidate>
              <div className="space-y-1.5">
                <Label htmlFor="emailOrUsername" className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Tên đăng nhập
                </Label>
                <div className="relative group">
                  <User className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-300 group-focus-within:text-primary transition-colors pointer-events-none z-10" />
                  <Input
                    id="emailOrUsername"
                    type="text"
                    placeholder="CCCD, số điện thoại hoặc tên đăng nhập"
                    value={emailOrUsername}
                    onChange={(e) => setEmailOrUsername(e.target.value)}
                    onBlur={() => void syncCaptchaRequirement()}
                    className="premium-input pl-10 pr-4 h-11 text-sm bg-gray-50 border-gray-200 focus:bg-white focus:border-primary/50 transition-colors"
                    required
                    autoComplete="username"
                    autoCapitalize="none"
                    disabled={isDisabled}
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="password" className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Mật khẩu
                </Label>
                <div className="relative group">
                  <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-300 group-focus-within:text-primary transition-colors pointer-events-none z-10" />
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    placeholder="Nhập mật khẩu của bạn"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="premium-input pl-10 pr-12 h-11 text-sm bg-gray-50 border-gray-200 focus:bg-white focus:border-primary/50 transition-colors"
                    required
                    autoComplete="current-password"
                    disabled={isDisabled}
                  />
                  <button
                    type="button"
                    onClick={togglePassword}
                    aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
                    className="absolute right-3 top-1/2 -translate-y-1/2 h-8 w-8 flex items-center justify-center text-gray-300 hover:text-gray-500 rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                    tabIndex={0}
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </div>

              {/* CAPTCHA — shown after 3 failed login attempts */}
              {needsCaptcha && (
                <div className="space-y-1.5 animate-fade-in-up">
                  <Label htmlFor="captchaCode" className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                    Mã xác nhận
                  </Label>
                  <div className="flex items-center gap-3">
                    {/* Image + inline loading skeleton. Same h-11 + rounded-lg as
                        the inputs so the row stays visually aligned. */}
                    {captchaImage ? (
                      <img
                        src={captchaImage}
                        alt="Hình ảnh mã xác nhận 5 chữ số — nhấn để tải hình khác"
                        className="h-11 rounded-lg border border-gray-200 bg-white select-none"
                        onClick={() => refreshCaptcha()}
                        style={{ cursor: "pointer" }}
                      />
                    ) : (
                      <div
                        className="h-11 rounded-lg border border-gray-200 bg-gray-50 flex items-center justify-center"
                        aria-hidden="true"
                      >
                        <Loader2 className="h-4 w-4 animate-spin text-gray-300" />
                      </div>
                    )}
                    <Input
                      id="captchaCode"
                      type="text"
                      inputMode="numeric"
                      placeholder="Nhập mã"
                      value={captchaCode}
                      onChange={(e) => setCaptchaCode(e.target.value.replace(/\D/g, "").slice(0, 5))}
                      className="premium-input flex-1 h-11 text-sm tracking-widest text-center bg-gray-50 border-gray-200 focus:bg-white focus:border-primary/50 transition-colors"
                      autoComplete="off"
                      required={needsCaptcha}
                      disabled={isDisabled || captchaLoading}
                    />
                    <button
                      type="button"
                      onClick={() => refreshCaptcha()}
                      disabled={captchaLoading || isDisabled}
                      aria-label="Tải hình mã xác nhận khác"
                      className="shrink-0 h-11 w-11 flex items-center justify-center rounded-lg border border-gray-200 bg-gray-50 text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:opacity-40 disabled:pointer-events-none"
                    >
                      <RefreshCw className={`h-4 w-4 ${captchaLoading ? "animate-spin" : ""}`} />
                    </button>
                  </div>
                </div>
              )}

              <Button
                type="submit"
                variant="default"
                className="w-full h-11 font-bold text-sm rounded-xl mt-2 premium-button bg-employee-600 text-white shadow-[0_8px_24px_rgba(0,158,69,0.22)] hover:bg-employee-700 hover:shadow-[0_10px_28px_rgba(0,122,55,0.28)]"
                disabled={isDisabled}
              >
                {loginMutation.isPending
                  ? <><Loader2 className="h-4 w-4 animate-spin mr-2" />Đang đăng nhập...</>
                  : "Đăng nhập"}
              </Button>
            </form>
          </div>

          {/* Card footer */}
          <div className="bg-gray-50 border-t border-gray-100 px-8 py-4">
            <p className="text-xs text-gray-400 text-center whitespace-nowrap">
              Ứng lương với{" "}
              <span className="font-semibold text-primary">TingTing</span>
              {" · "}An toàn{" · "}Bảo mật{" · "}Tiện lợi
            </p>
          </div>
        </div>
      </div>

      {/* Page footer */}
      <div className="absolute bottom-6 left-0 right-0 z-10">
        <div className="flex flex-col items-center gap-1.5">
          <p className="text-center text-xs text-gray-600 font-medium">
            © {new Date().getFullYear()} TingTing. Enterprise Payroll Solutions.
          </p>
          <div className="flex items-center gap-3 text-xs text-gray-500">
            <span>Điều khoản</span>
            <span>·</span>
            <span>Quyền riêng tư</span>
            <span>·</span>
            <span>Hỗ trợ</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;
