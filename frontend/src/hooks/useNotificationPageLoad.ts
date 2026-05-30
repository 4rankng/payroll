import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { notificationKeys } from '@/hooks/api/useNotifications';

const LAST_FETCH_KEY = 'notifications-last-fetch';
const FETCH_INTERVAL_MS = 3 * 60 * 1000; // 3 minutes

/**
 * Hook to force immediate notification fetch on page load if we haven't fetched in the last 3 minutes
 */
export const useNotificationPageLoad = () => {
  const queryClient = useQueryClient();
  const hasChecked = useRef(false);

  useEffect(() => {
    // Only run this check once per app load
    if (hasChecked.current) return;
    hasChecked.current = true;

    const checkAndFetchNotifications = () => {
      try {
        const lastFetchTime = localStorage.getItem(LAST_FETCH_KEY);
        const now = Date.now();

        // If no last fetch time or last fetch was more than 3 minutes ago
        if (!lastFetchTime || (now - parseInt(lastFetchTime)) > FETCH_INTERVAL_MS) {
          // Force invalidate queries to trigger immediate fetch
          queryClient.invalidateQueries({
            queryKey: notificationKeys.unread(),
            refetchType: 'active'
          });

          queryClient.invalidateQueries({
            queryKey: notificationKeys.lists(),
            refetchType: 'active'
          });

          // Update last fetch time
          localStorage.setItem(LAST_FETCH_KEY, now.toString());
        }
      } catch (error) {
        // If localStorage fails, just force a fetch to be safe
        queryClient.invalidateQueries({
          queryKey: notificationKeys.unread(),
          refetchType: 'active'
        });
      }
    };

    // Run the check after a small delay to allow initial queries to set up
    const timeoutId = setTimeout(checkAndFetchNotifications, 100);

    return () => clearTimeout(timeoutId);
  }, [queryClient]);

  // Update last fetch time whenever queries succeed
  useEffect(() => {
    const updateLastFetchTime = () => {
      try {
        localStorage.setItem(LAST_FETCH_KEY, Date.now().toString());
      } catch {
        // Ignore localStorage errors
      }
    };

    // Listen for successful query completions
    const unsubscribe = queryClient.getQueryCache().subscribe((event) => {
      if (
        event.type === 'updated' &&
        event.query.state.status === 'success' &&
        (event.query.queryKey[0] === 'notifications')
      ) {
        updateLastFetchTime();
      }
    });

    return unsubscribe;
  }, [queryClient]);
};