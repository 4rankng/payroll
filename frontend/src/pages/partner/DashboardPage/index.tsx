import { useState, useCallback, useMemo } from 'react';
import { format, startOfMonth } from 'date-fns';
import {
  Users,
  UserCheck,
  UserX,
  Banknote,
  CalendarDays,
  Crown,
  Sparkles,
  ArrowUpRight,
  ArrowDownRight,
  Minus,
} from 'lucide-react';
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

// ─── Trend chip ────────────────────────────────────────────────────────────
function TrendChip({ change }: { change: PartnerDashboardMoMChange | undefined }) {
  if (!change || !Number.isFinite(change.change_pct)) return null;
  const pct = change.change_pct;
  const isUp = pct > 0;
  const isFlat = pct === 0;
  const Icon = isFlat ? Minus : isUp ? ArrowUpRight : ArrowDownRight;
  const sign = pct >= 0 ? '+' : '';
  const label = `${sign}${pct.toFixed(0)}%`;
  return (
    <span
      className={cn(
        'inline-flex items-center gap-0.5 text-xs font-semibold',
        isUp && 'text-success',
        !isUp && !isFlat && 'text-destructive',
        isFlat && 'text-muted-foreground',
      )}
    >
      <Icon className="h-3 w-3" />
      {label}
    </span>
  );
}

// ─── Decorative sparkline (NOT backed by real time-series data) ────────────
// Visual reference only. The dashboard API exposes point-in-time values + a
// single MoM delta — not a daily/weekly series. These SVGs are static curves
// tinted by `currentColor` so they inherit the card's semantic tone.
// Marked aria-hidden so screen readers don't announce fake data.
const SPARK_PATHS = [
  'M0 602.49c80-41.832 240-108.916 400-209.159C560 293.09 640 21.354 800 101.278c160 79.923 240 652.286 400 691.67 160 39.384 240-473.81 400-494.752 160-20.943 320 312.03 400 390.038',
  'M0 623.854c80-59.416 240-298.176 400-297.076 160 1.1 240 343.78 400 302.574 160-41.206 240-415.29 400-508.605 160-93.314 240-22.06 400 42.033 160 64.095 320 222.751 400 278.439',
  'M0 449.109c80 24.34 240 157.918 400 121.701 160-36.216 240-233.648 400-302.785 160-69.137 240-128.846 400-42.9s240 420.493 400 472.63 320-169.555 400-211.943',
  'M0 131.28c80 52.463 240 237.611 400 262.315 160 24.703 240-137.143 400-138.8 160-1.656 240 25.982 400 130.517s240 438.49 400 392.16c160-46.33 320-499.048 400-623.81',
];
const SPARK_FILL_PATHS = [
  'M0 602.49c80-41.832 240-108.916 400-209.159C560 293.09 640 21.354 800 101.278c160 79.923 240 652.286 400 691.67 160 39.384 240-473.81 400-494.752 160-20.943 320 312.03 400 390.038V1000H0Z',
  'M0 623.854c80-59.416 240-298.176 400-297.076 160 1.1 240 343.78 400 302.574 160-41.206 240-415.29 400-508.605 160-93.314 240-22.06 400 42.033 160 64.095 320 222.751 400 278.439V1000H0Z',
  'M0 449.109c80 24.34 240 157.918 400 121.701 160-36.216 240-233.648 400-302.785 160-69.137 240-128.846 400-42.9s240 420.493 400 472.63 320-169.555 400-211.943V1000H0Z',
  'M0 131.28c80 52.463 240 237.611 400 262.315 160 24.703 240-137.143 400-138.8 160-1.656 240 25.982 400 130.517s240 438.49 400 392.16c160-46.33 320-499.048 400-623.81V1000H0Z',
];

