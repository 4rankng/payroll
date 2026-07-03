import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useState, useCallback } from "react";
import { Bell, UserCircle, Key, LogOut, MoreHorizontal } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { NotificationBadge } from "@/components/notifications";
import { NotificationSheet } from "@/components/notifications";
import { UserProfileSheet } from "@/components/sheets/UserProfileSheet";
import { UserAvatar } from "@/components/ui/user-avatar";
import { useBottomNav, useAuth } from "@/contexts";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";

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

  const isMoreActive =
    moreItems?.some((item) => item.path && location.pathname.startsWith(item.path)) ?? false;

  const navSlotClass =
    "group relative flex min-h-[64px] min-w-0 flex-1 touch-manipulation flex-col items-center justify-center gap-1 px-1 pt-1 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40";

  const navIconClass = (isActive: boolean) =>
    cn(
      "flex h-8 w-8 items-center justify-center rounded-2xl transition-colors",
      isActive
        ? "bg-primary/10 text-primary"
        : "text-muted-foreground group-hover:bg-accent group-hover:text-foreground",
    );

  const navLabelClass = (isActive: boolean) =>
    cn(
      "max-w-[4.5rem] truncate text-[11px] font-semibold leading-tight transition-colors",
      isActive ? "text-primary" : "text-muted-foreground",
    );

  const renderActiveIndicator = (isActive: boolean) =>
    isActive ? (
      <span className="absolute left-1/2 top-0 h-1 w-10 -translate-x-1/2 rounded-b-full bg-primary/90 animate-in fade-in" />
    ) : null;

  const sheetSurfaceClass =
    "flex h-auto max-h-[78dvh] flex-col overflow-hidden rounded-t-[28px] border-t border-white/70 bg-slate-50/95 p-0 shadow-[0_-24px_80px_-36px_rgba(15,23,42,0.65)] backdrop-blur-xl";

  const openGroupTitle = openGroup?.title ?? "Điều hướng";
  const openGroupDescription =
    openGroup === ACCOUNT_GROUP ? "Tài khoản và cài đặt cá nhân" : "Chọn điểm đến nhanh";
  const moreTitle = "Thêm";
  const moreDescription = "Các mục quản trị ít dùng hơn";

  const renderSheetChrome = (title: string, description: string) => (
    <>
      <div className="flex justify-center pt-3">
        <div className="h-1.5 w-12 rounded-full bg-slate-300" />
      </div>
      <SheetHeader className="px-5 pb-3 pt-4 text-left">
        <SheetTitle className="text-xl font-bold tracking-tight text-slate-950">
          {title}
        </SheetTitle>
        <p className="text-sm font-medium text-slate-500">{description}</p>
      </SheetHeader>
    </>
  );

  const renderNavTile = (
    item: NavLeaf,
    isActive: boolean,
    onSelect: () => void,
    tone: "default" | "danger" = "default",
  ) => (
    <button
      key={item.title}
      className={cn(
        "group flex min-h-[96px] flex-col items-center justify-center gap-2 rounded-2xl border p-3 text-center transition-all touch-manipulation",
        "active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40",
        isActive &&
          tone === "default" &&
          "border-primary/30 bg-primary text-primary-foreground shadow-[0_14px_30px_-20px_hsl(var(--primary)/0.8)]",
        !isActive &&
          tone === "default" &&
          "border-slate-200/80 bg-white text-slate-700 shadow-sm hover:border-primary/20 hover:bg-primary/5 hover:text-primary",
        tone === "danger" &&
          "border-rose-200/80 bg-rose-50 text-rose-700 hover:bg-rose-100",
      )}
      onClick={onSelect}
    >
      <div
        className={cn(
          "flex h-11 w-11 items-center justify-center rounded-2xl transition-colors",
          isActive && tone === "default"
            ? "bg-white/18 text-white"
            : tone === "danger"
              ? "bg-white/70 text-rose-600"
              : "bg-slate-100 text-slate-900 group-hover:bg-white",
        )}
      >
        <item.icon className="h-5 w-5" />
      </div>
      <span
        className={cn(
          "max-w-full text-[13px] font-semibold leading-tight",
          isActive && tone === "default"
            ? "text-primary-foreground"
            : tone === "danger"
              ? "text-rose-700"
              : "text-slate-600 group-hover:text-slate-950",
        )}
      >
        {item.title}
      </span>
    </button>
  );

  if (!isMobile) return null;

  if (override) {
    return (
      <nav
        className="fixed bottom-0 left-0 right-0 z-50 border-t border-border/40 bg-background/92 shadow-[0_-10px_30px_-18px_hsl(var(--primary)/0.35)] backdrop-blur-xl lg:hidden"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      >
        <div className="flex min-h-[60px] items-center px-4">{override.content}</div>
      </nav>
    );
  }

  return (
    <>
      <nav
        className="fixed bottom-0 left-0 right-0 z-50 border-t border-border/40 bg-background/92 shadow-[0_-10px_30px_-18px_hsl(var(--primary)/0.35)] backdrop-blur-xl lg:hidden"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
        aria-label="Điều hướng chính"
      >
        <div className="relative flex min-h-[64px] items-stretch px-2">
          {groups.map((group) => {
            const isActive = isGroupActive(group);

            if (group.path && !group.submenu) {
              return (
                <NavLink
                  key={group.path}
                  to={group.path}
                  end={group.end}
                  className={navSlotClass}
                  aria-label={group.title}
                >
                  {renderActiveIndicator(isActive)}
                  <div className={navIconClass(isActive)}>
                    <group.icon
                      className={cn("w-5 h-5", isActive && "stroke-[2.5px]")}
                    />
                  </div>
                  <span className={navLabelClass(isActive)}>
                    {group.title}
                  </span>
                </NavLink>
              );
            }

            return (
              <button
                key={group.title}
                className={navSlotClass}
                onClick={() => setOpenGroup(group)}
                aria-label={group.title}
                aria-haspopup="true"
                aria-expanded={openGroup?.title === group.title}
              >
                {renderActiveIndicator(isActive)}
                <div className={navIconClass(isActive)}>
                  <group.icon
                    className={cn("w-5 h-5", isActive && "stroke-[2.5px]")}
                  />
                </div>
                <span className={navLabelClass(isActive)}>
                  {group.title}
                </span>
              </button>
            );
          })}

          {/* Account button — always last slot */}
          <button
            className={navSlotClass}
            onClick={() => setOpenGroup(ACCOUNT_GROUP)}
            aria-label="Tài khoản"
            aria-haspopup="true"
            aria-expanded={openGroup === ACCOUNT_GROUP}
          >
            {renderActiveIndicator(openGroup === ACCOUNT_GROUP)}
            {user ? (
              <div className="relative">
                <UserAvatar
                  email={user.email}
                  name={user.name}
                  username={user.username}
                  size="sm"
                  className={cn(
                    "h-8 w-8 text-[10px] ring-1 ring-border/70 transition-colors",
                    openGroup === ACCOUNT_GROUP && "ring-primary/30",
                  )}
                />
                <NotificationBadge count={unreadCount} className="-right-2 -top-1" />
              </div>
            ) : (
              <div className={navIconClass(openGroup === ACCOUNT_GROUP)}>
                <UserCircle className="w-5 h-5" />
              </div>
            )}
            <span className={navLabelClass(openGroup === ACCOUNT_GROUP)}>
              Tài khoản
            </span>
          </button>

          {/* More menu */}
          {moreItems && moreItems.length > 0 && (
            <button
              className={navSlotClass}
              onClick={() => setIsMoreOpen(true)}
              aria-label="Thêm"
              aria-haspopup="true"
              aria-expanded={isMoreOpen}
            >
              {renderActiveIndicator(isMoreActive || isMoreOpen)}
              <div className={navIconClass(isMoreActive || isMoreOpen)}>
                <MoreHorizontal
                  className={cn(
                    "h-5 w-5",
                    (isMoreActive || isMoreOpen) && "stroke-[2.5px]",
                  )}
                />
              </div>
              <span className={navLabelClass(isMoreActive || isMoreOpen)}>
                Thêm
              </span>
            </button>
          )}
        </div>
      </nav>

      {/* Submenu sheet — for nav groups and account */}
      <Sheet
        open={!!openGroup}
        onOpenChange={(open) => !open && setOpenGroup(null)}
      >
        <SheetContent
          side="bottom"
          className={sheetSurfaceClass}
          title={openGroupTitle}
          description={openGroupDescription}
        >
          {renderSheetChrome(openGroupTitle, openGroupDescription)}

          {openGroup === ACCOUNT_GROUP ? (
            <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-[calc(env(safe-area-inset-bottom)+1.25rem)]">
              {user && (
                <div className="mb-3 flex items-center gap-3 rounded-2xl border border-slate-200/80 bg-white p-3 shadow-sm">
                  <UserAvatar
                    email={user.email}
                    name={user.name}
                    username={user.username}
                    size="lg"
                    className="h-10 w-10"
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-slate-950 truncate">
                      {user.name}
                    </p>
                    <p className="text-xs text-slate-500 truncate">
                      {user.email}
                    </p>
                  </div>
                </div>
              )}

              <div className="grid grid-cols-2 gap-2.5">
                {renderNavTile(
                  { title: "Thông báo", icon: Bell },
                  false,
                  () => {
                    setOpenGroup(null);
                    setIsNotificationSheetOpen(true);
                  },
                )}

                {renderNavTile(
                  { title: "Hồ sơ", icon: UserCircle },
                  false,
                  () => {
                    setOpenGroup(null);
                    setIsProfileOpen(true);
                  },
                )}

                {renderNavTile(
                  { title: "Đổi mật khẩu", icon: Key },
                  false,
                  () => {
                    setOpenGroup(null);
                    openModal(MODAL_IDS.CHANGE_PASSWORD);
                  },
                )}

                {renderNavTile(
                  { title: "Đăng xuất", icon: LogOut },
                  false,
                  () => {
                    setOpenGroup(null);
                    logout();
                  },
                  "danger",
                )}
              </div>
            </div>
          ) : (
            <div className="grid min-h-0 flex-1 grid-cols-2 gap-2.5 overflow-y-auto px-5 pb-[calc(env(safe-area-inset-bottom)+1.25rem)]">
              {openGroup?.submenu?.map((item) => {
                const isActive = item.path
                  ? location.pathname.startsWith(item.path)
                  : false;
                return renderNavTile(item, isActive, () => {
                      setOpenGroup(null);
                      if (item.onClick) {
                        item.onClick();
                      } else if (item.path) {
                        navigate(item.path);
                      }
                });
              })}
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* More menu sheet */}
      <Sheet open={isMoreOpen} onOpenChange={setIsMoreOpen}>
        <SheetContent
          side="bottom"
          className={sheetSurfaceClass}
          title={moreTitle}
          description={moreDescription}
        >
          {renderSheetChrome(moreTitle, moreDescription)}
          <div className="grid min-h-0 flex-1 grid-cols-2 gap-2.5 overflow-y-auto px-5 pb-[calc(env(safe-area-inset-bottom)+1.25rem)] sm:grid-cols-3">
            {moreItems?.map((item) => {
              const isActive = item.path
                ? location.pathname.startsWith(item.path)
                : false;
              return renderNavTile(item, isActive, () => {
                    setIsMoreOpen(false);
                    if (item.onClick) {
                      item.onClick();
                    } else if (item.path) {
                      navigate(item.path);
                    }
              });
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
