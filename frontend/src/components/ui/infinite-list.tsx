import React from 'react';
import { cn } from '@/lib/utils';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Skeleton } from '@/components/ui/skeleton';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface InfiniteListProps<T> {
  items: T[];
  renderItem: (item: T, index: number) => React.ReactNode;
  onLoadMore: () => void | Promise<void>;
  hasMore: boolean;
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  onRefresh?: () => void | Promise<void>;
  className?: string;
  itemClassName?: string;
  listClassName?: string;
  gap?: 'none' | 'xs' | 'sm' | 'md' | 'lg';
  emptyState?: React.ReactNode;
  getItemKey?: (item: T, index: number) => string | number;
  skeletonCount?: number;
  pullToRefreshEnabled?: boolean;
  virtualScrolling?: boolean;
  threshold?: number;
  showLoadMoreButton?: boolean;
}

const gapClasses = {
  none: '',
  xs: 'gap-1',
  sm: 'gap-2',
  md: 'gap-4',
  lg: 'gap-6',
};

const ItemSkeleton = ({ className }: { className?: string }) => (
  <div className={cn("p-4 border rounded-lg", className)}>
    <div className="flex items-start gap-3">
      <Skeleton className="h-10 w-10 rounded-full" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-4 w-[200px]" />
        <Skeleton className="h-3 w-[150px]" />
      </div>
    </div>
  </div>
);

const DefaultEmptyState = () => (
  <div className="flex flex-col items-center justify-center py-12 px-4">
    <p className="text-muted-foreground text-center">Không có dữ liệu</p>
  </div>
);

export function InfiniteList<T>({
  items,
  renderItem,
  onLoadMore,
  hasMore,
  isLoading,
  isError = false,
  onRetry,
  onRefresh,
  className,
  itemClassName,
  listClassName,
  gap = 'sm',
  emptyState,
  getItemKey,
  skeletonCount = 3,
  pullToRefreshEnabled = true,
  virtualScrolling = false,
  threshold = 0.8,
  showLoadMoreButton = true,
}: InfiniteListProps<T>) {
  const isMobile = useIsMobile();
  const [isRefreshing, setIsRefreshing] = React.useState(false);
  const [pullDistance, setPullDistance] = React.useState(0);
  const [isPulling, setIsPulling] = React.useState(false);
  const touchStartY = React.useRef<number>(0);
  const listRef = React.useRef<HTMLDivElement>(null);

  // Pull to refresh logic for mobile
  React.useEffect(() => {
    if (!pullToRefreshEnabled || !isMobile || !onRefresh) return;

    const handleTouchStart = (e: TouchEvent) => {
      if (window.scrollY === 0) {
        touchStartY.current = e.touches[0].clientY;
        setIsPulling(true);
      }
    };

    const handleTouchMove = (e: TouchEvent) => {
      if (!isPulling) return;
      
      const currentY = e.touches[0].clientY;
      const distance = Math.max(0, currentY - touchStartY.current);
      
      if (distance > 0 && window.scrollY === 0) {
        e.preventDefault();
        setPullDistance(Math.min(distance, 100));
      }
    };

    const handleTouchEnd = async () => {
      if (pullDistance > 60 && onRefresh) {
        setIsRefreshing(true);
        try {
          await onRefresh();
        } finally {
          setIsRefreshing(false);
        }
      }
      setPullDistance(0);
      setIsPulling(false);
    };

    const element = listRef.current;
    if (element) {
      element.addEventListener('touchstart', handleTouchStart, { passive: true });
      element.addEventListener('touchmove', handleTouchMove, { passive: false });
      element.addEventListener('touchend', handleTouchEnd);

      return () => {
        element.removeEventListener('touchstart', handleTouchStart);
        element.removeEventListener('touchmove', handleTouchMove);
        element.removeEventListener('touchend', handleTouchEnd);
      };
    }
  }, [isPulling, pullDistance, onRefresh, pullToRefreshEnabled, isMobile]);

  // Loading skeleton
  if (isLoading && items.length === 0) {
    return (
      <div className={cn("w-full", className)}>
        <div className={cn("space-y-2", listClassName)}>
          {Array.from({ length: skeletonCount }).map((_, i) => (
            <ItemSkeleton key={i} className={itemClassName} />
          ))}
        </div>
      </div>
    );
  }

  // Empty state
  if (!isLoading && items.length === 0) {
    return (
      <div className={cn("w-full", className)}>
        {emptyState || <DefaultEmptyState />}
      </div>
    );
  }

  return (
    <div ref={listRef} className={cn("relative w-full", className)}>
      {/* Pull to refresh indicator */}
      {pullToRefreshEnabled && isMobile && onRefresh && (
        <div
          className={cn(
            "absolute top-0 left-0 right-0 flex items-center justify-center transition-all",
            "bg-background/80 backdrop-blur-sm",
            pullDistance > 0 ? "opacity-100" : "opacity-0"
          )}
          style={{
            height: `${pullDistance}px`,
            transform: `translateY(-${pullDistance}px)`,
          }}
        >
          <div className={cn(
            "transition-transform",
            pullDistance > 60 ? "rotate-180" : ""
          )}>
            <RefreshCw className={cn(
              "h-5 w-5",
              isRefreshing && "animate-spin"
            )} />
          </div>
        </div>
      )}

      {/* Refresh button for desktop */}
      {!isMobile && onRefresh && (
        <div className="flex justify-end mb-2">
          <Button
            onClick={async () => {
              setIsRefreshing(true);
              try {
                await onRefresh();
              } finally {
                setIsRefreshing(false);
              }
            }}
            variant="ghost"
            size="sm"
            disabled={isRefreshing}
            className="gap-2"
          >
            <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />
            Làm mới
          </Button>
        </div>
      )}

      <InfiniteScrollContainer
        onLoadMore={onLoadMore}
        hasMore={hasMore}
        isLoading={isLoading}
        isError={isError}
        onRetry={onRetry}
        threshold={threshold}
        showLoadMoreButton={showLoadMoreButton}
      >
        <div className={cn(
          "w-full",
          gapClasses[gap],
          gap !== 'none' && "flex flex-col",
          listClassName
        )}>
          {items.map((item, index) => {
            const key = getItemKey ? getItemKey(item, index) : index;
            return (
              <div key={key} className={itemClassName}>
                {renderItem(item, index)}
              </div>
            );
          })}
        </div>
      </InfiniteScrollContainer>
    </div>
  );
}