function Sparkline({ index, className }: { index: number; className?: string }) {
  return (
    <svg
      aria-hidden="true"
      className={cn('w-full max-w-[112px] h-auto', className)}
      viewBox="0 0 2000 1000"
      preserveAspectRatio="none"
    >
      <path d={SPARK_FILL_PATHS[index]} fill="currentColor" opacity={0.12} />
      <path
        d={SPARK_PATHS[index]}
        fill="none"
        stroke="currentColor"
        strokeWidth={28}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

// ─── Stat card (a-c-statistics-11 inspired) ────────────────────────────────
type Tone = 'primary' | 'success' | 'warning' | 'info';

const TONE_TEXT: Record<Tone, string> = {
  primary: 'text-primary',
  success: 'text-success',
  warning: 'text-warning',
  info: 'text-info',
};

interface StatCardProps {
  label: string;
  value: string;
  icon: typeof Users;
  tone: Tone;
  sparkIndex: number;
  change?: PartnerDashboardMoMChange;
  caption?: string;
  onClick?: () => void;
}

function StatCard({ label, value, icon: Icon, tone, sparkIndex, change, caption, onClick }: StatCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      aria-label={`${label}: ${value}`}
      className={cn(
        'group relative flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-card p-4 text-left transition-all',
        'shadow-soft hover:shadow-card hover:-translate-y-0.5',
        onClick && 'cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        !onClick && 'cursor-default',
      )}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <Icon className={cn('h-3.5 w-3.5 shrink-0', TONE_TEXT[tone])} strokeWidth={2.2} />
          <span className="text-[10px] font-bold uppercase tracking-[0.08em] text-muted-foreground truncate">
            {label}
          </span>
        </div>
        <p className="mt-2 font-display text-2xl font-extrabold tabular-nums tracking-tight text-foreground leading-none">
          {value}
        </p>
        <div className="mt-2 flex items-center gap-2">
          {change ? <TrendChip change={change} /> : null}
          {caption && (
            <span className="text-[11px] text-muted-foreground truncate">{caption}</span>
          )}
        </div>
      </div>
      {/* Sparkline — decorative, inherits tone color via currentColor */}
      <div className={cn('relative w-[112px] shrink-0 self-stretch flex items-end', TONE_TEXT[tone])}>
        <div className="absolute inset-0 bg-linear-to-t from-card via-card/0 to-transparent pointer-events-none" />
        <Sparkline index={sparkIndex} className="relative" />
      </div>
    </button>
  );
}

// ─── Leaderboard row (simplified, single-tier) ─────────────────────────────
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
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-card p-4">
          <div className="flex-1 space-y-2">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-3 w-16" />
          </div>
          <Skeleton className="h-12 w-24 rounded" />
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

// ─── Banner header (a-c-page-headings-06 inspired, brand emerald only) ─────
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
      {/* Faint grid pattern overlay */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-[0.04] [background-image:linear-gradient(90deg,hsl(var(--foreground))_1px,transparent_1px),linear-gradient(0deg,hsl(var(--foreground))_1px,transparent_1px)] [background-size:24px_24px]"
      />
      <div className="relative flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
        <div className="min-w-0">
          <div className="inline-flex items-center gap-1.5 rounded-full bg-primary/10 px-2.5 py-1 text-[10.5px] font-bold uppercase tracking-[0.08em] text-primary ring-1 ring-inset ring-primary/20">
            <Sparkles className="h-3.5 w-3.5" />
            Bảng điều hành
          </div>
          <h1 className="mt-2 font-display text-2xl font-extrabold tracking-tight text-foreground sm:text-3xl">
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

// ─── Quick actions strip (mirrors mobile quickActions) ─────────────────────
// Skipped per plan deferral: the 4-card stat row already provides the primary
// navigation via onClick. Adding a separate quick-actions strip would duplicate.

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
            <StatCard
              label="Tổng chi trả"
              value={formatVND(data?.total_paid_vnd ?? 0)}
              icon={Banknote}
              tone="primary"
              sparkIndex={0}
              change={showMomBadges ? data?.mom_paid_amount : undefined}
              caption={!showMomBadges ? periodLabel : undefined}
              onClick={(data?.total_paid_vnd ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
            <StatCard
              label="Đang làm việc"
              value={(data?.active_employees ?? 0).toLocaleString('vi-VN')}
              icon={UserCheck}
              tone="success"
              sparkIndex={1}
              caption="bảng công 14 ngày qua"
              onClick={() => openSheet('active')}
            />
            <StatCard
              label="Có thể nghỉ"
              value={(data?.dropped_employees ?? 0).toLocaleString('vi-VN')}
              icon={UserX}
              tone="warning"
              sparkIndex={2}
              caption="không bảng công 14 ngày"
              onClick={(data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined}
            />
            <StatCard
              label="Đã thanh toán"
              value={(data?.paid_employees ?? 0).toLocaleString('vi-VN')}
              icon={Users}
              tone="info"
              sparkIndex={3}
              change={showMomBadges ? data?.mom_paid_employees : undefined}
              caption={!showMomBadges ? 'tất cả thời gian' : undefined}
              onClick={(data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
          </div>
        )}

        {/* ── Analytics grid: workforce donut + leaderboard ── */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Workforce donut — preserved unchanged */}
          <PartnerWorkforceOverviewCard
            active={data?.active_employees ?? 0}
            dropped={data?.dropped_employees ?? 0}
            paid={data?.paid_employees ?? 0}
            isLoading={isLoading}
          />

          {/* Leaderboard — simplified (no podium) */}
          <div className="md:col-span-2 rounded-2xl border border-border/60 bg-card p-5 shadow-soft">
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
