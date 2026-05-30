import { useState } from "react";
import { Button } from "@/components/ui/button";
import { UserAvatarDropdown } from "@/components/shared/UserAvatarDropdown";
import { NotificationBadge } from "@/components/notifications";
import { NotificationSheet } from "@/components/notifications";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { useBellAnimation } from "@/hooks/useBellAnimation";
import { useAppState } from "@/contexts";
import { cn } from "@/lib/utils";

interface MobileHeaderProps {
  className?: string;
}

export const MobileHeader = ({ className }: MobileHeaderProps) => {
  const { pageTitle } = useAppState();
  const [isNotificationSheetOpen, setIsNotificationSheetOpen] = useState(false);

  const { data: unreadData, isLoading: isLoadingCount } =
    useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;
  const bellIcon = useBellAnimation(unreadCount);

  return (
    <>
      <header
        className={cn(
          "flex h-14 shrink-0 items-center gap-2 px-3 border-b lg:hidden bg-background/95 backdrop-blur-md sticky top-0 z-50",
          className,
        )}
      >
        <span className="flex-1 typography-body-large font-semibold text-foreground truncate">
          {pageTitle || ""}
        </span>

        <div className="flex items-center shrink-0">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setIsNotificationSheetOpen(true)}
            className="relative h-10 w-10 touch-manipulation"
            aria-label={`Thông báo${unreadCount > 0 ? ` (${unreadCount} chưa đọc)` : ""}`}
          >
            <img
              src={bellIcon}
              alt="Thông báo"
              className={cn(
                "h-4 w-4 transition-all duration-1000",
                unreadCount > 0 && "animate-bell-swing",
              )}
            />
            {!isLoadingCount && <NotificationBadge count={unreadCount} />}
          </Button>

          <UserAvatarDropdown />
        </div>
      </header>

      <NotificationSheet
        isOpen={isNotificationSheetOpen}
        onClose={() => setIsNotificationSheetOpen(false)}
      />
    </>
  );
};
