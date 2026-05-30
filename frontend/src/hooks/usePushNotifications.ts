import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';

interface PushSubscriptionKeys {
  p256dh: string;
  auth: string;
}

interface PushSubscriptionData {
  endpoint: string;
  keys: PushSubscriptionKeys;
}

// Convert base64 string to Uint8Array (needed for VAPID key)
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; i++) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export type PushPermissionStatus = 'default' | 'granted' | 'denied' | 'unsupported';

// Module-level guard prevents duplicate calls across hook instances and StrictMode remounts
let _subscribing = false;

interface UsePushNotificationsReturn {
  /** Current permission status */
  permissionStatus: PushPermissionStatus;
  /** Whether the user is subscribed to push */
  isSubscribed: boolean;
  /** Whether push is supported on this browser/device */
  isSupported: boolean;
  /** Request permission and subscribe */
  requestPermission: () => Promise<boolean>;
  /** Unsubscribe from push notifications */
  unsubscribe: () => Promise<void>;
  /** Loading state */
  isLoading: boolean;
}

export function usePushNotifications(): UsePushNotificationsReturn {
  const [permissionStatus, setPermissionStatus] = useState<PushPermissionStatus>('default');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSupported] = useState(() => {
    if (typeof window === 'undefined') return false;
    return 'Notification' in window && 'serviceWorker' in navigator && 'PushManager' in window;
  });

  // Check current permission and subscription status on mount
  useEffect(() => {
    if (!isSupported) {
      setPermissionStatus('unsupported');
      return;
    }

    // Check notification permission
    if (Notification.permission === 'granted') {
      setPermissionStatus('granted');
    } else if (Notification.permission === 'denied') {
      setPermissionStatus('denied');
    } else {
      setPermissionStatus('default');
    }

    // Check if already subscribed
    checkSubscription();
  }, [isSupported]);

  const checkSubscription = async () => {
    try {
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.getSubscription();
      setIsSubscribed(!!subscription);
    } catch {
      // SW not ready yet
    }
  };

  const getVAPIDKey = async (): Promise<string | null> => {
    try {
      const response = await apiClient.get<{ publicKey: string }>(API_ENDPOINTS.push.vapidKey);
      return response.publicKey;
    } catch {
      console.error('Failed to fetch VAPID public key');
      return null;
    }
  };

  const sendSubscriptionToServer = async (subscription: PushSubscriptionData) => {
    await apiClient.post(API_ENDPOINTS.push.subscribe, {
      endpoint: subscription.endpoint,
      keys: subscription.keys,
      deviceType: /iPhone|iPad|iPod/.test(navigator.userAgent) ? 'ios' :
                  /Android/.test(navigator.userAgent) ? 'android' : 'web',
    });
  };

  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported || _subscribing) return false;
    _subscribing = true;
    setIsLoading(true);
    try {
      const permission = await Notification.requestPermission();

      if (permission !== 'granted') {
        setPermissionStatus(permission === 'denied' ? 'denied' : 'default');
        return false;
      }

      setPermissionStatus('granted');

      const vapidKey = await getVAPIDKey();
      if (!vapidKey) {
        console.error('Push notifications not configured on server');
        return false;
      }

      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidKey),
      });

      const subData = subscription.toJSON() as unknown as PushSubscriptionData;
      await sendSubscriptionToServer(subData);

      setIsSubscribed(true);
      return true;
    } catch (error) {
      console.error('Failed to subscribe to push notifications:', error);
      return false;
    } finally {
      _subscribing = false;
      setIsLoading(false);
    }
  }, [isSupported]);

  const unsubscribe = useCallback(async () => {
    if (_subscribing) return;
    _subscribing = true;
    setIsLoading(true);
    try {
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.getSubscription();

      if (subscription) {
        const endpoint = subscription.endpoint;
        await subscription.unsubscribe();

        await apiClient.post(API_ENDPOINTS.push.unsubscribe, { endpoint });
      }

      setIsSubscribed(false);
    } catch (error) {
      console.error('Failed to unsubscribe:', error);
    } finally {
      _subscribing = false;
      setIsLoading(false);
    }
  }, []);

  return {
    permissionStatus,
    isSubscribed,
    isSupported,
    requestPermission,
    unsubscribe,
    isLoading,
  };
}
