import { useEffect, useRef, useCallback, useState } from 'react';

interface UseInfiniteScrollOptions {
  threshold?: number;
  rootMargin?: string;
  root?: Element | null;
  hasMore: boolean;
  isLoading: boolean;
  onLoadMore: () => void | Promise<void>;
  enabled?: boolean;
}

interface UseInfiniteScrollReturn {
  observerRef: (node: HTMLElement | null) => void;
  isIntersecting: boolean;
  reset: () => void;
}

export function useInfiniteScroll({
  threshold = 0.8,
  rootMargin = '100px',
  root = null,
  hasMore,
  isLoading,
  onLoadMore,
  enabled = true,
}: UseInfiniteScrollOptions): UseInfiniteScrollReturn {
  const observer = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef(onLoadMore);
  const [isIntersecting, setIsIntersecting] = useState(false);

  // Update the ref when onLoadMore changes
  useEffect(() => {
    loadMoreRef.current = onLoadMore;
  }, [onLoadMore]);

  const reset = useCallback(() => {
    setIsIntersecting(false);
  }, []);

  const observerRef = useCallback(
    (node: HTMLElement | null) => {
      // Disconnect previous observer
      if (observer.current) {
        observer.current.disconnect();
      }

      // Exit if disabled or no more items to load
      if (!enabled || isLoading || !hasMore) {
        return;
      }

      // Create new observer
      if (node) {
        observer.current = new IntersectionObserver(
          (entries) => {
            const [entry] = entries;
            setIsIntersecting(entry.isIntersecting);
            
            if (entry.isIntersecting && hasMore && !isLoading) {
              loadMoreRef.current();
            }
          },
          {
            root,
            rootMargin,
            threshold,
          }
        );

        observer.current.observe(node);
      }
    },
    [enabled, hasMore, isLoading, root, rootMargin, threshold]
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (observer.current) {
        observer.current.disconnect();
      }
    };
  }, []);

  return {
    observerRef,
    isIntersecting,
    reset,
  };
}

// Hook for detecting mobile scroll end (useful for mobile momentum scrolling)
export function useMobileScrollEnd(
  onScrollEnd: () => void,
  delay: number = 150
) {
  const timeoutRef = useRef<NodeJS.Timeout>();

  useEffect(() => {
    const handleScroll = () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      timeoutRef.current = setTimeout(() => {
        const scrollHeight = document.documentElement.scrollHeight;
        const scrollTop = document.documentElement.scrollTop || document.body.scrollTop;
        const clientHeight = document.documentElement.clientHeight;

        // Check if user scrolled near bottom (within 100px)
        if (scrollHeight - scrollTop - clientHeight < 100) {
          onScrollEnd();
        }
      }, delay);
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    window.addEventListener('touchmove', handleScroll, { passive: true });

    return () => {
      window.removeEventListener('scroll', handleScroll);
      window.removeEventListener('touchmove', handleScroll);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [onScrollEnd, delay]);
}