import type { ReactNode } from 'react';
import { Activity, ArrowUpRight, Loader2, RefreshCcw, type LucideIcon } from 'lucide-react';

import { cn } from '@/lib/utils';

type DashboardTone = 'primary' | 'success' | 'warning' | 'neutral';

export interface DashboardMetricItem {
  label: string;
  value: string;
  context: string;
  /** Kept for API compatibility; the treasury strip renders a signal LED
   *  instead of an icon tile. */
  icon?: LucideIcon;
  tone: DashboardTone;
  onClick?: () => void;
}

export interface DashboardPriorityItem {
  title: string;
  value: string;
  detail: string;
  statusLabel: string;
  icon: LucideIcon;
  tone: DashboardTone;
  onClick?: () => void;
}

export interface DashboardActivityItem {
  label: string;
  value: string | number;
  onClick?: () => void;
}

interface DashboardMetricStripProps {
  items: DashboardMetricItem[];
}

interface DashboardPriorityListProps {
  items: DashboardPriorityItem[];
  isRefreshing: boolean;
  onRefresh: () => void;
}

interface DashboardActivityPanelProps {
  monthLabel: string;
  totalActive?: number;
  items: DashboardActivityItem[];
  isLoading?: boolean;
  hasError?: boolean;
}

interface DashboardAreaHeaderProps {
  eyebrow: string;
  title: string;
  description: string;
  actions?: ReactNode;
  className?: string;
}

const TONE_STYLES: Record<
  DashboardTone,
  { icon: string; iconSurface: string; badge: string }
> = {
  primary: {
    icon: 'text-primary',
    iconSurface: 'border-primary/15 bg-primary/5',
    badge: 'border-primary/20 bg-primary/5 text-primary',
  },
  success: {
    icon: 'text-emerald-700',
    iconSurface: 'border-emerald-200 bg-emerald-50',
    badge: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  },
  warning: {
    icon: 'text-amber-700',
    iconSurface: 'border-amber-200 bg-amber-50',
    badge: 'border-amber-200 bg-amber-50 text-amber-800',
  },
  neutral: {
    icon: 'text-muted-foreground',
    iconSurface: 'border-border bg-muted/25',
    badge: 'border-border bg-muted/25 text-muted-foreground',
  },
};

/** Signal LED tones — semantics only: primary/success emerald, warning amber,
 *  neutral muted. The strip's color carries meaning, never decoration. */
const SIGNAL_TONE: Record<DashboardTone, string> = {
  primary: 'bg-primary',
  success: 'bg-emerald-500',
  warning: 'bg-amber-500',
  neutral: 'bg-muted-foreground/40',
};

function MetricContent({ item }: { item: DashboardMetricItem }) {
  return (
    <>
      <div className="relative z-[1] flex items-center gap-1.5">
        <span className={cn('treasury-signal', SIGNAL_TONE[item.tone])} aria-hidden="true" />
        <p className="text-xs font-bold uppercase tracking-[0.12em] text-muted-foreground">
          {item.label}
        </p>
      </div>
      <p className="treasury-value relative z-[1] mt-2 break-words font-financial text-2xl font-semibold leading-tight tracking-normal text-foreground tabular-nums">
        {item.value}
      </p>
      <p className="relative z-[1] mt-1 text-xs leading-relaxed text-muted-foreground">
        {item.context}
      </p>
    </>
  );
}

export function DashboardMetricStrip({ items }: DashboardMetricStripProps) {
  // treasury-grid: fading seams between cells replace per-cell borders.
  const xlCols = items.length >= 4 ? 'xl:grid-cols-4' : 'xl:grid-cols-3';

  return (
    <section
      aria-label="Tóm tắt kỳ lương"
      className={cn(
        'treasury-grid grid grid-cols-1 overflow-hidden rounded-xl border border-border/80 bg-card sm:grid-cols-2',
        xlCols,
      )}
    >
      {items.map((item) => {
        const className = cn(
          'treasury-panel min-w-0 px-3 py-3 text-left sm:px-4',
          item.onClick &&
            'cursor-pointer transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring motion-reduce:transition-none',
        );

        const content = (
          <>
            {/* Measurement mesh — faint dot grid under the cell label. */}
            <div className="treasury-mesh" aria-hidden />
            <MetricContent item={item} />
          </>
        );

        return item.onClick ? (
          <button key={item.label} type="button" onClick={item.onClick} className={className}>
            {content}
          </button>
        ) : (
          <div key={item.label} className={className}>
            {content}
          </div>
        );
      })}
    </section>
  );
}

