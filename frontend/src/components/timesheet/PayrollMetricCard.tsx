import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

export interface PayrollMetricCardProps {
  label: string;
  /** Pre-formatted value (counts or currency string). */
  value: string;
  /** Extra line under the label, e.g. "0 yêu cầu sửa · 0 bị loại". */
  supportText?: string;
  /** Headline KPI — gets a navy base tint + larger figure. */
  primary?: boolean;
  /** Span both columns once the KPI grid has enough width for a two-column layout. */
  mobileSpan?: boolean;
  selected?: boolean;
  disabled?: boolean;
  isLoading?: boolean;
  onClick?: () => void;
  className?: string;
}

export const PayrollMetricCard = memo(function PayrollMetricCard({
  label,
  value,
  supportText,
  primary = false,
  mobileSpan = false,
  selected = false,
  disabled = false,
  isLoading = false,
  onClick,
  className,
}: PayrollMetricCardProps) {
  const spanCls = mobileSpan ? 'col-span-2' : '';
  if (isLoading) {
    return (
      <div
        className={cn(
          'flex h-full min-h-[64px] flex-col justify-center gap-1.5 rounded-lg border border-border/70 bg-card px-3 py-2.5 sm:min-h-[64px] sm:py-2',
          spanCls,
          className,
        )}
      >
        <div className="space-y-1.5">
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
        'group flex h-full min-h-[64px] min-w-0 overflow-hidden flex-col justify-center gap-1.5 rounded-lg border border-border/70 bg-card px-3 py-2.5 text-left sm:min-h-[64px] sm:py-2',
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
      <div className="min-w-0 flex flex-col gap-0.5">
        <span
          className={cn(
            'whitespace-nowrap font-financial font-bold leading-none tabular-nums tracking-tight',
            primary
              ? 'text-[clamp(1.125rem,2.2vw,1.75rem)]'
              // 16px on phones so full VND totals fit a half-width cell; the
              // desktop grid has room for the larger figure.
              : 'text-base sm:text-[1.5rem]',
            selected ? 'text-primary' : 'text-foreground',
          )}
        >
          {value}
        </span>
        <span className="text-[12px] font-medium leading-tight text-muted-foreground">
          {label}
        </span>
        {supportText && (
          <span className="mt-0.5 text-xs leading-tight text-muted-foreground/80">
            {supportText}
          </span>
        )}
      </div>
    </button>
  );
});
