import { Bell, BellOff, Loader2, Smartphone } from 'lucide-react';
import { usePushNotifications } from '@/hooks/usePushNotifications';
import { cn } from '@/lib/utils';

interface PushNotificationToggleProps {
  className?: string;
}

/**
 * Toggle component for web push notifications.
 * Shows current permission state and allows subscribe/unsubscribe.
 */
export const PushNotificationToggle = ({ className }: PushNotificationToggleProps) => {
  const {
    permissionStatus,
    isSubscribed,
    isSupported,
    requestPermission,
    unsubscribe,
    isLoading,
  } = usePushNotifications();

  if (!isSupported) {
    return (
      <div className={cn('flex items-center gap-3 p-3 rounded-xl bg-muted/50', className)}>
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-muted">
          <Smartphone className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-muted-foreground">Thông báo đẩy</p>
          <p className="text-xs text-muted-foreground/70">Trình duyệt không hỗ trợ</p>
        </div>
      </div>
    );
  }

  if (permissionStatus === 'denied') {
    return (
      <div className={cn('flex items-center gap-3 p-3 rounded-xl bg-red-50 border border-red-100', className)}>
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-red-100">
          <BellOff className="h-4 w-4 text-red-500" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-red-700">Thông báo đã bị chặn</p>
          <p className="text-xs text-red-500/70">Vui lòng bật lại trong cài đặt trình duyệt</p>
        </div>
      </div>
    );
  }

  const handleToggle = async () => {
    if (isSubscribed) {
      await unsubscribe();
    } else {
      await requestPermission();
    }
  };

  return (
    <button
      type="button"
      onClick={handleToggle}
      disabled={isLoading}
      className={cn(
        'flex items-center gap-3 p-3 rounded-xl border transition-all w-full text-left',
        isSubscribed
          ? 'bg-emerald-50 border-emerald-200 hover:bg-emerald-100'
          : 'bg-card border-border hover:bg-accent',
        isLoading && 'opacity-60 pointer-events-none',
        className
      )}
    >
      <div className={cn(
        'flex h-9 w-9 items-center justify-center rounded-full',
        isSubscribed ? 'bg-emerald-100' : 'bg-muted'
      )}>
        {isLoading ? (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        ) : isSubscribed ? (
          <Bell className="h-4 w-4 text-emerald-600" />
        ) : (
          <Bell className="h-4 w-4 text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className={cn('text-sm font-medium', isSubscribed ? 'text-emerald-700' : 'text-foreground')}>
          Thông báo đẩy
        </p>
        <p className="text-xs text-muted-foreground">
          {isSubscribed ? 'Đã bật — nhận thông báo ngay trên thiết bị' : 'Nhấn để bật thông báo đẩy'}
        </p>
      </div>
      {/* Toggle indicator */}
      <div className={cn(
        'w-10 h-6 rounded-full transition-colors flex items-center px-0.5',
        isSubscribed ? 'bg-emerald-500 justify-end' : 'bg-muted justify-start'
      )}>
        <div className="w-5 h-5 rounded-full bg-white shadow-sm" />
      </div>
    </button>
  );
};
