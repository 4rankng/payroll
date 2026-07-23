import { memo } from 'react';
import { X, Bell } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogClose } from '@/components/ui/dialog';
import { useMarkAsReadSilent } from '@/hooks/api/useNotifications';
import type { Notification } from '@/types/api/notification.types';
import {
  getNotificationStyle,
  formatNotificationDetailDate,
} from '@/utils/notification-styles';

interface NotificationDetailModalProps {
  notification: Notification | null;
  isOpen: boolean;
  onClose: () => void;
}

export const NotificationDetailModal = memo(function NotificationDetailModal({
  notification,
  isOpen,
  onClose,
}: NotificationDetailModalProps) {
  const markAsReadSilent = useMarkAsReadSilent();

  const handleClose = () => {
    if (notification && !notification.read_at) {
      markAsReadSilent.mutate(notification.id);
    }
    onClose();
  };

  if (!notification) return null;

  const style = getNotificationStyle(notification.type);

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent
        className="max-w-lg w-[90vw] bg-background border-border"
        contentPadding="none"
        title={notification.title}
        description={notification.message}
        hideCloseButton
      >
        {/* Header */}
        <div className="flex items-center gap-3 border-b border-emerald-900 bg-emerald-950 px-6 pb-4 pt-5">
          <div className={`flex-shrink-0 w-10 h-10 ${style.bgColor} rounded-full flex items-center justify-center`}>
            <Bell className={`h-5 w-5 ${style.iconColor}`} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-base font-semibold text-white leading-tight">
              {notification.title}
            </p>
            <p className="mt-0.5 text-xs text-emerald-200/75">
              {formatNotificationDetailDate(notification.created_at)}
            </p>
          </div>
          <DialogClose
            onClick={handleClose}
            className="w-8 h-8 flex-shrink-0 flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20 transition-colors outline-none focus:ring-2 focus:ring-white/30"
          >
            <X className="w-4 h-4 text-white" />
            <span className="sr-only">Đóng</span>
          </DialogClose>
        </div>

        {/* Body */}
        <div className="px-6 py-5">
          <p className="text-sm text-foreground leading-relaxed whitespace-pre-line">
            {notification.message}
          </p>
        </div>

        {/* Footer */}
        <div className="flex justify-end border-t border-border px-6 pb-5 pt-3">
          <Button
            onClick={handleClose}
            disabled={markAsReadSilent.isPending}
            className="min-w-[80px]"
          >
            Đóng
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
});
