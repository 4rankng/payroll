import { useState } from "react";
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

const CANOPY_GRADIENT =
  "linear-gradient(158deg, #0b9a53 0%, #08783e 52%, #065c32 100%)";

interface EmployeeCanopyProps {
  employeeName?: string;
  unreadCount?: number;
  onNotificationClick: () => void;
  onChangePassword: () => void;
  onLogout: () => void;
  model: EmployeeHomeViewModel;
  paidAmount: number;
  totalAmount: number;
}

export function EmployeeCanopy({
  employeeName,
  unreadCount,
  onNotificationClick,
  onChangePassword,
  onLogout,
  model,
  paidAmount,
  totalAmount,
}: EmployeeCanopyProps) {
  const [amountVisible, setAmountVisible] = useState(true);
  const todayLabel = format(new Date(), "EEEE, d 'tháng' M", { locale: vi });
  const progressPercent =
    totalAmount > 0 ? Math.min(100, Math.round((paidAmount / totalAmount) * 100)) : 0;
  const remainingAmount = Math.max(0, totalAmount - paidAmount);
  const showProgress = totalAmount > 0;

  return (
    <section
      className="relative isolate overflow-hidden rounded-b-[28px] px-4 pb-6 text-white"
      style={{
        backgroundImage: CANOPY_GRADIENT,
        paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.875rem)",
      }}
      aria-label={model.title}
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(circle_at_86%_-14%,rgba(255,255,255,0.28),transparent_58%)]"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-16 -top-20 -z-10 h-52 w-52 rounded-full bg-white/8 blur-2xl"
      />

      <div className="mx-auto flex max-w-lg items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="employee-type-body-sm truncate capitalize text-white/60">{todayLabel}</p>
          <h1 className="employee-type-header-name truncate text-white" title={employeeName}>
            {employeeName || "bạn"}
          </h1>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            onClick={onNotificationClick}
            aria-label="Thông báo"
            className="relative flex h-11 w-11 items-center justify-center rounded-xl border border-white/18 bg-white/14 text-white/90 backdrop-blur-sm transition-colors hover:bg-white/22"
          >
            <Bell className="h-5 w-5" strokeWidth={2.1} aria-hidden="true" />
            {unreadCount != null && unreadCount > 0 && (
              <span className="absolute -top-1.5 -right-1.5 min-w-[18px] rounded-full bg-[var(--employee-warning-border)] px-1 text-[11px] font-semibold leading-[18px] text-[var(--employee-warning-strong)]">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="flex h-11 w-11 items-center justify-center rounded-xl border border-white/18 bg-white/14 p-1 backdrop-blur-sm transition-colors hover:bg-white/22"
                aria-label="Menu tài khoản"
              >
                <img
                  src="/icons/employee-avatar.png"
                  alt=""
                  width={40}
                  height={40}
                  className="h-full w-full rounded-lg object-contain"
                />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="w-48 rounded-xl border-base-300 shadow-none"
              data-employee-ui=""
              data-theme="employee"
            >
              <DropdownMenuItem onClick={onChangePassword} className="gap-2.5 rounded-lg py-2.5">
                <Settings className="h-4 w-4 text-base-content/50" />
                <span className="type-body">Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onLogout} className="gap-2.5 rounded-lg py-2.5 text-error">
                <LogOut className="h-4 w-4" />
                <span className="type-body">Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="mx-auto mt-5 max-w-lg">
        <p className="employee-type-label text-white/65">{model.amountLabel}</p>
        <div className="mt-1.5 flex items-end justify-between gap-3">
          <p className="employee-type-hero-amount truncate text-white tabular-nums [text-shadow:0_2px_12px_rgba(0,0,0,0.18)]">
            {amountVisible ? model.amount : MASKED_AMOUNT}
          </p>
          <button
            type="button"
            onClick={() => setAmountVisible((visible) => !visible)}
            className="mb-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-white/18 bg-white/14 text-white/80 backdrop-blur-sm transition-colors hover:bg-white/22"
            aria-label={amountVisible ? "Ẩn số tiền" : "Hiện số tiền"}
            aria-pressed={!amountVisible}
          >
            {amountVisible ? (
              <Eye className="h-4 w-4" strokeWidth={1.9} aria-hidden="true" />
            ) : (
              <EyeOff className="h-4 w-4" strokeWidth={1.9} aria-hidden="true" />
            )}
          </button>
        </div>

        {showProgress && (
          <div className="mt-4">
            <div className="h-1.5 overflow-hidden rounded-full bg-white/20">
              <div
                className="h-full rounded-full bg-white transition-[width]"
                style={{ width: `${amountVisible ? progressPercent : 0}%` }}
              />
            </div>
            <div className="mt-2.5 flex items-baseline justify-between gap-3">
              <p className="employee-type-label-caps text-white/60">
                Đã nhận
                <span className="employee-type-inline-amount ml-1.5 block text-white tabular-nums sm:inline">
                  {amountVisible ? formatCurrency(paidAmount) : MASKED_AMOUNT}
                </span>
              </p>
              <p className="employee-type-label-caps text-right text-white/60">
                Còn lại
                <span className="employee-type-inline-amount ml-1.5 block text-white tabular-nums sm:inline">
                  {amountVisible ? formatCurrency(remainingAmount) : MASKED_AMOUNT}
                </span>
              </p>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}
