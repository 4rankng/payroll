import { useEffect, useRef, useCallback } from 'react';

interface UseInfiniteScrollOptions {
  root?: Element | null;
  rootMargin?: string;
  threshold?: number | number[];
}

/**
 * Observe a sentinel element and invoke `onLoadMore` when in view.
 * Keeps logic in a .ts hook per project conventions.
 */
export function useInfiniteScroll(
  onLoadMore: () => void,
  enabled: boolean,
  { root = null, rootMargin = '200px', threshold = 0 }: UseInfiniteScrollOptions = {}
) {
  const ref = useRef<HTMLDivElement | null>(null);

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      if (!enabled) return;
      const entry = entries[0];
      if (entry && entry.isIntersecting) {
        onLoadMore();
      }
    },
    [enabled, onLoadMore]
  );

  useEffect(() => {
    if (!ref.current) return;
    if (!enabled) return;
    const observer = new IntersectionObserver(handleIntersect, {
      root,
      rootMargin,
      threshold,
    });
    observer.observe(ref.current);
    return () => observer.disconnect();
  }, [handleIntersect, root, rootMargin, threshold, enabled]);

  return ref;
}

