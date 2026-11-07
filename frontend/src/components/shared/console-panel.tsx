/**
 * Staff day-console surface primitives.
 *
 * These mirror the latest toto.sg gallery vocabulary: a panel owns its own
 * kicker/title/actions (no detached section card above it), figures sit in
 * compact stat chips under the chart, and a delta reads against the previous
 * period rather than in a paragraph.
 *
 * Staff surfaces only. Never apply these to the employee portal or auth.
 */
import type { ComponentType, ReactNode, SVGProps } from 'react';
import { cn } from '@/lib/utils';

export type ConsoleTone = 'primary' | 'secondary' | 'gold' | 'down' | 'neutral';

const TONE_CHIP: Record<ConsoleTone, string> = {
  primary: 'bg-primary/10 text-primary ring-primary/30',
  secondary: 'bg-info/10 text-info ring-info/30',
  gold: 'bg-warning/10 text-warning ring-warning/30',
  down: 'bg-destructive/10 text-destructive ring-destructive/30',
  neutral: 'bg-muted text-muted-foreground ring-border',
};

/* ------------------------------------------------------------------ *
 * Panel
 * ------------------------------------------------------------------ */

export interface ConsolePanelProps {
  /** Anchor id — a section index can link to it. */
  id?: string;
  /** 9px mono label above the title. */
  kicker?: string;
  title?: string;
  /** One short meta line (period, count). Never a paragraph. */
  description?: string;
  /** Controls rendered after the title. */
  actions?: ReactNode;
  /** Rendered below the body, separated by a hairline. */
  footer?: ReactNode;
  /** Remove body padding — for edge-to-edge tables and lists. */
  isFlush?: boolean;
  className?: string;
  bodyClassName?: string;
  children: ReactNode;
}

/**
 * Console cell: 8px hairline panel with a mono header strip. The header is
 * part of the panel, so pages no longer stack a section header card above a
 * second card — the main source of wasted vertical space.
 */
export function ConsolePanel({
  id,
  kicker,
  title,
  description,
  actions,
  footer,
  isFlush = false,
  className,
  bodyClassName,
  children,
}: ConsolePanelProps) {
  const hasHeader = Boolean(kicker || title || actions);

  return (
    <section
      id={id}
      data-slot="console-panel"
      className={cn('console-panel flex min-w-0 flex-col', className)}
    >
      {hasHeader && (
        <header className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1.5 border-b border-border/70 bg-muted/40 px-3 py-2">
          <div className="min-w-0">
            {kicker && <p className="console-kicker text-primary">{kicker}</p>}
            {title && (
              <h2
                className={cn(
                  'font-display text-[13px] font-semibold leading-5 tracking-tight text-foreground',
                  kicker && 'mt-0.5',
                )}
              >
                {title}
              </h2>
            )}
            {description && (
              <p className="mt-0.5 max-w-prose text-[11px] leading-4 text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {actions && (
            <div className="ml-auto flex min-w-0 flex-wrap items-center gap-2">{actions}</div>
          )}
        </header>
      )}
      <div className={cn(isFlush ? '' : 'console-panel-body', bodyClassName)}>{children}</div>
      {footer && (
        <footer className="border-t border-border/70 px-3 py-2 text-[11px] leading-4 text-muted-foreground">
          {footer}
        </footer>
      )}
    </section>
  );
}

/* ------------------------------------------------------------------ *
 * Stat chip
 * ------------------------------------------------------------------ */

export interface ConsoleStatProps {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  /** Full figure behind a compact value. */
  title?: string;
  tone?: 'default' | 'primary' | 'positive' | 'negative';
  /**
   * `bare` drops the card chrome for metric strips inside a panel — the top
   * hairline does the separating. `card` is a standalone tile.
   */
  variant?: 'bare' | 'card';
  className?: string;
}

const STAT_TONE: Record<NonNullable<ConsoleStatProps['tone']>, string> = {
  default: 'text-foreground',
  primary: 'text-primary',
  positive: 'text-success',
  negative: 'text-destructive',
};

/** Compact metric: mono label, one figure, optional hint. No icon chrome. */
export function ConsoleStat({
  label,
  value,
  hint,
  title,
  tone = 'default',
  variant = 'bare',
  className,
}: ConsoleStatProps) {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-col gap-0.5',
        variant === 'card'
          ? 'rounded-lg border border-border/70 bg-card px-3 py-2'
          : 'border-t border-border/70 pt-2',
        className,
      )}
    >
      <span className="console-kicker truncate">{label}</span>
      <span
        className={cn(
          'console-value truncate font-display text-xl font-semibold leading-6 tracking-tight',
          STAT_TONE[tone],
        )}
        {...(title ? { title, 'aria-label': title } : {})}
      >
        {value}
      </span>
      {hint && (
        <span className="truncate text-[10px] leading-3 text-muted-foreground">{hint}</span>
      )}
    </div>
  );
}

export interface ConsoleStatRowProps {
  stats: ConsoleStatProps[];
  className?: string;
}

