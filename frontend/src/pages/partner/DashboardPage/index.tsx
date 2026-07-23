import { useState, useCallback, useMemo } from 'react';
import { format, startOfMonth } from 'date-fns';
import {
  Users,
  UserCheck,
  UserX,
  Banknote,
  CalendarDays,
  Crown,
  type LucideIcon,
} from 'lucide-react';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
import { PartnerWorkforceOverviewCard } from '@/components/partner-dashboard/PartnerWorkforceOverviewCard';
import { cn } from '@/lib/utils';
import type {
  TopPaidEmployeeItem,
  PartnerEmployeeListType,
  PartnerDashboardMoMChange,
} from '@/types/api/dashboard.types';
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
      className="flex flex-wrap items-center gap-1 rounded-xl bg-muted/60 p-1"
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
            'min-h-11 shrink-0 rounded-lg px-3 text-xs font-semibold transition-[background-color,color,box-shadow]',
            value === opt.value
              ? 'bg-card text-foreground shadow-sm ring-1 ring-border/60'
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
            'h-11 min-h-11 w-[132px] shrink-0 border-0 bg-transparent px-3 text-xs font-semibold shadow-none focus:ring-1 focus:ring-ring focus:ring-offset-0',
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

// ─── KPI stat row ──────────────────────────────────────────────────────────
// Stat cards compose the shared admin KpiHeroCard (also used by StatsCards,
// UserStatsCard, and ProjectStatusCard) rather than a partner-only clone, so
// the partner and admin dashboards stay visually and behaviorally in sync.
//
// The dashboard API exposes point-in-time values + a single MoM delta, not a
// daily/weekly series, so each card surfaces only the real MoM change — no
// fabricated sparkline or watermark that implies a trend the data can't support.
type PartnerStatColor = 'emerald' | 'teal' | 'amber' | 'blue';

interface PartnerStatProps {
  label: string;
  value: string | number;
  icon: LucideIcon;
  color: PartnerStatColor;
  change?: PartnerDashboardMoMChange;
  caption?: string;
  onClick?: () => void;
}

function PartnerStat({ label, value, icon, color, change, caption, onClick }: PartnerStatProps) {
  // Three honest MoM states mapped onto KpiHeroCard's trend/badge: up/down become
  // a trend pill; an exact 0% becomes a neutral badge. Nothing invents a delta.
  let trend: { value: string; positive: boolean } | undefined;
  let badge: { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral' } | undefined;
  if (change && Number.isFinite(change.change_pct)) {
    const pct = change.change_pct;
    if (pct === 0) {
      badge = { label: '0%', variant: 'neutral' };
    } else {
      const sign = pct > 0 ? '+' : '';
      trend = { value: `${sign}${pct.toFixed(0)}%`, positive: pct > 0 };
    }
  }

  return (
    <KpiHeroCard
      label={label}
      value={value}
      icon={icon}
      color={color}
      sublabel={caption}
      trend={trend}
      badge={badge}
      onClick={onClick}
      className="h-full"
    />
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
    <div className="group grid grid-cols-[2rem_2.25rem_minmax(0,1fr)] items-center gap-2 rounded-xl border border-transparent px-2 py-2 transition-colors hover:border-border/60 hover:bg-muted/30">
      <span className={cn('flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[11px] font-bold tabular-nums', badge)}>
        {rank}
      </span>
      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="truncate text-[13px] font-semibold text-foreground">{item.employee_name}</span>
          {!item.is_active && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[9px] font-semibold text-destructive shrink-0">
              <UserX className="h-2.5 w-2.5" />
              Nghỉ
            </span>
          )}
        </div>
        <span className="mt-0.5 block text-xs font-bold tabular-nums text-foreground">
          {formatVND(item.total_paid_vnd)}
        </span>
      </div>
    </div>
  );
}

