import { useState, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Mail, Phone, ArrowLeft, Loader2, CheckCircle2 } from "lucide-react";
import { useRequestPasswordReset } from "@/hooks/api/usePasswordReset";
import { useRequestZaloReset } from "@/hooks/api/useZaloReset";

/**
 * ForgotPassword — step 1 of the self-service password-reset flow.
 *
 * A single input field accepts either an email or a mobile number. The system
 * auto-detects which and routes accordingly:
 *   - **Email** (contains @) → Resend magic-link flow → "check your inbox" state.
 *   - **Mobile** (digits) → Zalo ZNS 6-digit OTP → navigate to /zalo-reset-password.
 *
 * Both channels only send to values registered in the database; unknown inputs
 * get the same generic success response (anti-enumeration). The success/advance
 * state is shown regardless of network outcome so a timing/connection
 * difference can't leak which emails/mobiles are registered.
 */
const ForgotPassword = () => {
  const [identifier, setIdentifier] = useState("");
  const [done, setDone] = useState(false);
  const navigate = useNavigate();

  const emailMutation = useRequestPasswordReset();
  const zaloMutation = useRequestZaloReset();

  /**
   * Detect whether the trimmed input is an email or a mobile number.
   * Email = contains "@". Anything else is treated as a mobile (digits
   * are extracted server-side by NormalizePhone).
   */
  const isEmail = (val: string): boolean => val.includes("@");

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = identifier.trim();
      if (!trimmed) return;

      if (isEmail(trimmed)) {
        // Email → Resend magic link. Stay on page, show "check inbox" state.
        try {
          await emailMutation.mutateAsync(trimmed);
        } catch {
          // Swallow: success state shown either way (anti-enumeration).
        } finally {
          setDone(true);
        }
      } else {
        // Mobile → Zalo OTP. Navigate to the OTP entry page.
        let sid = "";
        try {
          const res = await zaloMutation.mutateAsync(trimmed);
          sid = res.data?.otp_session_id ?? "";
        } catch {
          // Anti-enumeration: advance anyway. A dummy/empty session id means
          // /zalo-reset-password will reject the confirm with the same
          // "invalid code" message a real wrong-code would produce.
        }
        // Navigate with router state (NOT a query string) so the session id
        // doesn't leak via Referer or browser history.
        navigate("/zalo-reset-password", {
          state: { otp_session_id: sid, mobile: trimmed },
          replace: true,
        });
      }
    },
    [identifier, emailMutation, zaloMutation, navigate],
  );

  const isEmailMode = isEmail(identifier.trim());
  const pending = emailMutation.isPending || zaloMutation.isPending;

  return (
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
            <div>
              <p className="font-display text-[1.35rem] font-black leading-none tracking-[-0.03em]">TingTing</p>
            </div>
          </div>

          {!done ? (
            <>
              <div className="mb-5">
                <h2 className="font-display text-3xl font-black leading-tight tracking-[-0.04em] sm:text-4xl">
                  Quên mật khẩu?
                </h2>
                <p className="mt-2 text-sm text-base-content">
                  Nhập email hoặc số điện thoại đã đăng ký để đặt lại mật khẩu.
                </p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                <div className="space-y-2">
                  <label htmlFor="identifier" className="block text-xs font-bold text-base-content">
                    Email hoặc số điện thoại
                  </label>
                  <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
                    {/* Swap icon based on detected input type */}
                    {isEmailMode ? (
                      <Mail className="h-4 w-4 shrink-0 text-base-content" aria-hidden="true" />
                    ) : (
                      <Phone className="h-4 w-4 shrink-0 text-base-content" aria-hidden="true" />
                    )}
                    <input
                      id="identifier"
                      type="text"
                      inputMode={isEmailMode ? "email" : "tel"}
                      placeholder="email@cua-ban.vn  hoặc  0987 654 321"
                      value={identifier}
                      onChange={(e) => setIdentifier(e.target.value)}
                      className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content"
                      required
                      autoComplete="off"
                      disabled={pending}
                    />
                  </label>
                  {/* Channel hint — neutral; does NOT confirm the value exists in the DB. */}
                  {identifier.trim() && (
                    <p className="text-xs text-base-content">
                      {isEmailMode
                        ? "📧 Nếu email tồn tại, liên kết đặt lại sẽ gửi qua email."
                        : "💬 Nếu số điện thoại tồn tại, mã OTP sẽ gửi qua Zalo."}
                    </p>
                  )}
                </div>

                <button
                  type="submit"
                  className="ct-btn ct-btn-primary ct-btn-lg mt-2 h-12 w-full rounded-xl text-sm font-extrabold normal-case"
                  disabled={pending || !identifier.trim()}
                >
                  {pending ? (
                    <><Loader2 className="h-4 w-4 animate-spin" /> Đang gửi...</>
                  ) : (
                    "Gửi yêu cầu đặt lại"
                  )}
                </button>
              </form>

              <button
                type="button"
                onClick={() => navigate("/login", { replace: true })}
                className="mt-5 flex items-center gap-1.5 text-xs font-bold text-base-content hover:text-base-content"
              >
                <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
                Quay lại đăng nhập
              </button>
            </>
          ) : (
            <div className="py-2" role="status">
              <div className="mb-5 flex items-start gap-3 rounded-xl border border-success/30 bg-success/10 p-4">
                <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-success" aria-hidden="true" />
                <div>
                  <p className="text-sm font-bold text-base-content">
                    Nếu email tồn tại trong hệ thống
                  </p>
                  <p className="mt-1 text-sm leading-5 text-base-content">
                    Bạn sẽ nhận được liên kết đặt lại mật khẩu trong vài phút. Vui lòng kiểm tra hộp thư (kể cả thư rác).
                  </p>
                </div>
              </div>
              <Link
                to="/login"
                className="ct-btn ct-btn-outline ct-btn-lg flex h-12 w-full items-center justify-center gap-2 rounded-xl border-base-300 text-sm font-bold"
              >
                <ArrowLeft className="h-4 w-4" aria-hidden="true" />
                Quay lại đăng nhập
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ForgotPassword;
