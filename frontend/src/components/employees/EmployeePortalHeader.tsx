import { Bell, Settings, LogOut, UserCircle } from "lucide-react";
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
  return (
    <div className="bg-employee">
      <div
        className="max-w-2xl mx-auto px-4 pb-4 flex items-center justify-between gap-3"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.875rem)" }}
      >
        <div className="min-w-0">
          <p className="text-[16px] font-semibold leading-6 text-white/85">
            Xin chào
          </p>
          <h1 className="truncate text-[26px] font-bold leading-8 text-white">
            {employeeName || "bạn"}
          </h1>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            className="relative flex h-11 w-11 items-center justify-center rounded-full text-white transition-colors hover:bg-white/15"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell className="h-6 w-6" strokeWidth={2.1} />
            {unreadCount != null && unreadCount > 0 && (
              <span className="absolute top-1.5 right-1.5 h-4 min-w-4 rounded-full bg-red-500 px-1 text-white text-[9px] font-bold flex items-center justify-center">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="flex h-11 w-11 items-center justify-center rounded-full transition-colors hover:bg-white/15"
                aria-label="Menu tài khoản"
              >
                <UserCircle className="h-8 w-8 text-white/90" strokeWidth={2.1} />
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
                <span className="text-[15px]">Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={onLogout}
                className="gap-2.5 rounded-lg py-2.5 text-red-500"
              >
                <LogOut className="w-4 h-4" />
                <span className="text-[15px]">Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
