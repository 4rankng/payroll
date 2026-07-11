import { useState, useCallback, useMemo } from 'react';
import { format, startOfMonth } from 'date-fns';
import {
  Users,
  UserCheck,
  UserX,
  Banknote,
  ArrowUpRight,
  ArrowDownRight,
  Minus,
  Crown,
  Medal,
  Award,
  Sparkles,
} from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { PageHeader } from '@/components/shared/PageHeader';
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
    <div className="flex items-center gap-1 overflow-x-auto no-scrollbar pb-0.5 -mb-0.5" aria-label="Chọn kỳ dữ liệu">
      {monthOptions.slice(0, 4).map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          aria-pressed={value === opt.value}
          className={cn(
            'min-h-[44px] px-3 rounded-full text-xs font-semibold transition-colors shrink-0',
            value === opt.value
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
          )}
        >
          {opt.label}
        </button>
      ))}
      <select
        aria-label="Chọn tháng khác"
        value={monthOptions.slice(4).some((o) => o.value === value) ? value : ''}
        onChange={(e) => e.target.value && onChange(e.target.value)}
        className="min-h-[44px] rounded-full border border-transparent bg-transparent px-2.5 text-xs font-semibold text-muted-foreground outline-none transition-colors hover:bg-muted/60 hover:text-foreground focus:border-primary/30 shrink-0"
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

// ─── Trend chip (used in hero + tiles) ─────────────────────────────────────
function TrendChip({
  change,
  variant = 'on-dark',
}: {
  change: PartnerDashboardMoMChange | undefined;
  variant?: 'on-dark' | 'on-light';
}) {
  if (!change || !Number.isFinite(change.change_pct)) return null;
  const pct = change.change_pct;
  const isUp = pct > 0;
  const isFlat = pct === 0;
  const Icon = isFlat ? Minus : isUp ? ArrowUpRight : ArrowDownRight;
  const sign = pct >= 0 ? '+' : '';
  const label = `${sign}${pct.toFixed(0)}%`;

  if (variant === 'on-dark') {
    return (
      <span
        className={cn(
          'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold backdrop-blur-sm',
          isUp && 'bg-emerald-300/20 text-emerald-100',
          !isUp && !isFlat && 'bg-rose-300/20 text-rose-100',
          isFlat && 'bg-white/10 text-white/80',
        )}
      >
        <Icon className="h-3 w-3" />
        {label}
      </span>
    );
  }
  return (
    <span
      className={cn(
        'inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-semibold',
        isUp && 'bg-emerald-50 text-emerald-700',
        !isUp && !isFlat && 'bg-rose-50 text-rose-700',
        isFlat && 'bg-slate-100 text-slate-600',
      )}
    >
      <Icon className="h-2.5 w-2.5" />
      {label}
    </span>
  );
}

// ─── Compact stat tile (used in hero side column) ───────────────────────────
// Watermark style: small inline icon-label at top-left, big number below,
// and a LARGE faint outline icon as decorative art bleeding off the right
// edge of the card. Inspired by user-provided reference (Stripe/Linear feel).
const TILE_COLORS = {
  emerald: { iconText: 'text-emerald-600', watermark: 'text-emerald-500/15' },
  amber:   { iconText: 'text-amber-600',   watermark: 'text-amber-500/15' },
  blue:    { iconText: 'text-teal-600',    watermark: 'text-teal-500/15' },
} as const;

function StatTile({
  label,
  value,
  icon: Icon,
  color,
  sublabel,
  change,
  onClick,
}: {
  label: string;
  value: number;
  icon: typeof UserCheck;
  color: keyof typeof TILE_COLORS;
  sublabel?: string;
  change?: PartnerDashboardMoMChange;
  onClick?: () => void;
}) {
  const c = TILE_COLORS[color];
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      className={cn(
        'group relative w-full min-h-[88px] text-left rounded-2xl border border-border/60 bg-card px-4 py-4 overflow-hidden transition-all',
        'shadow-soft',
        onClick &&
          'hover:shadow-card hover:-translate-y-0.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        !onClick && 'cursor-default',
      )}
    >
      {/* Watermark decoration — large faint icon fully inside the card on the
          right side. Sits behind content; pointer-events-none keeps it from
          blocking clicks. */}
      <Icon
        className={cn(
          'absolute right-3 top-1/2 -translate-y-1/2 h-16 w-16 pointer-events-none',
          'transition-transform duration-300 group-hover:scale-105',
          c.watermark,
        )}
        strokeWidth={1.5}
      />

      <div className="relative">
        {/* Label row with small icon prefix */}
        <div className="flex items-center gap-1.5">
          <Icon className={cn('h-3 w-3', c.iconText)} strokeWidth={2.2} />
          <span className="text-[10px] font-bold uppercase tracking-normal text-muted-foreground">
            {label}
          </span>
        </div>

        {/* Big number */}
        <p className="mt-1.5 text-2xl font-extrabold tabular-nums leading-none text-foreground tracking-normal">
          {value.toLocaleString('vi-VN')}
        </p>

        {/* Trend / sublabel */}
        {(change || sublabel) && (
          <div className="mt-2 flex items-center gap-1.5">
            {change ? <TrendChip change={change} variant="on-light" /> : null}
            {sublabel && (
              <span className="text-[10.5px] text-muted-foreground leading-tight">{sublabel}</span>
            )}
          </div>
        )}
      </div>
    </button>
  );
}

