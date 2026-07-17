import React, { useState, useCallback, useMemo, useRef, useEffect } from "react";
import { NavLink, useLocation } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import {
  Users,
  Briefcase,
  Calendar,
  LogOut,
  Home,
  BookOpen,
  Settings,
  Landmark,
  HandCoins,
  Wallet,
  UserCog,
  UserCircle,
  Key,
  ChevronUp,
  ChevronDown,
  Bell,
  Activity,
  Clock,
  ClipboardList,
} from "lucide-react";
import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { authManager } from "@/lib/auth";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
  SidebarSeparator,
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


type MenuItem = {
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  path: string;
  end?: boolean;
  badge?: string;
  group: string;
};

const menuItems: MenuItem[] = [
  { title: "Tổng quan", icon: Home, path: "/admin", end: true, group: "top" },
  { title: "Người dùng", icon: UserCog, path: "/admin/users", group: "quan-ly" },
  { title: "Dự án", icon: Briefcase, path: "/admin/projects", group: "quan-ly" },
  { title: "Nhân viên", icon: Users, path: "/admin/employees", group: "quan-ly" },
  { title: "Bảng công", icon: Calendar, path: "/admin/timesheet", group: "quan-ly" },
  { title: "Ứng lương", icon: HandCoins, path: "/admin/advance-payments", group: "quan-ly" },
  { title: "Sổ Cái", icon: BookOpen, path: "/admin/ledger", group: "tai-chinh" },
  { title: "Khoản vay", icon: Landmark, path: "/admin/loans", group: "tai-chinh" },
  { title: "Kiểm tra API", icon: Activity, path: "/admin/system-health", group: "he-thong" },
  { title: "Lịch công việc", icon: Clock, path: "/admin/cron-health", group: "he-thong" },
  { title: "Nhật ký", icon: ClipboardList, path: "/admin/audit-log", group: "he-thong" },
  { title: "Cài Đặt", icon: Settings, path: "/admin/settings", group: "he-thong" },
  { title: "Quản lý ví", icon: Wallet, path: "/admin/wallet", group: "tai-chinh" },
];

const menuGroups = [
  { key: "top", label: null },
  { key: "quan-ly", label: "Quản lý" },
  { key: "tai-chinh", label: "Tài chính" },
  { key: "he-thong", label: "Hệ thống" },
] as const;

interface NavItemProps {
  item: MenuItem;
  isCollapsed: boolean;
  onNavigate: () => void;
}

