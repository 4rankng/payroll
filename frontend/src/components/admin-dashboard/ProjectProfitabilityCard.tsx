import { memo, useState, useCallback, useMemo, useRef } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer,
  type MouseHandlerDataParam,
} from 'recharts';
import { TrendingUp, TrendingDown, BarChart2, Eye, Users } from 'lucide-react';
import { format } from 'date-fns';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { DashboardSectionHeader } from '@/components/admin-dashboard/DashboardSectionHeader';
import { useProjectProfitability, useProjectWeeklyProfit } from '@/hooks/api/useDashboard';
import type { ProjectProfitabilityItem, ProjectWeeklySeries } from '@/types/api/dashboard.types';

// ─── constants ────────────────────────────────────────────────────────────────

const CHART_COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#0d9488',
  '#06b6d4', '#64748b', '#84cc16', '#f97316', '#0ea5e9',
];

function buildColorMap(series: ProjectWeeklySeries[]): Map<number, string> {
  const map = new Map<number, string>();
  series.forEach((s, i) => map.set(s.project_id, CHART_COLORS[i % CHART_COLORS.length]));
  return map;
}

function buildNameToIdMap(series: ProjectWeeklySeries[]): Map<string, number> {
  const map = new Map<string, number>();
  series.forEach(s => map.set(s.project_name, s.project_id));
  return map;
}

function formatTooltipDate(label: string): string {
  try {
    return format(new Date(label), 'dd/MM/yyyy');
  } catch {
    return label;
  }
}

// ─── helpers ─────────────────────────────────────────────────────────────────

import { formatFullCurrency as formatVND } from '@/utils/formatters';

// ─── Rank badge ───────────────────────────────────────────────────────────────

function RankBadge({ rank }: { rank: number }) {
  if (rank === 1) return (
    <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-amber-100 text-amber-700 text-xs font-bold">1</span>
  );
  if (rank === 2) return (
    <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-slate-100 text-muted-foreground text-xs font-bold">2</span>
  );
  if (rank === 3) return (
    <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-orange-100 text-orange-600 text-xs font-bold">3</span>
  );
  return <span className="text-xs text-muted-foreground tabular-nums w-6 text-center inline-block">{rank}</span>;
}

interface ChartTooltipProps {
  active?: boolean;
  payload?: Array<{ name: string; value: number; color: string }>;
  label?: string;
}

const ChartTooltip = ({ active, payload, label }: ChartTooltipProps) => {
  if (!active || !payload?.length) return null;
  const sorted = [...payload].filter(e => e.value !== 0).sort((a, b) => b.value - a.value);
  if (!sorted.length) return null;
  return (
    <div className="bg-card border border-border rounded-xl p-3 text-xs max-w-[220px]">
      <p className="text-muted-foreground mb-2 font-medium tracking-wide uppercase" style={{ fontSize: 11 }}>
        Lũy kế đến {label && typeof label === 'string' && label.includes('-') ? formatTooltipDate(label) : label}
      </p>
      {sorted.map((entry, i) => (
        <div key={i} className="flex items-center justify-between gap-3 py-0.5">
          <div className="flex items-center gap-1.5 min-w-0">
            <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: entry.color }} />
            <span className="truncate text-muted-foreground">{entry.name}</span>
          </div>
          <span className="font-semibold whitespace-nowrap tabular-nums" style={{ color: entry.color }}>
            {formatVND(entry.value)}
          </span>
        </div>
      ))}
    </div>
  );
};

// ─── Table row ────────────────────────────────────────────────────────────────

interface ProfitRowProps {
  item: ProjectProfitabilityItem;
  isHighlighted?: boolean;
  onHover?: (id: number | null) => void;
}

