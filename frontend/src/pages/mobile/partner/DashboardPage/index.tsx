import { useState, useMemo, useCallback } from 'react';
import { format, startOfMonth } from 'date-fns';
import { Users, UserCheck, UserX, Banknote, Trophy, BarChart3, ChevronLeft, ChevronRight } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
import { generateMonthOptions } from '@/utils/dateHelpers';
import { cn } from '@/lib/utils';
import type { TopPaidEmployeeItem, PartnerEmployeeListType } from '@/types/api/dashboard.types';

const ALL_VALUE = 'all';

const monthOptions = [
  { value: ALL_VALUE, label: 'Tất cả' },
  ...generateMonthOptions(12).map((o) => ({
    value: o.value,
    label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
  })),
];

function formatVND(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(1)}tỷ đ`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(0)}K đ`;
  return `${sign}${abs.toLocaleString('vi-VN')} đ`;
}

/* ───────────────────────────────────────────────
   MonthNavigator — clean ← Month Year → control
   ─────────────────────────────────────────────── */

function MonthNavigator({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const isAll = value === ALL_VALUE;
  const currentIdx = monthOptions.findIndex((o) => o.value === value);

  const goPrev = useCallback(() => {
    if (isAll) return;
    const idx = Math.max(1, currentIdx - 1); // skip past "Tất cả"
    onChange(monthOptions[idx].value);
  }, [currentIdx, isAll, onChange]);

  const goNext = useCallback(() => {
    if (isAll) return;
    const idx = Math.min(monthOptions.length - 1, currentIdx + 1);
    onChange(monthOptions[idx].value);
  }, [currentIdx, isAll, onChange]);

  const toggleAll = useCallback(() => {
    onChange(isAll ? monthOptions[1].value : ALL_VALUE);
  }, [isAll, onChange]);

  // Format display label: "T06/2026" → "Tháng 6, 2026"
  const displayLabel = useMemo(() => {
    if (isAll) return 'Tất cả';
    const opt = monthOptions.find((o) => o.value === value);
    return opt?.label ?? value;
  }, [value, isAll]);

  return (
    <div className="flex items-center justify-between gap-2">
      {/* Prev */}
      <button
        onClick={goPrev}
        disabled={isAll || currentIdx <= 1}
        className="flex h-9 w-9 items-center justify-center rounded-full bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors disabled:opacity-30 disabled:pointer-events-none shrink-0"
        aria-label="Tháng trước"
      >
        <ChevronLeft className="h-4 w-4" />
      </button>

      {/* Center: current month + "Tất cả" chip */}
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-sm font-semibold text-foreground truncate">
          {displayLabel}
        </span>
        <button
          onClick={toggleAll}
          className={cn(
            'px-2 py-0.5 rounded-full text-[10px] font-semibold transition-colors shrink-0',
            isAll
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground',
          )}
        >
          Tất cả
        </button>
      </div>

      {/* Next */}
      <button
        onClick={goNext}
        disabled={isAll || currentIdx >= monthOptions.length - 1}
        className="flex h-9 w-9 items-center justify-center rounded-full bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors disabled:opacity-30 disabled:pointer-events-none shrink-0"
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

const RANK_STYLES: Record<number, { badge: string; icon: string }> = {
  1: { badge: 'bg-amber-100 text-amber-700 border-amber-200', icon: 'text-amber-500' },
  2: { badge: 'bg-slate-100 text-slate-600 border-slate-200', icon: 'text-slate-400' },
  3: { badge: 'bg-orange-50 text-orange-600 border-orange-200', icon: 'text-orange-400' },
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
        {item.rank <= 3 ? (
          <Trophy className={cn('w-3 h-3', rank?.icon)} />
        ) : (
          <span>{item.rank}</span>
        )}
      </div>

      {/* Avatar */}
      <UserAvatar name={item.employee_name} size="sm" />

      {/* Name + progress */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium text-foreground truncate">{item.employee_name}</span>
          {item.is_active ? (
            <Badge variant="outline" className="text-[9px] px-1 py-0 h-4 border-emerald-300 text-emerald-700 bg-emerald-50/50 shrink-0">
              <UserCheck className="w-2.5 h-2.5 mr-0.5" />Đang làm
            </Badge>
          ) : (
            <Badge variant="outline" className="text-[9px] px-1 py-0 h-4 border-rose-200 text-rose-500 bg-rose-50/50 shrink-0">
              <UserX className="w-2.5 h-2.5 mr-0.5" />Nghỉ
            </Badge>
          )}
        </div>
        <div className="mt-1.5 h-1.5 rounded-full bg-muted/50 overflow-hidden">
          <div
            className="h-full rounded-full bg-gradient-to-r from-primary/40 to-primary/80 transition-all duration-700 ease-out"
            style={{ width: `${Math.max(pct, 2)}%` }}
          />
        </div>
      </div>

      {/* Amount */}
      <span className="text-[13px] font-semibold tabular-nums text-foreground shrink-0 pl-1">
        {formatVND(item.total_paid_vnd)}
      </span>
    </div>
  );
}

/* ───────────────────────────────────────────────
   PartnerDashboardMobile — main page
   ─────────────────────────────────────────────── */

const PartnerDashboardMobile = () => {
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

  return (
    <div className="flex flex-col gap-5 pb-24">
      {/* Header */}
      <MobilePageHeader
        title="Tổng quan"
        subtitle="Theo dõi nhân viên và thanh toán"
        icon={BarChart3}
        sticky={false}
        bordered={false}
      />

      {/* Month selector strip */}
      <div className="px-4">
        <MonthNavigator value={selectedMonth} onChange={handleMonthChange} />
      </div>

      {/* KPI Grid */}
      <div className="px-4">
        <div className="grid grid-cols-2 gap-3">
          {isLoading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="rounded-xl border border-border/40 bg-card p-4 space-y-2">
                <Skeleton className="h-7 w-7 rounded-lg" />
                <Skeleton className="h-6 w-14" />
                <Skeleton className="h-3 w-24" />
              </div>
            ))
          ) : (
            <>
              <KpiHeroCard label="Đang làm việc" value={data?.active_employees ?? 0} icon={UserCheck} color="emerald" sublabel="Có bảng công 14 ngày qua" onClick={() => openSheet('active')} />
              <KpiHeroCard label="Có thể nghỉ việc" value={data?.dropped_employees ?? 0} icon={UserX} color="amber" sublabel="Không bảng công 14 ngày qua" onClick={(data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined} />
              <KpiHeroCard label="Được trả lương" value={data?.paid_employees ?? 0} icon={Users} color="blue" sublabel={momEmployeesSublabel} onClick={(data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined} />
              <KpiHeroCard label="Tổng chi trả" value={data?.total_paid_vnd ?? 0} formattedValue={formatVND(data?.total_paid_vnd ?? 0)} icon={Banknote} color="violet" sublabel={momAmountSublabel} />
            </>
          )}
        </div>
      </div>

      {/* Top Employees Leaderboard */}
      <div className="px-4">
        <div className="flex items-center gap-2 mb-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/5 shrink-0">
            <Trophy className="h-3.5 w-3.5 text-primary/60" />
          </div>
          <h2 className="text-sm font-semibold text-foreground">Top nhân viên</h2>
          <span className="text-xs text-muted-foreground">— {periodLabel}</span>
        </div>

        <Card className="overflow-hidden border-border/40">
          <CardContent className="p-4">
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3 py-3 border-b border-border/30 last:border-0">
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
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted/50 mb-3">
                  <Trophy className="w-5 h-5 text-muted-foreground/30" />
                </div>
                <p className="text-sm font-medium text-muted-foreground">Chưa có dữ liệu thanh toán</p>
                <p className="text-xs text-muted-foreground/70 mt-1">Chọn tháng khác để xem</p>
              </div>
            ) : (
              topEmployees.map((item) => (
                <TopEmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
              ))
            )}
          </CardContent>
        </Card>
      </div>

      <PartnerEmployeeListSheet type={sheetType} month={apiMonth} onClose={closeSheet} />
    </div>
  );
};

export default PartnerDashboardMobile;
