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
        className="mx-auto flex max-w-lg items-center justify-between gap-3 px-3 pb-3 min-[390px]:px-4"
        style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.625rem)" }}
      >
        <div className="min-w-0">
          <p className="employee-type-header-greeting text-white/85">
            Xin chào
          </p>
          <h1 className="employee-type-header-name truncate text-white">
            {employeeName || "bạn"}
          </h1>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            className="relative flex h-11 w-11 items-center justify-center rounded-full text-white transition-colors hover:bg-white/15"
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
                className="flex h-11 w-11 items-center justify-center rounded-full transition-colors hover:bg-white/15"
                aria-label="Menu tài khoản"
              >
                <UserCircle className="h-7 w-7 text-white/90" strokeWidth={2.1} />
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
    </div>
  );
}
