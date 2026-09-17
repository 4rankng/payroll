import { type ReactNode } from 'react';
import { ChevronRight, type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

type MobileOperationTone = 'primary' | 'success' | 'warning' | 'danger' | 'neutral';

export interface MobileOperationMetric {
  label: string;
  value: ReactNode;
  helper?: ReactNode;
  icon?: LucideIcon;
  tone?: MobileOperationTone;
  onClick?: () => void;
}

export interface MobileOperationAction {
  label: string;
  icon: LucideIcon;
  onClick: () => void;
  badge?: ReactNode;
  disabled?: boolean;
}

export interface MobileTaskRow {
  title: string;
  description?: ReactNode;
  value?: ReactNode;
  icon?: LucideIcon;
  tone?: MobileOperationTone;
  onClick?: () => void;
}

interface MobileOperationsPanelProps {
  eyebrow?: ReactNode;
  title: string;
  subtitle?: ReactNode;
  primaryLabel: string;
  primaryValue: ReactNode;
  primaryHint?: ReactNode;
  metrics: MobileOperationMetric[];
  actions?: MobileOperationAction[];
  className?: string;
}

interface MobileTaskListProps {
  title: string;
  subtitle?: ReactNode;
  items: MobileTaskRow[];
  className?: string;
}

const toneClasses: Record<MobileOperationTone, { icon: string; value: string }> = {
  primary: { icon: 'bg-primary/10 text-primary', value: 'text-primary' },
  success: { icon: 'bg-success/10 text-success', value: 'text-emerald-700' },
  warning: { icon: 'bg-warning/10 text-warning', value: 'text-amber-700' },
  danger: { icon: 'bg-destructive/10 text-destructive', value: 'text-destructive' },
  neutral: { icon: 'bg-muted text-muted-foreground', value: 'text-foreground' },
};

export function MobileOperationsPanel({
  eyebrow,
  title,
  subtitle,
  primaryLabel,
  primaryValue,
  primaryHint,
  metrics,
  actions = [],
  className,
}: MobileOperationsPanelProps) {
  return (
    <section
      className={cn(
        'overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white shadow-none',
        className,
      )}
    >
      <div className="px-3 py-3">
        <div className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2">
          <div className="min-w-0">
            {eyebrow && (
              <div className="mb-1 inline-flex items-center text-xs font-semibold text-muted-foreground">
                {eyebrow}
              </div>
            )}
            <h2 className="font-display text-base font-extrabold leading-tight tracking-normal text-foreground">
              {title}
            </h2>
            {subtitle && (
              <p className="mt-1 text-xs font-medium leading-relaxed text-muted-foreground">
                {subtitle}
              </p>
            )}
          </div>
          <div className="min-w-0 text-right">
            <p className="text-xs font-semibold uppercase tracking-normal text-muted-foreground">
              {primaryLabel}
            </p>
            <p className="mt-1 break-words font-display text-xl font-extrabold leading-tight tracking-normal text-primary tabular-nums min-[420px]:text-2xl min-[420px]:leading-none">
              {primaryValue}
            </p>
            {primaryHint && (
              <p className="mt-1 text-xs leading-snug text-muted-foreground min-[420px]:ml-auto min-[420px]:max-w-[12rem]">
                {primaryHint}
              </p>
            )}
          </div>
        </div>
      </div>

      {metrics.length > 0 && (
        <div className="divide-y divide-border/60 border-t border-border/60">
          {metrics.map((metric, index) => (
            <OperationMetricCell
              key={`${metric.label}-${index}`}
              metric={metric}
            />
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
                  <Icon className="h-5 w-5 text-primary" />
                  <span className="max-w-full text-xs font-semibold leading-tight text-foreground">
                    {action.label}
                  </span>
                  {action.badge && (
                    <span className="absolute right-1.5 top-1.5 rounded-full bg-warning px-1.5 text-xs font-bold leading-4 text-amber-950">
                      {action.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </section>
  );
}

function OperationMetricCell({
  metric,
  className,
}: {
  metric: MobileOperationMetric;
  className?: string;
}) {
  const tone = toneClasses[metric.tone ?? 'neutral'];
  const Icon = metric.icon;
  const content = (
    <>
      {Icon && (
        <span className={cn('flex h-6 w-6 shrink-0 items-center justify-center rounded-md', tone.icon)}>
          <Icon className="h-4 w-4" />
        </span>
      )}
      <span className="grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2">
        <span className="block text-xs font-medium leading-tight text-muted-foreground">
          {metric.label}
        </span>
        <span className={cn('block break-words text-sm font-extrabold leading-tight tabular-nums', tone.value)}>
          {metric.value}
        </span>
        {metric.helper && (
          <span className="col-span-2 block text-xs leading-snug text-muted-foreground">
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

export function MobileTaskList({ title, subtitle, items, className }: MobileTaskListProps) {
  if (items.length === 0) return null;

  return (
    <section
      className={cn(
        'overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white shadow-none',
        className,
      )}
    >
      <div className="px-3 py-2">
        <h2 className="text-sm font-extrabold leading-tight text-foreground">{title}</h2>
        {subtitle && (
          <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">{subtitle}</p>
        )}
      </div>
      <div className="divide-y divide-border/60 border-t border-border/60">
        {items.map((item, index) => (
          <TaskRow key={`${item.title}-${index}`} item={item} />
        ))}
      </div>
    </section>
  );
}

function TaskRow({ item }: { item: MobileTaskRow }) {
  const tone = toneClasses[item.tone ?? 'neutral'];
  const Icon = item.icon;
  const content = (
    <>
      {Icon && (
        <span className={cn('flex h-7 w-7 shrink-0 items-center justify-center rounded-full', tone.icon)}>
          <Icon className="h-4 w-4" />
        </span>
      )}
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-semibold leading-tight text-foreground">{item.title}</span>
        {item.description && (
          <span className="mt-1 block text-xs leading-relaxed text-muted-foreground">
            {item.description}
          </span>
        )}
      </span>
      {(item.value || item.onClick) && (
        <span className="flex shrink-0 items-center gap-1.5">
          {item.value && (
            <span className={cn('text-right text-sm font-extrabold tabular-nums', tone.value)}>
              {item.value}
            </span>
          )}
          {item.onClick && <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />}
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
