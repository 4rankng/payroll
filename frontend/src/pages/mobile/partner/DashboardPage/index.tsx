import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth } from 'date-fns';
import {
  Users,
  UserCheck,
  UserX,
  Banknote,
  BarChart3,
  ChevronLeft,
  ChevronRight,
  FolderKanban,
  ClipboardList,
  History,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import {
  MobileOperationsPanel,
  MobileTaskList,
  type MobileOperationAction,
  type MobileOperationMetric,
  type MobileTaskRow,
} from '@/components/shared/MobileOperationsPanel';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
import { generateMonthOptions } from '@/utils/dateHelpers';
import { cn } from '@/lib/utils';
import type { TopPaidEmployeeItem, PartnerEmployeeListType } from '@/types/api/dashboard.types';
import { formatCompactCurrency as formatVND } from '@/utils/formatters';

const ALL_VALUE = 'all';

/* ───────────────────────────────────────────────
   MonthNavigator — clean ← Month Year → control
   ─────────────────────────────────────────────── */

function MonthNavigator({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  // If value is "all", default back to current month for navigation
  const effectiveValue = value === ALL_VALUE ? format(startOfMonth(new Date()), 'yyyy-MM') : value;
  const monthOnlyOptions = generateMonthOptions(12).map((o) => ({
    value: o.value,
    label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
  }));
  const currentIdx = monthOnlyOptions.findIndex((o) => o.value === effectiveValue);

  const goPrev = useCallback(() => {
    const idx = Math.max(0, currentIdx - 1);
    onChange(monthOnlyOptions[idx].value);
  }, [currentIdx, monthOnlyOptions, onChange]);

  const goNext = useCallback(() => {
    const idx = Math.min(monthOnlyOptions.length - 1, currentIdx + 1);
    onChange(monthOnlyOptions[idx].value);
  }, [currentIdx, monthOnlyOptions, onChange]);

  const displayLabel = useMemo(() => {
    const opt = monthOnlyOptions.find((o) => o.value === effectiveValue);
    return opt?.label ?? effectiveValue;
  }, [effectiveValue, monthOnlyOptions]);

  return (
    <div className="flex items-center justify-between gap-2">
      <button
        onClick={goPrev}
        disabled={currentIdx <= 0}
        className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:bg-muted/80 hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
        aria-label="Tháng trước"
      >
        <ChevronLeft className="h-4 w-4" />
      </button>

      <span className="text-sm font-semibold text-foreground tabular-nums">
        {displayLabel}
      </span>

      <button
        onClick={goNext}
        disabled={currentIdx >= monthOnlyOptions.length - 1}
        className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:bg-muted/80 hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
        aria-label="Tháng sau"
      >
        <ChevronRight className="h-4 w-4" />
      </button>
    </div>
  );
}

/* ───────────────────────────────────────────────
   TopEmployeeRow — clean leaderboard row
   ─────────────────────────────────────────────── */

const RANK_STYLES: Record<number, { badge: string }> = {
  1: { badge: 'bg-warning/10 text-warning border-warning/30' },
  2: { badge: 'bg-muted text-muted-foreground border-border' },
  3: { badge: 'bg-primary/10 text-primary border-primary/30' },
};

function TopEmployeeRow({ item, maxPaid }: { item: TopPaidEmployeeItem; maxPaid: number }) {
  const pct = maxPaid > 0 ? (item.total_paid_vnd / maxPaid) * 100 : 0;
  const rank = RANK_STYLES[item.rank] ?? null;

  return (
    <div className="flex items-center gap-3 py-3 border-b border-border/30 last:border-0 group">
      {/* Rank badge */}
      <div
        className={cn(
          'flex items-center justify-center w-7 h-7 rounded-full text-[11px] font-bold shrink-0 border',
          rank?.badge ?? 'bg-muted/40 text-muted-foreground border-border/40',
        )}
      >
        {item.rank}
      </div>

      {/* Avatar */}
      <UserAvatar name={item.employee_name} size="sm" />

      {/* Name + progress */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium text-foreground truncate">{item.employee_name}</span>
          {item.is_active ? (
            <Badge variant="outline" className="h-4 shrink-0 border-success/30 bg-success/10 px-1 py-0 text-[9px] text-success">
              <UserCheck className="w-2.5 h-2.5 mr-0.5" />Đang làm
            </Badge>
          ) : (
            <Badge variant="outline" className="h-4 shrink-0 border-destructive/30 bg-destructive/10 px-1 py-0 text-[9px] text-destructive">
              <UserX className="w-2.5 h-2.5 mr-0.5" />Nghỉ
            </Badge>
          )}
        </div>
        <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-gradient-to-r from-primary/40 to-primary/80 transition-all duration-700 ease-out"
            style={{ width: `${Math.max(pct, 2)}%` }}
          />
        </div>
      </div>

      {/* Amount */}
      <span className="text-[13px] font-semibold tabular-nums text-foreground shrink-0 pl-1">
        {formatVND(item.total_paid_vnd, { useVietnamese: true })}
      </span>
    </div>
  );
}

/* ───────────────────────────────────────────────
   PartnerDashboardMobile — main page
   ─────────────────────────────────────────────── */

const PartnerDashboardMobile = () => {
  const navigate = useNavigate();
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM'),
  );
  const [sheetType, setSheetType] = useState<PartnerEmployeeListType | null>(null);

  const handleMonthChange = useCallback((v: string) => setSelectedMonth(v), []);
  const openSheet = useCallback((type: PartnerEmployeeListType) => setSheetType(type), []);
  const closeSheet = useCallback(() => setSheetType(null), []);

  const apiMonth = selectedMonth === ALL_VALUE ? undefined : selectedMonth;
  const { data, isLoading } = usePartnerDashboard({ month: apiMonth });

  const periodLabel = useMemo(() => {
    if (selectedMonth === ALL_VALUE) return 'Tất cả';
    const [y, m] = selectedMonth.split('-');
    return `T${m}/${y}`;
  }, [selectedMonth]);

  const topEmployees = useMemo(() => data?.top_paid_employees ?? [], [data?.top_paid_employees]);
  const maxPaid = topEmployees[0]?.total_paid_vnd ?? 0;

  const momEmployeesSublabel = useMemo(() => {
    if (!data?.mom_paid_employees || selectedMonth === ALL_VALUE) return periodLabel;
    const pct = data.mom_paid_employees.change_pct;
    return `${pct >= 0 ? '+' : ''}${pct.toFixed(0)}% so tháng trước`;
  }, [data?.mom_paid_employees, periodLabel, selectedMonth]);

  const momAmountSublabel = useMemo(() => {
    if (!data?.mom_paid_amount || selectedMonth === ALL_VALUE) return periodLabel;
    const pct = data.mom_paid_amount.change_pct;
    return `${pct >= 0 ? '+' : ''}${pct.toFixed(0)}% so tháng trước`;
  }, [data?.mom_paid_amount, periodLabel, selectedMonth]);

  const quickActions = useMemo<MobileOperationAction[]>(() => [
    {
      label: 'Dự án',
      icon: FolderKanban,
      onClick: () => navigate('/partner/projects'),
    },
    {
      label: 'Nhân viên',
      icon: Users,
      onClick: () => navigate('/partner/employees'),
    },
    {
      label: 'Bảng công',
      icon: ClipboardList,
      onClick: () => navigate('/partner/timesheet'),
    },
    {
      label: 'Lịch sử',
      icon: History,
      onClick: () => navigate('/partner/timesheet/payment-history'),
    },
  ], [navigate]);

  const operationMetrics = useMemo<MobileOperationMetric[]>(() => [
    {
      label: 'Đang làm việc',
      value: (data?.active_employees ?? 0).toLocaleString('vi-VN'),
      helper: 'Có bảng công 14 ngày qua',
      icon: UserCheck,
      tone: 'success',
      onClick: () => openSheet('active'),
    },
    {
      label: 'Cần kiểm tra',
      value: (data?.dropped_employees ?? 0).toLocaleString('vi-VN'),
      helper: 'Không bảng công 14 ngày qua',
      icon: UserX,
      tone: (data?.dropped_employees ?? 0) > 0 ? 'warning' : 'neutral',
      onClick: (data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined,
    },
    {
      label: 'Được trả lương',
      value: (data?.paid_employees ?? 0).toLocaleString('vi-VN'),
      helper: momEmployeesSublabel,
      icon: Users,
      tone: 'primary',
      onClick: (data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined,
    },
    {
      label: 'Chi trả bình quân',
      value: formatVND(
        (data?.paid_employees ?? 0) > 0
          ? Math.round((data?.total_paid_vnd ?? 0) / (data?.paid_employees ?? 1))
          : 0,
        { useVietnamese: true },
      ),
      helper: periodLabel,
      icon: Banknote,
      tone: 'neutral',
    },
  ], [data, momEmployeesSublabel, openSheet, periodLabel]);

  const taskRows = useMemo<MobileTaskRow[]>(() => [
    {
      title: 'Nhân viên cần kiểm tra',
      description: 'Không có bảng công trong 14 ngày gần nhất',
      value: (data?.dropped_employees ?? 0).toLocaleString('vi-VN'),
      icon: UserX,
      tone: (data?.dropped_employees ?? 0) > 0 ? 'warning' : 'success',
      onClick: (data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined,
    },
    {
      title: 'Bảng công tháng này',
      description: 'Mở danh sách bảng công để rà soát theo dự án',
      value: periodLabel,
      icon: ClipboardList,
      tone: 'primary',
      onClick: () => navigate('/partner/timesheet'),
    },
    {
      title: 'Lịch sử thanh toán',
      description: 'Kiểm tra các kỳ đã chi trả cho nhân sự',
      value: 'Xem',
      icon: History,
      tone: 'neutral',
      onClick: () => navigate('/partner/timesheet/payment-history'),
    },
  ], [data?.dropped_employees, navigate, openSheet, periodLabel]);

  return (
    <MobilePageShell className="space-y-4">
      <MobilePageHeader
        title="Tổng quan"
        subtitle="Theo dõi nhân viên và thanh toán"
        icon={BarChart3}
        sticky={false}
        bordered={false}
      />

      <MobileSurface className="p-3">
        <MonthNavigator value={selectedMonth} onChange={handleMonthChange} />
      </MobileSurface>

      {isLoading ? (
        <MobileSurface className="p-4">
          <div className="space-y-3">
            <Skeleton className="h-7 w-36" />
            <Skeleton className="h-8 w-44" />
            <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-[76px] rounded-2xl" />
              ))}
            </div>
          </div>
        </MobileSurface>
      ) : (
        <MobileOperationsPanel
          eyebrow={periodLabel}
          title="Theo dõi chi trả"
          subtitle="Tình hình nhân sự và lương theo kỳ đang xem"
          primaryLabel="Tổng chi trả"
          primaryValue={formatVND(data?.total_paid_vnd ?? 0, { useVietnamese: true })}
          primaryHint={momAmountSublabel}
          metrics={operationMetrics}
          actions={quickActions}
        />
      )}

      <MobileTaskList
        title="Việc cần theo dõi"
        subtitle="Những mục ảnh hưởng đến bảng công và thanh toán"
        items={taskRows}
      />

      <MobileSurface className="p-4">
        <div className="mb-3 flex items-center gap-2">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10">
            <Users className="h-4 w-4 text-primary" />
          </div>
          <h2 className="text-sm font-semibold text-foreground">Chi trả theo nhân viên</h2>
          <span className="text-xs text-muted-foreground">— {periodLabel}</span>
        </div>

        <div className="divide-y divide-border/60">
          {isLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-3">
                <Skeleton className="w-7 h-7 rounded-full" />
                <Skeleton className="w-8 h-8 rounded-full" />
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-1.5 w-full rounded-full" />
                </div>
                <Skeleton className="h-3.5 w-16" />
              </div>
            ))
          ) : topEmployees.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
                <Users className="w-5 h-5 text-muted-foreground/40" />
              </div>
              <p className="text-sm font-medium text-muted-foreground">Chưa có dữ liệu thanh toán</p>
              <p className="text-xs text-muted-foreground/70 mt-1">Chọn tháng khác để xem</p>
            </div>
          ) : (
            topEmployees.map((item) => (
              <TopEmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
            ))
          )}
        </div>
      </MobileSurface>

      <PartnerEmployeeListSheet type={sheetType} month={apiMonth} onClose={closeSheet} />
    </MobilePageShell>
  );
};

export default PartnerDashboardMobile;
