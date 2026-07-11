import { memo } from 'react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

/**
 * Category is encoded by the icon tint, never by coloring the number:
 * - action → amber (needs attention)
 * - done   → emerald (complete)
 * - issue  → rose (problem)
 */
export type MetricCategory = 'action' | 'done' | 'issue';

export interface PayrollMetricCardProps {
  label: string;
  /** Pre-formatted value (counts or currency string). */
  value: string;
  icon: LucideIcon;
  category?: MetricCategory;
  /** Extra line under the label, e.g. "0 yêu cầu sửa · 0 bị loại". */
  supportText?: string;
  /** Headline KPI — gets a navy base tint + larger figure. */
  primary?: boolean;
  /** Span both columns below the full desktop layout so long currency values stay readable. */
  mobileSpan?: boolean;
  selected?: boolean;
  disabled?: boolean;
  isLoading?: boolean;
  onClick?: () => void;
  className?: string;
}

const categoryIconTint: Record<MetricCategory, string> = {
  action: 'bg-amber-50 text-amber-600 ring-amber-100',
  done: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
  issue: 'bg-rose-50 text-rose-600 ring-rose-100',
};

export const PayrollMetricCard = memo(function PayrollMetricCard({
  label,
  value,
  icon: Icon,
  category,
  supportText,
  primary = false,
  mobileSpan = false,
  selected = false,
  disabled = false,
  isLoading = false,
  onClick,
  className,
}: PayrollMetricCardProps) {
  // Long-value tiles keep two tracks at desktop widths so VND values remain
  // readable instead of protruding beyond their card.
  const spanCls = mobileSpan ? 'col-span-2 lg:col-span-2' : '';
  const tint = category
    ? categoryIconTint[category]
    : 'bg-muted text-muted-foreground ring-border/60';

  if (isLoading) {
    return (
      <div
        className={cn(
          'flex h-full min-h-[76px] flex-col gap-2 rounded-lg border border-border bg-card px-3.5 py-3',
          spanCls,
          className,
        )}
      >
        <Skeleton className="h-7 w-7 rounded-lg" />
        <div className="mt-auto space-y-1.5">
          <Skeleton className="h-5 w-20" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
    );
  }

  const interactive = !!onClick && !disabled;

  return (
    <button
      type="button"
      onClick={interactive ? onClick : undefined}
      disabled={!interactive}
      aria-pressed={selected}
      aria-label={`${label}: ${value}${supportText ? `. ${supportText}` : ''}`}
      title={supportText ? `${label} — ${supportText}` : undefined}
      className={cn(
        'group flex h-full min-h-[76px] min-w-0 flex-col gap-2 rounded-lg border bg-card px-3.5 py-3 text-left',
        'transition-colors duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
        spanCls,
        primary && !selected && 'bg-primary/[0.035] border-primary/15',
        interactive && 'cursor-pointer hover:border-primary/40 hover:bg-primary/[0.05] hover:shadow-sm active:scale-[0.995]',
        selected && 'border-primary bg-primary/[0.06] ring-1 ring-primary/25',
        !interactive && 'cursor-default',
        className,
      )}
    >
      <span
        className={cn(
          'inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ring-1',
          tint,
        )}
      >
        <Icon className="h-3.5 w-3.5" strokeWidth={2.25} />
      </span>

      <div className="mt-auto min-w-0 flex flex-col gap-0.5">
        <span
          className={cn(
            'whitespace-nowrap font-financial font-bold leading-none tabular-nums tracking-tight',
            primary
              ? 'text-[clamp(1.125rem,2.2vw,1.75rem)]'
              : 'text-[1.25rem] sm:text-[1.5rem]',
            selected ? 'text-primary' : 'text-foreground',
          )}
        >
          {value}
        </span>
        <span className="text-[12px] font-medium leading-tight text-muted-foreground">
          {label}
        </span>
        {supportText && (
          <span className="mt-0.5 text-[11px] leading-tight text-muted-foreground/80">
            {supportText}
          </span>
        )}
      </div>
    </button>
  );
});
