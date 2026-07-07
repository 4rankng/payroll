import { useState, useMemo, useRef, useEffect } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp } from 'lucide-react';
import { formatCompactCurrency, formatCurrency } from '@/utils/formatters';
import type { WalletDemandForecastResponse } from '@/types/api/wallet.types';

interface WalletDemandChartProps {
  data?: WalletDemandForecastResponse;
  isLoading?: boolean;
}

// Current period = amber headline; historical periods = muted slate/blue.
const CURRENT_COLOR = 'hsl(38 92% 50%)';
const HIST_COLORS = ['hsl(215 25% 55%)', 'hsl(200 80% 45%)'];

/**
 * Cohort line chart of cumulative advance-payment request volume per period.
 * One <Line> per period (current highlighted + last 2 completed). X-axis is the
 * cycle day label ("20/6" … "9/7"); Y-axis is cumulative VND. A dashed reference
 * line marks today's cycle day.
 */
export function WalletDemandChart({ data, isLoading }: WalletDemandChartProps) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);
  const [hidden, setHidden] = useState<Set<string>>(new Set());

  // Lazy-load via IntersectionObserver — matches FinancialChart pattern.
  useEffect(() => {
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          obs.disconnect();
        }
      },
      { rootMargin: '100px' },
    );
    if (cardRef.current) obs.observe(cardRef.current);
    return () => obs.disconnect();
  }, []);

  const periods = useMemo(() => data?.periods ?? [], [data?.periods]);
  const currentPeriod = periods.find((p) => p.is_current);

  // Pivot periods into Recharts rows keyed by cycle day. All periods share the
  // cycle-day index [1..maxCycleDay]; the X-axis label comes from the current
  // period's day_label (its calendar dates are the headline).
  const chartData = useMemo(() => {
    if (!periods.length || !currentPeriod) return [];
    const maxDay = currentPeriod.series.length;
    const lookup = new Map<string, Map<number, number>>();
    for (const p of periods) {
      const m = new Map<number, number>();
      for (const pt of p.series) m.set(pt.cycle_day, pt.amount);
      lookup.set(p.label, m);
    }
    const rows: Array<Record<string, number | string>> = [];
    for (let d = 1; d <= maxDay; d++) {
      const row: Record<string, number | string> = {
        cycle_day: d,
        day_label: currentPeriod.series[d - 1]?.day_label ?? `${d}`,
      };
      for (const p of periods) {
        const m = lookup.get(p.label);
        row[p.label] = m?.get(d) ?? 0;
      }
      rows.push(row);
    }
    return rows;
  }, [periods, currentPeriod]);

  const handleLegendClick = (value?: string | number) => {
    const key = typeof value === 'string' ? value : String(value);
    if (!key) return;
    setHidden((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  if (isLoading || !isVisible) {
    return (
      <div ref={cardRef} className="rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
        <Skeleton className="h-5 w-56 mb-2" />
        <Skeleton className="h-3.5 w-72 mb-4" />
        <Skeleton className="h-[260px] w-full" />
      </div>
    );
  }

  if (!data || !periods.length || !currentPeriod) {
    return (
      <div
        ref={cardRef}
        className="rounded-2xl border border-border/60 bg-card p-5 shadow-sm flex flex-col justify-center"
      >
        <p className="text-sm font-semibold text-foreground">Lượng yêu cầu ứng lương theo kỳ</p>
        <p className="text-xs text-muted-foreground mt-2">Chưa có dữ liệu</p>
      </div>
    );
  }

  const todayIdx = Math.min(Math.max(data.current_cycle_day, 1), currentPeriod.series.length) - 1;
  const todayLabel = currentPeriod.series[todayIdx]?.day_label;

  return (
    <div
      ref={cardRef}
      className="rounded-2xl border border-border/60 bg-card p-5 shadow-sm h-full flex flex-col"
    >
      <div className="flex items-center gap-2 mb-1">
        <TrendingUp className="h-4 w-4 text-amber-600" />
        <p className="text-sm font-semibold text-foreground">Lượng yêu cầu ứng lương theo kỳ</p>
      </div>
      <p className="text-[11.5px] text-muted-foreground mb-3">
        Theo ngày trong kỳ (20 → 9), 3 kỳ gần nhất
      </p>

      <div className="flex-1 w-full min-w-0">
        <ResponsiveContainer width="100%" height={260} minWidth={0}>
          <LineChart data={chartData} margin={{ top: 5, right: 12, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
            <XAxis
              dataKey="day_label"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              interval="preserveStartEnd"
              minTickGap={20}
            />
            <YAxis
              fontSize={10}
              tickLine={false}
              axisLine={false}
              width={56}
              tickFormatter={(v) => formatCompactCurrency(Number(v), { showSymbol: false })}
            />
            <Tooltip
              formatter={(value: number, name: string) => [formatCurrency(Number(value)), name]}
              contentStyle={{
                backgroundColor: 'hsl(var(--background))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '8px',
                fontSize: '11px',
                padding: '8px',
              }}
            />
            <Legend
              wrapperStyle={{ fontSize: '11px', paddingTop: '8px', cursor: 'pointer' }}
              iconType="line"
              iconSize={14}
              onClick={(d: { value?: string | number }) => handleLegendClick(d.value)}
            />
            {todayLabel && (
              <ReferenceLine
                x={todayLabel}
                stroke="hsl(38 92% 50%)"
                strokeDasharray="4 4"
                label={{
                  value: 'Hôm nay',
                  fontSize: 10,
                  fill: 'hsl(38 92% 45%)',
                  position: 'insideTopRight',
                }}
              />
            )}
            {periods.map((p, idx) => {
              const color = p.is_current ? CURRENT_COLOR : HIST_COLORS[idx % HIST_COLORS.length];
              return (
                <Line
                  key={p.label}
                  type="monotone"
                  dataKey={p.label}
                  stroke={color}
                  strokeWidth={p.is_current ? 2.5 : 1.5}
                  dot={false}
                  activeDot={{ r: 3 }}
                  hide={hidden.has(p.label)}
                />
              );
            })}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