// ─── Skeletons ─────────────────────────────────────────────────────────────
function StatRowSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="h-[140px] rounded-2xl border border-border/60 bg-card px-4 py-4 shadow-soft">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="mt-3 h-6 w-24" />
          <Skeleton className="mt-2 h-3 w-20" />
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
    <header className="rounded-2xl border border-border/60 bg-card px-5 py-4 shadow-soft sm:px-6 sm:py-5">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <div className="mb-2 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-primary">
            <CalendarDays className="h-4 w-4" />
            Kỳ báo cáo
          </div>
          <h1 className="font-display text-2xl font-extrabold tracking-tight text-foreground sm:text-[2rem]">
            Tổng quan
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Hoạt động nhân viên và tình hình thanh toán · <span className="font-semibold text-foreground">{periodLabel}</span>
          </p>
        </div>
        <div className="flex items-center lg:justify-end">
          <MonthSelector value={monthValue} onChange={onMonthChange} />
        </div>
      </div>
    </header>
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
    <div className="min-h-full p-4 lg:p-6">
      <div className="mx-auto max-w-[1400px] space-y-5">
        <BannerHeader
          periodLabel={periodLabel}
          monthValue={selectedMonth}
          onMonthChange={handleMonthChange}
        />

        {/* ── Stat row (4 cards) ── */}
        {isLoading ? (
          <StatRowSkeleton />
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4 animate-fade-in-up">
            <PartnerStat
              label="Tổng chi trả"
              value={formatVND(data?.total_paid_vnd ?? 0)}
              icon={Banknote}
              color="emerald"
              change={showMomBadges ? data?.mom_paid_amount : undefined}
              caption={!showMomBadges ? periodLabel : undefined}
              onClick={(data?.total_paid_vnd ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
            <PartnerStat
              label="Đang làm việc"
              value={data?.active_employees ?? 0}
              icon={UserCheck}
              color="teal"
              caption="bảng công 14 ngày qua"
              onClick={() => openSheet('active')}
            />
            <PartnerStat
              label="Có thể nghỉ"
              value={data?.dropped_employees ?? 0}
              icon={UserX}
              color="amber"
              caption="không bảng công 14 ngày"
              onClick={(data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined}
            />
            <PartnerStat
              label="Đã thanh toán"
              value={data?.paid_employees ?? 0}
              icon={Users}
              color="blue"
              change={showMomBadges ? data?.mom_paid_employees : undefined}
              caption={!showMomBadges ? 'tất cả thời gian' : undefined}
              onClick={(data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
          </div>
        )}

        {/* ── Analytics grid: workforce donut + leaderboard ── */}
        <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(280px,0.8fr)_minmax(0,2.2fr)]">
          {/* Workforce donut */}
          <PartnerWorkforceOverviewCard
            active={data?.active_employees ?? 0}
            dropped={data?.dropped_employees ?? 0}
            paid={data?.paid_employees ?? 0}
            isLoading={isLoading}
          />

          {/* Leaderboard */}
          <section className="rounded-2xl border border-border/60 bg-card p-5 shadow-soft">
            <div className="mb-4 flex items-start justify-between gap-3">
              <div className="flex items-start gap-2.5">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-warning/10 text-warning">
                  <Crown className="h-4 w-4" />
                </span>
                <div>
                  <h2 className="text-base font-bold text-foreground">Top nhân viên được trả lương</h2>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    Xếp hạng theo tổng chi trả {periodLabel}
                  </p>
                </div>
              </div>
              {!isLoading && topEmployees.length > 0 && (
                <span className="inline-flex items-center rounded-full bg-muted px-2.5 py-1 text-[11px] font-semibold text-muted-foreground">
                  {topEmployees.length} nhân viên
                </span>
              )}
            </div>

            {isLoading ? (
              <LeaderboardSkeleton />
            ) : topEmployees.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-muted/50 mb-3">
                  <Crown className="h-5 w-5 text-muted-foreground/40" />
                </div>
                <p className="text-[13px] font-medium text-foreground">Chưa có dữ liệu thanh toán</p>
                <p className="text-[11px] text-muted-foreground mt-1">
                  Sẽ hiển thị khi có nhân viên được thanh toán trong {periodLabel}.
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-x-4 gap-y-0.5 min-[1120px]:grid-cols-2">
                {topEmployees.map((item, idx) => (
                  <LeaderRow key={item.employee_id} item={item} rank={idx + 1} />
                ))}
              </div>
            )}
          </section>
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
