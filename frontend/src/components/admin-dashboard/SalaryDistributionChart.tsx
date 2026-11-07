import { useState, useMemo, useRef, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import {
  useSalaryDistribution,
  getCycleSections,
  buildFixedBins,
} from '@/hooks/admin-dashboard/useSalaryDistribution';
import { formatCompactVND, formatCurrency } from '@/utils/formatters';
import { ConsolePanel, ConsoleStatRow } from '@/components/shared/console-panel';
import { Skeleton } from '@/components/ui/skeleton';

interface SalaryDistributionChartProps {
  className?: string;
}

export function SalaryDistributionChart({ className }: SalaryDistributionChartProps) {
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
      { rootMargin: '200px', threshold: 0.1 },
    );
    if (cardRef.current) observer.observe(cardRef.current);
    const timer = setTimeout(() => {
      setIsVisible(true);
      observer.disconnect();
    }, 3000);
    return () => {
      observer.disconnect();
      clearTimeout(timer);
    };
  }, []);

  const { data, isLoading, isError } = useSalaryDistribution(isVisible);

  const cycleSections = useMemo(
    () => getCycleSections(data?.data as Record<string, unknown> | undefined),
    [data],
  );

  return (
    <ConsolePanel
      kicker="Phân tích chi trả"
      title="Phân bổ lương"
      className={className}
    >
      <div ref={cardRef}>
        {!isVisible || isLoading ? (
          <div className="space-y-3" aria-label="Đang tải phân bổ lương">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-16 w-full rounded-lg" />
            <Skeleton className="h-40 w-full rounded-lg" />
          </div>
        ) : isError || !data?.data ? (
          <div className="py-6 text-center text-[13px] text-destructive">
            Không thể tải dữ liệu
          </div>
        ) : cycleSections.length > 0 ? (
          <div className="space-y-5">
            {cycleSections.map((section) => (
              <HorizontalCycleSection
                key={section.cycle}
                label={section.label}
                values={section.values}
                summary={section.summary}
              />
            ))}
          </div>
        ) : (
          <div className="py-6 text-center text-[13px] text-muted-foreground">
            Không có dữ liệu cho khoảng thời gian này
          </div>
        )}
      </div>
    </ConsolePanel>
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

  const maxCount = useMemo(() => Math.max(...chartData.map((d) => d.count), 1), [chartData]);

  if (!summary) return null;

  return (
    <div>
      <p className="console-kicker">{label}</p>

      <ConsoleStatRow
        className="mb-3 mt-1.5"
        stats={[
          {
            label: 'TB',
            value: formatCompactVND(summary.mean),
            title: formatCurrency(summary.mean),
          },
          {
            label: 'Thấp nhất',
            value: formatCompactVND(summary.min),
            title: formatCurrency(summary.min),
          },
          {
            label: 'Cao nhất',
            value: formatCompactVND(summary.max),
            title: formatCurrency(summary.max),
          },
        ]}
      />

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
                width={150}
                tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))', width: 150 }}
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
                labelFormatter={(labelValue) => `Khung lương: ${labelValue}`}
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
