import { useState, useCallback, useMemo } from 'react';
import { format, startOfMonth } from 'date-fns';
import { Users, UserCheck, UserX, Banknote } from 'lucide-react';
import { UserX01 } from '@untitledui/icons';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageHeader } from '@/components/shared/PageHeader';
import { ConsoleKpi, ConsolePanel } from '@/components/shared/console-panel';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import { MissingMobileSection } from '@/components/employees/MissingMobileSection';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
import { PartnerWorkforceOverviewCard } from '@/components/partner-dashboard/PartnerWorkforceOverviewCard';
import { PartnerApprovalRequestsCard } from '@/components/partner-dashboard/PartnerApprovalRequestsCard';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/shared/EmptyState';
import { formatCompactVND } from '@/utils/formatters';
import type {
  TopPaidEmployeeItem,
  PartnerEmployeeListType,
} from '@/types/api/dashboard.types';
import type { Employee } from '@/types/api/employee.types';
import { monthOptions, formatVND } from './utils';

const ALL_VALUE = 'all';
const OTHER_MONTH_VALUE = 'other-month';

// ─── Month selector ────────────────────────────────────────────────────────
function MonthSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const primaryOptions = monthOptions.slice(0, 4);
  const otherOptions = monthOptions.slice(4);
  const isOtherMonth = otherOptions.some((option) => option.value === value);

  return (
    <div
      className="console-filter-bar rounded-lg bg-muted/60 p-1"
      role="group"
      aria-label="Chọn kỳ dữ liệu"
    >
      {primaryOptions.map((opt) => (
        <button
          type="button"
          key={opt.value}
          onClick={() => onChange(opt.value)}
          aria-pressed={value === opt.value}
          className={cn(
            'h-9 shrink-0 rounded-md px-3 text-xs font-semibold transition-[background-color,color,box-shadow]',
            'outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2',
            value === opt.value
              ? 'bg-card text-foreground shadow-sm ring-1 ring-inset ring-border/60'
              : 'text-muted-foreground hover:bg-card/70 hover:text-foreground',
          )}
        >
          {opt.label}
        </button>
      ))}
      <Select
        value={isOtherMonth ? value : OTHER_MONTH_VALUE}
        onValueChange={(nextValue) => {
          if (nextValue !== OTHER_MONTH_VALUE) onChange(nextValue);
        }}
      >
        <SelectTrigger
          aria-label="Chọn tháng khác"
          className={cn(
            'h-9 min-h-9 w-[132px] shrink-0 border-0 bg-transparent px-3 text-xs font-semibold shadow-none focus:ring-1 focus:ring-ring focus:ring-offset-0',
            isOtherMonth && 'bg-card text-foreground shadow-sm ring-1 ring-border/60',
          )}
        >
          <SelectValue placeholder="Tháng khác" />
        </SelectTrigger>
        <SelectContent align="end">
          <SelectItem value={OTHER_MONTH_VALUE} disabled>
            Tháng khác
          </SelectItem>
          {otherOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

// ─── Leaderboard row ───────────────────────────────────────────────────────
const RANK_BADGE = [
  'bg-warning/15 text-warning ring-1 ring-warning/30',
  'bg-muted text-muted-foreground ring-1 ring-border',
  'bg-warning/10 text-warning/80 ring-1 ring-warning/20',
];

function LeaderRow({ item, rank }: { item: TopPaidEmployeeItem; rank: number }) {
  const badge = RANK_BADGE[rank - 1] ?? 'bg-muted/60 text-muted-foreground ring-1 ring-border/60';
  return (
    <div className="group grid grid-cols-[2rem_2.25rem_minmax(0,1fr)] items-center gap-2 rounded-md px-2 py-2 transition-colors hover:bg-muted/40">
      <span className={cn('flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold tabular-nums', badge)}>
        {rank}
      </span>
      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="truncate text-[13px] font-semibold text-foreground">{item.employee_name}</span>
          {!item.is_active && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-destructive/10 px-1.5 py-0.5 text-xs font-semibold text-destructive shrink-0">
              <UserX01 className="h-2.5 w-2.5" aria-hidden="true" />
              Nghỉ
            </span>
          )}
        </div>
        <span className="console-value mt-0.5 block text-xs font-bold text-foreground">
          {formatVND(item.total_paid_vnd)}
        </span>
      </div>
    </div>
  );
}

// ─── Skeletons ─────────────────────────────────────────────────────────────
function StatRowSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="console-panel flex items-center gap-2.5 px-3 py-2">
          <Skeleton className="size-9 shrink-0 rounded-md" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <Skeleton className="h-2.5 w-16" />
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-2.5 w-20" />
          </div>
        </div>
      ))}
    </div>
  );
}

function LeaderboardSkeleton() {
  return (
    <div className="space-y-2 pt-2">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-2 py-2.5">
          <Skeleton className="h-7 w-7 rounded-full" />
          <Skeleton className="h-8 w-8 rounded-full" />
          <div className="flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-1.5 w-full rounded-full max-w-[140px]" />
          </div>
          <Skeleton className="h-3.5 w-20" />
        </div>
      ))}
    </div>
  );
}

// ─── Banner header (brand emerald only) ────────────────────────────────────
function BannerHeader({
  periodLabel,
  monthValue,
  onMonthChange,
}: {
  periodLabel: string;
  monthValue: string;
  onMonthChange: (v: string) => void;
}) {
  return (
    <PageHeader
      kicker="Đối tác"
      title="Tổng quan"
      meta={`Nhân sự và thanh toán · ${periodLabel}`}
    >
      <MonthSelector value={monthValue} onChange={onMonthChange} />
    </PageHeader>
  );
}

