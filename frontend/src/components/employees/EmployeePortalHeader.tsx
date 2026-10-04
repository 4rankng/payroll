import { Bell01, LogOut01, Settings01 } from "@untitledui/icons";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface EmployeePortalHeaderProps {
  employeeName?: string;
  unreadCount?: number;
  onNotificationClick: () => void;
  onChangePassword: () => void;
  onLogout: () => void;
}

export function EmployeePortalHeader({
  employeeName,
  unreadCount,
  onNotificationClick,
  onChangePassword,
  onLogout,
}: EmployeePortalHeaderProps) {
  const todayLabel = format(new Date(), "EEEE, d 'tháng' M", { locale: vi });

  return (
    <header className="sticky top-0 z-30 border-b border-[var(--employee-border)] bg-[var(--employee-surface)]">
      <div
        className="mx-auto flex min-h-0 max-w-6xl items-center justify-between gap-3 px-4 pb-3 sm:px-5 lg:px-8"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}
      >
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] p-1 transition-colors hover:border-[var(--employee-accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2 sm:h-12 sm:w-12"
                aria-label="Menu tài khoản"
              >
                <img
                  src="/icons/employee-avatar.png"
                  alt=""
                  width={48}
                  height={48}
                  className="h-full w-full rounded-full object-contain"
                />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="w-48 rounded-xl"
              data-employee-ui=""
              data-theme="employee"
            >
              <DropdownMenuItem onClick={onChangePassword} className="min-h-11 gap-2.5 rounded-lg py-2.5">
                <Settings01 className="h-4 w-4 text-fg-tertiary" aria-hidden="true" />
                <span className="employee-type-action">Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onLogout} className="min-h-11 gap-2.5 rounded-lg py-2.5 text-[var(--employee-error)] focus:text-[var(--employee-error)]">
                <LogOut01 className="h-4 w-4" aria-hidden="true" />
                <span className="employee-type-action">Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <div className="min-w-0">
            <div className="flex min-w-0 items-baseline gap-2">
              <h1 className="employee-type-header-name truncate text-[var(--employee-text)]" title={employeeName}>
                {employeeName || "bạn"}
              </h1>
              <span className="hidden h-1 w-1 shrink-0 rounded-full bg-utility-gray-300 sm:block" aria-hidden="true" />
              <p className="employee-type-header-date hidden truncate capitalize text-[var(--employee-text-secondary)] tabular-nums sm:block">{todayLabel}</p>
            </div>
            <p className="employee-type-header-date mt-0.5 truncate capitalize text-[var(--employee-text-secondary)] tabular-nums sm:hidden">{todayLabel}</p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            className="employee-icon-button relative !rounded-full"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell01 className="h-5 w-5" strokeWidth={2.1} aria-hidden="true" />
            {unreadCount != null && unreadCount > 0 && (
              <span className="employee-type-notification-badge employee-notification-pop absolute -right-1 -top-1 flex h-5 min-w-[20px] items-center justify-center rounded-full bg-[var(--employee-error)] px-1 text-white ring-2 ring-[var(--employee-surface)]">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
}