// ─── Top employees leaderboard ────────────────────────────────────────────
const PODIUM_DECOR = [
  { icon: Crown, color: 'text-amber-500', ring: 'ring-amber-200', accent: 'bg-amber-100 text-amber-700' },
  { icon: Medal, color: 'text-slate-400', ring: 'ring-slate-200', accent: 'bg-slate-100 text-slate-700' },
  { icon: Award, color: 'text-amber-700/70', ring: 'ring-orange-200', accent: 'bg-orange-100 text-orange-700' },
];

function PodiumCard({ item, maxPaid }: { item: TopPaidEmployeeItem; maxPaid: number }) {
  const decor = PODIUM_DECOR[item.rank - 1];
  const Icon = decor.icon;
  const pct = maxPaid > 0 ? Math.round((item.total_paid_vnd / maxPaid) * 100) : 0;
  return (
    <div className="relative flex flex-col items-center gap-2 rounded-2xl bg-gradient-to-b from-card to-muted/30 px-3 py-4 transition-all hover:from-card hover:to-muted/50">
      <div className="absolute top-2 right-2">
        <Icon className={cn('h-4 w-4', decor.color)} />
      </div>
      <UserAvatar
        name={item.employee_name}
        size="md"
        className={cn('ring-2 ring-offset-2 ring-offset-card shrink-0', decor.ring)}
      />
      <div className="text-center min-w-0 w-full">
        <p className="text-[12px] font-semibold text-foreground truncate leading-tight" title={item.employee_name}>
          {item.employee_name}
        </p>
        <p className="text-[14px] font-extrabold tabular-nums text-foreground leading-tight mt-0.5">
          {formatVND(item.total_paid_vnd)}
        </p>
        <div
          className={cn(
            'inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 mt-1.5 text-[9.5px] font-semibold',
            decor.accent,
          )}
        >
          {pct}% top
        </div>
      </div>
    </div>
  );
}

function LeaderRow({ item, maxPaid }: { item: TopPaidEmployeeItem; maxPaid: number }) {
  const pct = maxPaid > 0 ? Math.round((item.total_paid_vnd / maxPaid) * 100) : 0;
  return (
    <div className="group flex items-center gap-3 py-2.5 px-2 -mx-2 rounded-lg transition-colors hover:bg-muted/40">
      <span className="w-6 text-center text-[11px] font-bold tabular-nums text-muted-foreground">
        #{item.rank}
      </span>
      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-[13px] font-semibold text-foreground truncate">{item.employee_name}</span>
          {!item.is_active && (
            <span className="inline-flex items-center gap-0.5 rounded-full bg-rose-50 px-1.5 py-0.5 text-[9px] font-semibold text-rose-700 shrink-0">
              <UserX className="h-2.5 w-2.5" />
              Nghỉ
            </span>
          )}
        </div>
        <div className="mt-1 flex items-center gap-2">
          <div className="h-1 flex-1 rounded-full bg-muted/50 overflow-hidden max-w-[120px]">
            <div
              className="h-full rounded-full bg-gradient-to-r from-violet-400 to-violet-500"
              style={{ width: `${pct}%` }}
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

// ─── Skeletons ────────────────────────────────────────────────────────────
function HeroSkeleton() {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
      <div className="lg:col-span-3 rounded-2xl bg-muted/30 p-6 lg:p-8 space-y-3">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-12 w-48" />
        <Skeleton className="h-4 w-36" />
      </div>
      <div className="lg:col-span-2 space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[88px] w-full rounded-2xl" />
        ))}
      </div>
    </div>
  );
}

