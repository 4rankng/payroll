import { useState, type ReactNode } from "react";
import { Bell, Eye, EyeOff, LogOut, Settings } from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { EmployeeHomeViewModel } from "@/utils/employeePortal/mobileHome";
import { formatCurrency } from "@/utils/formatters";

const MASKED_AMOUNT = "••••••";

interface EmployeeCanopyWalletProps {
  model: EmployeeHomeViewModel;
  paidAmount: number;
  totalAmount: number;
}

interface EmployeeCanopyProps {
  employeeName?: string;
  unreadCount?: number;
  onNotificationClick: () => void;
  onChangePassword: () => void;
  onLogout: () => void;
  /** Omit to render identity chrome only — no amount/progress block. */
  wallet?: EmployeeCanopyWalletProps;
  /**
   * Period scope for everything below the canopy — e.g. the month navigator.
   * Rendered on the emerald surface so period selection sits with the identity
   * chrome instead of occupying a separate white card.
   */
  periodSlot?: ReactNode;
}

export function EmployeeCanopy({
  employeeName,
  unreadCount,
  onNotificationClick,
  onChangePassword,
  onLogout,
  wallet,
  periodSlot,
}: EmployeeCanopyProps) {
  const [amountVisible, setAmountVisible] = useState(true);
  const todayLabel = format(new Date(), "EEEE, d 'tháng' M", { locale: vi });
  const paidAmount = wallet?.paidAmount ?? 0;
  const totalAmount = wallet?.totalAmount ?? 0;
  const progressPercent =
    totalAmount > 0 ? Math.min(100, Math.round((paidAmount / totalAmount) * 100)) : 0;
  const remainingAmount = Math.max(0, totalAmount - paidAmount);
  const showProgress = Boolean(wallet) && totalAmount > 0;

  return (
    <section
      className="relative isolate overflow-hidden rounded-b-[32px] text-white"
      style={{
        // Anchored on the TingTing brand emerald (--employee-accent #08783e).
        // A tight, low-chroma range reads as payroll rather than consumer app;
        // bright mint (#10b981) is deliberately out of the ramp.
        background: "linear-gradient(168deg, #0a7a41 0%, #08783e 42%, #065c32 78%, #054d2a 100%)",
        paddingTop: "calc(env(safe-area-inset-top, 0px) + 1rem)",
        paddingBottom: wallet || periodSlot ? "1.5rem" : "1.25rem",
      }}
      aria-label={wallet?.model.title ?? "Thông tin tài khoản"}
    >
      {/* One restrained highlight. Stacked blurred orbs made the surface look
          hazy and washed out instead of like a solid brand block. */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10"
        style={{
          background:
            "radial-gradient(120% 80% at 88% -20%, rgba(255,255,255,0.10) 0%, transparent 60%)",
        }}
      />

      <div className="mx-auto max-w-lg px-4 sm:px-5">
        {/* Identity row */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="employee-type-header-date text-white/80">{todayLabel}</p>
            <h1
              className="employee-type-header-name mt-0.5 break-words text-white"
              title={employeeName}
            >
              {employeeName || "bạn"}
            </h1>
          </div>
          <div className="flex shrink-0 items-center gap-2.5">
            <button
              type="button"
              onClick={onNotificationClick}
              aria-label="Thông báo"
              className="relative flex h-11 w-11 items-center justify-center rounded-2xl border border-white/20 bg-white/15 text-white/90 backdrop-blur-md transition-all duration-200 hover:bg-white/25 hover:scale-105 active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--employee-accent)]"
            >
              <Bell className="h-[1.125rem] w-[1.125rem]" strokeWidth={2} aria-hidden="true" />
              {unreadCount != null && unreadCount > 0 && (
                <span className="absolute -right-1 -top-1 flex h-5 min-w-[20px] items-center justify-center rounded-full bg-white px-1 text-[0.6875rem] font-bold leading-none text-[var(--employee-accent-strong)] ring-2 ring-[#065c32]">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </span>
              )}
            </button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="flex h-11 w-11 items-center justify-center rounded-2xl border border-white/20 bg-white/15 p-1 backdrop-blur-md transition-all duration-200 hover:bg-white/25 hover:scale-105 active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--employee-accent)]"
                  aria-label="Menu tài khoản"
                >
                  <img
                    src="/icons/employee-avatar.png"
                    alt=""
                    width={40}
                    height={40}
                    className="h-full w-full rounded-xl object-contain"
                  />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                className="w-52 rounded-2xl border border-slate-300 bg-white p-1.5 shadow-xl shadow-slate-900/10"
                data-employee-ui=""
                data-theme="employee"
              >
                <DropdownMenuItem onClick={onChangePassword} className="min-h-11 gap-3 rounded-xl py-2.5 text-slate-700 focus:bg-slate-50 focus:text-slate-900">
                  <Settings className="h-4 w-4 text-slate-500" />
                  <span className="employee-type-action">Đổi mật khẩu</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator className="my-1 bg-slate-100" />
                <DropdownMenuItem onClick={onLogout} className="min-h-11 gap-3 rounded-xl py-2.5 text-red-600 focus:bg-red-50 focus:text-red-700">
                  <LogOut className="h-4 w-4" />
                  <span className="employee-type-action">Đăng xuất</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Wallet section */}
        {wallet && (
          <div className="mt-6">
            <p className="employee-type-label text-white/80">{wallet.model.amountLabel}</p>
            <div className="mt-2 flex items-end justify-between gap-3">
              <p className="employee-type-hero-amount min-w-0 break-words text-white tabular-nums">
                {amountVisible ? wallet.model.amount : MASKED_AMOUNT}
              </p>
              <button
                type="button"
                onClick={() => setAmountVisible((visible) => !visible)}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-white/20 bg-white/15 text-white/80 backdrop-blur-md transition-all duration-200 hover:bg-white/25 hover:scale-105 active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--employee-accent)]"
                aria-label={amountVisible ? "Ẩn số tiền" : "Hiện số tiền"}
                aria-pressed={!amountVisible}
              >
                {amountVisible ? (
                  <Eye className="h-4 w-4" strokeWidth={2} aria-hidden="true" />
                ) : (
                  <EyeOff className="h-4 w-4" strokeWidth={2} aria-hidden="true" />
                )}
              </button>
            </div>

            {showProgress && (
              <div className="mt-5">
                {/* Progress bar with glass effect */}
                <div className="relative h-2 overflow-hidden rounded-full bg-white/20">
                  <div
                    className="absolute inset-y-0 left-0 rounded-full bg-white shadow-[0_0_12px_rgba(255,255,255,0.4)] transition-[width] duration-700 ease-out"
                    style={{ width: `${amountVisible ? progressPercent : 0}%` }}
                  />
                </div>
                <div className="mt-3 grid grid-cols-2 gap-3">
                  <div className="min-w-0">
                    <span className="employee-type-label block text-white/80">Đã nhận</span>
                    <span className="employee-type-action mt-1 block break-words tabular-nums text-white">
                      {amountVisible ? formatCurrency(paidAmount) : MASKED_AMOUNT}
                    </span>
                  </div>
                  <div className="min-w-0 text-right">
                    <span className="employee-type-label block text-white/80">Còn lại</span>
                    <span className="employee-type-action mt-1 block break-words tabular-nums text-white">
                      {amountVisible ? formatCurrency(remainingAmount) : MASKED_AMOUNT}
                    </span>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Period scope for the content below — sits on the canopy so the
            month selector does not need a competing white card of its own. */}
        {periodSlot && <div className={wallet ? "mt-5" : "mt-4"}>{periodSlot}</div>}
      </div>
    </section>
  );
}
