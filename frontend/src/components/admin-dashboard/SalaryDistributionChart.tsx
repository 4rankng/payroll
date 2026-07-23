import { useState, useMemo, useRef, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { TrendingUp, ArrowDown, ArrowUp } from 'lucide-react';
import { useSalaryDistribution, getCycleSections, buildFixedBins } from '@/hooks/admin-dashboard/useSalaryDistribution';
import { formatCurrency } from '@/utils/formatters';
import { Card, CardContent } from '@/components/ui/card';

export function SalaryDistributionChart() {
  const [isVisible, setIsVisible] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: '200px', threshold: 0.1 }
    );
    if (cardRef.current) observer.observe(cardRef.current);
    const timer = setTimeout(() => {
      setIsVisible(true);
      observer.disconnect();
    }, 3000);
    return () => { observer.disconnect(); clearTimeout(timer); };
  }, []);

  const { data, isLoading, isError } = useSalaryDistribution(isVisible);

  const cycleSections = useMemo(() => getCycleSections(data?.data as Record<string, unknown> | undefined), [data]);

  // Loading state
  if (!isVisible || isLoading) {
    return (
      <Card ref={cardRef} className="shadow-none">
        <CardContent className="p-3 sm:p-4">
          <div className="h-[280px] flex items-center justify-center">
            <p className="text-sm text-muted-foreground">Đang tải dữ liệu...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  // Error state
  if (isError || !data?.data) {
    return (
      <Card ref={cardRef} className="shadow-none">
        <CardContent className="p-3 sm:p-4">
          <div className="h-[280px] flex items-center justify-center">
            <p className="text-sm text-financial-negative">Không thể tải dữ liệu</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card ref={cardRef} className="shadow-none">
      <CardContent className="p-3 sm:p-4">
        {cycleSections.length > 0 ? (
          <div className="space-y-6">
            {cycleSections.map((section) => (
              <HorizontalCycleSection key={section.cycle} label={section.label} values={section.values} summary={section.summary} />
            ))}
          </div>
        ) : (
          <div className="h-[280px] flex items-center justify-center">
            <p className="text-sm text-muted-foreground">Không có dữ liệu cho khoảng thời gian này</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface CycleSectionProps {
  label: string;
  values: number[];
  summary: {
    count: number;
    mean: number;
    median: number;
    min: number;
    max: number;
    std_dev: number;
  };
}

function HorizontalCycleSection({ label, values, summary }: CycleSectionProps) {
  const chartData = useMemo(() => (summary ? buildFixedBins(values) : []), [values, summary]);

  const maxCount = useMemo(() => Math.max(...chartData.map(d => d.count), 1), [chartData]);

  if (!summary) return null;

  return (
    <div>
      <h4 className="text-sm font-medium text-muted-foreground mb-2">{label}</h4>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-3 mb-3">
        {[
          { title: 'TB', value: formatCurrency(summary.mean), icon: TrendingUp },
          { title: 'Thấp nhất', value: formatCurrency(summary.min), icon: ArrowDown },
          { title: 'Cao nhất', value: formatCurrency(summary.max), icon: ArrowUp },
        ].map(({ title, value, icon: Icon }) => (
          <div key={title} className="p-2 rounded-xl border border-border/40 bg-muted/30">
            <div className="flex items-center gap-1.5 mb-0.5">
              <Icon className="h-3 w-3 text-muted-foreground/60" />
              <span className="text-xs font-medium text-muted-foreground">{title}</span>
            </div>
            <p className="text-sm font-semibold text-foreground">{value}</p>
          </div>
        ))}
      </div>

      {/* Horizontal bar chart */}
      {chartData.length > 0 && (
        <div
          className="w-full"
          role="img"
          aria-label={`${label}: phân bổ ${values.length} nhân viên theo khung lương. Trung bình ${formatCurrency(summary.mean)}, thấp nhất ${formatCurrency(summary.min)}, cao nhất ${formatCurrency(summary.max)}. ${chartData.map((item) => `${item.label}: ${item.count} nhân viên`).join('; ')}.`}
          style={{ height: chartData.length * 32 + 16, minWidth: 1 }}
        >
          <ResponsiveContainer width="100%" height={chartData.length * 32 + 16} minWidth={0}>
            <BarChart
              data={chartData}
              layout="vertical"
              margin={{ top: 0, right: 32, left: 4, bottom: 0 }}
              barSize={16}
            >
              <XAxis
                type="number"
                domain={[0, maxCount]}
                tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                tickLine={false}
                axisLine={false}
                tickCount={4}
              />
              <YAxis
                type="category"
                dataKey="label"
                width={90}
                tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))', width: 90 }}
                tickLine={false}
                axisLine={false}
              />
              <Tooltip
                cursor={{ fill: 'hsl(var(--muted))', opacity: 0.5 }}
                contentStyle={{
                  backgroundColor: 'hsl(var(--card))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                  fontSize: '12px',
                  boxShadow: 'none',
                }}
                formatter={(value: number) => [`${value} nhân viên`, 'Số lượng']}
                labelFormatter={(label) => `Khung lương: ${label}`}
              />
              <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                {chartData.map((_, i) => (
                  <Cell
                    key={i}
                    fill={`hsl(var(--primary) / ${0.4 + (chartData[i].count / maxCount) * 0.6})`}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
