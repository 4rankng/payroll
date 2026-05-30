import { useState, useMemo } from 'react';
import { Loader2, CheckCheck, Bell, X, ArrowLeft } from 'lucide-react';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { NotificationItem } from './NotificationItem';
import { NotificationDetailModal } from './NotificationDetailModal';
import {
  useInfiniteNotifications,
  useUnreadNotifications,
  useMarkAllAsRead
} from '@/hooks/api/useNotifications';
import type { Notification } from '@/types/api/notification.types';

import { EMPLOYEE_BRAND_COLOR } from '@/constants/branding';

type NotificationVariant = 'employee' | 'corporate';

const THEME = {
  employee: {
    headerBg: EMPLOYEE_BRAND_COLOR,
    activeTabText: 'text-green-700',
    activeTabCount: 'bg-green-100 text-green-700',
    loaderColor: EMPLOYEE_BRAND_COLOR,
  },
  corporate: {
    headerBg: 'hsl(220, 90%, 12%)',
    activeTabText: 'text-blue-700',
    activeTabCount: 'bg-blue-100 text-blue-700',
    loaderColor: 'hsl(220, 90%, 12%)',
  },
} as const;

type NotificationView = 'unread' | 'all';

interface NotificationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  variant?: NotificationVariant;
}

export const NotificationSheet = ({ isOpen, onClose, variant = 'employee' }: NotificationSheetProps) => {
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
      <Loader2 className="h-5 w-5 animate-spin" style={{ color: theme.loaderColor }} />
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
        <SheetContent side="right" className="!w-full sm:!w-[420px] p-0 flex flex-col h-full bg-gray-50">

          {/* Header */}
          <div className="text-white px-4 pb-4" style={{ background: theme.headerBg, paddingTop: "calc(env(safe-area-inset-top) + 20px)" }}>
            <div className="flex items-center justify-between mb-4">
              <button
                onClick={onClose}
                className="h-8 w-8 flex items-center justify-center rounded-full hover:bg-white/20 transition-colors"
                aria-label="Đóng"
              >
                <ArrowLeft className="h-5 w-5 text-white" />
              </button>
              <h2 className="text-base font-bold text-white">Thông báo</h2>
              {activeView === 'unread' && unreadNotifications.length > 0 ? (
                <button
                  onClick={() => markAllAsRead.mutate()}
                  disabled={markAllAsRead.isPending}
                  className="h-8 flex items-center gap-1.5 px-2 rounded-full hover:bg-white/20 transition-colors text-white/90 text-xs font-medium disabled:opacity-50"
                  aria-label="Đánh dấu tất cả đã đọc"
                >
                  {markAllAsRead.isPending
                    ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    : <CheckCheck className="h-3.5 w-3.5" />}
                  <span>Đọc tất cả</span>
                </button>
              ) : (
                <div className="w-8" />
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
                  className={`flex-1 flex items-center justify-center gap-1.5 py-1.5 rounded-lg text-sm font-semibold transition-all ${
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
