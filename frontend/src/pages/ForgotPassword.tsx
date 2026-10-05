import { useState, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft, CheckCircle, Mail01, PhoneCall01 } from "@untitledui/icons";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useRequestPasswordReset } from "@/hooks/api/usePasswordReset";
import { useRequestZaloReset } from "@/hooks/api/useZaloReset";
import { AuthBrandMark } from "@/components/auth/AuthBrandMark";
import { AuthCard } from "@/components/auth/AuthCard";
import { AuthSpinner } from "@/components/auth/AuthSpinner";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";
import { cx } from "@/utils/cx";

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
      className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-muted px-4 py-6 text-fg-primary"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <AuthCard>
        <AuthBrandMark className="mb-6" />

        {!done ? (
          <>
            <div className="mb-5">
              <h2 className="font-display text-display-md font-semibold leading-tight tracking-tight text-fg-primary">
                Quên mật khẩu?
              </h2>
              <p className="mt-2 text-sm text-fg-tertiary">
                Nhập email hoặc số điện thoại đã đăng ký để đặt lại mật khẩu.
              </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4" noValidate>
              <div className="space-y-1.5">
                <Label htmlFor="identifier">Email hoặc số điện thoại</Label>
                <div className="relative">
                  {/* Swap icon based on detected input type */}
                  {isEmailMode ? (
                    <Mail01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
                  ) : (
                    <PhoneCall01 className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-fg-quaternary" aria-hidden="true" />
                  )}
                  <Input
                    id="identifier"
                    type="text"
                    inputMode={isEmailMode ? "email" : "tel"}
                    placeholder="email@cua-ban.vn  hoặc  0987 654 321"
                    value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)}
                    className="pl-9 sm:pl-9"
                    required
                    autoComplete="off"
                    disabled={pending}
                  />
                </div>
                {/* Channel hint — neutral; does NOT confirm the value exists in the DB. */}
                {identifier.trim() && (
                  <p className="text-xs text-fg-tertiary">
                    {isEmailMode
                      ? "📧 Nếu email tồn tại, liên kết đặt lại sẽ gửi qua email."
                      : "💬 Nếu số điện thoại tồn tại, mã OTP sẽ gửi qua Zalo."}
                  </p>
                )}
              </div>

              <Button type="submit" className="mt-2 w-full" disabled={pending || !identifier.trim()}>
                {pending ? (
                  <>
                    <AuthSpinner className="size-4" /> Đang gửi...
                  </>
                ) : (
                  "Gửi yêu cầu đặt lại"
                )}
              </Button>
            </form>

            <Button
              type="button"
              variant="ghost"
              onClick={() => navigate("/login", { replace: true })}
              className="mt-5 text-xs"
            >
              <ArrowLeft className="size-4" aria-hidden="true" />
              Quay lại đăng nhập
            </Button>
          </>
        ) : (
          <div className="py-2" role="status">
            <div className="mb-5 flex items-start gap-3">
              <FeaturedIcon icon={CheckCircle} color="success" theme="light" size="md" />
              <div>
                <p className="text-sm font-bold text-fg-primary">
                  Nếu email tồn tại trong hệ thống
                </p>
                <p className="mt-1 text-sm leading-5 text-fg-secondary">
                  Bạn sẽ nhận được liên kết đặt lại mật khẩu trong vài phút. Vui lòng kiểm tra hộp thư (kể cả thư rác).
                </p>
              </div>
            </div>
            <Link to="/login" className={cx(buttonVariants({ variant: "outline" }), "w-full")}>
              <ArrowLeft className="size-4" aria-hidden="true" />
              Quay lại đăng nhập
            </Link>
          </div>
        )}
      </AuthCard>
    </div>
  );
};

export default ForgotPassword;
