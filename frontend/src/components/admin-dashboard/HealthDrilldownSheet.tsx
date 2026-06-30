import { useState } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { X, Inbox } from 'lucide-react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';

import { useFailedAttempts, useQuotaAnomalies } from '@/hooks/api/useDashboard';
import { formatCompactCurrency } from '@/utils/formatters';
import type { HealthDrilldownTarget } from './CheckInHealthStrip';

const PAGE_SIZE = 20;

interface HealthDrilldownSheetProps {
  target: HealthDrilldownTarget;
  month?: string;
  onClose: () => void;
}

const REASON_CATEGORY_LABELS: Record<string, string> = {
  geofence_not_configured: 'Chưa cấu hình vị trí',
  geofence_outside: 'Ngoài khu vực chấm công',
  check_in_not_enabled: 'Chưa cấp quyền chấm công',
  already_checked_in: 'Đã vào làm rồi',
  shift_not_configured: 'Chưa cấu hình ca',
  check_in_window: 'Ngoài giờ vào làm',
  check_out_window: 'Ngoài giờ tan ca',
  already_checked_out: 'Đã tan ca rồi',
  already_auto_rejected: 'Tự huỷ (hết giờ)',
  orphaned: 'Quá hạn tan ca',
  not_flexible_project: 'Không hỗ trợ tự chấm công',
  no_flexible_project: 'Không thuộc dự án tự chấm công',
  multiple_flexible: 'Thuộc nhiều dự án',
  no_attendance: 'Không có ca hôm nay',
  check_in_disabled: 'Chấm công bị khoá',
  other: 'Lỗi khác',
  // Attendance-based drilldown categories (not from attempt_classifier)
  open: 'Đang chờ checkout',
  zero_earning: 'Ca 0 đ',
  stuck_pending: 'Yêu cầu kẹt pending',
  request_failed: 'Yêu cầu lỗi',
};

const ATTEMPT_TYPE_LABELS: Record<string, string> = {
  check_in: 'Check-in',
  check_out: 'Check-out',
  request: 'Yêu cầu',
};

const ANOMALY_TYPE_LABELS: Record<string, string> = {
  drift: 'Lệch quota (salary ≠ max_adv)',
  missing: 'Thiếu row quota',
  stale: 'Quota cũ sau khi disable',
};

function labelFor(map: Record<string, string>, key: string): string {
  return map[key] ?? key;
}

