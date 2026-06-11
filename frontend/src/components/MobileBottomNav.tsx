import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useState, useCallback } from "react";
import { Bell, UserCircle, Key, LogOut, MoreVertical } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { NotificationBadge } from "@/components/notifications";
import { NotificationSheet } from "@/components/notifications";
import { UserProfileSheet } from "@/components/sheets/UserProfileSheet";
import { UserAvatar } from "@/components/ui/user-avatar";
import { useBottomNav, useAuth } from "@/contexts";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { useBellAnimation } from "@/hooks/useBellAnimation";

export interface NavLeaf {
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  path?: string;
  end?: boolean;
  onClick?: () => void;
}

export interface NavGroup {
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  path?: string;
  end?: boolean;
  submenu?: NavLeaf[];
}

interface MobileBottomNavProps {
  groups: NavGroup[];
  moreItems?: NavLeaf[];
}

export const MobileBottomNav = ({ groups, moreItems }: MobileBottomNavProps) => {
  const isMobile = useIsMobile();
  const location = useLocation();
  const navigate = useNavigate();
  const [openGroup, setOpenGroup] = useState<NavGroup | null>(null);
  const [isMoreOpen, setIsMoreOpen] = useState(false);

  const [isNotificationSheetOpen, setIsNotificationSheetOpen] = useState(false);
  const [isProfileOpen, setIsProfileOpen] = useState(false);

  const { override } = useBottomNav();
  const { user, logout } = useAuth();
  const { openModal } = useModalNavigation();

  const { data: unreadData } = useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;
  const bellIcon = useBellAnimation(unreadCount);

  const isGroupActive = useCallback(
    (group: NavGroup): boolean => {
      if (group.path) {
        return group.end
          ? location.pathname === group.path
          : location.pathname.startsWith(group.path);
      }
      return (
        group.submenu?.some(
          (item) => item.path && location.pathname.startsWith(item.path),
        ) ?? false
      );
    },
    [location.pathname],
  );

  if (!isMobile) return null;

  if (override) {
    return (
      <nav
        className="fixed bottom-0 left-0 right-0 z-50 lg:hidden bg-background border-t border-border/40"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      >
        <div className="h-14 flex items-center px-4">{override.content}</div>
      </nav>
    );
  }

  return (
    <>
      <nav
        className="fixed bottom-0 left-0 right-0 z-50 lg:hidden bg-background border-t border-border/40"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
        aria-label="Điều hướng chính"
      >
        <div className="flex items-stretch h-14 relative px-2">
          {groups.map((group) => {
            const isActive = isGroupActive(group);

            if (group.path && !group.submenu) {
              return (
                <NavLink
                  key={group.path}
                  to={group.path}
                  end={group.end}
                  className="flex-1 flex flex-col items-center justify-center gap-0.5 touch-manipulation relative group transition-all duration-300"
                  aria-label={group.title}
                >
                  {isActive && (
                    <span className="absolute top-0 left-1/2 -translate-x-1/2 w-10 h-1.5 bg-primary/90 rounded-b-full shadow-[0_4px_12px_rgba(var(--primary-rgb),0.5)] animate-in fade-in" />
                  )}
                  <div
                    className={cn(
                      "p-1.5 rounded-full transition-all duration-300",
                      isActive
                        ? "bg-primary/10 text-primary -translate-y-1"
                        : "text-muted-foreground group-hover:bg-accent",
                    )}
                  >
                    <group.icon
                      className={cn("w-5 h-5", isActive && "stroke-[2.5px]")}
                    />
                  </div>
                  <span
                    className={cn(
                      "text-[10px] font-medium leading-none transition-all duration-300",
                      isActive
                        ? "text-primary"
                        : "text-muted-foreground opacity-80",
                    )}
                  >
                    {group.title}
                  </span>
                </NavLink>
              );
            }

            return (
              <button
                key={group.title}
                className="flex-1 flex flex-col items-center justify-center gap-0.5 touch-manipulation relative group transition-all duration-300"
                onClick={() => setOpenGroup(group)}
                aria-label={group.title}
                aria-haspopup="true"
              >
                {isActive && (
                  <span className="absolute top-0 left-1/2 -translate-x-1/2 w-10 h-1.5 bg-primary/90 rounded-b-full shadow-[0_4px_12px_rgba(var(--primary-rgb),0.5)] animate-in fade-in" />
                )}
                <div
                  className={cn(
                    "p-1.5 rounded-full transition-all duration-300",
                    isActive
                      ? "bg-primary/10 text-primary -translate-y-1"
                      : "text-muted-foreground group-hover:bg-accent",
                  )}
                >
                  <group.icon
                    className={cn("w-5 h-5", isActive && "stroke-[2.5px]")}
                  />
                </div>
                <span
                  className={cn(
                    "text-[10px] font-medium leading-none transition-all duration-300",
                    isActive
                      ? "text-primary"
                      : "text-muted-foreground opacity-80",
                  )}
                >
                  {group.title}
                </span>
              </button>
            );
          })}

          {/* Account button — always last slot */}
          <button
            className="flex-1 flex flex-col items-center justify-center gap-0.5 touch-manipulation relative group transition-all duration-300"
            onClick={() => setOpenGroup(ACCOUNT_GROUP)}
            aria-label="Tài khoản"
            aria-haspopup="true"
          >
            {user ? (
              <UserAvatar
                email={user.email}
                name={user.name}
                username={user.username}
                size="sm"
                className="h-6 w-6 text-[10px]"
              />
            ) : (
              <div className="p-1.5 rounded-full text-muted-foreground group-hover:bg-accent">
                <UserCircle className="w-5 h-5" />
              </div>
            )}
            <span className="text-[10px] font-medium leading-none text-muted-foreground opacity-80">
              Tài khoản
            </span>
          </button>

          {/* More menu — compact ⋮ button */}
          {moreItems && moreItems.length > 0 && (
            <button
              className="w-11 flex flex-col items-center justify-center gap-0.5 touch-manipulation"
              onClick={() => setIsMoreOpen(true)}
              aria-label="Thêm"
            >
              <MoreVertical className="w-5 h-5 text-muted-foreground" />
            </button>
          )}
        </div>
      </nav>

      {/* Submenu sheet — for nav groups and account */}
      <Sheet
        open={!!openGroup}
        onOpenChange={(open) => !open && setOpenGroup(null)}
      >
        <SheetContent side="bottom" className="h-auto pb-safe shadow-[0_-4px_24px_rgba(0,0,0,0.08)]">
          {/* Drag handle */}
          <div className="flex justify-center pt-3 pb-1">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
          <SheetHeader className="px-5 pb-4">
            <SheetTitle className="text-base font-semibold text-left">
              {openGroup?.title}
            </SheetTitle>
          </SheetHeader>

          {openGroup === ACCOUNT_GROUP ? (
            <div className="px-5 pb-4">
              {user && (
                <div className="flex items-center gap-3 pb-3">
                  <UserAvatar
                    email={user.email}
                    name={user.name}
                    username={user.username}
                    size="lg"
                    className="h-10 w-10"
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-foreground truncate">
                      {user.name}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">
                      {user.email}
                    </p>
                  </div>
                </div>
              )}

              <div className="grid grid-cols-2 gap-2">
                <button
                  className="flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation hover:bg-accent text-foreground border border-transparent"
                  onClick={() => {
                    setOpenGroup(null);
                    setIsNotificationSheetOpen(true);
                  }}
                >
                  <div className="relative">
                    <Bell className="w-5 h-5 text-muted-foreground" />
                    {unreadCount > 0 && (
                      <span className="absolute -top-1.5 -right-1.5 h-4 w-4 rounded-full bg-destructive text-[9px] text-white flex items-center justify-center font-bold">
                        {unreadCount > 9 ? "9+" : unreadCount}
                      </span>
                    )}
                  </div>
                  <span className="text-xs font-medium leading-none text-muted-foreground">
                    Thông báo
                  </span>
                </button>

                <button
                  className="flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation hover:bg-accent text-foreground border border-transparent"
                  onClick={() => {
                    setOpenGroup(null);
                    setIsProfileOpen(true);
                  }}
                >
                  <UserCircle className="w-5 h-5 text-muted-foreground" />
                  <span className="text-xs font-medium leading-none text-muted-foreground">
                    Hồ sơ
                  </span>
                </button>

                <button
                  className="flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation hover:bg-accent text-foreground border border-transparent"
                  onClick={() => {
                    setOpenGroup(null);
                    openModal(MODAL_IDS.CHANGE_PASSWORD);
                  }}
                >
                  <Key className="w-5 h-5 text-muted-foreground" />
                  <span className="text-xs font-medium leading-none text-muted-foreground">
                    Đổi mật khẩu
                  </span>
                </button>

                <button
                  className="flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation hover:bg-destructive/10 text-destructive border border-transparent"
                  onClick={() => {
                    setOpenGroup(null);
                    logout();
                  }}
                >
                  <LogOut className="w-5 h-5" />
                  <span className="text-xs font-medium leading-none">
                    Đăng xuất
                  </span>
                </button>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2 px-5 pb-4">
              {openGroup?.submenu?.map((item) => {
                const isActive = item.path
                  ? location.pathname.startsWith(item.path)
                  : false;
                return (
                  <button
                    key={item.title}
                    className={cn(
                      "flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation",
                      isActive
                        ? "bg-primary/10 text-primary border border-primary/20"
                        : "hover:bg-accent text-foreground border border-transparent",
                    )}
                    onClick={() => {
                      setOpenGroup(null);
                      if (item.onClick) {
                        item.onClick();
                      } else if (item.path) {
                        navigate(item.path);
                      }
                    }}
                  >
                    <item.icon
                      className={cn("w-5 h-5", isActive && "text-primary")}
                    />
                    <span
                      className={cn(
                        "text-xs font-medium leading-none",
                        isActive ? "text-primary" : "text-muted-foreground",
                      )}
                    >
                      {item.title}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* More menu sheet */}
      <Sheet open={isMoreOpen} onOpenChange={setIsMoreOpen}>
        <SheetContent side="bottom" className="h-auto rounded-t-2xl pb-safe shadow-[0_-4px_24px_rgba(0,0,0,0.08)]">
          {/* Drag handle */}
          <div className="flex justify-center pt-2.5 pb-1">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
          <SheetHeader className="pb-3">
            <SheetTitle className="text-base font-semibold text-left">
              Thêm
            </SheetTitle>
          </SheetHeader>
          <div className="grid grid-cols-3 gap-2 pb-2">
            {moreItems?.map((item) => {
              const isActive = item.path
                ? location.pathname.startsWith(item.path)
                : false;
              return (
                <button
                  key={item.title}
                  className={cn(
                    "flex flex-col items-center justify-center gap-1.5 py-4 rounded-xl transition-colors touch-manipulation",
                    isActive
                      ? "bg-primary/10 text-primary border border-primary/20"
                      : "hover:bg-accent text-foreground border border-transparent",
                  )}
                  onClick={() => {
                    setIsMoreOpen(false);
                    if (item.onClick) {
                      item.onClick();
                    } else if (item.path) {
                      navigate(item.path);
                    }
                  }}
                >
                  <item.icon
                    className={cn("w-5 h-5", isActive && "text-primary")}
                  />
                  <span
                    className={cn(
                      "text-xs font-medium leading-none",
                      isActive ? "text-primary" : "text-muted-foreground",
                    )}
                  >
                    {item.title}
                  </span>
                </button>
              );
            })}
          </div>
        </SheetContent>
      </Sheet>

      <NotificationSheet
        variant="corporate"
        isOpen={isNotificationSheetOpen}
        onClose={() => setIsNotificationSheetOpen(false)}
      />

      <UserProfileSheet
        isOpen={isProfileOpen}
        onClose={() => setIsProfileOpen(false)}
      />
    </>
  );
};

const ACCOUNT_GROUP: NavGroup = {
  title: "Tài khoản",
  icon: UserCircle,
};