const NavItem = React.memo(({ item, isCollapsed, onNavigate }: NavItemProps) => {
  const location = useLocation();
  const isActive = item.end
    ? location.pathname === item.path
    : location.pathname.startsWith(item.path);

  const inner = (
    <NavLink to={item.path} className="block w-full" onClick={onNavigate}>
      <div
        className={cn(
          "relative flex items-center gap-2.5 rounded-xl transition-all duration-150 ease-out cursor-pointer select-none",
          isCollapsed
            ? isActive
              ? "h-9 w-9 justify-center mx-auto bg-card/[0.08] ring-1 ring-white/[0.12]"
              : "h-9 w-9 justify-center mx-auto"
            : "h-9 px-2.5",
          isActive
            ? "bg-card/[0.08] text-white shadow-[-3px_0_8px_-2px_hsl(var(--admin-accent)/0.15)]"
            : "text-white/50 hover:bg-card/10 hover:text-white/80 hover:translate-x-0.5"
        )}
      >
        {/* Gliding active pill — slides in with a scale animation */}
        {isActive && !isCollapsed && (
          <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-[hsl(var(--admin-accent))] rounded-r-full animate-pill-appear" />
        )}
        <item.icon
          className={cn(
            "shrink-0 transition-all duration-150",
            isCollapsed ? "w-[17px] h-[17px]" : "w-[15px] h-[15px]",
            // Icon morphs from outline-weight to accent color when active
            isActive ? "text-[hsl(var(--admin-accent))]" : "text-white/80 group-hover:text-white"
          )}
        />
        {!isCollapsed && (
          <span className={cn(
            "text-base leading-none truncate flex-1 transition-all duration-150",
            isActive ? "font-medium text-white" : "font-normal"
          )}>
            {item.title}
          </span>
        )}
        {item.badge && !isCollapsed && (
          <span className="ml-auto bg-destructive text-destructive-foreground text-[9px] px-1.5 py-0.5 rounded-full shrink-0 font-semibold">
            {item.badge}
          </span>
        )}
        {item.badge && isCollapsed && (
          <span className="absolute -top-0.5 -right-0.5 bg-destructive text-destructive-foreground text-[8px] w-3.5 h-3.5 rounded-full flex items-center justify-center font-bold">
            {item.badge}
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

interface NavGroupProps {
  label: string;
  groupKey: string;
  items: MenuItem[];
  isCollapsed: boolean;
  isExpanded: boolean;
  onToggle: (key: string) => void;
  onNavigate: () => void;
}

const NavGroup = React.memo(
  ({ label, groupKey, items, isCollapsed, isExpanded, onToggle, onNavigate }: NavGroupProps) => {
    const contentRef = useRef<HTMLDivElement>(null);
    const [contentHeight, setContentHeight] = useState<number | undefined>(undefined);

    // Re-measure whenever isCollapsed changes (remount scenario) or items change
    useEffect(() => {
      const el = contentRef.current;
      if (!el) return;
      // Measure immediately
      setContentHeight(el.scrollHeight);
      const ro = new ResizeObserver(() => setContentHeight(el.scrollHeight));
      ro.observe(el);
      return () => ro.disconnect();
    }, [isCollapsed, items]);

    if (isCollapsed) {
      return (
        <div className="flex flex-col items-center gap-1 py-1.5">
          <div className="w-5 h-px bg-card/[0.08] my-1" />
          {items.map((item) => (
            <NavItem key={item.path} item={item} isCollapsed={isCollapsed} onNavigate={onNavigate} />
          ))}
        </div>
      );
    }

    return (
      <div data-group={groupKey}>
        <button
          type="button"
          onClick={() => onToggle(groupKey)}
          className="flex items-center w-full px-2.5 py-1 mt-4 select-none cursor-pointer group/label"
          aria-expanded={isExpanded}
        >
          <span className="flex-1 text-left text-xs font-semibold uppercase tracking-[0.1em] text-white/50 group-hover/label:text-white/70 transition-colors">
            {label}
          </span>
          <ChevronDown
            className={cn(
              "w-3.5 h-3.5 shrink-0 text-white/40 group-hover/label:text-white/60 transition-all duration-200",
              isExpanded && "rotate-180"
            )}
          />
        </button>
        <div
          className={cn(
            "overflow-hidden",
            // Only animate when we have a measured height; skip transition on initial mount
            contentHeight !== undefined && "transition-[height,opacity] duration-300 ease-out"
          )}
          style={{
            height: isExpanded ? (contentHeight ?? "auto") : 0,
            opacity: isExpanded ? 1 : 0,
          }}
          aria-hidden={!isExpanded}
        >
          <div ref={contentRef}>
            <SidebarMenu className="px-2 pb-1 gap-0.5">
              {items.map((item) => (
                <NavItem key={item.path} item={item} isCollapsed={isCollapsed} onNavigate={onNavigate} />
              ))}
            </SidebarMenu>
          </div>
        </div>
      </div>
    );
  }
);
NavGroup.displayName = "NavGroup";

const AdminSidebar = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();
  const { isMobile, setOpenMobile, open, openMobile } = useSidebar();
  const { openModal } = useModalNavigation();
  const isCollapsed = !open && !openMobile;
  const { data: unreadData } = useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;
  const isAdvPartner = user?.role === 'adv_partner';

  const [showProfile, setShowProfile] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);

  const topItems = useMemo(() => {
    const items = menuItems.filter((i) => i.group === "top");
    if (isAdvPartner) return [];
    return items;
  }, [isAdvPartner]);

  const filteredMenuItems = useMemo(() => {
    if (!isAdvPartner) return menuItems;
    const basePath = "/adv-partner";
    const allowed = ["/admin/advance-payments", "/admin/users"];
    return menuItems
      .filter((i) => allowed.includes(i.path))
      .map((i) => ({
        ...i,
        path: i.path.replace("/admin", basePath),
        ...(i.path === "/admin/users" ? { title: "Nhân viên" } : {}),
      }));
  }, [isAdvPartner]);

  const activeGroupKey = useMemo(() => {
    for (const item of filteredMenuItems) {
      const isActive = item.end
        ? location.pathname === item.path
        : location.pathname.startsWith(item.path);
      if (isActive) return item.group;
    }
    return null;
  }, [location.pathname, filteredMenuItems]);

  const allGroupKeys = useMemo(
    () => new Set(menuGroups.filter((g) => g.label !== null).map((g) => g.key)),
    []
  );

  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(
    () => isAdvPartner
      ? (activeGroupKey && activeGroupKey !== "top" ? new Set([activeGroupKey]) : new Set<string>())
      : new Set(allGroupKeys)
  );

  const autoCollapsedRef = useRef(false);
  const overflowTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Reset auto-collapse state when sidebar opens so groups restore properly
  const prevCollapsedRef = useRef(isCollapsed);
  useEffect(() => {
    if (prevCollapsedRef.current && !isCollapsed) {
      if (autoCollapsedRef.current) {
        autoCollapsedRef.current = false;
        setExpandedGroups(
          isAdvPartner
            ? (activeGroupKey && activeGroupKey !== "top" ? new Set([activeGroupKey]) : new Set<string>())
            : new Set(allGroupKeys)
        );
      }
    }
    prevCollapsedRef.current = isCollapsed;
  }, [isCollapsed, activeGroupKey, isAdvPartner, allGroupKeys]);

  useEffect(() => {
    const el = contentRef.current;
    if (!el || isCollapsed) return;

    const checkOverflow = () => {
      if (overflowTimerRef.current) clearTimeout(overflowTimerRef.current);
      overflowTimerRef.current = setTimeout(() => {
        // Skip check if sidebar is currently collapsed (avoids false positives during animation)
        if (!contentRef.current || prevCollapsedRef.current) return;
        const overflowing = el.scrollHeight > el.clientHeight + 2;
        if (overflowing && !autoCollapsedRef.current) {
          autoCollapsedRef.current = true;
          setExpandedGroups(
            activeGroupKey && activeGroupKey !== "top" ? new Set([activeGroupKey]) : new Set()
          );
        } else if (!overflowing && autoCollapsedRef.current) {
          autoCollapsedRef.current = false;
          setExpandedGroups(
            isAdvPartner
              ? (activeGroupKey && activeGroupKey !== "top" ? new Set([activeGroupKey]) : new Set<string>())
              : new Set(allGroupKeys)
          );
        }
      }, 300);
    };

    const observer = new ResizeObserver(checkOverflow);
    observer.observe(el);
    checkOverflow();
    return () => {
      observer.disconnect();
      if (overflowTimerRef.current) clearTimeout(overflowTimerRef.current);
    };
  }, [isCollapsed, activeGroupKey, allGroupKeys, isAdvPartner]);

  useEffect(() => {
    if (activeGroupKey && activeGroupKey !== "top" && autoCollapsedRef.current) {
      setExpandedGroups(new Set([activeGroupKey]));
    }
  }, [activeGroupKey]);

  const handleToggleGroup = useCallback((key: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

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

  const groupedSections = useMemo(() => menuGroups.filter((g) => g.label !== null), []);

  return (
    <>
      <Sidebar
        collapsible="icon"
        className="border-r border-white/[0.06] bg-[hsl(var(--sidebar-background))]"
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
        <SidebarContent
          ref={contentRef}
          className="py-2 overflow-y-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
        >
          {/* Top items */}
          <SidebarMenu className={cn("px-2 gap-0.5", isCollapsed && "items-center")}>
            {topItems.map((item) => (
              <NavItem key={item.path} item={item} isCollapsed={isCollapsed} onNavigate={handleNavigate} />
            ))}
          </SidebarMenu>

          {/* Grouped sections */}
          {groupedSections.map(({ key, label }) => {
            const items = filteredMenuItems.filter((i) => i.group === key);
            if (!items.length) return null;
            return (
              <NavGroup
                key={key}
                label={label!}
                groupKey={key}
                items={items}
                isCollapsed={isCollapsed}
                isExpanded={expandedGroups.has(key)}
                onToggle={handleToggleGroup}
                onNavigate={handleNavigate}
              />
            );
          })}
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
                      <span className="text-base font-medium truncate leading-tight text-white/90">{user.name}</span>
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
                      <p className="text-xs text-muted-foreground">{isAdvPartner ? 'Quản lý ứng lương' : 'Quản trị viên'}</p>
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
            <p className="text-[10px] text-white/40 text-center pt-1 pb-1 tracking-wide select-none">
              v{__APP_VERSION__}
            </p>
          )}
        </SidebarFooter>
      </Sidebar>

      <UserProfileSheet isOpen={showProfile} onClose={() => setShowProfile(false)} />
    </>
  );
};

export default AdminSidebar;
