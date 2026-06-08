import React, { useState, useCallback, useMemo } from 'react';
import { useAppState } from '@/contexts';
import { ClipboardList, Loader2, SearchX } from 'lucide-react';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import { PageHeader } from '@/components/shared/PageHeader';
import { AuditLogFilters } from './AuditLogFilters';
import { AuditLogCard, AuditLogCardSkeleton } from './AuditLogCard';
import { AuditLogDetailSheet } from './AuditLogDetailSheet';
import type { AuditLogsQueryParams } from '@/types/api/audit.types';
import type { BackendAuditLog } from '@/types/api/audit.types';

type Filters = Omit<AuditLogsQueryParams, 'page' | 'pageSize'>;

export default function AuditLogPage() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle('Nhật ký hoạt động'); }, [setPageTitle]);

  const [filters, setFilters] = useState<Filters>({});
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);

  const {
    data,
    isLoading,
    isFetchingNextPage,
    fetchNextPage,
    hasNextPage,
  } = useInfiniteAuditLogs(filters);

  const logs = useMemo<BackendAuditLog[]>(
    () => data?.pages.flatMap((p) => p.data ?? []) ?? [],
    [data]
  );

  const totalRecords = data?.pages[0]?.pagination?.totalRecords ?? 0;

  const { observerRef } = useInfiniteScroll({
    hasMore: hasNextPage ?? false,
    isLoading: isFetchingNextPage,
    onLoadMore: () => { fetchNextPage(); },
    rootMargin: '300px',
    threshold: 0.1,
  });

  const handleFiltersChange = useCallback((next: Filters) => {
    setFilters(next);
  }, []);

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">

        <PageHeader
          icon={ClipboardList}
          title="Nhật ký hoạt động"
          description="Lịch sử thao tác của người dùng trong hệ thống"
        >
          {totalRecords > 0 && (
            <span className="text-xs font-medium text-muted-foreground tabular-nums">
              {totalRecords.toLocaleString('vi-VN')} bản ghi
            </span>
          )}
        </PageHeader>

        {/* Filters */}
        <AuditLogFilters filters={filters} onChange={handleFiltersChange} />

        {/* Initial loading */}
        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <AuditLogCardSkeleton key={i} />
            ))}
          </div>
        )}

        {/* Cards */}
        {!isLoading && logs.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <SearchX className="h-10 w-10 text-muted-foreground/40 mb-3" />
            <p className="text-sm font-medium text-foreground">Không có dữ liệu</p>
            <p className="text-xs text-muted-foreground mt-1">
              Thử thay đổi bộ lọc để xem kết quả khác
            </p>
          </div>
        )}

        {!isLoading && logs.length > 0 && (
          <div className="space-y-2.5">
            {logs.map((log) => (
              <AuditLogCard
                key={log.id}
                log={log}
                onClick={setSelectedLogId}
              />
            ))}
          </div>
        )}

        {/* Infinite scroll sentinel */}
        {hasNextPage && !isFetchingNextPage && (
          <div ref={observerRef} className="h-px w-full" aria-hidden="true" />
        )}

        {/* Loading more */}
        {isFetchingNextPage && (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            <span className="ml-2 text-sm text-muted-foreground">Đang tải thêm...</span>
          </div>
        )}

        {/* End of list */}
        {!hasNextPage && !isLoading && logs.length > 0 && (
          <p className="text-center py-4 text-xs text-muted-foreground">
            Đã hiển thị tất cả {logs.length.toLocaleString('vi-VN')} bản ghi
          </p>
        )}
      </div>

      <AuditLogDetailSheet
        logId={selectedLogId}
        onClose={() => setSelectedLogId(null)}
      />
    </div>
  );
}
