import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { notificationService } from '@/services/api/notification.service';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';
import type {
  Notification,
  PaginatedNotificationResponse,
  NotificationFilters,
  SendNotificationRequest,
} from '@/types/api/notification.types';

// Query keys
export const notificationKeys = {
  all: ['notifications'] as const,
  lists: () => [...notificationKeys.all, 'list'] as const,
  list: (filters: NotificationFilters) => [...notificationKeys.lists(), filters] as const,
  unread: () => [...notificationKeys.all, 'unread'] as const,
  unreadCount: () => [...notificationKeys.all, 'unread', 'count'] as const,
};

/**
 * Hook to fetch paginated notifications
 */
export const useNotifications = (filters: NotificationFilters = {}) => {
  return useQuery({
    queryKey: notificationKeys.list(filters),
    queryFn: () => notificationService.getNotifications(filters),
    refetchInterval: 3 * 60 * 1000, // 3 minutes
    refetchOnWindowFocus: true,
  });
};

/**
 * Hook to fetch notifications with infinite scroll
 */
export const useInfiniteNotifications = (filters: Omit<NotificationFilters, 'page'> = {}) => {
  return useInfiniteQuery({
    queryKey: [...notificationKeys.lists(), 'infinite', filters],
    queryFn: async ({ pageParam = 1 }) => {

      const result = await notificationService.getNotifications({
        ...filters,
        page: pageParam,
        pageSize: 20
      });

      return result;
    },
    refetchInterval: 3 * 60 * 1000, // 3 minutes
    refetchOnWindowFocus: true,
    getNextPageParam: (lastPage) => {


      // Handle if lastPage is an array (unexpected structure)
      if (Array.isArray(lastPage)) {

        return undefined;
      }

      if (!lastPage?.pagination) {

        return undefined;
      }

      const { page, totalPages } = lastPage.pagination;
      const nextPage = page < totalPages ? page + 1 : undefined;

      return nextPage;
    },
    initialPageParam: 1,
  });
};

/**
 * Hook to fetch unread notifications with count
 */
export const useUnreadNotifications = (enabled: boolean = true, forceRefresh: boolean = false) => {
  return useQuery({
    queryKey: notificationKeys.unread(),
    queryFn: () => notificationService.getUnreadNotifications(),
    enabled,
    staleTime: forceRefresh ? 0 : undefined,
    refetchInterval: 3 * 60 * 1000, // 3 minutes
    refetchOnWindowFocus: true,
    select: (data) => ({
      notifications: data.notifications,
      count: data.count
    })
  });
};

/**
 * Hook to fetch unread notification count
 * @deprecated Use useUnreadNotifications and access count property instead
 */
export const useUnreadCount = (enabled: boolean = true) => {
  const unreadQuery = useUnreadNotifications(enabled);
  return {
    ...unreadQuery,
    data: unreadQuery.data?.count || 0
  };
};

/**
 * Hook to mark a notification as read with optimistic updates and undo
 */
export const useMarkAsRead = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: number) => {
      // Perform optimistic update immediately
      const previousUnreadData = queryClient.getQueryData<{ notifications: Notification[], count: number }>(
        notificationKeys.unread()
      );

      // Store original data for potential rollback
      const rollbackData = { previousUnreadData };

      // Optimistically update unread notifications
      queryClient.setQueryData<{ notifications: Notification[], count: number }>(
        notificationKeys.unread(),
        (oldData) => {
          if (!oldData) return oldData;
          const filteredNotifications = oldData.notifications.filter(notification => notification.id !== notificationId);
          return {
            notifications: filteredNotifications,
            count: Math.max(0, oldData.count - 1)
          };
        }
      );

      // Notifications handled by API response only

      // Actually make the API call
      return notificationService.markAsRead(notificationId);
    },
    onSuccess: () => {
      // Server confirmed, invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
    onError: (error, notificationId) => {
      // Rollback optimistic updates on error
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });

      // Error notification will be shown by global handler using response.message
    },
  });
};

/**
 * Hook to mark a notification as read silently (without toast/undo)
 */
export const useMarkAsReadSilent = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: number) => {
      // Optimistically update unread notifications
      queryClient.setQueryData<{ notifications: Notification[], count: number }>(
        notificationKeys.unread(),
        (oldData) => {
          if (!oldData) return oldData;
          const filteredNotifications = oldData.notifications.filter(notification => notification.id !== notificationId);
          return {
            notifications: filteredNotifications,
            count: Math.max(0, oldData.count - 1)
          };
        }
      );

      // Update the main notifications list to mark as read
      queryClient.setQueryData<PaginatedNotificationResponse>(
        notificationKeys.list({}),
        (oldData) => {
          if (!oldData?.data) return oldData;
          return {
            ...oldData,
            data: oldData.data.map(notification =>
              notification.id === notificationId
                ? { ...notification, read_at: new Date().toISOString() }
                : notification
            )
          };
        }
      );

      // Actually make the API call
      return notificationService.markAsRead(notificationId);
    },
    onSuccess: () => {
      // Server confirmed, invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
    onError: () => {
      // Rollback optimistic updates on error
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
};

/**
 * Hook to mark all notifications as read
 */
export const useMarkAllAsRead = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => notificationService.markAllAsRead(),
    onSuccess: () => {
      // Update unread notifications to empty array with count 0
      queryClient.setQueryData<{ notifications: Notification[], count: number }>(
        notificationKeys.unread(),
        { notifications: [], count: 0 }
      );

      // Invalidate all notification queries
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });

      // Success notifications handled by API response only
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to refresh all notification data
 */
export const useRefreshNotifications = () => {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: notificationKeys.all });
  };
};

/**
 * Hook to send notification to a user (Admin only)
 */
export const useSendNotification = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SendNotificationRequest) => notificationService.sendNotification(data),
    onSuccess: (response) => {
      showSuccessNotification(response.message || 'Đã gửi thông báo thành công');

      // Invalidate notification queries to ensure recipient sees the new notification
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || 'Không thể gửi thông báo');
    },
  });
};
