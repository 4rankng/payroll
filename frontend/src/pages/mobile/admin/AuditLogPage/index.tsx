import React, { useState, useMemo } from 'react';
import { ClipboardList } from 'lucide-react';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import { AuditLogCard, AuditLogCardSkeleton } from '../../../admin/AuditLogPage/AuditLogCard';
import { AuditLogDetailSheet } from '../../../admin/AuditLogPage/AuditLogDetailSheet';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import type { BackendAuditLog } from '@/types/api/audit.types';

export default function AuditLogPageMobile() {
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);

  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } =
    useInfiniteAuditLogs({});

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

  return (
    <div className="pb-20">
      <MobilePageHeader
        title="Nhật ký hoạt động"
        icon={ClipboardList}
        subtitle={
          !isLoading && totalRecords > 0
            ? `${totalRecords.toLocaleString('vi-VN')} bản ghi`
            : undefined
        }
      />

      <div className="p-4 flex flex-col gap-3">
        {isLoading ? (
          <>
            {Array.from({ length: 6 }).map((_, i) => <AuditLogCardSkeleton key={i} />)}
          </>
        ) : logs.length === 0 ? (
          <EmptyState
            icon={ClipboardList}
            title="Không có bản ghi nào"
            description="Chưa có nhật ký hoạt động"
          />
        ) : (
          <>
            {logs.map((log) => (
              <AuditLogCard key={log.id} log={log} onClick={setSelectedLogId} />
            ))}
            {isFetchingNextPage && (
              <>
                {Array.from({ length: 3 }).map((_, i) => <AuditLogCardSkeleton key={i} />)}
              </>
            )}
            <div ref={observerRef} className="h-1" />
          </>
        )}
      </div>

      <AuditLogDetailSheet logId={selectedLogId} onClose={() => setSelectedLogId(null)} />
    </div>
  );
}
