import { useState, useMemo } from 'react';
import { Loader2, CheckCheck, Bell, X, ArrowLeft } from 'lucide-react';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { NotificationItem } from './NotificationItem';
import { NotificationDetailModal } from './NotificationDetailModal';
import { PushNotificationToggle } from './PushNotificationToggle';
import {
  useInfiniteNotifications,
  useUnreadNotifications,
  useMarkAllAsRead
} from '@/hooks/api/useNotifications';
import type { Notification } from '@/types/api/notification.types';

type NotificationVariant = 'employee' | 'corporate';

const THEME = {
  employee: {
    headerBg: 'bg-employee',
    activeTabText: 'text-employee-700',
    activeTabCount: 'bg-employee-100 text-employee-700',
    loaderColor: 'text-employee',
  },
  corporate: {
    headerBg: 'bg-employee',
    activeTabText: 'text-employee-700',
    activeTabCount: 'bg-employee-100 text-employee-700',
    loaderColor: 'text-employee',
  },
} as const;

type NotificationView = 'unread' | 'all';

interface NotificationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  variant?: NotificationVariant;
}

export const NotificationSheet = ({ isOpen, onClose, variant = 'employee' }: NotificationSheetProps) => {
  const isMobile = useIsMobile();
  const headerPaddingTop = `calc(env(safe-area-inset-top, 0px) + 0.875rem)`;
  const theme = THEME[variant];
  const [activeView, setActiveView] = useState<NotificationView>('unread');
  const [selectedNotification, setSelectedNotification] = useState<Notification | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const { data: unreadData, isLoading: isLoadingUnread, error: unreadError } = useUnreadNotifications(isOpen);
  const {
    data: infiniteNotificationsData,
    isLoading: isLoadingAll,
    error: allError,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage
  } = useInfiniteNotifications({});

  const markAllAsRead = useMarkAllAsRead();

  const unreadNotifications = unreadData?.notifications || [];
  const allNotifications = useMemo(() => {
    if (!infiniteNotificationsData?.pages) return [];
    return infiniteNotificationsData.pages
      .flatMap(page => (Array.isArray(page) ? page : page?.data || []))
      .filter(Boolean);
  }, [infiniteNotificationsData]);

  const unreadCount = unreadData?.count || 0;

  const handleNotificationClick = (notification: Notification) => {
    setSelectedNotification(notification);
    setIsModalOpen(true);
  };

  const renderEmpty = (message: string) => (
    <div className="flex flex-col items-center justify-center py-20 px-6">
      <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <Bell className="h-7 w-7 text-gray-300" />
      </div>
      <p className="text-sm font-medium text-gray-400">{message}</p>
    </div>
  );

  const renderLoading = () => (
    <div className="flex items-center justify-center py-16">
      <Loader2 className={`h-5 w-5 animate-spin ${theme.loaderColor}`} />
    </div>
  );

  const renderContent = () => {
    if (activeView === 'unread') {
      if (isLoadingUnread) return renderLoading();
      if (unreadError) return renderEmpty('Không thể tải thông báo');
      if (unreadNotifications.length === 0) return renderEmpty('Không có thông báo chưa đọc');
      return (
        <div className="divide-y divide-gray-100">
          {unreadNotifications.map((n) => (
            <NotificationItem key={n.id} notification={n} showMarkAsRead onClick={() => handleNotificationClick(n)} />
          ))}
        </div>
      );
    }

    if (isLoadingAll && allNotifications.length === 0) return renderLoading();
    if (allError && allNotifications.length === 0) return renderEmpty('Không thể tải thông báo');
    if (allNotifications.length === 0) return renderEmpty('Chưa có thông báo nào');

    return (
      <InfiniteScrollContainer
        onLoadMore={() => { fetchNextPage(); }}
        hasMore={!!hasNextPage}
        isLoading={isFetchingNextPage}
        isError={!!allError}
        className="h-full"
      >
        <div className="divide-y divide-gray-100">
          {allNotifications.map((n) => {
            if (!n?.id) return null;
            return <NotificationItem key={n.id} notification={n} showMarkAsRead onClick={() => handleNotificationClick(n)} />;
          })}
        </div>
      </InfiniteScrollContainer>
    );
  };

  return (
    <>
      <Sheet open={isOpen} onOpenChange={onClose}>
        <SheetContent
          side={isMobile ? "bottom" : "right"}
          title="Thông báo"
          description="Danh sách thông báo của bạn"
          data-theme="congtruong"
          className={cn(
            "!w-full sm:!w-[420px] p-0 flex flex-col overflow-hidden bg-white",
            isMobile
              ? "h-auto max-h-[92dvh] rounded-t-[1.5rem] shadow-[0_-10px_32px_rgba(16,24,40,0.16)]"
              : "h-full"
          )}
        >

          {/* Mobile drag handle */}
          {isMobile && (
            <div className={cn("flex justify-center pt-2.5 pb-1 flex-shrink-0", theme.headerBg)}>
              <div className="h-1 w-9 rounded-full bg-white/25" />
            </div>
          )}

          {/* Header */}
          <div
            className={cn("text-white px-4 pb-4", theme.headerBg)}
            style={{ paddingTop: headerPaddingTop }}
          >
            <div className="mb-3 flex items-center justify-between gap-2">
              <button
                onClick={onClose}
                className="ct-btn ct-btn-ghost ct-btn-circle h-11 min-h-11 w-11 border-0 bg-transparent p-0 text-white hover:bg-white/20"
                aria-label="Đóng"
              >
                <ArrowLeft className="h-6 w-6 text-white" />
              </button>
              <h2 className="min-w-0 flex-1 text-center text-base font-bold text-white">Thông báo</h2>
              {activeView === 'unread' && unreadNotifications.length > 0 ? (
                <button
                  onClick={() => markAllAsRead.mutate()}
                  disabled={markAllAsRead.isPending}
                  className="ct-btn ct-btn-ghost h-11 min-h-11 max-w-[6.75rem] shrink-0 gap-1.5 border-0 px-2 text-xs font-medium text-white/90 hover:bg-white/20 disabled:bg-transparent"
                  aria-label="Đánh dấu tất cả đã đọc"
                >
                  {markAllAsRead.isPending
                    ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    : <CheckCheck className="h-3.5 w-3.5" />}
                  <span className="min-w-0 truncate">Đọc tất cả</span>
                </button>
              ) : (
                <div className="w-11" />
              )}
            </div>

            {/* Tab switcher */}
            <div role="tablist" aria-label="Lọc thông báo" className="ct-tabs ct-tabs-box grid grid-cols-2 gap-1 rounded-xl bg-white/20 p-1">
              {([
                { value: 'unread' as const, label: 'Chưa đọc', count: unreadCount },
                { value: 'all'    as const, label: 'Tất cả',   count: null },
              ]).map((tab) => (
                <button
                  key={tab.value}
                  type="button"
                  role="tab"
                  aria-selected={activeView === tab.value}
                  onClick={() => setActiveView(tab.value)}
                  className={`ct-tab h-11 min-h-11 w-full justify-center gap-1.5 rounded-lg border-0 py-0 text-sm font-semibold transition-colors ${
                    activeView === tab.value
                      ? `ct-tab-active bg-white shadow-sm ${theme.activeTabText}`
                      : 'text-white/80 hover:text-white'
                  }`}
                >
                  {tab.label}
                  {tab.count != null && tab.count > 0 && (
                    <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full leading-none ${
                      activeView === tab.value ? theme.activeTabCount : 'bg-white/30 text-white'
                    }`}>
                      {tab.count > 99 ? '99+' : tab.count}
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>

          {/* Content */}
          <div className="min-h-0 flex-1 overflow-y-auto bg-white">
            {renderContent()}
          </div>

          {/* Push notification toggle — pinned footer */}
          <div
            className="border-t bg-white px-3 py-2"
            style={{ paddingBottom: "calc(env(safe-area-inset-bottom, 0px) + 0.5rem)" }}
          >
            <PushNotificationToggle />
          </div>

        </SheetContent>
      </Sheet>

      <NotificationDetailModal
        notification={selectedNotification}
        isOpen={isModalOpen}
        onClose={() => { setIsModalOpen(false); setSelectedNotification(null); }}
      />
    </>
  );
};
