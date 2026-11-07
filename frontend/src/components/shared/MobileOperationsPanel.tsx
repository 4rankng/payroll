import { type ReactNode } from 'react';
// W13a follow-up (approved): icon slots widened from LucideIcon so both lucide
// and @untitledui/icons components fit (UU's are plain FunctionComponents).
import type { ComponentType, SVGProps } from 'react';
import { ChevronRight } from '@untitledui/icons';
import { cn } from '@/lib/utils';
import { ConsolePanel } from '@/components/shared/console-panel';

type MobileOperationTone = 'primary' | 'success' | 'warning' | 'danger' | 'neutral';

export interface MobileOperationMetric {
  label: string;
  value: ReactNode;
  helper?: ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  tone?: MobileOperationTone;
  onClick?: () => void;
}

export interface MobileOperationAction {
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  onClick: () => void;
  badge?: ReactNode;
  disabled?: boolean;
}

export interface MobileTaskRow {
  title: string;
  description?: ReactNode;
  value?: ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  tone?: MobileOperationTone;
  onClick?: () => void;
}

interface MobileOperationsPanelProps {
  eyebrow?: string;
  title: string;
  subtitle?: string;
  primaryLabel: string;
  primaryValue: ReactNode;
  primaryHint?: ReactNode;
  metrics: MobileOperationMetric[];
  actions?: MobileOperationAction[];
  className?: string;
}

interface MobileTaskListProps {
  title: string;
  subtitle?: string;
  items: MobileTaskRow[];
  className?: string;
}

const TONE: Record<MobileOperationTone, { dot: string; value: string }> = {
  primary: { dot: 'bg-primary', value: 'text-foreground' },
  success: { dot: 'bg-success', value: 'text-success' },
  warning: { dot: 'bg-warning', value: 'text-warning' },
  danger: { dot: 'bg-destructive', value: 'text-destructive' },
  neutral: { dot: 'bg-muted-foreground/40', value: 'text-foreground' },
};

export function MobileOperationsPanel({
  eyebrow,
  title,
  primaryLabel,
  primaryValue,
  primaryHint,
  metrics,
  actions = [],
  className,
}: MobileOperationsPanelProps) {
  return (
    <ConsolePanel
      kicker={eyebrow}
      title={title}
      className={className}
      isFlush
      actions={
        <div className="text-right">
          <p className="console-kicker">{primaryLabel}</p>
          <p className="console-value mt-0.5 font-display text-xl font-semibold leading-6 tracking-tight text-foreground tabular-nums">
            {primaryValue}
          </p>
          {primaryHint && (
            <p className="mt-0.5 text-[10px] leading-3 text-muted-foreground">{primaryHint}</p>
          )}
        </div>
      }
    >
      {metrics.length > 0 && (
        <div className="divide-y divide-border/60">
          {metrics.map((metric, index) => (
            <OperationMetricCell key={`${metric.label}-${index}`} metric={metric} />
          ))}
        </div>
      )}

      {actions.length > 0 && (
        <div className="border-t border-border/60 px-1 py-1">
          <div className="grid grid-cols-4 gap-1">
            {actions.map((action) => {
              const Icon = action.icon;
              return (
                <button
                  key={action.label}
                  type="button"
                  onClick={action.onClick}
                  disabled={action.disabled}
                  className="relative flex min-h-[48px] min-w-0 touch-manipulation flex-col items-center justify-center gap-1 rounded-lg px-1.5 text-center transition-colors active:bg-muted disabled:pointer-events-none disabled:opacity-45"
                >
                  <Icon className="h-5 w-5 text-primary" aria-hidden="true" />
                  <span className="max-w-full text-[11px] font-semibold leading-tight text-foreground">
                    {action.label}
                  </span>
                  {action.badge && (
                    <span className="absolute right-1.5 top-1.5 rounded-full bg-warning px-1.5 text-[10px] font-bold leading-4 text-white">
                      {action.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </ConsolePanel>
  );
}

function OperationMetricCell({ metric, className }: { metric: MobileOperationMetric; className?: string }) {
  const tone = TONE[metric.tone ?? 'neutral'];

  const content = (
    <>
      <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', tone.dot)} aria-hidden="true" />
      <span className="min-w-0 flex-1">
        <span className="flex items-baseline justify-between gap-2">
          <span className="min-w-0 truncate text-[12px] leading-4 text-foreground">{metric.label}</span>
          <span className={cn('console-value shrink-0 text-[13px] font-semibold tabular-nums', tone.value)}>
            {metric.value}
          </span>
        </span>
        {metric.helper && (
          <span className="mt-0.5 block truncate text-[10px] leading-3 text-muted-foreground">
            {metric.helper}
          </span>
        )}
      </span>
    </>
  );

  const itemClassName = cn(
    'flex min-h-11 w-full items-center gap-2 px-3 py-2 text-left',
    metric.onClick && 'touch-manipulation transition-colors active:bg-muted',
    className,
  );

  if (metric.onClick) {
    return (
      <button type="button" onClick={metric.onClick} className={itemClassName}>
        {content}
      </button>
    );
  }

  return <div className={itemClassName}>{content}</div>;
}

/** Compact queue panel. Hidden entirely when there is no real row. */
export function MobileTaskList({ title, subtitle, items, className }: MobileTaskListProps) {
  if (items.length === 0) return null;

  return (
    <ConsolePanel
      kicker="Hàng đợi"
      title={title}
      description={subtitle}
      className={className}
      isFlush
    >
      <div className="divide-y divide-border/60">
        {items.map((item, index) => (
          <TaskRow key={`${item.title}-${index}`} item={item} />
        ))}
      </div>
    </ConsolePanel>
  );
}

function TaskRow({ item }: { item: MobileTaskRow }) {
  const tone = TONE[item.tone ?? 'neutral'];

  const content = (
    <>
      <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', tone.dot)} aria-hidden="true" />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[13px] font-semibold leading-tight text-foreground">
          {item.title}
        </span>
        {item.description && (
          <span className="mt-0.5 block text-[11px] leading-4 text-muted-foreground">
            {item.description}
          </span>
        )}
      </span>
      {(item.value || item.onClick) && (
        <span className="flex shrink-0 items-center gap-1.5">
          {item.value && (
            <span className={cn('console-value text-[13px] font-semibold tabular-nums', tone.value)}>
              {item.value}
            </span>
          )}
          {item.onClick && (
            <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          )}
        </span>
      )}
    </>
  );

  const className =
    'flex min-h-11 w-full items-center gap-2 px-3 py-2 text-left touch-manipulation transition-colors active:bg-muted';

  if (item.onClick) {
    return (
      <button type="button" onClick={item.onClick} className={className}>
        {content}
      </button>
    );
  }

  return <div className={className}>{content}</div>;
}
