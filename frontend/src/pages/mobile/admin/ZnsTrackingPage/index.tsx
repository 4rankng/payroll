import React, { useState, useCallback } from 'react';
import { useAppState } from '@/contexts';
import { MessageSquare01, FilterLines, XClose } from '@untitledui/icons';
import { useZnsMessages } from '@/hooks/api/useZnsMessages';
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorState } from '@/components/ui/error-state';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { PaginationControls } from '@/components/ui/pagination-controls';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { ZnsTrackingFilters } from '../../../admin/ZnsTrackingPage/ZnsTrackingFilters';
import { ZnsStatusBadge } from '../../../admin/ZnsTrackingPage/ZnsStatusBadge';
import { ZNS_FEATURE_LABELS } from '@/types/api/zns.types';
import type { ZnsMessage, ZnsMessagesQueryParams } from '@/types/api/zns.types';

const EMPTY_PARAMS: ZnsMessagesQueryParams = { page: 1, pageSize: 20 };

function formatDateTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString('vi-VN', {
    hour: '2-digit',
    minute: '2-digit',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });
}

/** One ZNS per card — no horizontal table on mobile, full wrap, no truncation. */
function ZnsCard({ row }: { row: ZnsMessage }) {
  return (
    <article className="rounded-xl border border-utility-gray-200 bg-white p-3.5">
      <div className="flex items-start justify-between gap-2">
        <p className="text-xs tabular-nums text-fg-secondary">{formatDateTime(row.created_at)}</p>
        <ZnsStatusBadge status={row.status} />
      </div>

      <dl className="mt-2.5 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-sm">
        <dt className="text-xs text-fg-tertiary">Tính năng</dt>
        <dd className="break-words text-fg-primary">
          {ZNS_FEATURE_LABELS[row.feature] ?? row.feature}
        </dd>

        <dt className="text-xs text-fg-tertiary">SĐT</dt>
        <dd className="break-all tabular-nums text-fg-primary">{row.phone || '—'}</dd>

        <dt className="text-xs text-fg-tertiary">Template</dt>
        <dd className="break-all text-fg-primary">{row.template_id || '—'}</dd>

        <dt className="text-xs text-fg-tertiary">Mã ZNS</dt>
        <dd className="break-all text-fg-primary">{row.provider_msg_id || '—'}</dd>
      </dl>

      {row.error_msg && (
        <p className="mt-2.5 break-words rounded-lg bg-error-50 px-2.5 py-2 text-xs text-error-700">
          {row.provider_code !== 0 && <span className="mr-1 tabular-nums">[{row.provider_code}]</span>}
          {row.error_msg}
        </p>
      )}
    </article>
  );
}

export default function ZnsTrackingPageMobile() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle('ZNS tracking'); }, [setPageTitle]);

  const [params, setParams] = useState<ZnsMessagesQueryParams>(EMPTY_PARAMS);
  const [isFilterOpen, setIsFilterOpen] = useState(false);
  const { data, isLoading, isError, refetch } = useZnsMessages(params);

  const rows = data?.data ?? [];
  const pagination = data?.pagination ?? null;

  const handleFiltersChange = useCallback((next: ZnsMessagesQueryParams) => {
    setParams({ ...next, page: 1 });
  }, []);

  return (
    <MobilePageShell>
      <MobilePageHeader
        title="ZNS tracking"
        icon={MessageSquare01}
        actions={
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="min-h-11 gap-1.5"
            onClick={() => setIsFilterOpen(true)}
          >
            <FilterLines className="h-4 w-4" />
            Bộ lọc
          </Button>
        }
      />

      <Sheet open={isFilterOpen} onOpenChange={setIsFilterOpen}>
        <SheetContent side="bottom" className="max-h-[85dvh] overflow-y-auto">
          <SheetHeader>
            <SheetTitle className="flex items-center justify-between">
              Bộ lọc ZNS
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="min-h-11 min-w-11"
                aria-label="Đóng bộ lọc"
                onClick={() => setIsFilterOpen(false)}
              >
                <XClose className="h-4 w-4" />
              </Button>
            </SheetTitle>
          </SheetHeader>

          <div className="flex flex-col gap-4 px-4 pb-6">
            <ZnsTrackingFilters filters={params} onChange={handleFiltersChange} />
            <div className="flex gap-2">
              <Input
                type="date"
                aria-label="Từ ngày"
                value={params.fromDate ?? ''}
                onChange={(e) => handleFiltersChange({ ...params, fromDate: e.target.value || undefined })}
              />
              <Input
                type="date"
                aria-label="Đến ngày"
                value={params.toDate ?? ''}
                onChange={(e) => handleFiltersChange({ ...params, toDate: e.target.value || undefined })}
              />
            </div>
            <Input
              type="search"
              aria-label="Tìm SĐT hoặc mã ZNS"
              placeholder="Tìm SĐT / mã ZNS…"
              value={params.q ?? ''}
              onChange={(e) => handleFiltersChange({ ...params, q: e.target.value || undefined })}
            />
            <Button
              type="button"
              variant="outline"
              className="min-h-11"
              onClick={() => {
                setParams(EMPTY_PARAMS);
                setIsFilterOpen(false);
              }}
            >
              Xóa lọc
            </Button>
          </div>
        </SheetContent>
      </Sheet>

      {isError ? (
        <ErrorState message="Không thể tải danh sách ZNS" onRetry={() => refetch()} />
      ) : isLoading ? (
        <p className="py-8 text-center text-sm text-fg-secondary">Đang tải dữ liệu…</p>
      ) : rows.length === 0 ? (
        <EmptyState title="Không có ZNS nào khớp bộ lọc" />
      ) : (
        <div className="flex flex-col gap-2.5 px-4 pb-6">
          {rows.map((row: ZnsMessage) => (
            <ZnsCard key={row.id} row={row} />
          ))}
          <PaginationControls
            pagination={pagination}
            onPageChange={(page) => setParams((p) => ({ ...p, page }))}
            onPageSizeChange={(pageSize) => setParams((p) => ({ ...p, pageSize, page: 1 }))}
          />
        </div>
      )}
    </MobilePageShell>
  );
}
