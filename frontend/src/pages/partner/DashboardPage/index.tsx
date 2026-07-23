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

// ─── Month selector ────────────────────────────────────────────────────────
function MonthSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="flex items-center gap-1 overflow-x-auto rounded-2xl border border-border/60 bg-card/80 p-1 no-scrollbar" aria-label="Chọn kỳ dữ liệu">
      {monthOptions.slice(0, 4).map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          aria-pressed={value === opt.value}
          className={cn(
            'min-h-[36px] px-3 rounded-xl text-xs font-semibold transition-colors shrink-0',
            value === opt.value
              ? 'bg-primary text-primary-foreground shadow-soft'
              : 'text-muted-foreground hover:bg-muted hover:text-foreground',
          )}
        >
          {opt.label}
        </button>
      ))}
      <select
        aria-label="Chọn tháng khác"
        value={monthOptions.slice(4).some((o) => o.value === value) ? value : ''}
        onChange={(e) => e.target.value && onChange(e.target.value)}
        className="min-h-[36px] rounded-xl border border-transparent bg-transparent px-2.5 text-xs font-semibold text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus:border-primary/30 shrink-0"
      >
        <option value="">Tháng khác…</option>
        {monthOptions.slice(4).map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
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

function LeaderRow({ item, rank, maxPaid }: { item: TopPaidEmployeeItem; rank: number; maxPaid: number }) {
  const pct = maxPaid > 0 ? Math.round((item.total_paid_vnd / maxPaid) * 100) : 0;
  const badge = RANK_BADGE[rank - 1] ?? 'bg-muted/60 text-muted-foreground ring-1 ring-border/60';
  return (
    <div className="group flex items-center gap-3 rounded-xl px-2 py-2.5 transition-colors hover:bg-muted/40">
      <span className={cn('flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[11px] font-bold tabular-nums', badge)}>
        {rank}
      </span>
      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-[13px] font-semibold text-foreground truncate">{item.employee_name}</span>
          {!item.is_active && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[9px] font-semibold text-destructive shrink-0">
              <UserX className="h-2.5 w-2.5" />
              Nghỉ
            </span>
          )}
        </div>
        <div className="mt-1.5 flex items-center gap-2">
          <div className="h-1 flex-1 rounded-full bg-muted/60 overflow-hidden max-w-[140px]">
            <div
              className="h-full rounded-full bg-gradient-to-r from-primary/60 to-primary transition-all duration-500"
              style={{ width: `${Math.max(pct, 2)}%` }}
            />
          </div>
          <span className="text-[10px] text-muted-foreground tabular-nums">{pct}%</span>
        </div>
      </div>
      <span className="text-[13px] font-bold tabular-nums text-foreground shrink-0">
        {formatVND(item.total_paid_vnd)}
      </span>
    </div>
  );
}

// ─── Skeletons ─────────────────────────────────────────────────────────────
function StatRowSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="h-[92px] rounded-xl border border-border/40 bg-card px-4 py-3 shadow-soft">
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
    <div className="relative overflow-hidden rounded-2xl border border-primary/15 bg-linear-to-br from-primary/15 via-primary/5 to-transparent">
      <div className="relative flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
        <div className="min-w-0">
          <h1 className="font-display text-2xl font-extrabold tracking-tight text-foreground sm:text-3xl">
            Tổng quan
          </h1>
          <p className="mt-1 text-[13px] text-muted-foreground">
            Theo dõi hoạt động nhân viên và tình hình thanh toán · <span className="font-medium text-foreground/80">{periodLabel}</span>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <CalendarDays className="hidden h-4 w-4 text-muted-foreground sm:block" />
          <MonthSelector value={monthValue} onChange={onMonthChange} />
        </div>
      </div>
    </div>
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
  const maxPaid = topEmployees[0]?.total_paid_vnd ?? 0;

  const showMomBadges = selectedMonth !== ALL_VALUE;

  return (
    <div className="min-h-full p-4 lg:p-6">
      <div className="mx-auto max-w-[1320px] space-y-5">
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
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Workforce donut */}
          <PartnerWorkforceOverviewCard
            active={data?.active_employees ?? 0}
            dropped={data?.dropped_employees ?? 0}
            paid={data?.paid_employees ?? 0}
            isLoading={isLoading}
          />

          {/* Leaderboard */}
          <div className="md:col-span-2 rounded-2xl border border-border/40 bg-card p-5 shadow-soft">
            <div className="mb-4 flex items-start justify-between gap-3">
              <div className="flex items-start gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning ring-1 ring-warning/20">
                  <Crown className="h-4 w-4" />
                </span>
                <div>
                  <h3 className="text-sm font-bold text-foreground">Top nhân viên được trả lương</h3>
                  <p className="text-[11px] text-muted-foreground mt-0.5">
                    Xếp hạng theo tổng chi trả {periodLabel}
                  </p>
                </div>
              </div>
              {!isLoading && topEmployees.length > 0 && (
                <span className="inline-flex items-center rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-semibold text-muted-foreground">
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
              <div className="space-y-0.5">
                {topEmployees.map((item, idx) => (
                  <LeaderRow key={item.employee_id} item={item} rank={idx + 1} maxPaid={maxPaid} />
                ))}
              </div>
            )}
          </div>
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
