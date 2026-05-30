import { useState } from "react";
import { Button } from "@/components/ui/button";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { NotificationBadge } from "@/components/notifications/NotificationBadge";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { useBellAnimation } from "@/hooks/useBellAnimation";
import { useAppState } from "@/contexts";
import { cn } from "@/lib/utils";

interface DesktopHeaderProps {
  className?: string;
  showSidebarToggle?: boolean;
}

export const DesktopHeader = ({
  className,
  showSidebarToggle = true,
}: DesktopHeaderProps) => {
  const { pageTitle, pageSubtitle } = useAppState();
  const [isNotificationSheetOpen, setIsNotificationSheetOpen] = useState(false);

  const { data: unreadData, isLoading: isLoadingCount } = useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;
  const bellIcon = useBellAnimation(unreadCount);

  return (
    <>
      <header
        className={cn(
          "flex h-11 shrink-0 items-center gap-3 px-4 border-b border-border/40 bg-background/80 backdrop-blur-sm z-40",
          className
        )}
      >
        {showSidebarToggle && (
          <SidebarTrigger className="hidden" />
        )}

        {/* Page title */}
        <div className="flex flex-col min-w-0 flex-1">
          {pageTitle && (
            <>
              <h1 className="text-sm font-semibold text-foreground truncate leading-tight">{pageTitle}</h1>
              {pageSubtitle && (
                <p className="text-xs text-muted-foreground truncate leading-tight">{pageSubtitle}</p>
              )}
            </>
          )}
        </div>

        {/* Notification bell */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setIsNotificationSheetOpen(true)}
          className="relative h-8 w-8 hover:bg-accent/50"
          aria-label={`Thông báo${unreadCount > 0 ? ` (${unreadCount} chưa đọc)` : ""}`}
        >
          <img
            src={bellIcon}
            alt="Thông báo"
            className={cn("h-4 w-4", unreadCount > 0 && "animate-bell-swing")}
          />
          {!isLoadingCount && <NotificationBadge count={unreadCount} />}
        </Button>
      </header>

      <NotificationSheet
        isOpen={isNotificationSheetOpen}
        onClose={() => setIsNotificationSheetOpen(false)}
      />
    </>
  );
};
