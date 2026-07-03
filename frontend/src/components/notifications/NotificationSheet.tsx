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
    activeTabText: 'text-green-700',
    activeTabCount: 'bg-green-100 text-green-700',
    loaderColor: 'text-employee',
  },
  corporate: {
    headerBg: 'bg-[hsl(220,90%,12%)]',
    activeTabText: 'text-blue-700',
    activeTabCount: 'bg-blue-100 text-blue-700',
    loaderColor: 'text-[hsl(220,90%,12%)]',
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
  const isEmployeeMobile = isMobile && variant === 'employee';
  // Employee full-screen sheet hugs the top safe-area tighter than the
  // rounded bottom-sheet variant.
  const headerPaddingTop = `calc(env(safe-area-inset-top, 0px) + ${
    isEmployeeMobile ? '0.875rem' : '1.25rem'
  })`;
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
          className={cn(
            "!w-full sm:!w-[420px] p-0 flex flex-col overflow-hidden bg-white",
            isMobile
              ? isEmployeeMobile
                ? "h-[100dvh] max-h-[100dvh] !rounded-none shadow-none"
                : "h-full max-h-[94dvh] rounded-t-2xl shadow-[0_-4px_24px_rgba(0,0,0,0.08)]"
              : "h-full"
          )}
        >

          {/* Mobile drag handle */}
          {isMobile && !isEmployeeMobile && (
            <div className={cn("flex justify-center pt-2.5 pb-1 flex-shrink-0", theme.headerBg)}>
              <div className="h-1 w-9 rounded-full bg-white/25" />
            </div>
          )}

          {/* Header */}
          <div
            className={cn("text-white px-4 pb-4", theme.headerBg, isEmployeeMobile && "px-5")}
            style={{ paddingTop: headerPaddingTop }}
          >
            <div className="flex items-center justify-between mb-4">
              <button
                onClick={onClose}
                className="h-11 w-11 -ml-2 flex items-center justify-center rounded-full hover:bg-white/20 transition-colors"
                aria-label="Đóng"
              >
                <ArrowLeft className="h-6 w-6 text-white" />
              </button>
              <h2 className="text-base font-bold text-white">Thông báo</h2>
              {activeView === 'unread' && unreadNotifications.length > 0 ? (
                <button
                  onClick={() => markAllAsRead.mutate()}
                  disabled={markAllAsRead.isPending}
                  className="h-11 flex items-center gap-1.5 px-2 rounded-full hover:bg-white/20 transition-colors text-white/90 text-xs font-medium disabled:opacity-50"
                  aria-label="Đánh dấu tất cả đã đọc"
                >
                  {markAllAsRead.isPending
                    ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    : <CheckCheck className="h-3.5 w-3.5" />}
                  <span>Đọc tất cả</span>
                </button>
              ) : (
                <div className="w-11" />
              )}
            </div>

            {/* Tab switcher */}
            <div className="flex bg-white/20 rounded-xl p-1 gap-1">
              {([
                { value: 'unread' as const, label: 'Chưa đọc', count: unreadCount },
                { value: 'all'    as const, label: 'Tất cả',   count: null },
              ]).map((tab) => (
                <button
                  key={tab.value}
                  onClick={() => setActiveView(tab.value)}
                  className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-lg text-sm font-semibold transition-all ${
                    activeView === tab.value
                      ? `bg-white shadow-sm ${theme.activeTabText}`
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
          <div className="flex-1 min-h-0 overflow-y-auto bg-white">
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
