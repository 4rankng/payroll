import { Bell, Settings, LogOut, UserCircle } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";

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
    <div style={{ background: EMPLOYEE_BRAND_COLOR }}>
      <div className="max-w-2xl mx-auto px-4 py-3.5 flex items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs font-medium text-white/70 uppercase tracking-wider">
            Xin chào
          </p>
          <h1 className="text-lg font-bold text-white truncate leading-tight">
            {employeeName || "bạn"}
          </h1>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            className="relative h-9 w-9 flex items-center justify-center rounded-full hover:bg-white/20 transition-colors text-white"
            onClick={onNotificationClick}
            aria-label="Thông báo"
          >
            <Bell className="h-5 w-5" />
            {unreadCount != null && unreadCount > 0 && (
              <span className="absolute top-1 right-1 h-3.5 w-3.5 rounded-full bg-red-500 text-white text-[8px] font-bold flex items-center justify-center">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="rounded-full h-9 w-9 flex items-center justify-center hover:bg-white/20 transition-colors"
                aria-label="Menu tài khoản"
              >
                <UserCircle className="h-7 w-7 text-white/80" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              className="w-48 rounded-xl shadow-lg border-gray-100"
            >
              <DropdownMenuItem
                onClick={onChangePassword}
                className="py-2 gap-2.5 rounded-lg"
              >
                <Settings className="w-4 h-4 text-gray-400" />
                <span className="text-sm">Đổi mật khẩu</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={onLogout}
                className="text-red-500 py-2 gap-2.5 rounded-lg"
              >
                <LogOut className="w-4 h-4" />
                <span className="text-sm">Đăng xuất</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
