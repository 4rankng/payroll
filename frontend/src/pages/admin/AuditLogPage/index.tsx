import { ErrorState } from "@/components/ui/error-state";
import React, { useState, useCallback, useMemo } from 'react';
import { useAppState } from '@/contexts';
import { ClipboardList, Loader2 } from 'lucide-react';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from '@/components/shared/AdminPageFrame';
import { PageHeader } from '@/components/shared/PageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
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
    isError,
    isFetchNextPageError,
    refetch,
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
    hasMore: !isError && (hasNextPage ?? false),
    isLoading: isFetchingNextPage,
    onLoadMore: () => { fetchNextPage(); },
    rootMargin: '300px',
    threshold: 0.1,
  });

  const handleFiltersChange = useCallback((next: Filters) => {
    setFilters(next);
  }, []);

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
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
      </AdminPageHeaderCard>

      {/* Filters */}
      <AdminSectionCard aria-label="Bộ lọc nhật ký">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-slate-800">Nhật ký</p>
          <div className="flex flex-1 items-center justify-end">
            <AuditLogFilters filters={filters} onChange={handleFiltersChange} />
          </div>
        </AdminFilterRow>
      </AdminSectionCard>

      {isError && (
        <div role="alert">
          <ErrorState
            message={isFetchNextPageError
              ? "Không thể tải thêm nhật ký. Các bản ghi đã tải vẫn được giữ lại."
              : "Không thể tải nhật ký hoạt động. Vui lòng thử lại."}
            onRetry={() => void (isFetchNextPageError ? fetchNextPage() : refetch())}
            className="px-4"
          />
        </div>
      )}

      {/* Initial loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <AuditLogCardSkeleton key={i} />
          ))}
        </div>
      )}

        {/* Cards */}
        {!isLoading && !isError && logs.length === 0 && (
          <EmptyState
            title="Không có dữ liệu"
            description="Thử thay đổi bộ lọc để xem kết quả khác"
            size="sm"
          />
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
        {!isError && hasNextPage && !isFetchingNextPage && (
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
        {!isError && !hasNextPage && !isLoading && logs.length > 0 && (
          <p className="text-center py-4 text-xs text-muted-foreground">
            Đã hiển thị tất cả {logs.length.toLocaleString('vi-VN')} bản ghi
          </p>
        )}

      <AuditLogDetailSheet
        logId={selectedLogId}
        onClose={() => setSelectedLogId(null)}
      />
    </AdminPageCanvas>
  );
}
