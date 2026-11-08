/**
 * Console chart primitives — plain SVG, no chart library.
 * Part of the staff day console instrument language: donut with centre
 * figure + stacked legend, half-circle gauge for one rate. Callers own the
 * gating (e.g. omit the gauge when the denominator is 0 — the gauge itself
 * returns null for a non-finite rate as a last resort).
 */
import type { CSSProperties, ReactElement } from 'react';

export interface ConsoleDonutSegment {
  /** Legend label (bank name, bucket name…). */
  label: string;
  /** Raw numeric value used for share math. */
  value: number;
  /** Pre-formatted figure for the legend row (count + amount…). */
  display: string;
  /** CSS color; falls back to the chart token ladder by index. */
  color?: string;
}

const CHART_TOKENS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
  'hsl(var(--chart-6))',
] as const;

const OTHER_COLOR = 'hsl(var(--chart-other))';
const NAMED_SEGMENT_CAP = 5;

export interface PreparedDonutSegment extends ConsoleDonutSegment {
  color: string;
  percentage: number;
}

/**
 * Collapses long tails into a trailing "Khác" slice so a donut never exceeds
 * six segments (top 5 named + Khác). Percentages are recomputed over the
 * collapsed set and rounded to whole percents.
 */
export const toDonutSegments = (
  segments: ReadonlyArray<ConsoleDonutSegment>,
  otherLabel = 'Khác',
): PreparedDonutSegment[] => {
  const usable = segments.filter((segment) => segment.value > 0);
  const named = usable.slice(0, NAMED_SEGMENT_CAP);
  const rest = usable.slice(NAMED_SEGMENT_CAP);

  const collapsed: ConsoleDonutSegment[] =
    rest.length > 0
      ? [
          ...named,
          {
            label: otherLabel,
            value: rest.reduce((total, segment) => total + segment.value, 0),
            display: rest.map((segment) => segment.display).join(' · '),
            color: undefined,
          },
        ]
      : named;

  const total = collapsed.reduce((total, segment) => total + segment.value, 0);

  return collapsed.map((segment, index) => ({
    ...segment,
    color: segment.color ?? (index < CHART_TOKENS.length ? CHART_TOKENS[index] : OTHER_COLOR),
    percentage: total > 0 ? Math.round((segment.value / total) * 100) : 0,
  }));
};

export interface ConsoleDonutProps {
  segments: ReadonlyArray<ConsoleDonutSegment>;
  centreValue: string;
  centreLabel: string;
  className?: string;
}

const RADIUS = 44;
const RING = 10;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

/** One donut: 10px ring segments from 12 o'clock with the headline figure
 *  beside it (never inside — full currency strings outgrow the hole), plus a
 *  stacked legend with whole-percent shares. Total ≤ 0 renders the bare
 *  track ring. */
export function ConsoleDonut({
  segments,
  centreValue,
  centreLabel,
  className,
}: ConsoleDonutProps): ReactElement {
  const prepared = toDonutSegments(segments);
  const total = prepared.reduce((total, segment) => total + segment.value, 0);

  let accumulated = 0;
  const arcs = prepared.map((segment) => {
    const length = total > 0 ? (segment.value / total) * CIRCUMFERENCE : 0;
    const arc = {
      color: segment.color,
      dasharray: `${length} ${CIRCUMFERENCE - length}`,
      dashoffset: -accumulated,
    };
    accumulated += length;
    return arc;
  });

  return (
    <div className={className}>
      <div className="flex items-center gap-4">
        <div className="relative w-[120px] shrink-0" role="img" aria-label={`${centreLabel} ${centreValue}`}>
          <svg viewBox="0 0 120 120" width={120} height={120}>
            <g transform="rotate(-90 60 60)">
              <circle
                cx={60}
                cy={60}
                r={RADIUS}
                fill="none"
                strokeWidth={RING}
                stroke="hsl(var(--border) / 0.5)"
              />
              {arcs.map((arc, index) => (
                <circle
                  key={index}
                  cx={60}
                  cy={60}
                  r={RADIUS}
                  fill="none"
                  strokeWidth={RING}
                  stroke={arc.color}
                  strokeDasharray={arc.dasharray}
                  strokeDashoffset={arc.dashoffset}
                />
              ))}
            </g>
          </svg>
        </div>
        {/* Figure sits BESIDE the ring, never inside it: a full currency
            string is wider than the 78px hole and used to collide with the
            ring segments. */}
        <div className="min-w-0 flex-1">
          <span className="console-value block break-words font-display text-xl font-bold leading-tight">
            {centreValue}
          </span>
          <span className="console-kicker mt-1 block">{centreLabel}</span>
        </div>
      </div>

      <ul className="mt-3 flex flex-col gap-1.5">
        {prepared.map((segment) => (
          <li key={segment.label} className="flex items-center justify-between gap-2 text-sm">
            <span className="flex min-w-0 items-center gap-2">
              <span
                aria-hidden="true"
                className="size-2.5 shrink-0 rounded-full"
                style={{ background: segment.color } as CSSProperties}
              />
              <span className="truncate">{segment.label}</span>
              <span className="console-value shrink-0 text-xs text-fg-tertiary">{segment.display}</span>
            </span>
            <span className="console-value shrink-0 font-financial text-xs font-semibold tabular-nums">
              {segment.percentage}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export interface ConsoleGaugeProps {
  /** Completion rate 0..1; values are clamped, non-finite renders nothing. */
  value: number;
  /** Formatted rate figure in the centre; defaults to whole percent. */
  display?: string;
  /** Mono kicker under the arc. */
  label: string;
  className?: string;
}

const GAUGE_RADIUS = 46;
const GAUGE_ARC_LENGTH = Math.PI * GAUGE_RADIUS;

/** Half-circle gauge for a single rate (completion %). Callers omit it when
 *  the denominator is 0; non-finite input renders null as a last resort. */
export function ConsoleGauge({ value, display, label, className }: ConsoleGaugeProps): ReactElement | null {
  if (!Number.isFinite(value)) return null;

  const clamped = Math.min(Math.max(value, 0), 1);
  const progress = `${GAUGE_ARC_LENGTH * clamped} ${GAUGE_ARC_LENGTH}`;
  const figure = display ?? `${Math.round(clamped * 100)}%`;

  return (
    <div className={className}>
      <svg viewBox="0 0 120 66" width={120} height={66} role="img" aria-label={`${label} ${figure}`}>
        <path
          d={`M ${60 - GAUGE_RADIUS} 60 A ${GAUGE_RADIUS} ${GAUGE_RADIUS} 0 0 1 ${60 + GAUGE_RADIUS} 60`}
          fill="none"
          strokeWidth={10}
          stroke="hsl(var(--border) / 0.5)"
        />
        <path
          d={`M ${60 - GAUGE_RADIUS} 60 A ${GAUGE_RADIUS} ${GAUGE_RADIUS} 0 0 1 ${60 + GAUGE_RADIUS} 60`}
          fill="none"
          strokeWidth={10}
          stroke="hsl(var(--primary))"
          strokeDasharray={progress}
        />
      </svg>
      <div className="-mt-4 flex flex-col items-center gap-1">
        <span className="console-value font-display text-xl font-bold leading-none">{figure}</span>
        <span className="console-kicker">{label}</span>
      </div>
    </div>
  );
}
