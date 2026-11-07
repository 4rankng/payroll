// W13a follow-up (approved): icon slots widened from LucideIcon so both lucide
// and @untitledui/icons components fit (UU's are plain FunctionComponents).
import { useState } from 'react';
import type { ComponentType, SVGProps } from 'react';
import { ArrowUpRight, ChevronDown, Loader2, RefreshCcw } from 'lucide-react';

import { cn } from '@/lib/utils';
import { ConsolePanel } from '@/components/shared/console-panel';

type DashboardTone = 'primary' | 'success' | 'warning' | 'neutral';

export interface DashboardMetricItem {
  label: string;
  value: string;
  /** Exact figure behind a compact `value` — surfaces as the cell's
   *  title/aria-label so compact KPI text never hides the real number. */
  fullValue?: string;
  context: string;
  /** Kept for API compatibility; the treasury strip renders a signal LED
   *  instead of an icon tile. */
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  tone: DashboardTone;
  onClick?: () => void;
}

export interface DashboardPriorityItem {
  title: string;
  value: string;
  detail: string;
  statusLabel: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
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
  className?: string;
}

interface DashboardActivityPanelProps {
  monthLabel: string;
  totalActive?: number;
  items: DashboardActivityItem[];
  /** Per-project active logins for the same month; hidden when empty. */
  projectItems?: DashboardActivityItem[];
  isLoading?: boolean;
  hasError?: boolean;
  className?: string;
}

/** Signal LED tones — semantics only: primary/success emerald, warning amber,
 *  neutral muted. The strip's color carries meaning, never decoration. */
const SIGNAL_TONE: Record<DashboardTone, string> = {
  primary: 'bg-primary',
  success: 'bg-emerald-500',
  warning: 'bg-amber-500',
  neutral: 'bg-muted-foreground/40',
};

const STATUS_TONE: Record<DashboardTone, string> = {
  primary: 'text-primary',
  success: 'text-success',
  warning: 'text-warning',
  neutral: 'text-muted-foreground',
};

function MetricContent({ item }: { item: DashboardMetricItem }) {
  return (
    <>
      <div className="relative z-[1] flex items-center gap-1.5">
        <span className={cn('treasury-signal', SIGNAL_TONE[item.tone])} aria-hidden="true" />
        <p className="console-kicker">{item.label}</p>
      </div>
      <p
        className="console-value treasury-value relative z-[1] mt-1.5 break-words font-financial text-xl font-semibold leading-tight tracking-normal text-foreground tabular-nums"
        {...(item.fullValue ? { title: item.fullValue, 'aria-label': item.fullValue } : {})}
      >
        {item.value}
      </p>
      <p className="relative z-[1] mt-0.5 text-[11px] leading-4 text-muted-foreground">
        {item.context}
      </p>
    </>
  );
}

