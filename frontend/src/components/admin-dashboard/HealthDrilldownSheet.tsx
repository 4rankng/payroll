import { useState, type ReactNode } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from '@/components/ui/sheet';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { ErrorState } from '@/components/ui/error-state';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import {
  AlertCircle,
  AlertTriangle,
  Ban,
  Banknote,
  CalendarClock,
  CalendarX2,
  CheckCircle2,
  Clock,
  Crosshair,
  Hourglass,
  Layers,
  Lock,
  LogIn,
  LogOut,
  Map,
  MapPin,
  MapPinOff,
  SatelliteDish,
  ShieldAlert,
  Smartphone,
  X,
  type LucideIcon,
} from 'lucide-react';
import { format, parseISO, subDays } from 'date-fns';
import { vi } from 'date-fns/locale';
import { EmptyState as SharedEmptyState } from '@/components/shared/EmptyState';

import { useFailedAttempts, useQuotaAnomalies } from '@/hooks/api/useDashboard';
import { useAdminAttendances } from '@/hooks/api/useAdminAttendance';
import { formatFullCurrency } from '@/utils/formatters';
import { getFailedAttemptOverrideViewState } from '@/utils/failedAttemptOverride';
import { formatDistanceMeters, formatGeofenceDistanceDelta } from '@/utils/geoDistance';
import type { AdminFailedAttempt, QuotaAnomalyRow } from '@/types/api/dashboard.types';
import type { AdminAttendanceResponse } from '@/types/api/attendance.types';
import type { HealthDrilldownTarget } from './CheckInHealthStrip';
import { FailedAttemptLocationMap, formatGpsAccuracy } from './FailedAttemptLocationMap';
import { AttendanceLocationMap } from './AttendanceLocationMap';
import type { AttemptTone, ReasonSeverity } from './LocationMap';

const PAGE_SIZE = 20;

interface HealthDrilldownSheetProps {
  target: HealthDrilldownTarget;
  month?: string;
  periodStart?: string;
  periodEnd?: string;
  onClose: () => void;
}

// ─── Reason severity model ────────────────────────────────────────────────
// Each failed-attempt reason carries a severity + icon so an admin can scan a
// long list and instantly separate dangerous geofence breaches from benign
// GPS hiccups or process states.
//   danger  → worker is somewhere they should not be (geofence breach)
//   warning → tech / timing friction (GPS, shift window)
//   info    → permission / config blocking self-attendance
//   neutral → benign process state (already checked in, etc.)

type Severity = 'danger' | 'warning' | 'info' | 'neutral';

interface ReasonMeta {
  label: string;
  severity: Severity;
  icon: LucideIcon;
}

