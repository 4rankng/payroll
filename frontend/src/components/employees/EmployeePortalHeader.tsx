import { Bell, Settings, LogOut, UserCircle } from "lucide-react";
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
    <header className="border-b border-[#E4E7EC] bg-white">
      <div
        className="mx-auto flex min-h-16 max-w-lg items-center justify-between gap-3 px-4 py-2"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.5rem)" }}
      >
        <div className="min-w-0">
          <h1 className="employee-type-header-name truncate text-[#101828]">
            {employeeName || "bạn"}
          </h1>
          <p className="employee-type-body-sm mt-0.5 capitalize text-[#667085]">{todayLabel}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            className="relative flex h-11 w-11 items-center justify-center rounded-full text-[#475467] transition-colors hover:bg-[#F2F4F7] focus-visible:ring-2 focus-visible:ring-[#07883F]"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell className="h-5 w-5" strokeWidth={2.1} />
            {unreadCount != null && unreadCount > 0 && (
              <span className="type-count-badge absolute top-1.5 right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-white">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="flex h-11 w-11 items-center justify-center rounded-full bg-[#07883F] text-white transition-colors hover:bg-[#067647] focus-visible:ring-2 focus-visible:ring-[#07883F] focus-visible:ring-offset-2"
                aria-label="Menu tài khoản"
              >
                <UserCircle className="h-6 w-6" strokeWidth={2.1} />
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