const ProfitRow = memo(({ item, isHighlighted, onHover }: ProfitRowProps) => {
  const isPositive = item.net_profit_vnd > 0;
  const isNegative = item.net_profit_vnd < 0;

  return (
    <tr
      className={`group border-b border-border/20 last:border-0 transition-colors duration-150
        ${isHighlighted ? 'bg-primary/5 ring-1 ring-primary/10' : 'hover:bg-muted/30'}`}
      onMouseEnter={() => onHover?.(item.project_id)}
      onMouseLeave={() => onHover?.(null)}
    >
      <td className="py-2 pl-3 pr-2 w-8">
        <RankBadge rank={item.rank} />
      </td>
      <td className="py-2 px-2">
        <div className="font-medium text-foreground leading-tight text-xs">{item.project_name}</div>
        <div className="text-[11px] text-muted-foreground mt-0.5">{item.client_name}</div>
      </td>
      <td className="py-2 px-2 text-center">
        <div className="inline-flex items-center gap-1 text-[11px] text-muted-foreground">
          <Users className="h-3 w-3 text-muted-foreground/50" />
          <span className="tabular-nums font-medium">{item.employee_count}</span>
        </div>
      </td>
      <td className="py-2 pl-2 pr-3 text-right min-w-[90px]">
        <div className={`text-xs font-semibold tabular-nums flex items-center justify-end gap-1
          ${isPositive ? 'text-financial-positive' : isNegative ? 'text-financial-negative' : 'text-muted-foreground'}`}>
          {isPositive ? <TrendingUp className="h-3 w-3" /> : isNegative ? <TrendingDown className="h-3 w-3" /> : null}
          {formatVND(item.net_profit_vnd)}
        </div>
      </td>
    </tr>
  );
});
ProfitRow.displayName = 'ProfitRow';

// ─── Skeleton rows for "load more" ────────────────────────────────────────────

const SkeletonRows = ({ count }: { count: number }) => (
  <>
    {Array.from({ length: count }).map((_, i) => (
      <tr key={i} className="border-b border-border/20 last:border-0">
        <td className="py-2 pl-3 pr-2 w-8">
          <div className="w-6 h-6 rounded-full bg-muted animate-shimmer bg-[length:200%_100%]
            [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
        </td>
        <td className="py-2 px-2">
          <div className="h-3 w-28 rounded bg-muted animate-shimmer bg-[length:200%_100%]
            [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)] mb-1" />
          <div className="h-2.5 w-16 rounded bg-muted animate-shimmer bg-[length:200%_100%]
            [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
        </td>
        <td className="py-2 px-2 text-center">
          <div className="h-3 w-6 rounded bg-muted animate-shimmer bg-[length:200%_100%]
            [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)] mx-auto" />
        </td>
        <td className="py-2 pl-2 pr-3 text-right">
          <div className="h-3 w-16 rounded bg-muted animate-shimmer bg-[length:200%_100%]
            [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)] ml-auto" />
        </td>
      </tr>
    ))}
  </>
);

// ─── Main card ────────────────────────────────────────────────────────────────

const INITIAL_VISIBLE_ROWS = 6;

