import { useEffect, useRef, useMemo } from 'react';
import { toast } from 'sonner';
import { useNavigate } from 'react-router-dom';
import { useUnreadNotifications } from './api/useNotifications';
import { Notification } from '@/types/api/notification.types';
import { notificationService } from '@/services/api/notification.service';
import { getNotificationNavigation, buildNavigationUrl } from '@/utils/notification-navigation';
import { getNotificationPreview } from '@/utils/notification-helpers';
import { vibrateOnNotification } from '@/utils/vibration';

/**
 * Hook to show toast notifications for new incoming notifications
 * Compares current notifications with previous ones to detect new ones
 */
export const useNotificationToasts = (enabled: boolean = true) => {
  const navigate = useNavigate();
  const { data } = useUnreadNotifications(enabled);
  const notifications = useMemo(() => data?.notifications || [], [data?.notifications]);
  const previousNotificationsRef = useRef<Notification[]>([]);

  useEffect(() => {
    if (!enabled || notifications.length === 0) {
      previousNotificationsRef.current = notifications;
      return;
    }

    const previousNotifications = previousNotificationsRef.current;
    
    // If this is the first load, don't show toasts
    if (previousNotifications.length === 0) {
      previousNotificationsRef.current = notifications;
      return;
    }

    // Find new notifications by comparing IDs
    const previousIds = new Set(previousNotifications.map(n => n.id));
    const newNotifications = notifications.filter(n => !previousIds.has(n.id));

    // Haptic feedback for new notifications
    if (newNotifications.length > 0) {
      vibrateOnNotification();
    }

    // Show toast for each new notification
    newNotifications.forEach((notification, index) => {
      const toastId = `notification-${notification.id}-${Date.now()}-${index}`;
      
      toast(notification.title, {
        id: toastId,
        description: getNotificationPreview(notification.message),
        duration: 6000, // 6 seconds
        action: {
          label: "Xem",
          onClick: async () => {
            try {
              // Mark notification as read before navigation
              await notificationService.markAsRead(notification.id);
              
              // Navigate to specific page based on notification type
              const navigationTarget = getNotificationNavigation(notification);
              if (navigationTarget) {
                const url = buildNavigationUrl(navigationTarget);
                navigate(url);
              } else {
                console.warn('No navigation target found for notification:', notification);
              }
            } catch (error) {
              console.error('Failed to handle notification action:', error);
            }
          }
        },
        onDismiss: async () => {
          try {
            // Mark notification as read when toast is dismissed
            await notificationService.markAsRead(notification.id);
          } catch (error) {
            console.error('Failed to mark notification as read:', error);
          }
        }
      });
    });

    // Update the reference
    previousNotificationsRef.current = notifications;
  }, [notifications, enabled, navigate]);

  return {
    notificationCount: notifications.length,
  };
};
