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
        'group flex items-start gap-3 px-4 py-3.5 cursor-pointer transition-colors',
        isUnread ? 'bg-white hover:bg-gray-50' : 'bg-white hover:bg-gray-50 opacity-75',
        className
      )}
      onClick={handleClick}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleClick(); } }}
      tabIndex={0}
      role="button"
      aria-label={`${notification.title}. ${isUnread ? 'Chưa đọc' : 'Đã đọc'}.`}
    >
      {/* Accent dot */}
      <div className="mt-1.5 shrink-0">
        <div className="w-2.5 h-2.5 rounded-full" style={{ background: isUnread ? accent : '#D1D5DB' }} />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2 mb-0.5">
          <h4 className={cn('text-sm leading-snug truncate', isUnread ? 'font-semibold text-gray-900' : 'font-medium text-gray-500')}>
            {notification.title}
          </h4>
          <div className="flex items-center gap-1.5 shrink-0">
            <span className="text-xs text-gray-400 tabular-nums">{formatNotificationDate(notification.created_at)}</span>
            {isUnread && <div className="w-1.5 h-1.5 rounded-full shrink-0 bg-employee" />}
          </div>
        </div>

        <div className="flex items-end justify-between gap-2">
          <p className="text-xs text-gray-400 leading-relaxed line-clamp-2 flex-1">
            {getNotificationPreview(notification.message)}
          </p>
          {isUnread && showMarkAsRead && (
            <button
              className="h-6 w-6 flex items-center justify-center rounded-full opacity-0 group-hover:opacity-100 transition-opacity shrink-0 hover:bg-green-50"
              onClick={handleMarkAsRead}
              disabled={markAsRead.isPending}
              aria-label="Đánh dấu đã đọc"
            >
              <Check className="h-3.5 w-3.5 text-employee" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