// ─── Main page ─────────────────────────────────────────────────────────────
const PartnerDashboardPage = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM'),
  );
  const [sheetType, setSheetType] = useState<PartnerEmployeeListType | null>(null);

  const handleMonthChange = useCallback((value: string) => setSelectedMonth(value), []);
  const openSheet = useCallback((type: PartnerEmployeeListType) => setSheetType(type), []);
  const closeSheet = useCallback(() => setSheetType(null), []);

  // Employee rows in the mobile-warning section open the employee details
  // sheet through the URL-driven modal system — same wiring as the partner
  // EmployeesPage's handleEmployeeClick.
  const { openEmployeeDetails } = useEmployeeModals();
  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const apiMonth = selectedMonth === ALL_VALUE ? undefined : selectedMonth;
  const { data, isLoading } = usePartnerDashboard({ month: apiMonth });

  const periodLabel = useMemo(() => {
    if (selectedMonth === ALL_VALUE) return 'tất cả thời gian';
    const [y, m] = selectedMonth.split('-');
    return `tháng ${m}/${y}`;
  }, [selectedMonth]);

  const topEmployees = useMemo(() => data?.top_paid_employees ?? [], [data?.top_paid_employees]);
  const showMomBadges = selectedMonth !== ALL_VALUE;

  return (
    <div className="min-h-full p-3 sm:p-4">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-3">
        <BannerHeader
          periodLabel={periodLabel}
          monthValue={selectedMonth}
          onMonthChange={handleMonthChange}
        />

        {/* ── KPI strip ── */}
        {isLoading ? (
          <StatRowSkeleton />
        ) : (
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4 motion-safe:animate-fade-in-up">
            <ConsoleKpi
              label="Tổng chi trả"
              value={formatCompactVND(data?.total_paid_vnd ?? 0)}
              title={formatVND(data?.total_paid_vnd ?? 0)}
              icon={Banknote}
              tone="primary"
              delta={
                showMomBadges && (data?.total_paid_vnd ?? 0) > 0
                  ? { value: data?.mom_paid_amount?.change_pct ?? null }
                  : undefined
              }
              hint={!showMomBadges ? periodLabel : undefined}
              onClick={(data?.total_paid_vnd ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
            <ConsoleKpi
              label="Đang làm việc"
              value={(data?.active_employees ?? 0).toLocaleString('vi-VN')}
              icon={UserCheck}
              tone="secondary"
              hint="bảng công 14 ngày qua"
              onClick={() => openSheet('active')}
            />
            <ConsoleKpi
              label="Có thể nghỉ"
              value={(data?.dropped_employees ?? 0).toLocaleString('vi-VN')}
              icon={UserX}
              tone="gold"
              hint="không bảng công 14 ngày"
              onClick={
                (data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined
              }
            />
            <ConsoleKpi
              label="Đã thanh toán"
              value={(data?.paid_employees ?? 0).toLocaleString('vi-VN')}
              icon={Users}
              tone="neutral"
              delta={
                showMomBadges && (data?.paid_employees ?? 0) > 0
                  ? { value: data?.mom_paid_employees?.change_pct ?? null }
                  : undefined
              }
              hint={!showMomBadges ? 'tất cả thời gian' : undefined}
              onClick={
                (data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined
              }
            />
          </div>
        )}

        <PartnerApprovalRequestsCard />

        {/* Mobile warnings render nothing when clean (no dead card). */}
        <MissingMobileSection onEmployeeClick={handleEmployeeClick} />

        {/* ── Analytics gallery: workforce donut + leaderboard ── */}
        <div className="grid grid-cols-1 items-stretch gap-3 lg:grid-cols-12">
          <PartnerWorkforceOverviewCard
            active={data?.active_employees ?? 0}
            dropped={data?.dropped_employees ?? 0}
            paid={data?.paid_employees ?? 0}
            isLoading={isLoading}
            className="h-full lg:col-span-4"
          />

          <ConsolePanel
            kicker="Chi trả"
            title="Top nhân viên"
            description={periodLabel}
            isFlush
            className="h-full lg:col-span-8"
            bodyClassName="flex flex-1 flex-col"
            actions={
              !isLoading && topEmployees.length > 0 ? (
                <span className="console-kicker">{topEmployees.length} nhân viên</span>
              ) : undefined
            }
          >
            <div
              className={cn(
                'px-3 py-2',
                !isLoading && topEmployees.length === 0 && 'flex flex-1 items-center justify-center',
              )}
            >
              {isLoading ? (
                <LeaderboardSkeleton />
              ) : topEmployees.length === 0 ? (
                <EmptyState
                showIllustration={false}
                  title="Chưa có dữ liệu thanh toán"
                  description={`Chưa có chi trả trong ${periodLabel}.`}
                  size="sm"
                />
              ) : (
                <div className="grid grid-cols-1 gap-x-4 gap-y-0.5 min-[1120px]:grid-cols-2">
                  {topEmployees.map((item, idx) => (
                    <LeaderRow key={item.employee_id} item={item} rank={idx + 1} />
                  ))}
                </div>
              )}
            </div>
          </ConsolePanel>
        </div>

        <PartnerEmployeeListSheet
          type={sheetType}
          month={apiMonth}
          onClose={closeSheet}
        />
      </div>
    </div>
  );
};

export default PartnerDashboardPage;
