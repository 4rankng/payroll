import { useEffect, useCallback } from 'react';
import { useNotificationToasts } from '@/hooks/useNotificationToasts';
import { useNotificationPageLoad } from '@/hooks/useNotificationPageLoad';
import { usePushNotifications } from '@/hooks/usePushNotifications';
import { useUnreadNotifications } from '@/hooks/api/useNotifications';
import { authManager } from '@/lib/auth';

interface NotificationProviderProps {
  children: React.ReactNode;
}

/**
 * Provider component that manages the full notification lifecycle:
 * - In-app toast notifications for new items
 * - Web push auto-subscription on login
 * - PWA badge count on app icon
 * - Foreground push handler (shows native notification even when app is open)
 */
export const NotificationProvider = ({ children }: NotificationProviderProps) => {
  const isAuthenticated = authManager.isTokenValid();
  const { isSubscribed, isSupported, requestPermission } = usePushNotifications();
  const { data: unreadData } = useUnreadNotifications(isAuthenticated);

  // Toast notifications for new items
  useNotificationToasts(isAuthenticated);

  // Force immediate fetch on load (consolidated from headers/nav)
  useNotificationPageLoad();

  // Auto-subscribe to push on login
  useEffect(() => {
    if (!isAuthenticated || !isSupported) return;
    if (isSubscribed) return;
    if (Notification.permission === 'denied') return;

    requestPermission();
  }, [isAuthenticated, isSupported, isSubscribed, requestPermission]);

  // Update PWA badge count (Android/iOS home screen icon)
  const updateBadge = useCallback(async (count: number) => {
    try {
      // @ts-expect-error - setAppBadge is not yet in all TS lib defs
      if ('setAppBadge' in navigator) {
        if (count > 0) {
          // @ts-expect-error
          await navigator.setAppBadge(count);
        } else {
          // @ts-expect-error
          await navigator.clearAppBadge();
        }
      }
    } catch {
      // Badge API not supported or failed — non-critical
    }
  }, []);

  // Keep badge count in sync with unread notifications
  useEffect(() => {
    if (!isAuthenticated) return;
    const count = unreadData?.count || 0;
    updateBadge(count);
  }, [unreadData?.count, isAuthenticated, updateBadge]);

  // Listen for messages from service worker (push received, notification clicked)
  useEffect(() => {
    if (!isSupported || !isAuthenticated) return;

    const handleMessage = async (event: MessageEvent) => {
      // Foreground push notification — show native notification
      if (event.data?.type === 'PUSH_NOTIFICATION') {
        const { title, body } = event.data.payload || {};
        if (title && Notification.permission === 'granted') {
          new Notification(title, {
            body: body || '',
            icon: '/favicon.png',
            badge: '/favicon.png',
            tag: 'tingting-foreground',
          });
        }
      }
      // Notification click from SW — app is already focused, navigate
      if (event.data?.type === 'NOTIFICATION_CLICK') {
        const { url } = event.data.payload || {};
        if (url && url !== '/') {
          window.location.href = url;
        }
      }
    };

    navigator.serviceWorker?.addEventListener('message', handleMessage);
    return () => {
      navigator.serviceWorker?.removeEventListener('message', handleMessage);
    };
  }, [isSupported, isAuthenticated]);

  return <>{children}</>;
};
