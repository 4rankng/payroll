import React, { forwardRef } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Loader2, AlertCircle, ChevronDown } from 'lucide-react';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useIsMobile } from '@/hooks/use-mobile';

interface InfiniteScrollContainerProps {
  children: React.ReactNode;
  onLoadMore: () => void | Promise<void>;
  hasMore: boolean;
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  className?: string;
  loadingComponent?: React.ReactNode;
  errorComponent?: React.ReactNode;
  endComponent?: React.ReactNode;
  threshold?: number;
  rootMargin?: string;
  showLoadMoreButton?: boolean;
  loadMoreText?: string;
  containerRef?: React.RefObject<HTMLDivElement>;
}

const DefaultLoadingComponent = () => (
  <div className="flex items-center justify-center py-8">
    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
    <span className="ml-2 typography-body-medium text-muted-foreground">Đang tải thêm...</span>
  </div>
);

const DefaultErrorComponent = ({ onRetry }: { onRetry?: () => void }) => (
  <div className="flex flex-col items-center justify-center py-8 px-4">
    <AlertCircle className="h-8 w-8 text-destructive mb-2" />
    <p className="typography-body-medium text-muted-foreground mb-4">Không thể tải thêm dữ liệu</p>
    {onRetry && (
      <Button onClick={onRetry} variant="outline" size="sm">
        Thử lại
      </Button>
    )}
  </div>
);

const DefaultEndComponent = () => (
  <div className="flex items-center justify-center py-6">
    <p className="typography-body-medium text-muted-foreground">Đã tải hết dữ liệu</p>
  </div>
);

export const InfiniteScrollContainer = forwardRef<
  HTMLDivElement,
  InfiniteScrollContainerProps
>(
  (
    {
      children,
      onLoadMore,
      hasMore,
      isLoading,
      isError = false,
      onRetry,
      className,
      loadingComponent,
      errorComponent,
      endComponent,
      threshold = 0.8,
      rootMargin = '100px',
      showLoadMoreButton = true,
      loadMoreText = 'Tải thêm',
      containerRef,
    },
    ref
  ) => {
    const isMobile = useIsMobile();
    const { observerRef } = useInfiniteScroll({
      threshold,
      rootMargin,
      hasMore: hasMore && !isError,
      isLoading,
      onLoadMore,
      enabled: true,
    });

    const handleLoadMore = () => {
      if (!isLoading && hasMore && !isError) {
        onLoadMore();
      }
    };

    return (
      <div
        ref={ref || containerRef}
        className={cn(
          'relative w-full',
          'overflow-auto',
          isMobile && 'touch-pan-y',
          className
        )}
      >
        {children}
        
        {/* Intersection Observer Trigger */}
        {hasMore && !isError && !isLoading && (
          <div
            ref={observerRef}
            className="h-px w-full"
            aria-hidden="true"
          />
        )}

        {/* Loading State */}
        {isLoading && (loadingComponent || <DefaultLoadingComponent />)}

        {/* Error State */}
        {isError && !isLoading && (errorComponent || <DefaultErrorComponent onRetry={onRetry} />)}

        {/* End State */}
        {!hasMore && !isLoading && !isError && (endComponent || <DefaultEndComponent />)}

        {/* Manual Load More Button (Fallback) */}
        {showLoadMoreButton && hasMore && !isLoading && !isError && (
          <div className="flex items-center justify-center py-4">
            <Button
              onClick={handleLoadMore}
              variant="outline"
              size="sm"
              className="gap-2"
            >
              <ChevronDown className="h-4 w-4" />
              {loadMoreText}
            </Button>
          </div>
        )}
      </div>
    );
  }
);

InfiniteScrollContainer.displayName = 'InfiniteScrollContainer';

// Specialized version for tables
interface InfiniteScrollTableContainerProps extends Omit<InfiniteScrollContainerProps, 'children'> {
  children: React.ReactNode;
  tableFooter?: React.ReactNode;
}

export const InfiniteScrollTableContainer: React.FC<InfiniteScrollTableContainerProps> = ({
  children,
  tableFooter,
  ...props
}) => {
  return (
    <InfiniteScrollContainer {...props}>
      {children}
      {tableFooter}
    </InfiniteScrollContainer>
  );
};

// Hook for managing infinite scroll state
interface UseInfiniteScrollStateOptions<T> {
  initialData?: T[];
  pageSize?: number;
}

export function useInfiniteScrollState<T>({
  initialData = [],
  pageSize = 20,
}: UseInfiniteScrollStateOptions<T> = {}) {
  const [data, setData] = React.useState<T[]>(initialData);
  const [page, setPage] = React.useState(1);
  const [hasMore, setHasMore] = React.useState(true);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isError, setIsError] = React.useState(false);

  const loadMore = React.useCallback(async (fetchFn: (page: number, pageSize: number) => Promise<T[]>) => {
    if (isLoading || !hasMore) return;
    
    setIsLoading(true);
    setIsError(false);
    
    try {
      const newData = await fetchFn(page, pageSize);
      
      if (newData.length < pageSize) {
        setHasMore(false);
      }
      
      setData(prev => [...prev, ...newData]);
      setPage(prev => prev + 1);
    } catch (error) {
      setIsError(true);
      console.error('Failed to load more data:', error);
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize, isLoading, hasMore]);

  const reset = React.useCallback(() => {
    setData(initialData);
    setPage(1);
    setHasMore(true);
    setIsLoading(false);
    setIsError(false);
  }, [initialData]);

  return {
    data,
    page,
    hasMore,
    isLoading,
    isError,
    loadMore,
    reset,
    setData,
    setHasMore,
  };
}