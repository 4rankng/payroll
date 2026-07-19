import { memo, useState, useCallback, useMemo } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer,
} from 'recharts';
import { TrendingUp } from 'lucide-react';
import { format } from 'date-fns';
import { Skeleton } from '@/components/ui/skeleton';
import { useProjectProfitability, useProjectWeeklyProfit } from '@/hooks/api/useDashboard';
import type { ProjectProfitabilityItem, ProjectWeeklySeries } from '@/types/api/dashboard.types';

// ─── constants ────────────────────────────────────────────────────────────────

const COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#0d9488',
  '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#0ea5e9',
];

function buildColorMap(series: ProjectWeeklySeries[]): Map<number, string> {
  const map = new Map<number, string>();
  series.forEach((s, i) => map.set(s.project_id, COLORS[i % COLORS.length]));
  return map;
}

function formatTooltipDate(label: string): string {
  try {
    return format(new Date(label), 'dd/MM/yyyy');
  } catch {
    return label;
  }
}

function fmtVND(v: number): string {
  const abs = Math.abs(v);
  const sign = v < 0 ? '-' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(1)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(0)}K`;
  return `${sign}${abs.toLocaleString('vi-VN')}`;
}

// ─── Tooltip ─────────────────────────────────────────────────────────────────

interface TooltipProps {
  active?: boolean;
  payload?: Array<{ name: string; value: number; color: string }>;
  label?: string;
}

const MiniTooltip = ({ active, payload, label }: TooltipProps) => {
  if (!active || !payload?.length) return null;
  const sorted = [...payload].filter(e => e.value !== 0).sort((a, b) => b.value - a.value);
  if (!sorted.length) return null;
  return (
    <div className="bg-card border border-border/60 rounded-xl p-2 shadow-sm max-w-[200px]">
      <p className="text-muted-foreground mb-1" style={{ fontSize: 11 }}>Lũy kế đến {label && typeof label === 'string' && label.includes('-') ? formatTooltipDate(label) : label}</p>
      {sorted.map((e, i) => (
        <div key={i} className="flex items-center justify-between gap-1.5 mb-0.5">
          <div className="flex items-center gap-1 min-w-0">
            <span className="rounded-full flex-shrink-0" style={{ width: 5, height: 5, backgroundColor: e.color }} />
            <span className="truncate text-foreground" style={{ fontSize: 11 }}>{e.name}</span>
          </div>
          <span className="font-semibold tabular-nums flex-shrink-0" style={{ fontSize: 11, color: e.color }}>{fmtVND(e.value)}</span>
        </div>
      ))}
    </div>
  );
};

// ─── Legend chips ─────────────────────────────────────────────────────────────

interface ChipProps { label: string; color: string; active: boolean; onPress: () => void; }

const Chip = memo(({ label, color, active, onPress }: ChipProps) => (
  <button
    onClick={onPress}
    className="flex items-center gap-1 whitespace-nowrap flex-shrink-0 transition-opacity active:scale-95"
    style={{ opacity: active ? 1 : 0.3, fontSize: 11, lineHeight: '16px', padding: '1px 5px' }}
  >
    <span className="rounded-full flex-shrink-0" style={{ width: 6, height: 6, backgroundColor: color }} />
    <span style={{ color: active ? 'hsl(var(--foreground))' : 'hsl(var(--muted-foreground))', fontWeight: active ? 600 : 400 }}>
      {label}
    </span>
  </button>
));
Chip.displayName = 'Chip';

// ─── Ranking row ──────────────────────────────────────────────────────────────

const RankRow = memo(({ item, idx }: { item: ProjectProfitabilityItem; idx: number }) => {
  const profit = item.net_profit_vnd;
  const profitColor = profit > 0 ? 'hsl(var(--financial-positive, 142 71% 45%))' : profit < 0 ? 'hsl(var(--financial-negative, 0 84% 60%))' : 'hsl(var(--muted-foreground))';
  return (
    <tr className="border-b border-border/40 last:border-0">
      <td className="py-2 pl-4 pr-2 text-muted-foreground tabular-nums" style={{ fontSize: 11, width: 28 }}>{idx + 1}</td>
      <td className="py-2 px-2" style={{ fontSize: 11 }}>
        <p className="font-semibold text-foreground leading-tight truncate" style={{ maxWidth: 110 }}>{item.project_name}</p>
        <p className="text-muted-foreground leading-tight truncate" style={{ fontSize: 11, maxWidth: 110 }}>{item.client_name}</p>
      </td>
      <td className="py-2 px-2 text-right tabular-nums text-muted-foreground" style={{ fontSize: 11 }}>
        {fmtVND(item.total_payout_vnd)}
      </td>
      <td className="py-2 px-2 text-right tabular-nums text-muted-foreground" style={{ fontSize: 11 }}>
        {fmtVND(item.total_received_vnd)}
      </td>
      <td className="py-2 pl-2 pr-4 text-right tabular-nums font-semibold" style={{ fontSize: 11, color: profitColor }}>
        {fmtVND(profit)}
      </td>
    </tr>
  );
});
RankRow.displayName = 'RankRow';

// ─── Main ─────────────────────────────────────────────────────────────────────

export const ProjectProfitabilityMobile = memo(() => {
  const { data: profitData, isLoading: tableLoading } = useProjectProfitability();
  const { data: weeklyData, isLoading: chartLoading } = useProjectWeeklyProfit(365);

  const [hiddenIds, setHiddenIds] = useState<Set<number>>(new Set());
  const [showAll, setShowAll] = useState(false);
  const toggleId = useCallback((id: number) => {
    setHiddenIds(prev => { const n = new Set(prev); if (n.has(id)) { n.delete(id); } else { n.add(id); } return n; });
  }, []);

  const allSeries = useMemo(
    () => (weeklyData?.series ?? []).filter(s => s.total_profit !== 0).slice(0, 8),
    [weeklyData]
  );
  const colorMap = useMemo(() => buildColorMap(allSeries), [allSeries]);

  const chartData = useMemo(() => {
    if (!weeklyData?.days?.length || !allSeries.length) return [];
    return weeklyData.days.map((day, wi) => {
      const pt: Record<string, string | number> = { week: day };
      allSeries.forEach(s => { pt[s.project_name] = s.data[wi] ?? 0; });
      return pt;
    });
  }, [weeklyData, allSeries]);

  const rankedProjects = useMemo(
    () => profitData?.projects ?? [],
    [profitData]
  );

  const visibleProjects = useMemo(
    () => showAll ? rankedProjects : rankedProjects.slice(0, 5),
    [rankedProjects, showAll]
  );

  if (tableLoading && chartLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-[190px] w-full rounded-xl" />
        <Skeleton className="h-[200px] w-full rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-3">

      {/* ── Chart card ── */}
      <div className="bg-card border border-border/60 rounded-xl overflow-hidden shadow-sm">
        <div className="flex items-center gap-2 px-4 py-2.5 bg-muted/40 border-b border-border/40">
          <TrendingUp className="h-3.5 w-3.5 text-primary shrink-0" />
          <span className="text-xs font-semibold uppercase tracking-wider text-foreground">Lợi Nhuận Tuần</span>
          <span className="text-muted-foreground ml-auto" style={{ fontSize: 11 }}>Lũy kế 84 ngày gần nhất</span>
        </div>

        {/* legend chips */}
        {allSeries.length > 0 && (
          <div className="px-3 pt-2 pb-0">
            <div className="flex gap-0.5 overflow-x-auto" style={{ scrollbarWidth: 'none' }}>
              {allSeries.map(s => (
                <Chip
                  key={s.project_id}
                  label={s.project_name}
                  color={colorMap.get(s.project_id) ?? '#94a3b8'}
                  active={!hiddenIds.has(s.project_id)}
                  onPress={() => toggleId(s.project_id)}
                />
              ))}
            </div>
          </div>
        )}

        {/* chart */}
        <div className="px-1 pb-2 pt-1">
          {chartLoading ? (
            <Skeleton className="h-[140px] w-full rounded-xl" />
          ) : !chartData.length ? (
            <div className="h-[140px] flex items-center justify-center text-muted-foreground text-xs">Chưa có dữ liệu</div>
          ) : (
            <ResponsiveContainer width="100%" height={140} minWidth={0}>
              <LineChart data={chartData} margin={{ top: 6, right: 6, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border) / 0.3)" />
                <XAxis dataKey="week" stroke="hsl(var(--muted-foreground))" fontSize={9} tickLine={false}
                  tickFormatter={v => (v as string).slice(5)} interval="preserveStartEnd" />
                <YAxis stroke="hsl(var(--muted-foreground))" fontSize={9} tickLine={false} axisLine={false}
                  tickFormatter={fmtVND} width={38} />
                <Tooltip content={<MiniTooltip />} allowEscapeViewBox={{ x: false, y: true }} position={{ y: 0 }} />
                {allSeries.map(s => (
                  <Line key={s.project_id} type="monotone" dataKey={s.project_name}
                    stroke={colorMap.get(s.project_id)} strokeWidth={1.5}
                    dot={false} activeDot={hiddenIds.has(s.project_id) ? false : { r: 2.5 }}
                    hide={hiddenIds.has(s.project_id)} />
                ))}
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* ── Ranking table ── */}
      <div className="bg-card border border-border/60 rounded-xl overflow-hidden shadow-sm">
        <div className="flex items-center gap-2 px-4 py-2.5 bg-muted/40 border-b border-border/40">
          <TrendingUp className="h-3.5 w-3.5 text-primary shrink-0" />
          <span className="text-xs font-semibold uppercase tracking-wider text-foreground">Xếp Hạng Lợi Nhuận</span>
          <span className="text-muted-foreground ml-auto" style={{ fontSize: 11 }}>12 tháng gần nhất</span>
        </div>

        {tableLoading ? (
          <div className="p-4 space-y-2">
            {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-9 w-full rounded" />)}
          </div>
        ) : !rankedProjects.length ? (
          <p className="py-6 text-center text-muted-foreground text-xs">Chưa có dữ liệu</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border/40">
                  <th className="py-1.5 pl-4 pr-2 text-left text-muted-foreground font-medium" style={{ fontSize: 11 }}>#</th>
                  <th className="py-1.5 px-2 text-left text-muted-foreground font-medium" style={{ fontSize: 11 }}>Dự án</th>
                  <th className="py-1.5 px-2 text-right text-muted-foreground font-medium" style={{ fontSize: 11 }}>Đã trả NV</th>
                  <th className="py-1.5 px-2 text-right text-muted-foreground font-medium" style={{ fontSize: 11 }}>Doanh thu</th>
                  <th className="py-1.5 pl-2 pr-4 text-right text-muted-foreground font-medium" style={{ fontSize: 11 }}>Lợi nhuận</th>
                </tr>
              </thead>
              <tbody>
                {visibleProjects.map((item, idx) => (
                  <RankRow key={item.project_id} item={item} idx={idx} />
                ))}
              </tbody>
              {rankedProjects.length > 5 && (
                <tfoot>
                  <tr>
                    <td colSpan={5} className="py-2 text-center">
                      <button
                        onClick={() => setShowAll(prev => !prev)}
                        className="text-primary text-xs font-semibold active:opacity-70"
                      >
                        {showAll ? 'Thu gọn' : `Xem tất cả (${rankedProjects.length})`}
                      </button>
                    </td>
                  </tr>
                </tfoot>
              )}
            </table>
          </div>
        )}
      </div>

    </div>
  );
});

ProjectProfitabilityMobile.displayName = 'ProjectProfitabilityMobile';