const REASON_META: Record<string, ReasonMeta> = {
  geofence_outside:        { label: 'Ngoài khu vực chấm công',      severity: 'danger',  icon: ShieldAlert },
  geofence_not_configured: { label: 'Chưa cấu hình vị trí',         severity: 'warning', icon: MapPinOff },
  gps_denied:              { label: 'Chưa cấp quyền GPS',           severity: 'warning', icon: Ban },
  gps_timeout:             { label: 'GPS phản hồi chậm',            severity: 'warning', icon: SatelliteDish },
  gps_unavailable:         { label: 'Không lấy được vị trí GPS',    severity: 'warning', icon: SatelliteDish },
  gps_unsupported:         { label: 'Thiết bị không hỗ trợ GPS',    severity: 'warning', icon: Smartphone },
  gps_inaccurate:          { label: 'GPS không chính xác',          severity: 'warning', icon: Crosshair },
  check_in_not_enabled:    { label: 'Chưa cấp quyền chấm công',     severity: 'info',    icon: Lock },
  already_checked_in:      { label: 'Đã vào làm rồi',               severity: 'neutral', icon: CheckCircle2 },
  shift_not_configured:    { label: 'Chưa cấu hình ca',             severity: 'warning', icon: CalendarClock },
  check_in_window:         { label: 'Ngoài giờ vào làm',             severity: 'warning', icon: CalendarClock },
  check_out_window:        { label: 'Ngoài giờ tan ca',              severity: 'warning', icon: CalendarClock },
  already_checked_out:     { label: 'Đã tan ca rồi',                severity: 'neutral', icon: CheckCircle2 },
  already_auto_rejected:   { label: 'Tự động huỷ (hết giờ)',        severity: 'neutral', icon: Clock },
  orphaned:                { label: 'Quá hạn tan ca',               severity: 'warning', icon: Hourglass },
  not_flexible_project:    { label: 'Không hỗ trợ tự chấm công',    severity: 'info',    icon: Lock },
  no_flexible_project:     { label: 'Không thuộc dự án tự chấm công', severity: 'info',  icon: Lock },
  multiple_flexible:       { label: 'Thuộc nhiều dự án',            severity: 'info',    icon: Layers },
  no_attendance:           { label: 'Không có ca hôm nay',          severity: 'neutral', icon: CalendarX2 },
  check_in_disabled:       { label: 'Chấm công bị khoá',            severity: 'info',    icon: Lock },
  other:                   { label: 'Lỗi khác',                     severity: 'neutral', icon: AlertCircle },
  // Attendance-based drilldown categories (not from attempt_classifier)
  open:                    { label: 'Đang chờ checkout',            severity: 'info',    icon: Clock },
  zero_earning:            { label: 'Ca 0 ₫',                       severity: 'warning', icon: Banknote },
  stuck_pending:           { label: 'Yêu cầu kẹt pending',          severity: 'warning', icon: Hourglass },
  request_failed:          { label: 'Yêu cầu lỗi',                  severity: 'danger',  icon: AlertTriangle },
};

const DEFAULT_REASON_META: ReasonMeta = { label: '', severity: 'neutral', icon: AlertCircle };

function reasonMeta(category: string): ReasonMeta {
  const meta = REASON_META[category];
  return meta ? meta : { ...DEFAULT_REASON_META, label: category };
}

