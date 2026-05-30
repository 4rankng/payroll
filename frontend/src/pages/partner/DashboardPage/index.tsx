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
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
} from 'recharts';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { PageHeader } from '@/components/shared/PageHeader';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
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
    <div className="flex items-center gap-1 overflow-x-auto no-scrollbar pb-0.5 -mb-0.5">
      {monthOptions.slice(0, 4).map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={cn(
            'px-3 py-1.5 rounded-full text-xs font-medium transition-all shrink-0',
            value === opt.value
              ? 'bg-foreground text-background shadow-sm'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
          )}
        >
          {opt.label}
        </button>
      ))}
      <select
        value={monthOptions.slice(4).some((o) => o.value === value) ? value : ''}
        onChange={(e) => e.target.value && onChange(e.target.value)}
        className="px-2.5 py-1.5 rounded-full text-xs font-medium bg-transparent text-muted-foreground border-0 outline-none cursor-pointer hover:bg-muted/60 hover:text-foreground transition-colors shrink-0"
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
  blue:    { iconText: 'text-blue-600',    watermark: 'text-blue-500/15' },
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
        'group relative w-full text-left rounded-2xl bg-card px-4 py-4 overflow-hidden transition-all',
        'shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_2px_6px_-2px_rgba(15,15,30,0.04)]',
        onClick &&
          'hover:shadow-[0_2px_4px_-1px_rgba(15,15,30,0.08),0_8px_16px_-6px_rgba(15,15,30,0.08)] hover:-translate-y-0.5 cursor-pointer',
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
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            {label}
          </span>
        </div>

        {/* Big number */}
        <p className="mt-1.5 text-2xl font-extrabold tabular-nums leading-none text-foreground tracking-tight">
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

// ─── Workforce donut ──────────────────────────────────────────────────────
function WorkforceDonut({
  active,
  dropped,
  paid,
}: {
  active: number;
  dropped: number;
  paid: number;
}) {
  const data = useMemo(
    () => [
      { name: 'Đang làm', value: active, color: '#10b981' },
      { name: 'Đã thanh toán', value: paid, color: '#3b82f6' },
      { name: 'Có thể nghỉ', value: dropped, color: '#f59e0b' },
    ],
    [active, dropped, paid],
  );
  const total = active + dropped + paid;

  if (total === 0) {
    return (
      <div className="flex h-[180px] items-center justify-center text-[12px] text-muted-foreground">
        Chưa có dữ liệu
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="relative h-[180px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={55}
              outerRadius={80}
              paddingAngle={2}
              dataKey="value"
              startAngle={90}
              endAngle={-270}
              stroke="none"
            >
              {data.map((entry) => (
                <Cell key={entry.name} fill={entry.color} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span className="text-2xl font-extrabold tabular-nums text-foreground leading-none">
            {(active + dropped).toLocaleString('vi-VN')}
          </span>
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mt-1">
            Tổng nhân viên
          </span>
        </div>
      </div>
      <div className="space-y-1.5">
        {data.map((d) => (
          <div key={d.name} className="flex items-center justify-between text-[12px]">
            <div className="flex items-center gap-2">
              <span className="h-2 w-2 rounded-full" style={{ background: d.color }} />
              <span className="text-foreground">{d.name}</span>
            </div>
            <span className="font-semibold tabular-nums text-foreground">{d.value.toLocaleString('vi-VN')}</span>
          </div>
        ))}
      </div>
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
    <div className="p-4 lg:p-8 max-w-[1320px] mx-auto space-y-6">
      <PageHeader
        title="Tổng quan"
        description="Theo dõi hoạt động nhân viên và tình hình thanh toán"
      >
        <MonthSelector value={selectedMonth} onChange={handleMonthChange} />
      </PageHeader>

      {/* ── HERO SECTION ── */}
      {isLoading ? (
        <HeroSkeleton />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-4 animate-fade-in-up">
          {/* Left: big payout hero card */}
          <div className="lg:col-span-3 relative overflow-hidden rounded-2xl p-6 lg:p-8 text-white shadow-[0_20px_50px_-20px_rgba(76,29,149,0.45)]">
            {/* Background gradient + decorative blobs */}
            <div className="absolute inset-0 bg-gradient-to-br from-[#1e1b4b] via-[#4c1d95] to-[#6b21a8]" />
            <div className="absolute -right-12 -top-12 h-56 w-56 rounded-full bg-fuchsia-500/30 blur-3xl pointer-events-none" />
            <div className="absolute -left-10 bottom-0 h-44 w-44 rounded-full bg-indigo-500/25 blur-3xl pointer-events-none" />
            <div className="absolute inset-0 opacity-[0.07] pointer-events-none [background-image:radial-gradient(white_1px,transparent_1px)] [background-size:20px_20px]" />

            <div className="relative">
              <div className="flex items-center gap-1.5">
                <Sparkles className="h-3.5 w-3.5 text-violet-200" />
                <span className="text-[10.5px] font-bold uppercase tracking-[0.18em] text-violet-100/80">
                  Tổng chi trả · {periodLabel}
                </span>
              </div>
              <div className="mt-3 flex items-baseline gap-2.5 flex-wrap">
                <span className="font-display font-extrabold tabular-nums tracking-tight leading-none text-[clamp(2rem,5vw,3.5rem)]">
                  {formatVND(data?.total_paid_vnd ?? 0)}
                </span>
                {showMomBadges && <TrendChip change={data?.mom_paid_amount} variant="on-dark" />}
              </div>
              <p className="mt-3 text-[13px] text-violet-100/75 leading-relaxed">
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
                  <Banknote className="h-3.5 w-3.5 text-violet-200" />
                  <span className="text-violet-100/70">Cập nhật theo thời gian thực</span>
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
        <div className="rounded-2xl bg-card p-5 shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_2px_8px_-4px_rgba(15,15,30,0.05)]">
          <div className="mb-4">
            <h3 className="text-sm font-bold text-foreground">Cơ cấu nhân sự</h3>
            <p className="text-[11px] text-muted-foreground mt-0.5">Phân loại theo trạng thái hoạt động</p>
          </div>
          {isLoading ? (
            <Skeleton className="h-[200px] w-full rounded-xl" />
          ) : (
            <WorkforceDonut
              active={data?.active_employees ?? 0}
              dropped={data?.dropped_employees ?? 0}
              paid={data?.paid_employees ?? 0}
            />
          )}
        </div>

        {/* Top employees leaderboard */}
        <div className="lg:col-span-2 rounded-2xl bg-card p-5 shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_2px_8px_-4px_rgba(15,15,30,0.05)]">
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
