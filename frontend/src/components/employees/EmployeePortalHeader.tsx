import { Bell, Settings, LogOut } from "lucide-react";
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
    <header className="sticky top-0 z-30 border-b border-base-300/80 bg-base-100/90 backdrop-blur-xl supports-[backdrop-filter]:bg-base-100/80">
      <div
        className="ct-navbar mx-auto min-h-0 max-w-6xl gap-3 px-4 pb-3 sm:px-5 lg:px-8"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}
      >
        <div className="ct-navbar-start min-w-0 flex-1 gap-3">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="ct-avatar ct-placeholder h-11 w-11 shrink-0 rounded-[15px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 sm:h-12 sm:w-12"
                aria-label="Menu tài khoản"
              >
                <div className="h-11 w-11 overflow-hidden rounded-[15px] bg-primary/10 ring-1 ring-primary/15 sm:h-12 sm:w-12">
                  <img
                    src="/icons/employee-avatar.png"
                    alt=""
                    width={48}
                    height={48}
                    className="h-full w-full object-cover"
                  />
                </div>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="w-48 rounded-xl border-base-300 shadow-lg"
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
          <div className="min-w-0">
            <div className="flex min-w-0 items-baseline gap-2">
              <h1 className="employee-type-header-name truncate text-base-content" title={employeeName}>
                {employeeName || "bạn"}
              </h1>
              <span className="hidden h-1 w-1 shrink-0 rounded-full bg-secondary sm:block" aria-hidden="true" />
              <p className="employee-type-header-date hidden truncate capitalize text-base-content/55 tabular-nums sm:block">{todayLabel}</p>
            </div>
            <p className="employee-type-header-date mt-0.5 truncate capitalize text-base-content/55 tabular-nums sm:hidden">{todayLabel}</p>
          </div>
        </div>
        <div className="ct-navbar-end w-auto shrink-0 gap-2">
          <button
            type="button"
            className="ct-btn ct-btn-ghost ct-btn-circle ct-indicator employee-icon-button"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell className="h-5 w-5" strokeWidth={2.1} />
            {unreadCount != null && unreadCount > 0 && (
              <span className="ct-indicator-item ct-badge ct-badge-error ct-badge-sm employee-type-notification-badge employee-notification-pop min-w-[18px] px-1 text-error-content ring-2 ring-base-100">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
}
