import { useState, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Mail, AlertCircle, ArrowLeft, Loader2, CheckCircle2 } from "lucide-react";
import { useRequestPasswordReset } from "@/hooks/api/usePasswordReset";

/**
 * ForgotPassword — step 1 of the self-service password-reset flow.
 *
 * The user enters their email; the backend ALWAYS returns the same success
 * message whether or not the email exists (anti-enumeration). This page honors
 * that contract client-side too: the success state is shown regardless of
 * success or network error, so a timing/connection difference can't leak which
 * emails are registered.
 */
const ForgotPassword = () => {
  const [email, setEmail] = useState("");
  const [done, setDone] = useState(false);
  const navigate = useNavigate();
  const mutation = useRequestPasswordReset();

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = email.trim();
      if (!trimmed) return;
      try {
        await mutation.mutateAsync(trimmed);
      } catch {
        // Swallow: the success state is shown either way (anti-enumeration).
      } finally {
        // Flip to the success state regardless of outcome. The backend always
        // returns 200; a network error must not reveal whether the email exists.
        setDone(true);
      }
    },
    [email, mutation],
  );

  return (
    <div
      data-admin-ui=""
      data-theme="congtruong"
      className="relative flex min-h-dvh w-full items-center justify-center overflow-x-hidden bg-base-200 px-4 py-6 text-base-content"
      style={{ paddingTop: "env(safe-area-inset-top)", paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <div className="ct-card w-full max-w-[470px] border border-base-300 bg-base-100 shadow-[0_24px_70px_-42px_hsl(var(--neutral)/0.38)]">
        <div className="ct-card-body gap-0 p-6 sm:p-8">
          <div className="mb-6 flex items-center gap-3">
            <img src="/logo-square.png" alt="TingTing logo" className="h-12 w-12 object-contain" />
            <div>
              <p className="font-display text-[1.35rem] font-black leading-none tracking-[-0.03em]">TingTing</p>
            </div>
          </div>

          {!done ? (
            <>
              <div className="mb-6">
                <h2 className="font-display text-3xl font-black leading-tight tracking-[-0.04em] sm:text-4xl">
                  Quên mật khẩu?
                </h2>
                <p className="mt-2 text-sm text-base-content/60">
                  Nhập email đăng ký, chúng tôi sẽ gửi liên kết đặt lại mật khẩu.
                </p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                <div className="space-y-2">
                  <label htmlFor="email" className="block text-xs font-bold text-base-content/65">
                    Email
                  </label>
                  <label className="ct-input ct-input-bordered flex h-12 w-full items-center gap-3 rounded-xl border-base-300 bg-base-200/55 px-4 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary/10">
                    <Mail className="h-4 w-4 shrink-0 text-base-content/35" aria-hidden="true" />
                    <input
                      id="email"
                      type="email"
                      placeholder="email@cua-ban.vn"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-base-content/35"
                      required
                      autoComplete="email"
                      autoCapitalize="none"
                      disabled={mutation.isPending}
                    />
                  </label>
                </div>

                <button
                  type="submit"
                  className="ct-btn ct-btn-primary ct-btn-lg mt-2 h-12 w-full rounded-xl text-sm font-extrabold normal-case shadow-[0_12px_28px_-14px_rgba(8,120,62,0.8)]"
                  disabled={mutation.isPending || !email.trim()}
                >
                  {mutation.isPending ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Đang gửi...
                    </>
                  ) : (
                    "Gửi liên kết đặt lại"
                  )}
                </button>
              </form>

              <button
                type="button"
                onClick={() => navigate("/login", { replace: true })}
                className="mt-5 flex items-center gap-1.5 text-xs font-bold text-base-content/55 hover:text-base-content"
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
                  <p className="mt-1 text-sm leading-5 text-base-content/70">
                    Bạn sẽ nhận được hướng dẫn đặt lại mật khẩu trong vài phút. Vui lòng kiểm tra hộp thư (kể cả thư rác).
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
