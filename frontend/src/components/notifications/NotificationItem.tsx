import { cn } from '@/lib/utils';
import { Check } from 'lucide-react';
import type { Notification } from '@/types/api/notification.types';
import { useMarkAsRead, useMarkAsReadSilent } from '@/hooks/api/useNotifications';
import { getNotificationPreview } from '@/utils/notification-helpers';
import { getNotificationAccentColor, formatNotificationDate } from '@/utils/notification-styles';

interface NotificationItemProps {
  notification: Notification;
  showMarkAsRead?: boolean;
  onClick?: () => void;
  className?: string;
}

export const NotificationItem = ({ notification, showMarkAsRead = true, onClick, className }: NotificationItemProps) => {
  const markAsRead = useMarkAsRead();
  const markAsReadSilent = useMarkAsReadSilent();
  const isUnread = !notification.read_at;
  const accent = getNotificationAccentColor(notification.type);

  const handleClick = () => {
    if (isUnread) markAsReadSilent.mutate(notification.id);
    onClick?.();
  };

  const handleMarkAsRead = (e: React.MouseEvent) => {
    e.stopPropagation();
    markAsRead.mutate(notification.id);
  };

  return (
    <div
      className={cn(
        'group flex items-start gap-3 bg-base-100 px-4 py-3.5 transition-colors hover:bg-base-200/60',
        className
      )}
    >
      {/* Accent dot */}
      <div className="mt-1.5 shrink-0">
        <div className="w-2.5 h-2.5 rounded-full" style={{ background: isUnread ? accent : '#D1D5DB' }} />
      </div>

      <button
        type="button"
        className="min-h-11 min-w-0 flex-1 rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        onClick={handleClick}
        aria-label={`${notification.title}. ${isUnread ? 'Chưa đọc' : 'Đã đọc'}.`}
      >
        <span className="mb-0.5 flex items-start justify-between gap-2">
          <span className={cn('line-clamp-2 break-words text-sm leading-snug', isUnread ? 'font-semibold text-gray-900' : 'font-medium text-gray-600')}>
            {notification.title}
          </span>
          <span className="flex items-center gap-1.5 shrink-0">
            <span className="text-xs text-gray-600 tabular-nums">{formatNotificationDate(notification.created_at)}</span>
            {isUnread && <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-success" />}
          </span>
        </span>

          <span className="text-xs text-gray-600 leading-relaxed line-clamp-2">
            {getNotificationPreview(notification.message)}
          </span>
      </button>
      {isUnread && showMarkAsRead && (
        <button
          type="button"
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full transition-opacity hover:bg-success/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:opacity-0 sm:group-hover:opacity-100 sm:focus-visible:opacity-100"
          onClick={handleMarkAsRead}
          disabled={markAsRead.isPending}
          aria-label="Đánh dấu đã đọc"
        >
          <Check className="h-4 w-4 text-emerald-700" />
        </button>
      )}
    </div>
  );
};
