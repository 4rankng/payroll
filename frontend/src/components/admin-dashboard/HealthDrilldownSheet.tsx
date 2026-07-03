import { useState, type ReactNode } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { X, Inbox } from 'lucide-react';
import { format, parseISO, subDays } from 'date-fns';
import { vi } from 'date-fns/locale';

import { useFailedAttempts, useQuotaAnomalies } from '@/hooks/api/useDashboard';
import { useAdminAttendances } from '@/hooks/api/useAdminAttendance';
import { formatCompactCurrency } from '@/utils/formatters';
import { formatDistanceMeters, formatGeofenceDistanceDelta } from '@/utils/geoDistance';
import type { AdminFailedAttempt, QuotaAnomalyRow } from '@/types/api/dashboard.types';
import type { AdminAttendanceResponse } from '@/types/api/attendance.types';
import type { HealthDrilldownTarget } from './CheckInHealthStrip';

const PAGE_SIZE = 20;

interface HealthDrilldownSheetProps {
  target: HealthDrilldownTarget;
  month?: string;
  periodStart?: string;
  periodEnd?: string;
  onClose: () => void;
}

const REASON_CATEGORY_LABELS: Record<string, string> = {
  geofence_not_configured: 'Chưa cấu hình vị trí',
  geofence_outside: 'Ngoài khu vực chấm công',
  gps_denied: 'Chưa cấp quyền GPS',
  gps_timeout: 'GPS phản hồi chậm',
  gps_unavailable: 'Không lấy được vị trí GPS',
  gps_unsupported: 'Thiết bị không hỗ trợ GPS',
  gps_inaccurate: 'GPS không chính xác',
  check_in_not_enabled: 'Chưa cấp quyền chấm công',
  already_checked_in: 'Đã vào làm rồi',
  shift_not_configured: 'Chưa cấu hình ca',
  check_in_window: 'Ngoài giờ vào làm',
  check_out_window: 'Ngoài giờ tan ca',
  already_checked_out: 'Đã tan ca rồi',
  already_auto_rejected: 'Tự động huỷ (hết giờ)',
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

export function HealthDrilldownSheet({ target, month, periodStart, periodEnd, onClose }: HealthDrilldownSheetProps) {
  const isMobile = useIsMobile();
  const isOpen = target !== null;
  const title = deriveTitle(target, month);

  return (
    <Sheet open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <SheetContent
        side={isMobile ? 'bottom' : 'right'}
        className={cn(
          'w-full p-0 flex flex-col sm:w-[min(920px,calc(100vw-2rem))]',
          isMobile && 'rounded-t-2xl max-h-[94dvh] shadow-[0_-4px_24px_rgba(0,0,0,0.08)]',
        )}
      >
        {/* Mobile drag handle */}
        {isMobile && (
          <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
        )}

        <SheetHeader className="border-b bg-background/95 px-4 py-3 flex-shrink-0 sm:px-6">
          <div className="flex items-start justify-between gap-3">
            <SheetTitle className="text-left text-base font-semibold leading-snug text-foreground sm:text-lg">
              {title}
            </SheetTitle>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9 shrink-0 rounded-full text-muted-foreground">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-4 sm:p-6">
          {target?.type === 'failed-attempts' && (
            <FailedAttemptsTable
              category={target.category}
              attemptType={target.attemptType}
              periodStart={periodStart}
              periodEnd={periodEnd}
            />
          )}
          {target?.type === 'quota-anomaly' && (
            <QuotaAnomalyTable anomalyType={target.anomalyType} month={month} />
          )}
          {target?.type === 'attendance-list' && (
            <AttendanceRowsTable
              status={target.status}
              successfulCheckout={target.successfulCheckout}
              zeroEarning={target.zeroEarning}
              periodStart={periodStart}
              periodEnd={periodEnd}
              emptyLabel={target.emptyLabel}
            />
          )}
          {target?.type === 'successful-checkouts' && (
            <AttendanceRowsTable
              successfulCheckout
              periodStart={periodStart}
              periodEnd={periodEnd}
              emptyLabel="Không có ai chấm công ra thành công trong tháng này"
            />
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
  if (target.type === 'attendance-list') {
    return `${target.label}${monthSuffix}`;
  }
  if (target.type === 'successful-checkouts') {
    return `Chấm công ra thành công${monthSuffix}`;
  }
  return '';
}

// ─── Failed attempts table ────────────────────────────────────────────────

function inclusivePeriodEnd(periodEnd?: string): string | undefined {
  if (!periodEnd) return undefined;
  return format(subDays(parseISO(periodEnd), 1), 'yyyy-MM-dd');
}

function fallbackToday(): string {
  const today = new Date();
  return `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
}

function FailedAttemptsTable({
  category,
  attemptType,
  periodStart,
  periodEnd,
}: {
  category?: string;
  attemptType?: 'check_in' | 'check_out';
  periodStart?: string;
  periodEnd?: string;
}) {
  const [page, setPage] = useState(1);

  const { data, isLoading, isFetching } = useFailedAttempts({
    type: attemptType,
    category,
    from: periodStart,
    to: inclusivePeriodEnd(periodEnd),
    page,
    pageSize: PAGE_SIZE,
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
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          Tổng <span className="font-medium text-foreground tabular-nums">{total.toLocaleString('vi-VN')}</span> bản ghi
          {isFetching ? ' — đang tải…' : ''}
        </p>
      </div>

      <div className="hidden overflow-hidden rounded-lg border border-border/60 bg-card shadow-sm sm:block">
        <Table className="table-fixed">
          <colgroup>
            <col className="w-[16%]" />
            <col className="w-[9%]" />
            <col className="w-[26%]" />
            <col className="w-[12%]" />
            <col className="w-[18%]" />
            <col className="w-[19%]" />
          </colgroup>
          <TableHeader className="bg-muted/35">
            <TableRow className="hover:bg-transparent">
              <Th>Nhân viên</Th>
              <Th>Loại</Th>
              <Th>Địa điểm</Th>
              <Th>Khoảng cách</Th>
              <Th>Lý do</Th>
              <Th>Thời gian</Th>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id} className="hover:bg-muted/25">
                <Td className="font-medium text-foreground">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td>{labelFor(ATTEMPT_TYPE_LABELS, row.attempt_type)}</Td>
                <Td>
                  <CheckpointLocation row={row} />
                </Td>
                <Td>
                  <CheckpointDistance row={row} />
                </Td>
                <Td>{labelFor(REASON_CATEGORY_LABELS, row.reason_category)}</Td>
                <Td className="whitespace-nowrap text-muted-foreground tabular-nums">
                  {format(parseISO(row.created_at), 'dd/MM/yyyy HH:mm', { locale: vi })}
                </Td>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="space-y-2 sm:hidden">
        {rows.map((row) => (
          <FailedAttemptCard key={row.id} row={row} />
        ))}
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
      <p className="text-sm text-muted-foreground">
        <span className="font-medium text-foreground tabular-nums">{rows.length.toLocaleString('vi-VN')}</span> bản ghi
        {isFetching ? ' — đang tải…' : ''}
      </p>

      <div className="hidden overflow-hidden rounded-lg border border-border/60 bg-card shadow-sm sm:block">
        <Table className="table-fixed">
          <colgroup>
            <col className="w-[16%]" />
            <col className="w-[20%]" />
            <col className="w-[13%]" />
            <col className="w-[13%]" />
            <col className="w-[13%]" />
            <col className="w-[25%]" />
          </colgroup>
          <TableHeader className="bg-muted/35">
            <TableRow className="hover:bg-transparent">
              <Th>Nhân viên</Th>
              <Th>Dự án</Th>
              <Th>Lương</Th>
              <Th>Max adv</Th>
              <Th>Mong đợi</Th>
              <Th>Lý do</Th>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row, i) => (
              <TableRow key={`${row.employee_id}-${row.project_id}-${row.for_month}-${i}`} className="hover:bg-muted/25">
                <Td className="font-medium text-foreground">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td className="text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatCompactCurrency(row.salary)}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatCompactCurrency(row.max_adv_amount)}</Td>
                <Td className="whitespace-nowrap tabular-nums text-muted-foreground">{formatCompactCurrency(row.expected_max)}</Td>
                <Td className="text-muted-foreground">
                  <span className="block whitespace-normal break-words leading-relaxed">{row.reason}</span>
                </Td>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="space-y-2 sm:hidden">
        {rows.map((row, i) => (
          <QuotaAnomalyCard key={`${row.employee_id}-${row.project_id}-${row.for_month}-${i}`} row={row} />
        ))}
      </div>
    </div>
  );
}

// ─── Attendance rows table ─────────────────────────────────────────────────

function AttendanceRowsTable({
  status,
  successfulCheckout,
  zeroEarning,
  periodStart,
  periodEnd,
  emptyLabel,
}: {
  status?: 'checked_in' | 'orphaned' | 'rejected';
  successfulCheckout?: boolean;
  zeroEarning?: boolean;
  periodStart?: string;
  periodEnd?: string;
  emptyLabel: string;
}) {
  const [page, setPage] = useState(1);
  const isRejected = status === 'rejected';

  const queryStart = periodStart ?? fallbackToday();
  const queryEnd = inclusivePeriodEnd(periodEnd) ?? queryStart;

  const { data, isLoading, isFetching } = useAdminAttendances({
    status,
    successful_checkout: successfulCheckout,
    zero_earning: zeroEarning,
    date_field: 'check_in_time',
    from_date: queryStart,
    to_date: queryEnd,
    page,
    pageSize: PAGE_SIZE,
  });

  const rows: AdminAttendanceResponse[] = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  if (isLoading) {
    return <TableSkeleton cols={7} rows={5} />;
  }

  if (!rows.length) {
    return <EmptyState label={emptyLabel} />;
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          Tổng <span className="font-medium text-foreground tabular-nums">{total.toLocaleString('vi-VN')}</span> nhân viên
          {isFetching ? ' — đang tải…' : ''}
        </p>
      </div>

      <div className="hidden overflow-hidden rounded-lg border border-border/60 bg-card shadow-sm sm:block">
        <Table className="table-fixed">
          {isRejected ? (
            <colgroup>
              <col className="w-[16%]" />
              <col className="w-[12%]" />
              <col className="w-[11%]" />
              <col className="w-[11%]" />
              <col className="w-[18%]" />
              <col className="w-[20%]" />
              <col className="w-[12%]" />
            </colgroup>
          ) : (
            <colgroup>
              <col className="w-[18%]" />
              <col className="w-[14%]" />
              <col className="w-[12%]" />
              <col className="w-[12%]" />
              <col className="w-[12%]" />
              <col className="w-[12%]" />
              <col className="w-[20%]" />
            </colgroup>
          )}
          <TableHeader className="bg-muted/35">
            <TableRow className="hover:bg-transparent">
              <Th>Nhân viên</Th>
              <Th>Dự án</Th>
              <Th>Vào làm</Th>
              <Th>Cổng vào</Th>
              {isRejected ? (
                <>
                  <Th>Tự động huỷ lúc</Th>
                  <Th>Địa điểm</Th>
                  <Th>Khoảng cách</Th>
                </>
              ) : (
                <>
                  <Th>Tan ca</Th>
                  <Th>Cổng ra</Th>
                  <Th>Lương ca</Th>
                </>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id} className="hover:bg-muted/25">
                <Td className="font-medium text-foreground">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td className="text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</Td>
                <Td className="whitespace-nowrap tabular-nums text-muted-foreground">
                  {format(parseISO(row.check_in_time), 'HH:mm', { locale: vi })}
                </Td>
                <Td className="text-muted-foreground">{row.check_in_gate || '—'}</Td>
                {isRejected ? (
                  <>
                    <Td className="whitespace-nowrap text-muted-foreground tabular-nums">{formatDateTime(row.rejected_at)}</Td>
                    <Td>
                      <AttendanceCheckpointLocation row={row} />
                    </Td>
                    <Td>
                      <AttendanceCheckpointDistance row={row} />
                    </Td>
                  </>
                ) : (
                  <>
                    <Td className="whitespace-nowrap tabular-nums text-muted-foreground">
                      {row.check_out_time ? format(parseISO(row.check_out_time), 'HH:mm', { locale: vi }) : '—'}
                    </Td>
                    <Td className="text-muted-foreground">{row.check_out_gate || '—'}</Td>
                    <Td className="whitespace-nowrap tabular-nums font-medium text-financial-positive">
                      {row.earning_amount != null && row.earning_amount > 0
                        ? formatCompactCurrency(row.earning_amount)
                        : '—'}
                    </Td>
                  </>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="space-y-2 sm:hidden">
        {rows.map((row) => (
          isRejected ? <RejectedAttendanceCard key={row.id} row={row} /> : <SuccessfulCheckoutCard key={row.id} row={row} />
        ))}
      </div>

      <Pagination page={page} totalPages={totalPages} onChange={setPage} />
    </div>
  );
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  return format(parseISO(value), 'dd/MM/yyyy HH:mm', { locale: vi });
}

function SuccessfulCheckoutCard({ row }: { row: AdminAttendanceResponse }) {
  return (
    <div className="rounded-lg border border-border/60 bg-card p-3 shadow-sm">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
          <p className="mt-1 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
        </div>
        {row.earning_amount != null && row.earning_amount > 0 && (
          <span className="shrink-0 rounded-full bg-emerald-50 px-2 py-1 text-xs font-medium text-financial-positive">
            {formatCompactCurrency(row.earning_amount)}
          </span>
        )}
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1 border-y border-border/50 py-2 text-xs">
        <DetailLine label="Vào làm">
          {format(parseISO(row.check_in_time), 'HH:mm', { locale: vi })} · {row.check_in_gate || '—'}
        </DetailLine>
        <DetailLine label="Tan ca">
          {row.check_out_time ? format(parseISO(row.check_out_time), 'HH:mm', { locale: vi }) : '—'}
          {row.check_out_gate ? ` · ${row.check_out_gate}` : ''}
        </DetailLine>
      </div>
    </div>
  );
}

function RejectedAttendanceCard({ row }: { row: AdminAttendanceResponse }) {
  return (
    <div className="rounded-lg border border-border/60 bg-card p-3 shadow-sm">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
          <p className="mt-1 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
        </div>
        <div className="shrink-0 rounded-md bg-muted px-2.5 py-1.5 text-right">
          <p className="text-sm font-semibold text-foreground tabular-nums">
            {formatDistanceMeters(row.nearest_checkpoint_distance_meters)}
          </p>
          <p className="text-[11px] leading-tight text-muted-foreground">tới điểm chấm</p>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1 border-y border-border/50 py-2 text-xs">
        <DetailLine label="Vào làm">
          {format(parseISO(row.check_in_time), 'HH:mm', { locale: vi })} · {row.check_in_gate || '—'}
        </DetailLine>
        <DetailLine label="Tự động huỷ lúc">{formatDateTime(row.rejected_at)}</DetailLine>
      </div>
      <DetailLine label="Điểm gần nhất">{attendanceCheckpointDetail(row)}</DetailLine>
      <DetailLine label="Lý do">{row.salary_reject_reason || 'Tự động huỷ do quá hạn tan ca'}</DetailLine>
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

function Th({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <TableHead className={cn('h-auto whitespace-normal px-4 py-3 text-[11px] font-semibold uppercase tracking-wide', className)}>
      {children}
    </TableHead>
  );
}

function Td({ children, className }: { children: ReactNode; className?: string }) {
  return <TableCell className={cn('px-4 py-4 align-top leading-relaxed whitespace-normal break-words', className)}>{children}</TableCell>;
}

function AttendanceCheckpointLocation({ row }: { row: AdminAttendanceResponse }) {
  const checkpointName = row.nearest_checkpoint_name?.trim() || row.check_in_gate || 'Điểm chấm gần nhất';
  const delta = formatGeofenceDistanceDelta(row.nearest_checkpoint_distance_meters, row.geofence_radius_meters);

  return (
    <div className="min-w-0">
      <p className="font-medium text-foreground">{checkpointName}</p>
      {delta ? <p className="mt-1 text-xs text-muted-foreground">{delta}</p> : null}
    </div>
  );
}

function AttendanceCheckpointDistance({ row }: { row: AdminAttendanceResponse }) {
  const distance = row.nearest_checkpoint_distance_meters;

  if (distance == null) {
    return <span className="text-sm font-medium text-muted-foreground">Chưa có khoảng cách</span>;
  }

  return <span className="font-semibold text-foreground tabular-nums">{formatDistanceMeters(distance)}</span>;
}

function attendanceCheckpointDetail(row: AdminAttendanceResponse): string {
  if (row.nearest_checkpoint_distance_meters == null) {
    return 'Thiếu tọa độ hoặc điểm chấm';
  }

  const checkpointName = row.nearest_checkpoint_name?.trim() || row.check_in_gate || 'Điểm chấm gần nhất';
  const delta = formatGeofenceDistanceDelta(row.nearest_checkpoint_distance_meters, row.geofence_radius_meters);
  return delta ? `${checkpointName} · ${delta}` : checkpointName;
}

function CheckpointLocation({ row }: { row: AdminFailedAttempt }) {
  const checkpointName = row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất';
  const delta = formatGeofenceDistanceDelta(row.nearest_checkpoint_distance_meters, row.geofence_radius_meters);

  return (
    <div className="min-w-0">
      <p className="font-medium text-foreground">{checkpointName}</p>
      {delta ? <p className="mt-1 text-xs text-muted-foreground">{delta}</p> : null}
    </div>
  );
}

function CheckpointDistance({ row }: { row: AdminFailedAttempt }) {
  const distance = row.nearest_checkpoint_distance_meters;

  if (distance == null) {
    return <span className="text-sm font-medium text-muted-foreground">Chưa có khoảng cách</span>;
  }

  return <span className="font-semibold text-foreground tabular-nums">{formatDistanceMeters(distance)}</span>;
}

function checkpointDetail(row: AdminFailedAttempt): string {
  if (row.nearest_checkpoint_distance_meters == null) {
    return 'Thiếu tọa độ hoặc điểm chấm';
  }

  const checkpointName = row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất';
  const delta = formatGeofenceDistanceDelta(row.nearest_checkpoint_distance_meters, row.geofence_radius_meters);
  return delta ? `${checkpointName} · ${delta}` : checkpointName;
}

function FailedAttemptCard({ row }: { row: AdminFailedAttempt }) {
  return (
    <div className="rounded-lg border border-border/60 bg-card p-3 shadow-sm">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
          <p className="mt-1 text-xs text-muted-foreground tabular-nums">
            {format(parseISO(row.created_at), 'dd/MM/yyyy HH:mm', { locale: vi })}
          </p>
        </div>
        <div className="shrink-0 rounded-md bg-muted px-2.5 py-1.5 text-right">
          <p className="text-sm font-semibold text-foreground tabular-nums">
            {formatDistanceMeters(row.nearest_checkpoint_distance_meters)}
          </p>
          <p className="text-[11px] leading-tight text-muted-foreground">tới điểm chấm</p>
        </div>
      </div>
      <DetailLine label="Điểm gần nhất">{checkpointDetail(row)}</DetailLine>
      <DetailLine label="Lý do">{labelFor(REASON_CATEGORY_LABELS, row.reason_category)}</DetailLine>
      <DetailLine label="Loại">{labelFor(ATTEMPT_TYPE_LABELS, row.attempt_type)}</DetailLine>
    </div>
  );
}

function QuotaAnomalyCard({ row }: { row: QuotaAnomalyRow }) {
  return (
    <div className="rounded-lg border border-border/60 bg-card p-3 shadow-sm">
      <div className="mb-2">
        <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
        <p className="mt-1 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
      </div>
      <div className="grid grid-cols-3 gap-2 border-y border-border/50 py-2 text-xs">
        <Metric label="Lương" value={formatCompactCurrency(row.salary)} />
        <Metric label="Max adv" value={formatCompactCurrency(row.max_adv_amount)} />
        <Metric label="Mong đợi" value={formatCompactCurrency(row.expected_max)} />
      </div>
      <DetailLine label="Lý do">{row.reason}</DetailLine>
    </div>
  );
}

function DetailLine({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="mt-2 text-sm leading-relaxed">
      <span className="font-medium text-muted-foreground">{label}: </span>
      <span className="break-words text-foreground">{children}</span>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-muted-foreground">{label}</p>
      <p className="mt-1 font-medium text-foreground tabular-nums">{value}</p>
    </div>
  );
}

function Pagination({ page, totalPages, onChange }: { page: number; totalPages: number; onChange: (p: number) => void }) {
  if (totalPages <= 1) return null;
  return (
    <div className="flex items-center justify-between gap-3 pt-1">
      <Button
        variant="outline"
        size="sm"
        className="min-w-20"
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
        className="min-w-20"
        disabled={page >= totalPages}
        onClick={() => onChange(Math.min(totalPages, page + 1))}
      >
        Sau
      </Button>
    </div>
  );
}
