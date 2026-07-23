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
  success: { icon: 'bg-success/10 text-success', value: 'text-success' },
  warning: { icon: 'bg-warning/10 text-warning', value: 'text-warning' },
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
        'overflow-hidden rounded-[28px] border border-[hsl(var(--surface-border))] bg-white shadow-none',
        className,
      )}
    >
      <div className="px-4 pb-4 pt-4">
        <div className="flex flex-col items-start gap-3 min-[420px]:flex-row min-[420px]:justify-between">
          <div className="min-w-0">
            {eyebrow && (
              <div className="mb-2 inline-flex min-h-7 items-center rounded-full bg-muted px-2.5 text-[11px] font-semibold text-muted-foreground">
                {eyebrow}
              </div>
            )}
            <h2 className="font-display text-[20px] font-extrabold leading-tight tracking-normal text-foreground">
              {title}
            </h2>
            {subtitle && (
              <p className="mt-1 text-xs font-medium leading-relaxed text-muted-foreground">
                {subtitle}
              </p>
            )}
          </div>
          <div className="min-w-0 w-full text-left min-[420px]:w-auto min-[420px]:shrink-0 min-[420px]:text-right">
            <p className="text-[11px] font-semibold uppercase tracking-normal text-muted-foreground">
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
        <div className="grid grid-cols-1 border-t border-border/60 min-[420px]:grid-cols-2">
          {metrics.map((metric, index) => (
            <OperationMetricCell
              key={`${metric.label}-${index}`}
              metric={metric}
              className={cn(
                index % 2 === 1 && 'min-[420px]:border-l',
                index < 2 && 'min-[420px]:border-t-0',
              )}
            />
          ))}
        </div>
      )}

      {actions.length > 0 && (
        <div className="border-t border-border/60 px-2 py-2">
          <div className="grid grid-cols-4 gap-1">
            {actions.map((action) => {
              const Icon = action.icon;
              return (
                <button
                  key={action.label}
                  type="button"
                  onClick={action.onClick}
                  disabled={action.disabled}
                  className="relative flex min-h-[66px] min-w-0 touch-manipulation flex-col items-center justify-center gap-1 rounded-2xl px-1.5 text-center transition-colors active:bg-muted disabled:pointer-events-none disabled:opacity-45"
                >
                  <Icon className="h-5 w-5 text-primary" />
                  <span className="max-w-full text-[11px] font-semibold leading-tight text-foreground">
                    {action.label}
                  </span>
                  {action.badge && (
                    <span className="absolute right-1.5 top-1.5 rounded-full bg-warning px-1.5 text-[11px] font-bold leading-4 text-warning-foreground">
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
        <span className={cn('flex h-8 w-8 shrink-0 items-center justify-center rounded-full', tone.icon)}>
          <Icon className="h-4 w-4" />
        </span>
      )}
      <span className="min-w-0 flex-1">
        <span className="block text-xs font-medium leading-tight text-muted-foreground">
          {metric.label}
        </span>
        <span className={cn('mt-1 block break-words text-base font-extrabold leading-tight tabular-nums', tone.value)}>
          {metric.value}
        </span>
        {metric.helper && (
          <span className="mt-0.5 block text-[11px] leading-snug text-muted-foreground">
            {metric.helper}
          </span>
        )}
      </span>
    </>
  );

  const itemClassName = cn(
    'flex min-h-[76px] items-start gap-3 border-t border-border/60 px-4 py-3 text-left first:border-t-0',
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
        'overflow-hidden rounded-[28px] border border-[hsl(var(--surface-border))] bg-white shadow-none',
        className,
      )}
    >
      <div className="px-4 py-3">
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
        <span className={cn('flex h-9 w-9 shrink-0 items-center justify-center rounded-full', tone.icon)}>
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
    'flex min-h-[64px] w-full items-center gap-3 px-4 py-3 text-left touch-manipulation transition-colors active:bg-muted';

  if (item.onClick) {
    return (
      <button type="button" onClick={item.onClick} className={className}>
        {content}
      </button>
    );
  }

  return <div className={className}>{content}</div>;
}
