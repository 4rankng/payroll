import React, { useState, useMemo } from 'react';
import { useAppState } from '@/contexts';
import { SearchX } from 'lucide-react';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import { AuditLogCard, AuditLogCardSkeleton } from '../../../admin/AuditLogPage/AuditLogCard';
import { AuditLogDetailSheet } from '../../../admin/AuditLogPage/AuditLogDetailSheet';
import type { BackendAuditLog } from '@/types/api/audit.types';

export default function AuditLogPageMobile() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle('Nhật ký hoạt động'); }, [setPageTitle]);

  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);

  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } =
    useInfiniteAuditLogs({});

  const logs = useMemo<BackendAuditLog[]>(
    () => data?.pages.flatMap((p) => p.data) ?? [],
    [data]
  );
  const totalRecords = data?.pages[0]?.pagination.totalRecords ?? 0;

  const { observerRef } = useInfiniteScroll({
    hasMore: hasNextPage ?? false,
    isLoading: isFetchingNextPage,
    onLoadMore: fetchNextPage,
    rootMargin: '300px',
    threshold: 0.1,
  });

  return (
    <div className="p-4 pb-20 space-y-4">
      <div>
        <h1 className="text-xl font-bold text-foreground">Nhật ký</h1>
        {!isLoading && totalRecords > 0 && (
          <p className="text-xs text-muted-foreground mt-0.5">
            {totalRecords.toLocaleString('vi-VN')} bản ghi
          </p>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => <AuditLogCardSkeleton key={i} />)}
        </div>
      ) : logs.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <SearchX className="w-10 h-10 text-muted-foreground/30 mb-3" />
          <p className="font-semibold">Không có bản ghi nào</p>
          <p className="text-sm text-muted-foreground mt-1">Chưa có nhật ký hoạt động</p>
        </div>
      ) : (
        <>
          <div className="space-y-3">
            {logs.map((log) => (
              <AuditLogCard key={log.id} log={log} onClick={setSelectedLogId} />
            ))}
          </div>
          {isFetchingNextPage && (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => <AuditLogCardSkeleton key={i} />)}
            </div>
          )}
          <div ref={observerRef} className="h-1" />
        </>
      )}

      <AuditLogDetailSheet logId={selectedLogId} onClose={() => setSelectedLogId(null)} />
    </div>
  );
}
