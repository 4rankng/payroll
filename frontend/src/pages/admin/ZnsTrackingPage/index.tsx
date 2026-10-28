import React, { useState, useCallback } from 'react';
import { useAppState } from '@/contexts';
// UU PRO restyle: lucide → verified @untitledui/icons.
import { MessageSquare01 } from '@untitledui/icons';
import { useZnsMessages } from '@/hooks/api/useZnsMessages';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from '@/components/shared/AdminPageFrame';
import { PageHeader } from '@/components/shared/PageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorState } from '@/components/ui/error-state';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { PaginationControls } from '@/components/ui/pagination-controls';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ZnsTrackingFilters } from './ZnsTrackingFilters';
import { ZnsStatusBadge } from './ZnsStatusBadge';
import {
  ZNS_FEATURE_LABELS,
} from '@/types/api/zns.types';
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

export default function ZnsTrackingPage() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle('ZNS tracking'); }, [setPageTitle]);

  const [params, setParams] = useState<ZnsMessagesQueryParams>(EMPTY_PARAMS);
  const { data, isLoading, isError, refetch } = useZnsMessages(params);

  const rows = data?.data ?? [];
  const pagination = data?.pagination ?? null;

  const handleFiltersChange = useCallback((next: ZnsMessagesQueryParams) => {
    setParams({ ...next, page: 1 });
  }, []);

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          icon={MessageSquare01}
          title="ZNS tracking"
          description="Toàn bộ ZNS hệ thống đã gửi — mọi tính năng, mới nhất trước"
        >
          {pagination && pagination.totalRecords > 0 && (
            <span className="text-xs font-medium text-muted-foreground tabular-nums">
              {pagination.totalRecords.toLocaleString('vi-VN')} bản ghi
            </span>
          )}
        </PageHeader>
      </AdminPageHeaderCard>

      <AdminSectionCard aria-label="Bộ lọc ZNS">
        <AdminFilterRow>
          <ZnsTrackingFilters filters={params} onChange={handleFiltersChange} />
          <Input
            type="date"
            aria-label="Từ ngày"
            value={params.fromDate ?? ''}
            onChange={(e) => handleFiltersChange({ ...params, fromDate: e.target.value || undefined })}
            className="w-[150px]"
          />
          <Input
            type="date"
            aria-label="Đến ngày"
            value={params.toDate ?? ''}
            onChange={(e) => handleFiltersChange({ ...params, toDate: e.target.value || undefined })}
            className="w-[150px]"
          />
          <Input
            type="search"
            aria-label="Tìm SĐT hoặc mã ZNS"
            placeholder="Tìm SĐT / mã ZNS…"
            value={params.q ?? ''}
            onChange={(e) => handleFiltersChange({ ...params, q: e.target.value || undefined })}
            className="min-w-[180px] flex-1"
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="min-h-11 shrink-0"
            onClick={() => setParams(EMPTY_PARAMS)}
          >
            Xóa lọc
          </Button>
        </AdminFilterRow>
      </AdminSectionCard>

      {isError ? (
        <ErrorState message="Không thể tải danh sách ZNS" onRetry={() => refetch()} />
      ) : isLoading ? (
        <AdminSectionCard aria-label="Đang tải">
          <p className="py-8 text-center text-sm text-fg-secondary">Đang tải dữ liệu…</p>
        </AdminSectionCard>
      ) : rows.length === 0 ? (
        <AdminSectionCard aria-label="Trống">
          <EmptyState title="Không có ZNS nào khớp bộ lọc" />
        </AdminSectionCard>
      ) : (
        <AdminSectionCard aria-label="Danh sách ZNS">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">Thời gian gửi</TableHead>
                <TableHead>Tính năng</TableHead>
                <TableHead>Template</TableHead>
                <TableHead>SĐT</TableHead>
                <TableHead>Trạng thái</TableHead>
                <TableHead>Mã ZNS</TableHead>
                <TableHead>Lỗi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row: ZnsMessage) => (
                <TableRow key={row.id}>
                  <TableCell className="whitespace-nowrap align-top tabular-nums">
                    {formatDateTime(row.created_at)}
                  </TableCell>
                  <TableCell className="align-top">
                    {ZNS_FEATURE_LABELS[row.feature] ?? row.feature}
                  </TableCell>
                  <TableCell className="max-w-[140px] align-top break-all">
                    {row.template_id || '—'}
                  </TableCell>
                  <TableCell className="whitespace-nowrap align-top tabular-nums">
                    {row.phone || '—'}
                  </TableCell>
                  <TableCell className="align-top">
                    <ZnsStatusBadge status={row.status} />
                  </TableCell>
                  <TableCell className="max-w-[160px] align-top break-all">
                    {row.provider_msg_id || '—'}
                  </TableCell>
                  <TableCell className="min-w-[200px] max-w-[360px] align-top">
                    {row.error_msg ? (
                      <span className="block break-words text-error-700">
                        {row.provider_code !== 0 && (
                          <span className="mr-1 tabular-nums">[{row.provider_code}]</span>
                        )}
                        {row.error_msg}
                      </span>
                    ) : (
                      <span className="text-fg-tertiary">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <PaginationControls
            pagination={pagination}
            onPageChange={(page) => setParams((p) => ({ ...p, page }))}
            onPageSizeChange={(pageSize) => setParams((p) => ({ ...p, pageSize, page: 1 }))}
          />
        </AdminSectionCard>
      )}
    </AdminPageCanvas>
  );
}
