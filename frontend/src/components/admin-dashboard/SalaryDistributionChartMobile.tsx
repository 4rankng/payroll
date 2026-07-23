import { useState, useMemo, useRef, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { BarChart3 } from 'lucide-react';
import { useSalaryDistribution, getCycleSections, buildFixedBins } from '@/hooks/admin-dashboard/useSalaryDistribution';
import { formatCurrency } from '@/utils/formatters';
import { Skeleton } from '@/components/ui/skeleton';

export function SalaryDistributionChartMobile() {
  const [isVisible, setIsVisible] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) { setIsVisible(true); observer.disconnect(); } },
      { rootMargin: '100px' }
    );
    if (ref.current) observer.observe(ref.current);
    return () => observer.disconnect();
  }, []);

  const { data, isLoading, isError, fromDate, toDate } = useSalaryDistribution(isVisible);

  const cycleSections = useMemo(() => getCycleSections(data?.data as Record<string, unknown> | undefined), [data]);

  if (!isVisible || isLoading) {
    return (
      <div ref={ref} className="bg-card border border-border/60 rounded-xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-2.5 bg-muted/40 border-b border-border/40">
          <BarChart3 className="h-3.5 w-3.5 text-primary shrink-0" />
          <span className="text-xs font-semibold uppercase tracking-wider text-foreground">Phân Bổ Lương</span>
        </div>
        <div className="p-4 space-y-3">
          <div className="grid grid-cols-3 gap-2">
            {[0,1,2].map(i => <Skeleton key={i} className="h-14 rounded-xl" />)}
          </div>
          <Skeleton className="h-48 rounded-xl" />
        </div>
      </div>
    );
  }

  if (isError || !data?.data) {
    return (
      <div ref={ref} className="bg-card border border-border/60 rounded-xl p-4">
        <p className="text-sm text-muted-foreground text-center py-8">Không thể tải dữ liệu</p>
      </div>
    );
  }

  const { meta } = data;

  return (
    <div ref={ref} className="bg-card border border-border/60 rounded-xl overflow-hidden">
      <div className="p-4 space-y-4">
        {cycleSections.length > 0 ? cycleSections.map((section) => (
          <MobileCycleSection key={section.cycle} section={section} />
        )) : (
          <p className="text-sm text-muted-foreground text-center py-8">Không có dữ liệu cho khoảng thời gian này</p>
        )}
      </div>
    </div>
  );
}

interface CycleSection {
  cycle: string;
  label: string;
  values: number[];
  summary: { count: number; mean: number; median: number; min: number; max: number; std_dev: number };
}

function MobileCycleSection({ section }: { section: CycleSection }) {
  const { values, summary, label } = section;

  const chartData = useMemo(() => (summary ? buildFixedBins(values) : []), [values, summary]);

  const maxCount = useMemo(() => Math.max(...chartData.map(d => d.count), 1), [chartData]);

  if (!summary) return null;

  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground mb-2">{label}</p>
      {/* Summary stats */}
      <div className="mb-3 grid grid-cols-1 gap-2 min-[520px]:grid-cols-3">
        {[
          { label: 'Trung bình', value: formatCurrency(summary.mean) },
          { label: 'Thấp nhất', value: formatCurrency(summary.min) },
          { label: 'Cao nhất', value: formatCurrency(summary.max) },
        ].map(({ label: l, value }) => (
          <div key={l} className="min-w-0 rounded-xl border border-border/30 bg-muted/30 px-3 py-2.5">
            <p className="text-xs text-muted-foreground leading-none mb-1">{l}</p>
            <p className="break-words text-xs font-bold leading-snug text-foreground tabular-nums">{value}</p>
          </div>
        ))}
      </div>

      {/* Chart */}
      {chartData.length > 0 && (
        <div
          className="w-full"
          role="img"
          aria-label={`${label}: phân bổ ${values.length} nhân viên theo khung lương. Trung bình ${formatCurrency(summary.mean)}, thấp nhất ${formatCurrency(summary.min)}, cao nhất ${formatCurrency(summary.max)}. ${chartData.map((item) => `${item.label}: ${item.count} nhân viên`).join('; ')}.`}
          style={{ height: chartData.length * 36 + 24, minWidth: 1 }}
        >
          <ResponsiveContainer width="100%" height={chartData.length * 36 + 24} minWidth={0}>
            <BarChart data={chartData} layout="vertical" margin={{ top: 0, right: 32, left: 4, bottom: 0 }} barSize={18}>
              <XAxis type="number" domain={[0, maxCount]} tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }} tickLine={false} axisLine={false} tickCount={4} />
              <YAxis type="category" dataKey="label" width={135} tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))', width: 135 }} tickLine={false} axisLine={false} />
              <Tooltip
                cursor={{ fill: 'hsl(var(--muted))', opacity: 0.5 }}
                contentStyle={{ backgroundColor: 'hsl(var(--background))', border: '1px solid hsl(var(--border))', borderRadius: '8px', fontSize: '11px', padding: '6px 10px' }}
                formatter={(value: number) => [`${value} NV`, 'Số lượng']}
                labelFormatter={(label) => `Lương: ${label}`}
              />
              <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                {chartData.map((_, i) => (
                  <Cell key={i} fill={`hsl(var(--primary) / ${0.4 + (chartData[i].count / maxCount) * 0.6})`} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