// Badge text hues are darkened (rose-700 / amber-700) so they clear WCAG AA
// 4.5:1 against the soft tinted backgrounds — the raw --warning/--destructive
// tokens read ~2.7:1 / ~3.9:1, which fails for small badge text.
const SEVERITY_STYLES: Record<Severity, { badge: string; text: string; dot: string }> = {
  danger:  { badge: 'bg-rose-50 text-rose-700 border-rose-200',       text: 'text-rose-700',  dot: 'bg-rose-500' },
  warning: { badge: 'bg-amber-50 text-amber-700 border-amber-200',    text: 'text-amber-700', dot: 'bg-amber-500' },
  info:    { badge: 'bg-primary/10 text-primary border-primary/20',   text: 'text-primary',   dot: 'bg-primary' },
  neutral: { badge: 'bg-muted text-muted-foreground border-border/60', text: 'text-foreground', dot: 'bg-muted-foreground/45' },
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
          'flex w-full flex-col p-0 shadow-none sm:w-[min(920px,calc(100vw-2rem))]',
          isMobile && 'max-h-[94dvh] rounded-t-2xl',
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
              <Button variant="ghost" size="icon" aria-label="Đóng" className="h-11 w-11 shrink-0 rounded-full text-muted-foreground">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-3 sm:p-6">
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
    const catLabel = target.category ? reasonMeta(target.category).label : '';
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
  const [mapRow, setMapRow] = useState<AdminFailedAttempt | null>(null);

  const { data, isLoading, isFetching, isError, refetch } = useFailedAttempts({
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

  if (isError) {
    return <div role="alert"><ErrorState onRetry={() => void refetch()} /></div>;
  }

  if (!rows.length) {
    return <EmptyState label="Không có lần chấm thất bại" />;
  }

  return (
    <div className="space-y-3">
      <CountSummary total={total} isFetching={isFetching} noun="bản ghi" />

      <div className="hidden overflow-hidden rounded-2xl border border-border bg-card shadow-none sm:block">
        <div className="grid grid-cols-[minmax(160px,1.15fr)_minmax(220px,1.55fr)_minmax(112px,.75fr)_minmax(180px,1.35fr)_minmax(90px,.55fr)] border-b border-border bg-muted/50 px-4 py-3">
          <FailedAttemptHeader>Nhân viên</FailedAttemptHeader>
          <FailedAttemptHeader>Lý do</FailedAttemptHeader>
          <FailedAttemptHeader>Khoảng cách</FailedAttemptHeader>
          <FailedAttemptHeader>Địa điểm</FailedAttemptHeader>
          <FailedAttemptHeader>Thời gian</FailedAttemptHeader>
        </div>
        <div className="divide-y divide-border/60">
          {rows.map((row) => (
            <FailedAttemptDesktopRow
              key={row.id}
              row={row}
              onOpenMap={() => setMapRow(row)}
            />
          ))}
        </div>
      </div>

      <div className="space-y-3 sm:hidden">
        {rows.map((row) => (
          <FailedAttemptCard
            key={row.id}
            row={row}
            onOpenMap={() => setMapRow(row)}
          />
        ))}
      </div>

      <Pagination page={page} totalPages={totalPages} onChange={setPage} />

      <FailedAttemptMapDialog row={mapRow} onClose={() => setMapRow(null)} />
    </div>
  );
}

function FailedAttemptMapDialog({ row, onClose }: { row: AdminFailedAttempt | null; onClose: () => void }) {
  return (
    <Dialog open={row !== null} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent
        title="Bản đồ lần chấm công"
        hideCloseButton
        className="inset-0 max-h-none max-w-none translate-x-0 translate-y-0 gap-0 rounded-none border-0 shadow-none"
        contentPadding="none"
      >
        {row ? (
          <FailedAttemptLocationMap
            key={row.id}
            row={row}
            attemptLabel={labelFor(ATTEMPT_TYPE_LABELS, row.attempt_type)}
            reasonLabel={reasonMeta(row.reason_category).label}
            reasonSeverity={reasonMeta(row.reason_category).severity}
            timeLabel={format(parseISO(row.created_at), 'dd/MM/yyyy HH:mm', { locale: vi })}
            onClose={onClose}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

// ─── Quota anomaly table ──────────────────────────────────────────────────

function QuotaAnomalyTable({ anomalyType, month }: { anomalyType: string; month?: string }) {
  const { data: rows, isLoading, isFetching, isError, refetch } = useQuotaAnomalies(anomalyType, month);

  if (isLoading) {
    return <TableSkeleton cols={5} rows={4} />;
  }

  if (isError) {
    return <div role="alert"><ErrorState onRetry={() => void refetch()} /></div>;
  }

  if (!rows || rows.length === 0) {
    return <EmptyState label="Không có bất thường quota" />;
  }

  return (
    <div className="space-y-3">
      <CountSummary total={rows.length} isFetching={isFetching} noun="bản ghi" />

      <div className="hidden overflow-hidden rounded-xl border border-border/60 bg-card shadow-none sm:block">
        <Table className="table-fixed">
          <colgroup>
            <col className="w-[16%]" />
            <col className="w-[20%]" />
            <col className="w-[13%]" />
            <col className="w-[13%]" />
            <col className="w-[13%]" />
            <col className="w-[25%]" />
          </colgroup>
          <TableHeader className="bg-muted/40">
            <TableRow className="hover:bg-transparent border-border/50">
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
              <TableRow key={`${row.employee_id}-${row.project_id}-${row.for_month}-${i}`} className="border-border/50 transition-colors hover:bg-muted/30">
                <Td className="font-medium text-foreground">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td className="text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatFullCurrency(row.salary)}</Td>
                <Td className="whitespace-nowrap tabular-nums">{formatFullCurrency(row.max_adv_amount)}</Td>
                <Td className="whitespace-nowrap tabular-nums text-muted-foreground">{formatFullCurrency(row.expected_max)}</Td>
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
  const [mapRow, setMapRow] = useState<AdminAttendanceResponse | null>(null);
  const isRejected = status === 'rejected';
  const drawerVariant: AttendanceDrawerVariant = isRejected
    ? 'rejected'
    : zeroEarning
      ? 'zero'
      : successfulCheckout
        ? 'success'
        : 'open';

  const queryStart = periodStart ?? fallbackToday();
  const queryEnd = inclusivePeriodEnd(periodEnd) ?? queryStart;

  const { data, isLoading, isFetching, isError, refetch } = useAdminAttendances({
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

  if (isError) {
    return <div role="alert"><ErrorState onRetry={() => void refetch()} /></div>;
  }

  if (!rows.length) {
    return <EmptyState label={emptyLabel} />;
  }

  return (
    <div className="space-y-3">
      <CountSummary total={total} isFetching={isFetching} noun="nhân viên" />

      <div className="hidden overflow-hidden rounded-xl border border-border/60 bg-card shadow-none sm:block">
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
          <TableHeader className="bg-muted/40">
            <TableRow className="hover:bg-transparent border-border/50">
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
              <TableRow key={row.id} className="border-border/50 transition-colors hover:bg-muted/30">
                <Td className="font-medium text-foreground">{row.employee_name ?? `#${row.employee_id}`}</Td>
                <Td className="text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</Td>
                <Td className="whitespace-nowrap text-muted-foreground">
                  <TimeCell iso={row.check_in_time} />
                </Td>
                <Td className="text-muted-foreground">
                  <span className="block">{row.check_in_gate || '—'}</span>
                  <span className="mt-1 block text-xs text-muted-foreground">{formatGpsAccuracy(row.check_in_accuracy)}</span>
                </Td>
                {isRejected ? (
                  <>
                    <Td className="whitespace-nowrap text-muted-foreground tabular-nums">{formatDateTime(row.rejected_at)}</Td>
                    <Td>
                      <AttendanceCheckpointLocation row={row} />
                      <MapButton onClick={() => setMapRow(row)} />
                    </Td>
                    <Td>
                      <AttendanceCheckpointDistance row={row} />
                    </Td>
                  </>
                ) : (
                  <>
                    <Td className="whitespace-nowrap text-muted-foreground">
                      {row.check_out_time ? <TimeCell iso={row.check_out_time} /> : '—'}
                    </Td>
                    <Td className="text-muted-foreground">
                      <span className="block">{row.check_out_gate || '—'}</span>
                      <span className="mt-1 block text-xs text-muted-foreground">{formatGpsAccuracy(row.check_out_accuracy)}</span>
                      <MapButton onClick={() => setMapRow(row)} />
                    </Td>
                    <Td className="whitespace-nowrap tabular-nums font-medium text-financial-positive">
                      {row.earning_amount != null && row.earning_amount > 0
                        ? formatFullCurrency(row.earning_amount)
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
          isRejected ? (
            <RejectedAttendanceCard key={row.id} row={row} onOpenMap={() => setMapRow(row)} />
          ) : (
            <SuccessfulCheckoutCard key={row.id} row={row} onOpenMap={() => setMapRow(row)} />
          )
        ))}
      </div>

      <Pagination page={page} totalPages={totalPages} onChange={setPage} />

      <AttendanceMapDialog row={mapRow} variant={drawerVariant} onClose={() => setMapRow(null)} />
    </div>
  );
}

type AttendanceDrawerVariant = 'success' | 'zero' | 'rejected' | 'open';

function attendanceMapBadge(variant: AttendanceDrawerVariant): { label: string; tone: AttemptTone } {
  switch (variant) {
    case 'rejected':
      return { label: 'Đã huỷ', tone: 'slate' };
    case 'zero':
      return { label: 'Ca 0 ₫', tone: 'amber' };
    case 'success':
      return { label: 'Tan ca', tone: 'blue' };
    default:
      return { label: 'Đang làm', tone: 'slate' };
  }
}

function attendanceMapReason(
  variant: AttendanceDrawerVariant,
  row: AdminAttendanceResponse,
): { label: string; severity: ReasonSeverity } | null {
  if (variant === 'rejected') {
    return { label: row.salary_reject_reason || 'Tự động huỷ do quá hạn tan ca', severity: 'warning' };
  }
  if (variant === 'zero') {
    return { label: 'Ca không có lương', severity: 'warning' };
  }
  return null;
}

function formatAttendanceRange(row: AdminAttendanceResponse): string {
  const start = format(parseISO(row.check_in_time), 'dd/MM HH:mm', { locale: vi });
  const end = row.check_out_time ? format(parseISO(row.check_out_time), 'HH:mm', { locale: vi }) : null;
  return end ? `${start} → ${end}` : start;
}

function AttendanceMapDialog({
  row,
  variant,
  onClose,
}: {
  row: AdminAttendanceResponse | null;
  variant: AttendanceDrawerVariant;
  onClose: () => void;
}) {
  const badge = attendanceMapBadge(variant);
  const reason = row ? attendanceMapReason(variant, row) : null;
  return (
    <Dialog open={row !== null} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent
        title="Bản đồ chấm công"
        hideCloseButton
        className="inset-0 max-h-none max-w-none translate-x-0 translate-y-0 gap-0 rounded-none border-0 shadow-none"
        contentPadding="none"
      >
        {row ? (
          <AttendanceLocationMap
            key={row.id}
            row={row}
            badge={badge}
            reasonLabel={reason?.label}
            reasonSeverity={reason?.severity}
            timeLabel={formatAttendanceRange(row)}
            onClose={onClose}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function MapButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="mt-2 min-h-11 gap-1.5 px-3 text-xs font-medium text-muted-foreground hover:text-foreground"
      onClick={onClick}
    >
      <Map className="h-3.5 w-3.5" />
      Xem bản đồ
    </Button>
  );
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  return format(parseISO(value), 'dd/MM/yyyy HH:mm', { locale: vi });
}

function SuccessfulCheckoutCard({ row, onOpenMap }: { row: AdminAttendanceResponse; onOpenMap: () => void }) {
  return (
    <div className="rounded-xl border border-border/60 bg-card p-3 shadow-none">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div className="flex items-center gap-2.5 min-w-0">
          <Monogram name={row.employee_name} />
          <div className="min-w-0">
            <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
          </div>
        </div>
        {row.earning_amount != null && row.earning_amount > 0 && (
          <span className="shrink-0 rounded-full bg-success/10 px-2 py-1 text-xs font-semibold text-financial-positive">
            {formatFullCurrency(row.earning_amount)}
          </span>
        )}
      </div>
      <div className="grid grid-cols-1 gap-x-4 gap-y-1 border-y border-border/50 py-2 text-xs min-[380px]:grid-cols-2">
        <DetailLine label="Vào làm">
          {format(parseISO(row.check_in_time), 'HH:mm dd/MM', { locale: vi })} · {row.check_in_gate || '—'}
        </DetailLine>
        <DetailLine label="Tan ca">
          {row.check_out_time ? format(parseISO(row.check_out_time), 'HH:mm dd/MM', { locale: vi }) : '—'}
          {row.check_out_gate ? ` · ${row.check_out_gate}` : ''}
        </DetailLine>
        <DetailLine label="GPS vào">{formatGpsAccuracy(row.check_in_accuracy)}</DetailLine>
        <DetailLine label="GPS ra">{formatGpsAccuracy(row.check_out_accuracy)}</DetailLine>
      </div>
      <div className="mt-2">
        <Button type="button" variant="outline" size="sm" className="min-h-11 w-full gap-1.5" onClick={onOpenMap}>
          <Map className="h-4 w-4" />
          Xem bản đồ
        </Button>
      </div>
    </div>
  );
}

function RejectedAttendanceCard({ row, onOpenMap }: { row: AdminAttendanceResponse; onOpenMap: () => void }) {
  return (
    <div className="rounded-xl border border-border/60 bg-card p-3 shadow-none">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div className="flex items-center gap-2.5 min-w-0">
          <Monogram name={row.employee_name} />
          <div className="min-w-0">
            <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
          </div>
        </div>
        <div className="shrink-0 rounded-md bg-muted px-2.5 py-1.5 text-right">
          <p className="text-sm font-semibold text-foreground tabular-nums">
            {formatDistanceMeters(row.nearest_checkpoint_distance_meters)}
          </p>
          <p className="text-xs leading-tight text-muted-foreground">tới điểm chấm</p>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-x-4 gap-y-1 border-y border-border/50 py-2 text-xs min-[380px]:grid-cols-2">
        <DetailLine label="Vào làm">
          {format(parseISO(row.check_in_time), 'HH:mm dd/MM', { locale: vi })} · {row.check_in_gate || '—'}
        </DetailLine>
        <DetailLine label="Tự động huỷ lúc">{formatDateTime(row.rejected_at)}</DetailLine>
        <DetailLine label="GPS vào">{formatGpsAccuracy(row.check_in_accuracy)}</DetailLine>
        <DetailLine label="GPS ra">{formatGpsAccuracy(row.check_out_accuracy)}</DetailLine>
      </div>
      <DetailLine label="Điểm gần nhất">{attendanceCheckpointDetail(row)}</DetailLine>
      <DetailLine label="Lý do">{row.salary_reject_reason || 'Tự động huỷ do quá hạn tan ca'}</DetailLine>
      <div className="mt-2">
        <Button type="button" variant="outline" size="sm" className="min-h-11 w-full gap-1.5" onClick={onOpenMap}>
          <Map className="h-4 w-4" />
          Xem bản đồ
        </Button>
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
    <div className="rounded-xl border border-dashed border-border/60 bg-muted/20 px-4">
      <SharedEmptyState title={label} size="sm" />
    </div>
  );
}

function CountSummary({ total, isFetching, noun }: { total: number; isFetching: boolean; noun: string }) {
  return (
    <div className="flex items-center gap-2">
      <div className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-card px-3 py-1.5 shadow-none">
        <span className="text-xs font-medium text-muted-foreground">Tổng</span>
        <span className="font-display text-sm font-bold tabular-nums text-foreground">
          {total.toLocaleString('vi-VN')}
        </span>
        <span className="text-xs text-muted-foreground">{noun}</span>
      </div>
      {isFetching ? <span className="text-xs text-muted-foreground">đang tải…</span> : null}
    </div>
  );
}

function Th({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <TableHead className={cn('h-auto whitespace-normal px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground', className)}>
      {children}
    </TableHead>
  );
}

function Td({ children, className }: { children: ReactNode; className?: string }) {
  return <TableCell className={cn('px-4 py-3.5 align-top leading-relaxed whitespace-normal break-words', className)}>{children}</TableCell>;
}

const MONOGRAM_PALETTE = [
  'bg-blue-500/10 text-blue-600',
  'bg-emerald-500/10 text-emerald-700',
  'bg-amber-500/10 text-amber-700',
  'bg-teal-500/10 text-teal-700',
  'bg-rose-500/10 text-rose-600',
  'bg-sky-500/10 text-sky-700',
  'bg-primary/10 text-primary',
];

function monogramColor(seed: string): string {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  }
  return MONOGRAM_PALETTE[hash % MONOGRAM_PALETTE.length];
}

function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function Monogram({ name }: { name?: string | null }) {
  const label = name?.trim() || '?';
  return (
    <span
      className={cn(
        'flex h-8 w-8 shrink-0 items-center justify-center rounded-full font-display text-xs font-bold uppercase',
        monogramColor(label),
      )}
      aria-hidden
    >
      {initialsOf(label)}
    </span>
  );
}

function AttemptTypeBadge({ attemptType }: { attemptType: string }) {
  const label = labelFor(ATTEMPT_TYPE_LABELS, attemptType);
  const Icon = attemptType === 'check_in' ? LogIn : attemptType === 'check_out' ? LogOut : Clock;
  const tone =
    attemptType === 'check_in'
      ? 'bg-primary/10 text-primary border-primary/20'
      : attemptType === 'check_out'
        ? 'bg-teal-500/10 text-teal-700 border-teal-500/20'
        : 'bg-muted text-muted-foreground border-border/60';
  return (
    <span className={cn('inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs font-medium leading-tight', tone)}>
      <Icon className="h-3 w-3 shrink-0" />
      <span className="min-w-0 whitespace-normal break-words">{label}</span>
    </span>
  );
}

function SeverityBadge({ meta }: { meta: ReasonMeta }) {
  const s = SEVERITY_STYLES[meta.severity];
  const Icon = meta.icon;
  return (
    <span className={cn('inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-1 text-xs font-medium leading-tight', s.badge)}>
      <Icon className="h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 whitespace-normal break-words">{meta.label}</span>
    </span>
  );
}

function TimeCell({ iso }: { iso: string }) {
  const time = format(parseISO(iso), 'HH:mm', { locale: vi });
  const date = format(parseISO(iso), 'dd/MM/yyyy', { locale: vi });
  return (
    <div className="whitespace-nowrap">
      <span className="block text-sm font-medium tabular-nums text-foreground">{time}</span>
      <span className="block text-xs tabular-nums text-muted-foreground">{date}</span>
    </div>
  );
}

function severityForDistance(distance?: number | null, radius?: number | null): Severity {
  if (distance == null || !Number.isFinite(distance)) return 'neutral';
  if (distance >= 1000) return 'danger'; // ≥ 1 km from checkpoint is a serious breach
  if (radius != null && radius > 0 && distance > radius) return 'warning';
  return 'neutral';
}

function ContextualDistance({ row }: { row: AdminFailedAttempt }) {
  const distance = row.nearest_checkpoint_distance_meters;

  if (distance == null || !Number.isFinite(distance)) {
    return <span className="text-xs font-medium text-muted-foreground">Chưa có khoảng cách</span>;
  }

  const severity = severityForDistance(distance, row.geofence_radius_meters);
  const delta = formatGeofenceDistanceDelta(distance, row.geofence_radius_meters);
  const s = SEVERITY_STYLES[severity];

  return (
    <div className="inline-flex flex-col gap-0.5">
      <span className={cn('font-display text-base font-bold leading-none tabular-nums', s.text)}>
        {formatDistanceMeters(distance)}
      </span>
      {delta ? <span className="text-xs leading-tight text-muted-foreground">{delta}</span> : null}
    </div>
  );
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
  const accuracy = formatGpsAccuracy(row.accuracy);

  return (
    <div className="min-w-0">
      <p className="flex items-start gap-1 font-medium leading-snug text-foreground">
        <MapPin className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="min-w-0 break-words">{checkpointName}</span>
      </p>
      <p className="mt-1 pl-[18px] text-xs text-muted-foreground">{accuracy}</p>
    </div>
  );
}

function checkpointDetail(row: AdminFailedAttempt): string {
  if (row.nearest_checkpoint_distance_meters == null) {
    return 'Thiếu tọa độ hoặc điểm chấm';
  }

  const checkpointName = row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất';
  const delta = formatGeofenceDistanceDelta(row.nearest_checkpoint_distance_meters, row.geofence_radius_meters);
  return delta ? `${checkpointName} · ${delta}` : checkpointName;
}

function FailedAttemptOverrideAction({ row }: { row: AdminFailedAttempt }) {
  const state = getFailedAttemptOverrideViewState(row.reason_category, row.resolved_at);

  // The manual override ("Ghi nhận chấm công") button was removed from the
  // dashboard; this now only surfaces the "already recorded" badge for attempts
  // resolved previously (e.g. via the backend API).
  if (!state.canOverride || !state.isResolved) return null;
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-emerald-200">
      <CheckCircle2 className="h-3 w-3" />
      {state.resolvedLabel}
    </span>
  );
}

function FailedAttemptHeader({ children }: { children: ReactNode }) {
  return (
    <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
      {children}
    </span>
  );
}

function FailedAttemptDesktopRow({
  row,
  onOpenMap,
}: {
  row: AdminFailedAttempt;
  onOpenMap: () => void;
}) {
  const meta = reasonMeta(row.reason_category);

  return (
    <div className="grid min-h-[92px] grid-cols-[minmax(160px,1.15fr)_minmax(220px,1.55fr)_minmax(112px,.75fr)_minmax(180px,1.35fr)_minmax(90px,.55fr)] px-4 py-3.5 transition-colors hover:bg-muted/40">
      <div className="flex min-w-0 items-start gap-2.5 pr-4">
        <Monogram name={row.employee_name} />
        <p className="min-w-0 break-words text-sm font-medium leading-snug text-foreground">
          {row.employee_name ?? `#${row.employee_id}`}
        </p>
      </div>

      <div className="flex min-w-0 flex-col items-start gap-1.5 pr-4">
        <AttemptTypeBadge attemptType={row.attempt_type} />
        <SeverityBadge meta={meta} />
        <FailedAttemptOverrideAction row={row} />
      </div>

      <div className="min-w-0 pr-4">
        <ContextualDistance row={row} />
      </div>

      <div className="min-w-0 pr-4">
        <CheckpointLocation row={row} />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-2 min-h-11 gap-1.5 rounded-full px-3 text-xs font-semibold text-muted-foreground hover:bg-muted/60 hover:text-foreground"
          onClick={onOpenMap}
        >
          <Map className="h-3.5 w-3.5" />
          Xem bản đồ
        </Button>
      </div>

      <div className="min-w-0">
        <TimeCell iso={row.created_at} />
      </div>
    </div>
  );
}

function FailedAttemptCard({
  row,
  onOpenMap,
}: {
  row: AdminFailedAttempt;
  onOpenMap: () => void;
}) {
  const meta = reasonMeta(row.reason_category);

  return (
    <article className="rounded-xl border border-border bg-card p-3 shadow-none">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-2.5">
          <Monogram name={row.employee_name} />
          <div className="min-w-0">
            <p className="break-words text-sm font-medium leading-snug text-foreground">
              {row.employee_name ?? `#${row.employee_id}`}
            </p>
            <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
              <MapPin className="h-3.5 w-3.5 shrink-0" />
              <span className="min-w-0 truncate">{row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất'}</span>
            </p>
          </div>
        </div>
        <div className="shrink-0 text-right">
          <TimeCell iso={row.created_at} />
        </div>
      </div>

      <div className="mt-3 grid grid-cols-1 gap-2 rounded-lg border border-border/60 bg-muted/50 p-3 min-[380px]:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <div className="min-w-0">
          <p className="mb-1 text-xs font-semibold uppercase text-muted-foreground">Khoảng cách</p>
          <ContextualDistance row={row} />
        </div>
        <div className="min-w-0 border-t border-border pt-2 min-[380px]:border-l min-[380px]:border-t-0 min-[380px]:pl-3 min-[380px]:pt-0">
          <p className="mb-1 text-xs font-semibold uppercase text-muted-foreground">Địa điểm</p>
          <p className="break-words text-sm font-semibold text-foreground">
            {row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất'}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">{formatGpsAccuracy(row.accuracy)}</p>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <AttemptTypeBadge attemptType={row.attempt_type} />
        <SeverityBadge meta={meta} />
      </div>

      <div className="mt-3 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="min-h-11 flex-1 gap-1.5 rounded-full text-xs font-semibold text-foreground"
          onClick={onOpenMap}
        >
          <Map className="h-4 w-4" />
          Xem bản đồ
        </Button>
        <FailedAttemptOverrideAction row={row} />
      </div>
    </article>
  );
}

function QuotaAnomalyCard({ row }: { row: QuotaAnomalyRow }) {
  return (
    <div className="rounded-xl border border-border/60 bg-card p-3 shadow-none">
      <div className="mb-2 flex items-center gap-2.5">
        <Monogram name={row.employee_name} />
        <div className="min-w-0">
          <p className="font-medium leading-snug text-foreground">{row.employee_name ?? `#${row.employee_id}`}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{row.project_name ?? `#${row.project_id}`}</p>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-2 border-y border-border/50 py-2 text-xs min-[380px]:grid-cols-3">
        <Metric label="Lương" value={formatFullCurrency(row.salary)} />
        <Metric label="Max adv" value={formatFullCurrency(row.max_adv_amount)} />
        <Metric label="Mong đợi" value={formatFullCurrency(row.expected_max)} />
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
      <p className="mt-1 break-words font-medium text-foreground tabular-nums">{value}</p>
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
        className="h-11 min-w-[44px] sm:min-w-20"
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
        className="h-11 min-w-[44px] sm:min-w-20"
        disabled={page >= totalPages}
        onClick={() => onChange(Math.min(totalPages, page + 1))}
      >
        Sau
      </Button>
    </div>
  );
}
