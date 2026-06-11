import { useState } from 'react';
import { createPortal } from 'react-dom';
import { Bell, X, Loader2 } from 'lucide-react';
import { NotificationSheet } from '@/components/notifications/NotificationSheet';
import { NotificationBadge } from '@/components/notifications/NotificationBadge';
import { useUnreadNotifications } from '@/hooks/api/useNotifications';
import { usePushNotifications } from '@/hooks/usePushNotifications';
import { useBellAnimation } from '@/hooks/useBellAnimation';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { authManager } from '@/lib/auth';
import { cn } from '@/lib/utils';

export const NotificationFAB = () => {
  const isMobile = useIsMobile();
  const [isNotificationSheetOpen, setIsNotificationSheetOpen] = useState(false);

  const { data: unreadData, isLoading } = useUnreadNotifications();
  const unreadCount = unreadData?.count || 0;
  const bellIcon = useBellAnimation(unreadCount);

  const { isSubscribed, isSupported, permissionStatus, requestPermission } = usePushNotifications();

  const handleBellClick = () => {
    setIsNotificationSheetOpen(true);
  };

  const role = authManager.getUserRole();
  const isAdminOrPartner = role === 'admin' || role === 'partner';

  const hasUnread = unreadCount > 0;

  return (
    <>
      {hasUnread && (
        <button
          onClick={handleBellClick}
          className={cn(
            'fixed right-4 z-50 flex items-center justify-center rounded-full shadow-sm transition-all duration-300',
            isAdminOrPartner
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'text-white active:opacity-90',
            'active:scale-95',
            'animate-in slide-in-from-bottom-4 fade-in duration-300',
            isMobile ? 'bottom-20 h-12 w-12' : 'bottom-6 h-12 w-12'
          )}
          aria-label={`Thông báo (${unreadCount} chưa đọc)`}
        >
          <img
            src={bellIcon}
            alt=""
            className="h-5 w-5 brightness-0 invert animate-bell-swing"
            aria-hidden="true"
          />
          <NotificationBadge count={unreadCount} className="absolute -top-1 -right-1" />
        </button>
      )}

      <NotificationSheet variant="corporate" isOpen={isNotificationSheetOpen} onClose={() => setIsNotificationSheetOpen(false)} />
    </>
  );
};
