import React, { useState, useCallback } from "react";
import { NavLink, useNavigate, useLocation } from "react-router-dom";
import {
  Briefcase,
  Users,
  Calendar,
  LogOut,
  UserCircle,
  Key,
  ChevronUp,
  LayoutDashboard,
  Bell,
  ReceiptText,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { authManager } from "@/lib/auth";
import {
  getSidebarCollapsedState,
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { UserAvatar } from "@/components/ui/user-avatar";
import { UserProfileSheet } from "@/components/sheets/UserProfileSheet";
import { useAuth } from "@/contexts";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";


const menuItems = [
  { title: "Tổng quan", icon: LayoutDashboard, path: "/partner/dashboard", end: true },
  { title: "Dự án", icon: Briefcase, path: "/partner/projects", end: false },
  { title: "Nhân viên", icon: Users, path: "/partner/employees" },
  { title: "Bảng công", icon: Calendar, path: "/partner/timesheet", end: true },
  { title: "Bút toán ngân hàng", icon: ReceiptText, path: "/partner/timesheet/payment-history" },
];

interface NavItemProps {
  item: typeof menuItems[number];
  isCollapsed: boolean;
  onNavigate: () => void;
}

const NavItem = React.memo(({ item, isCollapsed, onNavigate }: NavItemProps) => {
  const location = useLocation();
  const isActive = item.end
    ? location.pathname === item.path
    : location.pathname.startsWith(item.path);

  const inner = (
    <NavLink to={item.path} end={item.end} className="block w-full" onClick={onNavigate}>
      <div
        className={cn(
          "relative flex items-center gap-2.5 rounded-xl transition-all duration-200 cursor-pointer select-none",
          isCollapsed
            ? isActive
              ? "h-9 w-9 justify-center mx-auto bg-card/[0.08] ring-1 ring-white/[0.12]"
              : "h-9 w-9 justify-center mx-auto"
            : "min-h-9 py-1.5 px-3",
          isActive
            ? "bg-card/[0.08] text-white shadow-[-3px_0_8px_-2px_hsl(var(--partner-accent)/0.15)]"
            : "text-white/50 hover:bg-card/10 hover:text-white/80 hover:translate-x-0.5"
        )}
      >
        {isActive && !isCollapsed && (
          <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-partner-accent rounded-r-full" />
        )}
        <item.icon
          className={cn(
            "shrink-0",
            isCollapsed ? "w-[17px] h-[17px]" : "w-4 h-4",
            isActive ? "text-[hsl(var(--partner-accent))]" : "text-white/80"
          )}
        />
        {!isCollapsed && (
          <span className={cn(
            "text-base leading-snug truncate flex-1",
            isActive ? "font-semibold text-white" : "font-medium"
          )}>
            {item.title}
          </span>
        )}
      </div>
    </NavLink>
  );

  if (isCollapsed) {
    return (
      <SidebarMenuItem>
        <Tooltip>
          <TooltipTrigger asChild>{inner}</TooltipTrigger>
          <TooltipContent side="right" align="center" className="font-medium">
            {item.title}
          </TooltipContent>
        </Tooltip>
      </SidebarMenuItem>
    );
  }

  return <SidebarMenuItem>{inner}</SidebarMenuItem>;
});
NavItem.displayName = "NavItem";

const PartnerSidebar = () => {
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const { isMobile, setOpenMobile, open, openMobile } = useSidebar();
  const { openModal } = useModalNavigation();
  const isCollapsed = getSidebarCollapsedState({ isMobile, open, openMobile });
  const { data: unreadData } = useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;

  const [showProfile, setShowProfile] = useState(false);

  const handleNavigate = useCallback(() => {
    if (isMobile) setOpenMobile(false);
  }, [isMobile, setOpenMobile]);

  const handleLogout = useCallback(() => {
    authManager.removeToken();
    localStorage.removeItem("userRole");
    localStorage.removeItem("username");
    logout();
    navigate("/login");
  }, [logout, navigate]);

  return (
    <>
      <Sidebar
        collapsible="icon"
        className="partner-sidebar-surface border-r border-white/[0.06] bg-[hsl(var(--sidebar-background))]"
      >
        {/* Header — logo */}
        <SidebarHeader className="p-0 shrink-0 border-b border-white/[0.06]">
          <div className="flex items-center justify-center h-14">
            {isCollapsed ? (
              <img src="/logo-square.png" alt="TingTing" className="h-7 w-7 object-contain" />
            ) : (
              <img src="/tingting-white.png" alt="TingTing" className="h-7 object-contain" />
            )}
          </div>
        </SidebarHeader>

        {/* Nav */}
        <SidebarContent className="py-2 overflow-y-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
          <SidebarMenu className={cn("px-2 gap-0.5", isCollapsed && "items-center")}>
            {menuItems.map((item) => (
              <NavItem key={item.path} item={item} isCollapsed={isCollapsed} onNavigate={handleNavigate} />
            ))}
          </SidebarMenu>
        </SidebarContent>

        {/* Footer — user menu */}
        <SidebarSeparator className="bg-card/[0.06] m-0 shrink-0" />
        <SidebarFooter className="p-2 shrink-0">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className={cn(
                  "relative flex items-center w-full rounded-xl transition-all duration-200 cursor-pointer outline-none",
                  "bg-card/[0.04] border border-white/[0.06]",
                  "hover:bg-card/[0.08] hover:border-white/[0.1]",
                  isCollapsed ? "h-9 w-9 justify-center mx-auto" : "h-auto min-h-[40px] px-2.5 py-2 gap-2.5"
                )}
              >
                {isCollapsed && user && (
                  <span className="text-xs font-bold text-white/80">
                    {user.name?.charAt(0)?.toUpperCase() || "U"}
                  </span>
                )}
                {!isCollapsed && user && (
                  <>
                    <div className="flex flex-col min-w-0 flex-1 text-left">
                      <span className="text-[10px] text-white/45 truncate leading-tight uppercase font-semibold tracking-wide">Xin chào</span>
                      <span className="text-[12.5px] font-semibold truncate leading-tight text-white/90">{user.name}</span>
                    </div>
                    <ChevronUp className="w-3.5 h-3.5 shrink-0 text-white/45" />
                  </>
                )}
                {unreadCount > 0 && (
                  <span className={cn(
                    "absolute -top-1.5 -right-1.5 h-4 min-w-4 flex items-center justify-center px-1",
                    "text-[10px] font-semibold rounded-full bg-red-500 text-white",
                    "animate-badge-pulse"
                  )}>
                    {unreadCount > 99 ? '99+' : unreadCount}
                  </span>
                )}
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="w-56 z-[9999]"
              side="top"
              align={isCollapsed ? "center" : "end"}
              sideOffset={8}
            >
              {user && (
                <>
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex flex-col gap-0.5">
                      <p className="text-sm font-semibold">{user.name}</p>
                      <p className="text-xs text-muted-foreground">{user.email}</p>
                      <p className="text-xs text-muted-foreground">Quản lý</p>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                </>
              )}
              <DropdownMenuItem onClick={() => setShowProfile(true)}>
                <UserCircle className="mr-2 h-4 w-4" />
                Thông tin cá nhân
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => openModal(MODAL_IDS.CHANGE_PASSWORD)}>
                <Key className="mr-2 h-4 w-4" />
                Đổi mật khẩu
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => openModal(MODAL_IDS.NOTIFICATION_SHEET)}>
                <Bell className="mr-2 h-4 w-4" />
                Thông báo
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout} className="text-destructive focus:text-destructive">
                <LogOut className="mr-2 h-4 w-4" />
                Đăng xuất
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {!isCollapsed && (
            <p className="text-[10px] text-white/70 text-center pt-1 pb-1 tracking-wide select-none">
              v{__APP_VERSION__}
            </p>
          )}
        </SidebarFooter>
      </Sidebar>

      <UserProfileSheet isOpen={showProfile} onClose={() => setShowProfile(false)} />
    </>
  );
};

export default PartnerSidebar;