export function DashboardPriorityList({
  items,
  isRefreshing,
  onRefresh,
}: DashboardPriorityListProps) {
  return (
    <section className="overflow-hidden rounded-xl border border-border/80 bg-card">
      <div className="flex flex-col gap-2.5 border-b border-border/70 px-3 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            Hàng đợi vận hành
          </p>
          <h2 className="mt-1 text-base font-semibold text-foreground">Việc cần xử lý</h2>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            Ưu tiên các bước có thể làm chậm chốt công và giải ngân kỳ lương.
          </p>
        </div>
        <button
          type="button"
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-busy={isRefreshing}
          className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-border px-3 text-xs font-semibold text-foreground transition-colors hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 sm:min-h-9 motion-reduce:transition-none"
        >
          {isRefreshing ? (
            <Loader2
              className="h-3.5 w-3.5 animate-spin motion-reduce:animate-none"
              aria-hidden="true"
            />
          ) : (
            <RefreshCcw className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          {isRefreshing ? 'Đang cập nhật' : 'Làm mới số liệu'}
        </button>
      </div>

      <div className="divide-y divide-border/60">
        {items.map((item) => {
          const styles = TONE_STYLES[item.tone];
          const content = (
            <>
              <div
                className={cn(
                  'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border sm:h-8 sm:w-8',
                  styles.iconSurface,
                )}
              >
                <item.icon className={cn('h-4 w-4', styles.icon)} aria-hidden="true" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="text-sm font-semibold text-foreground">{item.title}</h3>
                  <span
                    className={cn(
                      'rounded-full border px-2 py-0.5 text-xs font-semibold',
                      styles.badge,
                    )}
                  >
                    {item.statusLabel}
                  </span>
                </div>
                <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{item.detail}</p>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <span className="font-display text-lg font-semibold text-foreground tabular-nums">
                  {item.value}
                </span>
                {item.onClick ? <ArrowUpRight className="h-4 w-4 text-muted-foreground" /> : null}
              </div>
            </>
          );

          return item.onClick ? (
            <button
              key={item.title}
              type="button"
              onClick={item.onClick}
              className="flex min-h-[68px] w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:min-h-[60px] sm:px-4 sm:py-2 motion-reduce:transition-none"
            >
              {content}
            </button>
          ) : (
            <div key={item.title} className="flex min-h-[68px] items-center gap-3 px-3 py-2.5 sm:min-h-[60px] sm:px-4 sm:py-2">
              {content}
            </div>
          );
        })}
      </div>
    </section>
  );
}

export function DashboardActivityPanel({
  monthLabel,
  totalActive,
  items,
  isLoading = false,
  hasError = false,
}: DashboardActivityPanelProps) {
  return (
    <section className="overflow-hidden rounded-xl border border-border/80 bg-card">
      <div className="border-b border-border/70 px-3 py-3 sm:px-4">
        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          Tín hiệu sử dụng · {monthLabel}
        </p>
        <div className="mt-2 flex items-end justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold text-foreground">Tài khoản hoạt động</h2>
            <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
              Nhân viên đã đăng nhập theo hình thức trả lương.
            </p>
          </div>
          <div className="text-right">
            <p className="font-display text-2xl font-semibold text-foreground tabular-nums">
              {isLoading || hasError || totalActive === undefined
                ? '--'
                : totalActive.toLocaleString('vi-VN')}
            </p>
            <p className="text-xs text-muted-foreground">tổng hoạt động</p>
          </div>
        </div>
      </div>

      <div
        className="divide-y divide-border/60"
        aria-busy={isLoading}
        aria-live="polite"
      >
        {hasError ? (
          <p className="px-4 py-5 text-sm text-destructive sm:px-5">
            Không thể tải dữ liệu hoạt động. Vui lòng làm mới để thử lại.
          </p>
        ) : isLoading ? (
          Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="flex min-h-11 items-center justify-between px-3 py-2.5 sm:min-h-9 sm:px-4 sm:py-2">
              <span className="h-3 w-24 animate-pulse rounded bg-muted motion-reduce:animate-none" />
              <span className="h-4 w-8 animate-pulse rounded bg-muted motion-reduce:animate-none" />
            </div>
          ))
        ) : items.map((item) => (
          <button
            key={item.label}
            type="button"
            onClick={item.onClick}
            disabled={!item.onClick}
            className="flex min-h-11 w-full items-center justify-between gap-3 px-3 py-2.5 text-left transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring disabled:cursor-default disabled:hover:bg-transparent sm:min-h-9 sm:px-4 sm:py-2 motion-reduce:transition-none"
          >
            <span className="flex items-center gap-2 text-sm text-foreground">
              <Activity className="h-4 w-4 text-primary" aria-hidden="true" />
              {item.label}
            </span>
            <span className="font-display text-base font-semibold text-foreground tabular-nums">
              {typeof item.value === 'number'
                ? item.value.toLocaleString('vi-VN')
                : item.value}
            </span>
          </button>
        ))}
      </div>
    </section>
  );
}

export function DashboardAreaHeader({
  eyebrow,
  title,
  description,
  actions,
  className,
}: DashboardAreaHeaderProps) {
  return (
    <div
      className={cn(
        'flex flex-col gap-2 border-b border-border/70 pb-2 sm:flex-row sm:items-end sm:justify-between',
        className,
      )}
    >
      <div>
        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          {eyebrow}
        </p>
        <h2 className="mt-1 text-lg font-semibold tracking-tight text-foreground">{title}</h2>
        <p className="mt-1 max-w-3xl text-xs leading-relaxed text-muted-foreground">{description}</p>
      </div>
      {actions ? <div className="shrink-0">{actions}</div> : null}
    </div>
  );
}
