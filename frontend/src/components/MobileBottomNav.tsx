import { useCallback, useState } from "react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { Bell, Key, LogOut, MoreHorizontal, UserCircle } from "lucide-react";
import { NotificationBadge, NotificationSheet } from "@/components/notifications";
import { UserProfileSheet } from "@/components/sheets/UserProfileSheet";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { UserAvatar } from "@/components/ui/user-avatar";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useAuth, useBottomNav } from "@/contexts";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { useIsMobile } from "@/hooks/useBreakpoint";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { cn } from "@/lib/utils";

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

const ACCOUNT_GROUP: NavGroup = {
  title: "Tài khoản",
  icon: UserCircle,
};

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
          (item) => item.path && location.pathname.startsWith(item.path)
        ) ?? false
      );
    },
    [location.pathname]
  );

  const isMoreActive =
    moreItems?.some((item) => item.path && location.pathname.startsWith(item.path)) ?? false;

  const navSlotClass =
    "group relative flex min-h-[68px] min-w-0 flex-1 touch-manipulation flex-col items-center justify-center gap-1 px-1 pb-1 pt-2 transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--secondary))] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))]";

  const navIconClass = (isActive: boolean) =>
    cn(
      "flex h-9 w-9 items-center justify-center rounded-2xl border transition-all duration-200",
      isActive
        ? "border-transparent bg-[hsl(var(--primary))] text-white shadow-[0_16px_26px_-20px_hsl(var(--primary)/0.85)]"
        : "border-transparent bg-white/65 text-[hsl(var(--muted-foreground))] group-hover:border-[hsl(var(--border))] group-hover:text-[hsl(var(--foreground))]"
    );

  const navLabelClass = (isActive: boolean) =>
    cn(
      "max-w-[4.5rem] truncate text-[11px] font-semibold leading-tight transition-colors",
      isActive ? "text-[hsl(var(--foreground))]" : "text-[hsl(var(--muted-foreground))]"
    );

  const renderActiveIndicator = (isActive: boolean) =>
    isActive ? (
      <span className="absolute left-1/2 top-0 h-1 w-11 -translate-x-1/2 rounded-b-full bg-[hsl(var(--secondary))]" />
    ) : null;

  const sheetSurfaceClass =
    "admin-mobile-sheet flex h-auto max-h-[78dvh] flex-col overflow-hidden rounded-t-[30px] border-x-0 border-b-0 border-t border-white/70 bg-slate-50/95 px-0 pb-0 pt-0 shadow-[0_-24px_80px_-36px_rgba(15,23,42,0.65)] backdrop-blur-xl";

  const openGroupTitle = openGroup?.title ?? "Điều hướng";
  const openGroupDescription =
    openGroup === ACCOUNT_GROUP ? "Tài khoản và cài đặt cá nhân" : "Mục công việc truy cập nhanh";
  const moreTitle = "Thêm";
  const moreDescription = "Các mục quản trị ít dùng hơn";

  const renderSheetChrome = (title: string, description: string) => (
    <>
      <div className="flex justify-center pt-2.5">
        <div className="h-1.5 w-12 rounded-full bg-[hsl(var(--border))]" />
      </div>
      <SheetHeader className="px-4 pb-2.5 pt-3 text-left sm:px-5">
        <SheetTitle className="text-lg font-semibold tracking-tight text-[hsl(var(--foreground))]">
          {title}
        </SheetTitle>
        <p className="admin-subtle-copy text-sm">{description}</p>
      </SheetHeader>
    </>
  );

  const renderNavTile = (
    item: NavLeaf,
    isActive: boolean,
    onSelect: () => void,
    tone: "default" | "danger" = "default"
  ) => (
    <li key={item.title} className="min-w-0">
      <button
        className={cn(
          "admin-mobile-tile group grid min-h-14 w-full grid-cols-[2.25rem_minmax(0,1fr)] items-center gap-3 rounded-xl border border-slate-200/80 bg-white px-3 py-2 text-left shadow-sm transition-all touch-manipulation",
          "active:scale-[0.985] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--secondary))] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))]",
          tone === "default" &&
            isActive &&
            "border-[hsl(var(--primary))/0.25] bg-[hsl(var(--primary))] text-white shadow-[0_14px_26px_-22px_hsl(var(--primary)/0.9)]",
          tone === "default" &&
            !isActive &&
            "text-[hsl(var(--foreground))] hover:border-[hsl(var(--border))] hover:bg-white",
          tone === "danger" &&
            "border-rose-200 bg-rose-50 text-rose-700 hover:bg-rose-100"
        )}
        onClick={onSelect}
        aria-current={isActive ? "page" : undefined}
      >
        <span
          className={cn(
            "flex h-9 w-9 items-center justify-center rounded-xl border transition-colors",
            tone === "danger"
              ? "border-white/80 bg-white/75 text-rose-600"
              : isActive
                ? "border-white/10 bg-white/15 text-white"
                : "border-[hsl(var(--border))/0.8] bg-white/80 text-[hsl(var(--primary))]"
          )}
        >
          <item.icon className="h-[18px] w-[18px]" aria-hidden="true" />
        </span>
        <span
          className={cn(
            "min-w-0 line-clamp-2 break-words text-[13px] font-semibold leading-tight",
            tone === "danger"
              ? "text-rose-700"
              : isActive
                ? "text-white"
                : "text-[hsl(var(--foreground))]"
          )}
        >
          {item.title}
        </span>
      </button>
    </li>
  );

  if (!isMobile) return null;

  if (override) {
    return (
      <nav
        className="admin-mobile-nav fixed bottom-0 left-0 right-0 z-50 border-t border-border/40 bg-background/95 shadow-[0_-10px_30px_-18px_hsl(var(--primary)/0.35)] backdrop-blur-xl lg:hidden"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      >
        <div className="flex min-h-[64px] items-center px-4">{override.content}</div>
      </nav>
    );
  }

  return (
    <>
      <nav
        className="admin-mobile-nav fixed bottom-0 left-0 right-0 z-50 border-t border-border/40 bg-background/95 shadow-[0_-10px_30px_-18px_hsl(var(--primary)/0.35)] backdrop-blur-xl lg:hidden"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
        aria-label="Điều hướng chính"
      >
        <div className="relative flex min-h-[72px] items-stretch px-2">
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
                    <group.icon className={cn("h-5 w-5", isActive && "stroke-[2.4px]")} />
                  </div>
                  <span className={navLabelClass(isActive)}>{group.title}</span>
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
                  <group.icon className={cn("h-5 w-5", isActive && "stroke-[2.4px]")} />
                </div>
                <span className={navLabelClass(isActive)}>{group.title}</span>
              </button>
            );
          })}

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
                    "h-9 w-9 border border-white/70 bg-white/80 text-[10px] transition-all",
                    openGroup === ACCOUNT_GROUP && "ring-2 ring-[hsl(var(--secondary))/0.35]"
                  )}
                />
                <NotificationBadge count={unreadCount} className="-right-2 -top-1" />
              </div>
            ) : (
              <div className={navIconClass(openGroup === ACCOUNT_GROUP)}>
                <UserCircle className="h-5 w-5" />
              </div>
            )}
            <span className={navLabelClass(openGroup === ACCOUNT_GROUP)}>Tài khoản</span>
          </button>

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
                  className={cn("h-5 w-5", (isMoreActive || isMoreOpen) && "stroke-[2.4px]")}
                />
              </div>
              <span className={navLabelClass(isMoreActive || isMoreOpen)}>Thêm</span>
            </button>
          )}
        </div>
      </nav>

      <Sheet open={!!openGroup} onOpenChange={(open) => !open && setOpenGroup(null)}>
        <SheetContent
          side="bottom"
          className={sheetSurfaceClass}
          title={openGroupTitle}
          description={openGroupDescription}
        >
          {renderSheetChrome(openGroupTitle, openGroupDescription)}

          {openGroup === ACCOUNT_GROUP ? (
            <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-[calc(env(safe-area-inset-bottom)+1rem)] sm:px-5">
              {user && (
                <div className="admin-mobile-tile mb-2.5 flex items-center gap-3 rounded-2xl p-3">
                  <UserAvatar
                    email={user.email}
                    name={user.name}
                    username={user.username}
                    size="lg"
                    className="h-11 w-11"
                  />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-[hsl(var(--foreground))]">
                      {user.name}
                    </p>
                    <p className="truncate text-xs text-[hsl(var(--muted-foreground))]">
                      {user.email}
                    </p>
                  </div>
                </div>
              )}

              <ul className="ct-menu ct-menu-sm !grid grid-cols-2 gap-2 bg-transparent p-0 sm:grid-cols-4">
                {renderNavTile({ title: "Thông báo", icon: Bell }, false, () => {
                  setOpenGroup(null);
                  setIsNotificationSheetOpen(true);
                })}
                {renderNavTile({ title: "Hồ sơ", icon: UserCircle }, false, () => {
                  setOpenGroup(null);
                  setIsProfileOpen(true);
                })}
                {renderNavTile({ title: "Đổi mật khẩu", icon: Key }, false, () => {
                  setOpenGroup(null);
                  openModal(MODAL_IDS.CHANGE_PASSWORD);
                })}
                {renderNavTile({ title: "Đăng xuất", icon: LogOut }, false, () => {
                  setOpenGroup(null);
                  logout();
                }, "danger")}
              </ul>
            </div>
          ) : (
            <ul className="ct-menu ct-menu-sm !grid min-h-0 flex-1 grid-cols-2 gap-2 overflow-y-auto bg-transparent px-4 pb-[calc(env(safe-area-inset-bottom)+1rem)] pt-0 sm:px-5">
              {openGroup?.submenu?.map((item) => {
                const isActive = item.path ? location.pathname.startsWith(item.path) : false;
                return renderNavTile(item, isActive, () => {
                  setOpenGroup(null);
                  if (item.onClick) item.onClick();
                  else if (item.path) navigate(item.path);
                });
              })}
            </ul>
          )}
        </SheetContent>
      </Sheet>

      <Sheet open={isMoreOpen} onOpenChange={setIsMoreOpen}>
        <SheetContent
          side="bottom"
          className={sheetSurfaceClass}
          title={moreTitle}
          description={moreDescription}
        >
          {renderSheetChrome(moreTitle, moreDescription)}
          <ul className="ct-menu ct-menu-sm !grid min-h-0 flex-1 grid-cols-2 gap-2 overflow-y-auto bg-transparent px-4 pb-[calc(env(safe-area-inset-bottom)+1rem)] pt-0 sm:grid-cols-3 sm:px-5">
            {moreItems?.map((item) => {
              const isActive = item.path ? location.pathname.startsWith(item.path) : false;
              return renderNavTile(item, isActive, () => {
                setIsMoreOpen(false);
                if (item.onClick) item.onClick();
                else if (item.path) navigate(item.path);
              });
            })}
          </ul>
        </SheetContent>
      </Sheet>

      <NotificationSheet
        variant="corporate"
        isOpen={isNotificationSheetOpen}
        onClose={() => setIsNotificationSheetOpen(false)}
      />

      <UserProfileSheet isOpen={isProfileOpen} onClose={() => setIsProfileOpen(false)} />
    </>
  );
};