export function DashboardMetricStrip({ items }: DashboardMetricStripProps) {
  // treasury-grid: fading seams between cells replace per-cell borders.
  const cols = items.length >= 4 ? 'lg:grid-cols-4' : 'lg:grid-cols-3';

  return (
    <section
      aria-label="Tóm tắt kỳ lương"
      className={cn(
        'treasury-grid grid grid-cols-1 overflow-hidden rounded-lg border border-border bg-card sm:grid-cols-2',
        cols,
      )}
    >
      {items.map((item) => {
        const className = cn(
          'treasury-panel min-w-0 px-3 py-2.5 text-left sm:px-4',
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
  className,
}: DashboardPriorityListProps) {
  return (
    <ConsolePanel
      kicker="Hàng đợi vận hành"
      title="Việc cần xử lý"
      isFlush
      className={className}
      actions={
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
      }
    >
      <div className="divide-y divide-border/60">
        {items.length === 0 ? (
          <div className="flex min-h-[64px] items-center gap-3 px-3 py-2 sm:min-h-[52px]">
            <span className="h-1.5 w-1.5 rounded-full bg-success" aria-hidden="true" />
            <p className="text-[13px] text-foreground">Không có việc cần xử lý trong kỳ.</p>
          </div>
        ) : items.map((item) => {
          const content = (
            <>
              <span
                className={cn('h-1.5 w-1.5 shrink-0 rounded-full', SIGNAL_TONE[item.tone])}
                aria-hidden="true"
              />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                  <h3 className="text-[13px] font-semibold text-foreground">{item.title}</h3>
                  <span className={cn('console-kicker', STATUS_TONE[item.tone])}>
                    {item.statusLabel}
                  </span>
                </div>
                <p className="mt-0.5 line-clamp-1 text-[11px] leading-4 text-muted-foreground">
                  {item.detail}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-1.5">
                <span className="console-value font-display text-lg font-semibold leading-6 text-foreground tabular-nums">
                  {item.value}
                </span>
                {item.onClick ? (
                  <ArrowUpRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                ) : null}
              </div>
            </>
          );

          return item.onClick ? (
            <button
              key={item.title}
              type="button"
              onClick={item.onClick}
              className="flex min-h-[64px] w-full items-center gap-3 px-3 py-2 text-left transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:min-h-[52px] motion-reduce:transition-none"
            >
              {content}
            </button>
          ) : (
            <div
              key={item.title}
              className="flex min-h-[64px] items-center gap-3 px-3 py-2 sm:min-h-[52px]"
            >
              {content}
            </div>
          );
        })}
      </div>
    </ConsolePanel>
  );
}

export function DashboardActivityPanel({
  monthLabel,
  totalActive,
  items,
  projectItems = [],
  isLoading = false,
  hasError = false,
  className,
}: DashboardActivityPanelProps) {
  const [projectsExpanded, setProjectsExpanded] = useState(false);

  return (
    <ConsolePanel
      kicker={`Tín hiệu sử dụng · ${monthLabel}`}
      title="Tài khoản hoạt động"
      isFlush
      className={className}
      actions={
        <div className="text-right">
          <p className="console-value font-display text-xl font-semibold leading-6 text-foreground tabular-nums">
            {isLoading || hasError || totalActive === undefined
              ? '--'
              : totalActive.toLocaleString('vi-VN')}
          </p>
          <p className="console-kicker">tổng hoạt động</p>
        </div>
      }
    >
      <div className="divide-y divide-border/60" aria-busy={isLoading} aria-live="polite">
        {hasError ? (
          <p className="px-3 py-4 text-[13px] text-destructive">
            Không thể tải dữ liệu hoạt động. Vui lòng làm mới để thử lại.
          </p>
        ) : isLoading ? (
          Array.from({ length: 3 }).map((_, index) => (
            <div
              key={index}
              className="flex min-h-11 items-center justify-between px-3 py-2 sm:min-h-9"
            >
              <span className="h-3 w-24 animate-pulse rounded bg-muted motion-reduce:animate-none" />
              <span className="h-4 w-8 animate-pulse rounded bg-muted motion-reduce:animate-none" />
            </div>
          ))
        ) : items.length === 0 ? (
          <p className="px-3 py-4 text-[13px] text-muted-foreground">
            Chưa có hoạt động trong kỳ.
          </p>
        ) : (
          items.map((item) => (
            <button
              key={item.label}
              type="button"
              onClick={item.onClick}
              disabled={!item.onClick}
              className="flex min-h-11 w-full items-center justify-between gap-3 px-3 py-2 text-left transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring disabled:cursor-default disabled:hover:bg-transparent sm:min-h-9 motion-reduce:transition-none"
            >
              <span className="truncate text-[13px] text-foreground">{item.label}</span>
              <span className="console-value font-display text-base font-semibold text-foreground tabular-nums">
                {typeof item.value === 'number'
                  ? item.value.toLocaleString('vi-VN')
                  : item.value}
              </span>
            </button>
          ))
        )}
        {projectItems.length > 0 && (
          <div className="border-t border-border/60">
            <button
              type="button"
              aria-expanded={projectsExpanded}
              onClick={() => setProjectsExpanded((expanded) => !expanded)}
              className="flex min-h-11 w-full items-center justify-between gap-3 px-3 py-2 text-left transition-colors hover:bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:min-h-9"
            >
              <span className="console-kicker">Theo dự án</span>
              <span className="flex items-center gap-1.5 text-[11px] tabular-nums text-muted-foreground">
                {projectItems.length} dự án
                <ChevronDown
                  className={cn('h-4 w-4 transition-transform', projectsExpanded && 'rotate-180')}
                  aria-hidden="true"
                />
              </span>
            </button>
            {projectsExpanded &&
              projectItems.map((item) => (
                <div
                  key={item.label}
                  className="flex min-h-11 items-center justify-between gap-3 px-3 py-2 sm:min-h-9"
                >
                  <span className="truncate text-[13px] text-foreground">{item.label}</span>
                  <span className="console-value font-display text-base font-semibold text-foreground tabular-nums">
                    {typeof item.value === 'number'
                      ? item.value.toLocaleString('vi-VN')
                      : item.value}
                  </span>
                </div>
              ))}
          </div>
        )}
      </div>
    </ConsolePanel>
  );
}