// Specialized card list for mobile
interface InfiniteCardListProps<T> extends Omit<InfiniteListProps<T>, 'renderItem'> {
  renderCard: (item: T, index: number) => React.ReactNode;
  cardsPerRow?: 1 | 2;
}

export function InfiniteCardList<T>({
  items,
  renderCard,
  cardsPerRow = 1,
  className,
  listClassName,
  ...props
}: InfiniteCardListProps<T>) {
  const isMobile = useIsMobile();
  const actualCardsPerRow = isMobile ? 1 : cardsPerRow;

  return (
    <InfiniteList
      items={items}
      renderItem={renderCard}
      className={className}
      listClassName={cn(
        actualCardsPerRow === 2 && "grid grid-cols-2 gap-4",
        listClassName
      )}
      {...props}
    />
  );
}

// Helper hook for infinite list state management
export function useInfiniteListState<T>(
  fetchFn: (page: number, pageSize: number) => Promise<{ data: T[]; hasMore: boolean }>,
  options?: {
    pageSize?: number;
    initialData?: T[];
  }
) {
  const [items, setItems] = React.useState<T[]>(options?.initialData || []);
  const [page, setPage] = React.useState(1);
  const [hasMore, setHasMore] = React.useState(true);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isError, setIsError] = React.useState(false);

  const loadMore = React.useCallback(async () => {
    if (isLoading || !hasMore) return;

    setIsLoading(true);
    setIsError(false);

    try {
      const result = await fetchFn(page, options?.pageSize || 20);
      setItems(prev => [...prev, ...result.data]);
      setHasMore(result.hasMore);
      setPage(prev => prev + 1);
    } catch (error) {
      setIsError(true);
      console.error('Failed to load more items:', error);
    } finally {
      setIsLoading(false);
    }
  }, [page, isLoading, hasMore, fetchFn, options?.pageSize]);

  const refresh = React.useCallback(async () => {
    setItems([]);
    setPage(1);
    setHasMore(true);
    setIsError(false);
    
    setIsLoading(true);
    try {
      const result = await fetchFn(1, options?.pageSize || 20);
      setItems(result.data);
      setHasMore(result.hasMore);
      setPage(2);
    } catch (error) {
      setIsError(true);
      console.error('Failed to refresh items:', error);
    } finally {
      setIsLoading(false);
    }
  }, [fetchFn, options?.pageSize]);

  const retry = React.useCallback(() => {
    loadMore();
  }, [loadMore]);

  return {
    items,
    hasMore,
    isLoading,
    isError,
    loadMore,
    refresh,
    retry,
  };
}