export const ProjectProfitabilityCard = memo(() => {
  const { data: profitData, isLoading: tableLoading } = useProjectProfitability();
  const { data: weeklyData, isLoading: chartLoading } = useProjectWeeklyProfit(365);

  const [hiddenIds, setHiddenIds] = useState<Set<number>>(new Set());
  // focusedId: when a legend chip is clicked, all other lines dim to 20% opacity
  const [focusedId, setFocusedId] = useState<number | null>(null);
  const [showAllRows, setShowAllRows] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hoveredProjectId, setHoveredProjectId] = useState<number | null>(null);
  const loadMoreTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const toggleId = useCallback((id: number) => {
    setHiddenIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }, []);

  /**
   * Legend chip click:
   * - If already focused on this project → clear focus (show all)
   * - Otherwise → focus this project (dim others to 20%)
   */
  const handleLegendClick = useCallback((id: number) => {
    setFocusedId(prev => (prev === id ? null : id));
  }, []);

  const showAll = useCallback(() => {
    setHiddenIds(new Set());
    setFocusedId(null);
  }, []);

  /** Simulate skeleton loading when "Xem thêm" is clicked */
  const handleShowMore = useCallback(() => {
    setIsLoadingMore(true);
    if (loadMoreTimerRef.current) clearTimeout(loadMoreTimerRef.current);
    loadMoreTimerRef.current = setTimeout(() => {
      setIsLoadingMore(false);
      setShowAllRows(true);
    }, 600);
  }, []);

  const allSeries = useMemo(() => weeklyData?.series.slice(0, 8) ?? [], [weeklyData]);
  const colorMap = useMemo(() => buildColorMap(allSeries), [allSeries]);
  const nameToId = useMemo(() => buildNameToIdMap(allSeries), [allSeries]);

  const chartData = useMemo(() => {
    if (!weeklyData?.days?.length || !allSeries.length) return [];
    return weeklyData.days.map((day, wi) => {
      const point: Record<string, string | number> = { week: day };
      allSeries.forEach(s => { point[s.project_name] = s.data[wi] ?? 0; });
      return point;
    });
  }, [weeklyData, allSeries]);

  if (tableLoading && chartLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-[300px] w-full rounded-xl" />
        <Skeleton className="h-[320px] w-full rounded-xl" />
      </div>
    );
  }

  const hasFocus = focusedId !== null;
  const hasHidden = hiddenIds.size > 0;

  return (
    <div className="space-y-4">
      {/* ── Weekly line chart ── */}
      <Card className="shadow-none">
        <CardHeader className="p-3 sm:p-4 pb-3">
          <div className="flex items-center justify-between gap-2">
            {(hasHidden || hasFocus) && (
              <button
                onClick={showAll}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors duration-150"
              >
                <Eye className="h-3 w-3" />
                Hiện tất cả
              </button>
            )}
          </div>

          {/* Legend chips — clicking one dims all others to 20% (cross-fade) */}
          {allSeries.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {allSeries.map(s => {
                const hidden = hiddenIds.has(s.project_id);
                const dimmed = hasFocus && focusedId !== s.project_id;
                const color = colorMap.get(s.project_id) ?? '#94a3b8';
                return (
                  <button
                    key={s.project_id}
                    type="button"
                    aria-pressed={!hidden && focusedId === s.project_id}
                    onClick={() => {
                      if (hidden) {
                        toggleId(s.project_id);
                      } else {
                        handleLegendClick(s.project_id);
                      }
                    }}
                    className={`flex min-h-11 items-center gap-1.5 rounded-xl border px-2 py-1 text-xs select-none
                      transition-all duration-200
                      ${hidden
                        ? 'border-border/30 bg-muted/30 text-muted-foreground/40'
                        : dimmed
                          ? 'border-border/20 bg-card text-muted-foreground/40 opacity-50'
                          : 'border-border bg-card text-foreground hover:border-primary/30'
                      }`}
                    title={hidden ? 'Hiện dòng này' : hasFocus && focusedId === s.project_id ? 'Bỏ lọc' : 'Lọc dự án này'}
                  >
                    <span
                      className="w-2 h-2 rounded-sm flex-shrink-0 transition-colors duration-200"
                      style={{ backgroundColor: hidden || dimmed ? 'hsl(var(--muted-foreground) / 0.3)' : color }}
                    />
                    <span className={hidden ? 'line-through' : ''}>{s.project_name}</span>
                  </button>
                );
              })}
            </div>
          )}
        </CardHeader>

        <CardContent className="p-3 sm:p-4 pt-0">
          {chartLoading ? (
            <Skeleton className="h-[240px] w-full rounded-xl" />
          ) : !chartData.length ? (
            <div className="h-[240px] flex items-center justify-center text-muted-foreground text-sm">
              Chưa có dữ liệu
            </div>
          ) : (
            <div
              role="img"
              aria-label="Biểu đồ đường thể hiện lợi nhuận lũy kế theo tuần của các dự án trong 12 tháng gần nhất. Có thể dùng các nút tên dự án phía trên để lọc dữ liệu."
            >
              <ResponsiveContainer width="100%" height={240} minWidth={0}>
                <LineChart
                data={chartData}
                margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
                onMouseMove={(e) => {
                  // Recharts 3.x removed `activePayload` from the public
                  // MouseHandlerDataParam type, but the runtime still passes it.
                  // Augment the type locally so we don't need `any`.
                  const state = e as MouseHandlerDataParam & {
                    activePayload?: Array<{ name?: string }>;
                  };
                  if (state?.activePayload?.length) {
                    const name = state.activePayload[0]?.name;
                    if (name) {
                      const id = nameToId.get(name);
                      if (id !== undefined && id !== hoveredProjectId) setHoveredProjectId(id);
                    }
                  }
                }}
                onMouseLeave={() => setHoveredProjectId(null)}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border) / 0.3)" vertical={false} />
                <XAxis
                  dataKey="week"
                  stroke="hsl(var(--muted-foreground))"
                  fontSize={10}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={v => { const s = v as string; const d = new Date(s); return isNaN(d.getTime()) ? s : `${d.getDate().toString().padStart(2,'0')}/${(d.getMonth()+1).toString().padStart(2,'0')}`; }}
                />
                <YAxis
                  stroke="hsl(var(--muted-foreground))"
                  fontSize={10}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v: number) => formatVND(v, { showSymbol: false })}
                  width={86}
                />
                <Tooltip
                  content={<ChartTooltip />}
                  cursor={{ stroke: 'hsl(var(--border))', strokeWidth: 1, strokeDasharray: '4 2' }}
                />
                {allSeries.map(s => {
                  const isHidden = hiddenIds.has(s.project_id);
                  const isFocused = focusedId === s.project_id;
                  const isDimmed = hasFocus && !isFocused;
                  const isHovered = hoveredProjectId === s.project_id;

                  // Opacity: dimmed lines go to 0.2 (cross-fade effect)
                  const opacity = isHidden ? 0 : isDimmed ? 0.2 : 1;
                  const strokeWidth = isHidden ? 0 : (isFocused || isHovered) ? 3 : 1.8;

                  return (
                    <Line
                      key={s.project_id}
                      type="monotone"
                      dataKey={s.project_name}
                      stroke={colorMap.get(s.project_id)}
                      strokeWidth={strokeWidth}
                      strokeOpacity={opacity}
                      dot={false}
                      activeDot={
                        isHidden
                          ? false
                          : {
                              r: (isFocused || isHovered) ? 5 : 3,
                              strokeWidth: 0,
                              // pulse scale via inline style — activeDot renders as SVG circle
                              style: { transition: 'r 0.15s ease' },
                            }
                      }
                      hide={isHidden}
                      style={{ transition: 'stroke-width 0.2s ease, stroke-opacity 0.25s ease' }}
                      isAnimationActive={true}
                      animationDuration={800}
                      animationEasing="ease-in-out"
                    />
                  );
                })}
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ── Ranking table ── */}
      <DashboardSectionHeader title="Xếp Hạng Lợi Nhuận" subtitle="12 tháng gần nhất" icon={TrendingUp} />
      <Card className="shadow-none">
        <CardContent className="p-0">
          {tableLoading ? (
            <div className="px-3 pb-3 space-y-2">
              {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10 w-full rounded" />)}
            </div>
          ) : !profitData?.projects?.length ? (
            <div className="py-8 text-center text-muted-foreground text-xs">Chưa có dữ liệu dự án</div>
          ) : (
            <>
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border/40">
                    <th className="py-2 pl-3 pr-2 text-left text-[11px] font-semibold text-muted-foreground uppercase tracking-wider w-8">#</th>
                    <th className="py-2 px-2 text-left text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">Dự án</th>
                    <th className="py-2 px-2 text-center text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">NV</th>
                    <th className="py-2 pl-2 pr-3 text-right text-[11px] font-semibold text-foreground uppercase tracking-wider">Lợi nhuận</th>
                  </tr>
                </thead>
                <tbody>
                  {profitData.projects
                    .slice(0, showAllRows ? undefined : INITIAL_VISIBLE_ROWS)
                    .map(item => (
                      <ProfitRow
                        key={item.project_id}
                        item={item}
                        isHighlighted={hoveredProjectId === item.project_id}
                        onHover={setHoveredProjectId}
                      />
                    ))}
                  {/* Skeleton rows while "load more" is in flight */}
                  {isLoadingMore && (
                    <SkeletonRows count={Math.min(profitData.projects.length - INITIAL_VISIBLE_ROWS, 6)} />
                  )}
                </tbody>
              </table>

              {!showAllRows && !isLoadingMore && profitData.projects.length > INITIAL_VISIBLE_ROWS && (
                <div className="px-3 py-2 border-t border-border/20">
                  <Button
                    onClick={handleShowMore}
                    variant="ghost"
                    size="sm"
                    className="w-full text-xs text-muted-foreground hover:text-foreground"
                  >
                    Xem thêm {profitData.projects.length - INITIAL_VISIBLE_ROWS} dự án
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
});

ProjectProfitabilityCard.displayName = 'ProjectProfitabilityCard';
