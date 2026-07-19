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
      <div className={cn('flex min-h-16 items-center gap-3 rounded-xl bg-muted/50 p-3', className)}>
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
      <div className={cn('flex min-h-16 items-center gap-3 rounded-xl border border-red-100 bg-red-50 p-3', className)}>
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
    <div
      className={cn(
        'flex min-h-16 items-center gap-3 rounded-xl border p-3 transition-colors',
        isSubscribed
          ? 'border-success/20 bg-success/5'
          : 'border-base-300 bg-base-100',
        isLoading && 'opacity-60',
        className
      )}
    >
      <div className={cn(
        'flex h-9 w-9 items-center justify-center rounded-full',
        isSubscribed ? 'bg-success/10' : 'bg-base-200'
      )}>
        {isLoading ? (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        ) : isSubscribed ? (
          <Bell className="h-4 w-4 text-success" />
        ) : (
          <Bell className="h-4 w-4 text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className={cn('text-sm font-semibold', isSubscribed ? 'text-success' : 'text-base-content')}>
          Thông báo đẩy
        </p>
        <p className="text-xs text-base-content/55">
          {isSubscribed ? 'Đã bật — nhận thông báo ngay trên thiết bị' : 'Bật thông báo đẩy trên thiết bị'}
        </p>
      </div>
      <input
        type="checkbox"
        className="ct-toggle ct-toggle-success shrink-0"
        checked={isSubscribed}
        onChange={() => { void handleToggle(); }}
        disabled={isLoading}
        aria-label={isSubscribed ? 'Tắt thông báo đẩy' : 'Bật thông báo đẩy'}
      />
    </div>
  );
};
