import { Bell, Settings, LogOut, UserRound } from "lucide-react";
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
    <header className="sticky top-0 z-30 border-b border-[var(--employee-border)] bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/88">
      <div
        className="mx-auto grid max-w-lg grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-4 pb-3"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}
      >
        <div className="min-w-0">
          <h1 className="employee-type-header-name truncate text-[var(--employee-text)]" title={employeeName}>
            {employeeName || "bạn"}
          </h1>
          <p className="employee-type-header-date mt-0.5 truncate capitalize text-[var(--employee-text-secondary)] tabular-nums">{todayLabel}</p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            className="employee-icon-button relative"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell className="h-5 w-5" strokeWidth={2.1} />
            {unreadCount != null && unreadCount > 0 && (
              <span className="employee-type-notification-badge employee-notification-pop absolute right-1 top-1 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-[#D92D20] px-1 text-white ring-2 ring-white">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="employee-icon-button border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]"
                aria-label="Menu tài khoản"
              >
                <UserRound className="h-5 w-5" strokeWidth={2.1} />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="w-48 rounded-xl shadow-lg border-gray-100"
            >
              <DropdownMenuItem
                onClick={onChangePassword}
                className="gap-2.5 rounded-lg py-2.5"
              >
                <Settings className="w-4 h-4 text-gray-400" />
                <span className="type-body">Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={onLogout}
                className="gap-2.5 rounded-lg py-2.5 text-red-500"
              >
                <LogOut className="w-4 h-4" />
                <span className="type-body">Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