/** Two or three chips under a chart, always in one row. */
export function ConsoleStatRow({ stats, className }: ConsoleStatRowProps) {
  if (stats.length === 0) return null;

  return (
    <div
      className={cn(
        'grid gap-2',
        stats.length === 2 ? 'grid-cols-2' : 'grid-cols-3',
        className,
      )}
    >
      {stats.map((stat) => (
        <ConsoleStat key={stat.label} {...stat} />
      ))}
    </div>
  );
}

/* ------------------------------------------------------------------ *
 * Delta
 * ------------------------------------------------------------------ */

export interface ConsoleDeltaProps {
  /** Signed change. `null`/`undefined` renders an em dash. */
  value: number | null | undefined;
  /** Unit printed after the figure. Defaults to `%`. */
  suffix?: string;
  digits?: number;
  /** `false` for metrics where a rise is bad (expenses). Defaults to true. */
  positiveIsGood?: boolean;
  className?: string;
}

/**
 * Signed change against the previous period. This is the payroll analogue of
 * toto's observed-vs-expected mark: the delta lives beside the figure, never
 * in a sentence underneath it.
 */
export function ConsoleDelta({
  value,
  suffix = '%',
  digits = 1,
  positiveIsGood = true,
  className,
}: ConsoleDeltaProps) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return <span className={cn('console-value text-[11px] text-muted-foreground', className)}>—</span>;
  }

  const neutral = Math.abs(value) < 0.05;
  const good = positiveIsGood ? value > 0 : value < 0;

  return (
    <span
      className={cn(
        'console-value inline-flex items-center gap-0.5 text-[11px] font-semibold leading-4',
        neutral ? 'text-muted-foreground' : good ? 'text-success' : 'text-destructive',
        className,
      )}
      aria-label={`${value > 0 ? '+' : ''}${value.toFixed(digits)}${suffix} so với kỳ trước`}
    >
      {!neutral && <span aria-hidden="true">{value > 0 ? '↑' : '↓'}</span>}
      {value > 0 && !neutral ? '+' : value < 0 ? '−' : ''}
      {Math.abs(value).toFixed(digits)}
      {suffix}
    </span>
  );
}

/* ------------------------------------------------------------------ *
 * KPI card
 * ------------------------------------------------------------------ */

export interface ConsoleKpiProps {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  tone?: ConsoleTone;
  /** Optional change against the previous period. */
  delta?: { value: number | null | undefined; suffix?: string; positiveIsGood?: boolean };
  /** Full figure behind a compact value. */
  title?: string;
  onClick?: () => void;
  className?: string;
}

/** Horizontal KPI tile: icon chip, mono label, one figure, optional delta. */
export function ConsoleKpi({
  label,
  value,
  hint,
  icon: Icon,
  tone = 'primary',
  delta,
  title,
  onClick,
  className,
}: ConsoleKpiProps) {
  const Root = onClick ? 'button' : 'div';

  return (
    <Root
      type={onClick ? 'button' : undefined}
      onClick={onClick}
      className={cn(
        'console-panel flex min-w-0 items-center gap-2.5 px-3 py-2 text-left',
        onClick &&
          'cursor-pointer transition-[border-color,background-color] duration-100 hover:border-primary/40 hover:bg-muted/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 motion-reduce:transition-none',
        className,
      )}
    >
      {Icon && (
        <span
          className={cn(
            'grid size-9 shrink-0 place-items-center rounded-md ring-1 ring-inset',
            TONE_CHIP[tone],
          )}
        >
          <Icon className="size-4" aria-hidden="true" />
        </span>
      )}
      <div className="min-w-0">
        <p className="console-kicker truncate">{label}</p>
        <p
          className="console-value mt-0.5 truncate font-display text-xl font-semibold leading-6 tracking-tight text-foreground"
          {...(title ? { title, 'aria-label': title } : {})}
        >
          {value}
        </p>
        {(hint || delta) && (
          <div className="mt-0.5 flex min-w-0 items-center gap-1.5">
            {delta && (
              <ConsoleDelta
                value={delta.value}
                suffix={delta.suffix}
                positiveIsGood={delta.positiveIsGood}
              />
            )}
            {hint && (
              <span className="truncate text-[10px] leading-3 text-muted-foreground">{hint}</span>
            )}
          </div>
        )}
      </div>
    </Root>
  );
}

/* ------------------------------------------------------------------ *
 * Section heading
 * ------------------------------------------------------------------ */

export interface ConsoleSectionHeadingProps {
  id?: string;
  kicker?: string;
  title: string;
  /** One short meta line (count, period). */
  meta?: ReactNode;
  actions?: ReactNode;
  className?: string;
}

/** 16px section heading between packed galleries. No explanatory paragraph. */
export function ConsoleSectionHeading({
  id,
  kicker,
  title,
  meta,
  actions,
  className,
}: ConsoleSectionHeadingProps) {
  return (
    <div
      className={cn(
        'flex flex-wrap items-end justify-between gap-x-4 gap-y-1 border-b border-border/70 pb-2',
        className,
      )}
    >
      <div className="min-w-0">
        {kicker && <p className="console-kicker">{kicker}</p>}
        <h2
          id={id}
          className="font-display text-base font-semibold leading-6 tracking-tight text-foreground"
        >
          {title}
        </h2>
        {meta && <p className="mt-0.5 font-financial text-[11px] text-muted-foreground">{meta}</p>}
      </div>
      {actions && <div className="flex min-w-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}