export function HealthDrilldownSheet({ target, month, onClose }: HealthDrilldownSheetProps) {
  const isMobile = useIsMobile();
  const isOpen = target !== null;
  const title = deriveTitle(target, month);

  return (
    <Sheet open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <SheetContent
        side={isMobile ? 'bottom' : 'right'}
        className={cn(
          'w-full sm:w-[640px] p-0 flex flex-col',
          isMobile && 'rounded-t-2xl max-h-[94dvh] shadow-[0_-4px_24px_rgba(0,0,0,0.08)]',
        )}
      >
        {/* Mobile drag handle */}
        {isMobile && (
          <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
        )}

        <SheetHeader className="px-4 py-3 border-b flex-shrink-0">
          <div className="flex items-center justify-between">
            <SheetTitle className="text-sm font-semibold">{title}</SheetTitle>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full text-muted-foreground">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-4">
          {target?.type === 'failed-attempts' && (
            <FailedAttemptsTable category={target.category} attemptType={target.attemptType} />
          )}
          {target?.type === 'quota-anomaly' && (
            <QuotaAnomalyTable anomalyType={target.anomalyType} month={month} />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function deriveTitle(target: HealthDrilldownTarget, month?: string): string {
  if (!target) return '';
  const monthSuffix = month ? ` — ${month}` : '';
  if (target.type === 'failed-attempts') {
    const typeLabel = target.attemptType
      ? labelFor(ATTEMPT_TYPE_LABELS, target.attemptType)
      : '';
    const catLabel = target.category ? labelFor(REASON_CATEGORY_LABELS, target.category) : '';
    const filter = [catLabel, typeLabel].filter(Boolean).join(' — ');
    return `Lần chấm thất bại${filter ? ' — ' + filter : ''}${monthSuffix}`;
  }
  if (target.type === 'quota-anomaly') {
    const anomLabel = labelFor(ANOMALY_TYPE_LABELS, target.anomalyType);
    return `Bất thường quota — ${anomLabel}${monthSuffix}`;
  }
  return '';
}

// ─── Failed attempts table ────────────────────────────────────────────────

function FailedAttemptsTable({ category, attemptType }: { category?: string; attemptType?: 'check_in' | 'check_out' }) {
  const [page, setPage] = useState(1);

  const { data, isLoading, isFetching } = useFailedAttempts({
    type: attemptType,
    category,
    page,
    page_size: PAGE_SIZE,
  });

  const rows = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  if (isLoading) {
    return <TableSkeleton cols={4} rows={5} />;
  }

  if (!rows.length) {
    return <EmptyState label="Không có lần chấm thất bại" />;
  }

  return (
    <div className="space-y-3">
      <div className="text-xs text-muted-foreground">
                        Tổng {total.toLocaleString('vi-VN')} bản ghi
                        {isFetching ? ' — đang tải…' : ''}
      </div>

      <div className="overflow-x-auto rounded-lg border border-border/40">
        <table className="w-full text-sm">
          <thead className="bg-muted/40">
            <tr className="text-left text-xs uppercase tracking-wider text-muted-foreground">
              <Th>Nhân viên</Th>
              <Th>Loại</Th>
              <Th>Lý do</Th>
              <Th>Thông báo lỗi</Th>
              <Th>Thời gian</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border/30">
            {rows.map((row) => (
              <tr key={row.id} className="hover:bg-muted/20">
                <Td className="font-medium">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td>{labelFor(ATTEMPT_TYPE_LABELS, row.attempt_type)}</Td>
                <Td>{labelFor(REASON_CATEGORY_LABELS, row.reason_category)}</Td>
                <Td className="max-w-[220px] truncate text-muted-foreground" title={row.error_message ?? ''}>
                  {row.error_message ?? '—'}
                </Td>
                <Td className="whitespace-nowrap text-muted-foreground">
                  {format(parseISO(row.created_at), 'dd/MM/yyyy HH:mm', { locale: vi })}
                </Td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} totalPages={totalPages} onChange={setPage} />
    </div>
  );
}

// ─── Quota anomaly table ──────────────────────────────────────────────────

function QuotaAnomalyTable({ anomalyType, month }: { anomalyType: string; month?: string }) {
  const { data: rows, isLoading, isFetching } = useQuotaAnomalies(anomalyType, month);

  if (isLoading) {
    return <TableSkeleton cols={5} rows={4} />;
  }

  if (!rows || rows.length === 0) {
    return <EmptyState label="Không có bất thường quota" />;
  }

  return (
    <div className="space-y-3">
      <div className="text-xs text-muted-foreground">
        {rows.length.toLocaleString('vi-VN')} bản ghi{isFetching ? ' — đang tải…' : ''}
      </div>

      <div className="overflow-x-auto rounded-lg border border-border/40">
        <table className="w-full text-sm">
          <thead className="bg-muted/40">
            <tr className="text-left text-xs uppercase tracking-wider text-muted-foreground">
              <Th>Nhân viên</Th>
              <Th>Dự án</Th>
              <Th>Lương</Th>
              <Th>Max adv</Th>
              <Th>Mong đợi</Th>
              <Th>Lý do</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border/30">
            {rows.map((row, i) => (
              <tr key={`${row.employee_id}-${row.project_id}-${row.for_month}-${i}`} className="hover:bg-muted/20">
                <Td className="font-medium">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td className="text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatCompactCurrency(row.salary)}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatCompactCurrency(row.max_adv_amount)}</Td>
                <Td className="whitespace-nowrap tabular-nums text-muted-foreground">{formatCompactCurrency(row.expected_max)}</Td>
                <Td className="max-w-[200px] truncate text-muted-foreground" title={row.reason}>
                  {row.reason}
                </Td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── Shared subcomponents ─────────────────────────────────────────────────

function TableSkeleton({ cols, rows }: { cols: number; rows: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex gap-3">
          {Array.from({ length: cols }).map((__, c) => (
            <Skeleton key={c} className="h-4 flex-1 rounded" />
          ))}
        </div>
      ))}
    </div>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-40 text-muted-foreground text-sm gap-2">
      <Inbox className="h-8 w-8 opacity-30" />
      <span>{label}</span>
    </div>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-3 py-2 font-semibold">{children}</th>;
}

function Td({ children, className }: { children: React.ReactNode; className?: string }) {
  return <td className={cn('px-3 py-2 align-top', className)}>{children}</td>;
}

function Pagination({ page, totalPages, onChange }: { page: number; totalPages: number; onChange: (p: number) => void }) {
  if (totalPages <= 1) return null;
  return (
    <div className="flex items-center justify-between gap-2">
      <Button
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onChange(Math.max(1, page - 1))}
      >
        Trước
      </Button>
      <span className="text-xs text-muted-foreground tabular-nums">
        {page} / {totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={page >= totalPages}
        onClick={() => onChange(Math.min(totalPages, page + 1))}
      >
        Sau
      </Button>
    </div>
  );
}
