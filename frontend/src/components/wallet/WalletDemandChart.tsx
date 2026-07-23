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
import { formatCurrency, formatFullCurrency } from '@/utils/formatters';
import type { WalletDemandForecastResponse } from '@/types/api/wallet.types';

interface WalletDemandChartProps {
  data?: WalletDemandForecastResponse;
  isLoading?: boolean;
}

type WalletDemandPeriod = WalletDemandForecastResponse['periods'][number];
type ChartPeriod = WalletDemandPeriod & { displayLabel: string };

const SERIES_COLORS = ['hsl(151 87% 25%)', 'hsl(171 66% 34%)', 'hsl(215 16% 47%)'];

const getCumulativeAt = (period: WalletDemandPeriod, cycleDay: number) => {
  if (cycleDay <= 0) return 0;
  if (!period.series.length) return 0;
  if (cycleDay <= period.series.length) return period.series[cycleDay - 1]?.amount ?? 0;
  return period.series[period.series.length - 1]?.amount ?? 0;
};

const getFinalAmount = (period: WalletDemandPeriod) =>
  period.series.length ? period.series[period.series.length - 1]?.amount ?? 0 : 0;

const getRemainingFromDay = (period: WalletDemandPeriod, cycleDay: number) =>
  Math.max(0, getFinalAmount(period) - getCumulativeAt(period, cycleDay - 1));

const mean = (values: number[]) => {
  if (!values.length) return 0;
  return Math.round(values.reduce((sum, value) => sum + value, 0) / values.length);
};

const getForecastLabel = (period: WalletDemandPeriod) =>
  `Dự báo ${period.label.replace(/^Kỳ\s+/, '')}`;

/**
 * Cohort line chart of remaining net advance-payment request volume per period.
 * Backend amounts come from actual requests (request_amount - fee), not quota.
 * One <Line> per period (current + last 2 completed). X-axis is the cycle day
 * label ("20/6" … "8/7"); Y-axis is VND requested from each day through cutoff.
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

  const periods = useMemo(() => (data?.periods ?? []).slice(0, 3), [data?.periods]);
  const currentPeriod = periods.find((p) => p.is_current);
  const currentCycleDay = data?.current_cycle_day ?? 0;
  const chartPeriods = useMemo<ChartPeriod[]>(() => {
    const currentMaxDay = currentPeriod?.series.length ?? 0;
    return periods.map((period) => ({
      ...period,
      displayLabel:
        period.is_current && currentCycleDay < currentMaxDay
          ? getForecastLabel(period)
          : period.label,
    }));
  }, [currentCycleDay, currentPeriod?.series.length, periods]);

  // Pivot periods into Recharts rows keyed by cycle day. Values are remaining
  // demand from that day through cutoff, not cumulative demand since period start.
  const chartData = useMemo(() => {
    if (!chartPeriods.length || !currentPeriod) return [];
    const maxDay = currentPeriod.series.length;
    const historicalPeriods = chartPeriods.filter((p) => !p.is_current && getFinalAmount(p) > 0);
    const rows: Array<Record<string, number | string>> = [];
    for (let d = 1; d <= maxDay; d++) {
      const row: Record<string, number | string> = {
        cycle_day: d,
        day_label: currentPeriod.series[d - 1]?.day_label ?? `${d}`,
      };
      for (const p of chartPeriods) {
        row[p.displayLabel] =
          p.is_current && currentCycleDay < maxDay
            ? mean(historicalPeriods.map((hist) => getRemainingFromDay(hist, d)))
            : getRemainingFromDay(p, d);
      }
      rows.push(row);
    }
    return rows;
  }, [chartPeriods, currentCycleDay, currentPeriod]);

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
      <div ref={cardRef} className="h-full rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
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
        className="flex h-full flex-col justify-center rounded-2xl border border-border/60 bg-card p-5 shadow-sm"
      >
        <p className="text-sm font-semibold text-foreground">Nhu cầu ứng lương còn lại theo kỳ</p>
        <p className="text-xs text-muted-foreground mt-2">Chưa có dữ liệu</p>
      </div>
    );
  }

  const todayLabel =
    data.current_cycle_day >= 1 && data.current_cycle_day <= currentPeriod.series.length
      ? currentPeriod.series[data.current_cycle_day - 1]?.day_label
      : undefined;

  return (
    <div
      ref={cardRef}
      className="flex h-full flex-col rounded-2xl border border-border/60 bg-card p-5 shadow-sm"
    >
      <div className="flex items-center gap-2 mb-1">
        <TrendingUp className="h-4 w-4 text-primary" />
        <p className="text-sm font-semibold text-foreground">Nhu cầu ứng lương còn lại theo kỳ</p>
      </div>
      <p className="text-[11.5px] text-muted-foreground mb-3">
        Theo số tiền yêu cầu sau phí, 3 kỳ gần nhất
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
              width={86}
              tickFormatter={(v) => formatFullCurrency(Number(v), { showSymbol: false })}
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
                stroke="hsl(151 87% 25%)"
                strokeDasharray="4 4"
                label={{
                  value: 'Hôm nay',
                  fontSize: 11,
                  fill: 'hsl(151 87% 25%)',
                  position: 'insideTopRight',
                }}
              />
            )}
            {chartPeriods.map((p, idx) => {
              const color = SERIES_COLORS[idx % SERIES_COLORS.length];
              return (
                <Line
                  key={p.displayLabel}
                  type="monotone"
                  dataKey={p.displayLabel}
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