function LeaderboardSkeleton() {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[120px] rounded-2xl" />
        ))}
      </div>
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded-lg" />
        ))}
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────
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
  const podiumItems = topEmployees.slice(0, 3);
  const restItems = topEmployees.slice(3);

  const showMomBadges = selectedMonth !== ALL_VALUE;

  return (
    <div className="p-4 lg:p-8 max-w-[1320px] mx-auto space-y-5">
      <div className="rounded-2xl border border-white/80 bg-white/85 px-5 py-4 shadow-[0_18px_50px_-40px_rgba(8,120,62,0.28)] backdrop-blur">
        <PageHeader
          title="Tổng quan"
          description="Theo dõi hoạt động nhân viên và tình hình thanh toán"
        >
          <MonthSelector value={selectedMonth} onChange={handleMonthChange} />
        </PageHeader>
      </div>

      {/* ── HERO SECTION ── */}
      {isLoading ? (
        <HeroSkeleton />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-4 animate-fade-in-up">
          {/* Left: big payout hero card */}
          <div className="lg:col-span-3 relative overflow-hidden rounded-2xl border border-emerald-200/15 p-6 lg:p-8 text-white shadow-[0_22px_54px_-32px_rgba(6,69,46,0.58)]">
            <div className="absolute inset-0 bg-[linear-gradient(135deg,#06452E_0%,#08783E_62%,#15905A_100%)]" />
            <div className="absolute -right-20 -top-24 h-72 w-72 rounded-full bg-emerald-300/10 blur-3xl" />
            <div className="absolute inset-0 opacity-[0.08] pointer-events-none [background-image:linear-gradient(90deg,white_1px,transparent_1px),linear-gradient(0deg,white_1px,transparent_1px)] [background-size:28px_28px]" />

            <div className="relative">
              <div className="flex items-center gap-1.5">
                <Sparkles className="h-3.5 w-3.5 text-emerald-200" />
                <span className="text-[10.5px] font-bold uppercase tracking-normal text-white/75">
                  Tổng chi trả · {periodLabel}
                </span>
              </div>
              <div className="mt-3 flex items-baseline gap-2.5 flex-wrap">
                <span className="font-display text-4xl font-extrabold tabular-nums tracking-normal leading-none sm:text-5xl">
                  {formatVND(data?.total_paid_vnd ?? 0)}
                </span>
                {showMomBadges && <TrendChip change={data?.mom_paid_amount} variant="on-dark" />}
              </div>
              <p className="mt-3 max-w-2xl text-[13px] text-white/75 leading-relaxed">
                Đã thanh toán cho{' '}
                <span className="font-semibold text-white">{data?.paid_employees ?? 0}</span> nhân viên trong{' '}
                {periodLabel}.{' '}
                {data?.mom_paid_employees && showMomBadges && (
                  <>
                    {data.mom_paid_employees.change_amount >= 0 ? 'Tăng' : 'Giảm'}{' '}
                    <span className="font-semibold text-white">
                      {Math.abs(data.mom_paid_employees.change_amount)}
                    </span>{' '}
                    người so với tháng trước.
                  </>
                )}
              </p>

              <div className="mt-5 flex items-center gap-4 text-[11.5px]">
                <div className="flex items-center gap-1.5">
                  <Banknote className="h-3.5 w-3.5 text-emerald-200" />
                  <span className="text-white/70">Cập nhật theo thời gian thực</span>
                </div>
              </div>
            </div>
          </div>

          {/* Right: 3 stat tiles stacked */}
          <div className="lg:col-span-2 grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-1 gap-3">
            <StatTile
              label="Đang làm việc"
              value={data?.active_employees ?? 0}
              icon={UserCheck}
              color="emerald"
              sublabel="bảng công 14 ngày qua"
              onClick={() => openSheet('active')}
            />
            <StatTile
              label="Có thể nghỉ"
              value={data?.dropped_employees ?? 0}
              icon={UserX}
              color="amber"
              sublabel="không bảng công 14 ngày"
              onClick={(data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined}
            />
            <StatTile
              label="Đã thanh toán"
              value={data?.paid_employees ?? 0}
              icon={Users}
              color="blue"
              change={showMomBadges ? data?.mom_paid_employees : undefined}
              sublabel={!showMomBadges ? 'tất cả thời gian' : undefined}
              onClick={(data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined}
            />
          </div>
        </div>
      )}

      {/* ── ANALYTICS GRID: workforce donut + leaderboard ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Workforce donut */}
        <PartnerWorkforceOverviewCard
          active={data?.active_employees ?? 0}
          dropped={data?.dropped_employees ?? 0}
          paid={data?.paid_employees ?? 0}
          isLoading={isLoading}
        />

        {/* Top employees leaderboard */}
        <div className="lg:col-span-2 rounded-2xl border border-border/60 bg-card p-5 shadow-soft">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-foreground flex items-center gap-1.5">
                <Crown className="h-3.5 w-3.5 text-amber-500" />
                Top nhân viên được trả lương cao nhất
              </h3>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                Xếp hạng theo tổng chi trả {periodLabel}
              </p>
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
            <div className="space-y-4">
              {/* Podium: top 3 special cards */}
              {podiumItems.length > 0 && (
                <div
                  className={cn(
                    'grid gap-2',
                    podiumItems.length === 1 && 'grid-cols-1',
                    podiumItems.length === 2 && 'grid-cols-2',
                    podiumItems.length === 3 && 'grid-cols-3',
                  )}
                >
                  {podiumItems.map((item) => (
                    <PodiumCard key={item.employee_id} item={item} maxPaid={maxPaid} />
                  ))}
                </div>
              )}

              {/* Rest: compact rows */}
              {restItems.length > 0 && (
                <div className="space-y-0.5 pt-2 border-t border-border/40">
                  {restItems.map((item) => (
                    <LeaderRow key={item.employee_id} item={item} maxPaid={maxPaid} />
                  ))}
                </div>
              )}
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
  );
};

export default PartnerDashboardPage;
