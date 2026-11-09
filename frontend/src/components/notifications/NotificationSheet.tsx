import { useState, useMemo } from 'react';
import { Loader2, CheckCheck, X, ArrowLeft } from 'lucide-react';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { NotificationItem } from './NotificationItem';
import { NotificationDetailModal } from './NotificationDetailModal';
import { EmptyState } from '@/components/shared/EmptyState';
import { PushNotificationToggle } from './PushNotificationToggle';
import {
  useInfiniteNotifications,
  useUnreadNotifications,
  useMarkAllAsRead
} from '@/hooks/api/useNotifications';
import type { Notification } from '@/types/api/notification.types';

type NotificationVariant = 'employee' | 'corporate';

// Header/tab chrome is token-only (bg-card/bg-muted/foreground/border) so it
// follows the console's light/dark themes and the employee portal without a
// per-variant palette — a static green slab is wrong on both canvases.
// `variant` stays only as the daisyUI theme scope (data-theme below).
type NotificationView = 'unread' | 'all';

interface NotificationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  variant?: NotificationVariant;
}

export const NotificationSheet = ({ isOpen, onClose, variant = 'employee' }: NotificationSheetProps) => {
  const isMobile = useIsMobile();
  const headerPaddingTop = `calc(env(safe-area-inset-top, 0px) + 0.875rem)`;
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

  const renderEmpty = (message: string) => <EmptyState title={message} size="sm" />;

  const renderLoading = () => (
    <div className="flex items-center justify-center py-16">
      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
    </div>
  );

  const renderContent = () => {
    if (activeView === 'unread') {
      if (isLoadingUnread) return renderLoading();
      if (unreadError) return renderEmpty('Không thể tải thông báo');
      if (unreadNotifications.length === 0) return renderEmpty('Không có thông báo chưa đọc');
      return (
        <div className="divide-y divide-border">
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
        <div className="divide-y divide-border">
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
          data-theme={variant === 'employee' ? 'employee' : 'congtruong'}
          className={cn(
            "w-full lg:w-[420px] flex flex-col overflow-hidden bg-card p-0 text-card-foreground shadow-none",
            isMobile
              ? "h-[100dvh] max-h-none rounded-none"
              : "h-full"
          )}
        >
          {/* Header */}
          <div
            className="border-b border-border bg-card px-4 pb-4 text-foreground"
            style={{ paddingTop: headerPaddingTop }}
          >
            <div className="mb-3 flex items-center justify-between gap-2">
              <button
                onClick={onClose}
                className="inline-flex items-center justify-center rounded-full h-11 min-h-11 w-11 border-0 bg-transparent p-0 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                aria-label="Đóng"
              >
                <ArrowLeft className="h-6 w-6" />
              </button>
              <h2 className="min-w-0 flex-1 text-center text-base font-bold text-foreground">Thông báo</h2>
              {activeView === 'unread' && unreadNotifications.length > 0 ? (
                <button
                  onClick={() => markAllAsRead.mutate()}
                  disabled={markAllAsRead.isPending}
                  className="inline-flex items-center justify-center gap-1.5 h-11 min-h-11 max-w-[6.75rem] shrink-0 rounded-md border-0 bg-transparent px-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:bg-transparent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
            <div role="tablist" aria-label="Lọc thông báo" className="grid grid-cols-2 gap-1 rounded-lg bg-muted p-1">
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
                  className={`inline-flex items-center justify-center h-11 min-h-11 w-full gap-1.5 rounded-lg border-0 py-0 text-sm font-semibold transition-colors ${
                    activeView === tab.value
                      ? 'bg-card text-foreground shadow-xs'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {tab.label}
                  {tab.count != null && tab.count > 0 && (
                    <span className={`text-xs font-bold px-1.5 py-0.5 rounded-full leading-none ${
                      activeView === tab.value ? 'bg-success/10 text-success' : 'bg-foreground/10 text-foreground'
                    }`}>
                      {tab.count > 99 ? '99+' : tab.count}
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>

          {/* Content */}
          <div className="min-h-0 flex-1 overflow-y-auto bg-card">
            {renderContent()}
          </div>

          {/* Push notification toggle — pinned footer */}
          <div
            className="border-t border-border bg-card px-3 py-2"
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
