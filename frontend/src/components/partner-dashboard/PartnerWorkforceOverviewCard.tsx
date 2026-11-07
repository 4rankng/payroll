import { useMemo } from 'react';
import { Cell, Pie, PieChart, ResponsiveContainer } from 'recharts';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/shared/EmptyState';
import { ConsolePanel, ConsoleStat } from '@/components/shared/console-panel';

interface PartnerWorkforceOverviewCardProps {
  active: number;
  dropped: number;
  paid: number;
  isLoading?: boolean;
  className?: string;
}

export function PartnerWorkforceOverviewCard({
  active,
  dropped,
  paid,
  isLoading = false,
  className,
}: PartnerWorkforceOverviewCardProps) {
  return (
    <ConsolePanel
      kicker="Cơ cấu"
      title="Nhân sự"
      description="Theo trạng thái hoạt động"
      className={className}
    >
      {isLoading ? (
        <Skeleton className="h-[200px] w-full rounded-lg" />
      ) : (
        <PartnerWorkforceDonut active={active} dropped={dropped} paid={paid} />
      )}
    </ConsolePanel>
  );
}

function PartnerWorkforceDonut({
  active,
  dropped,
  paid,
}: {
  active: number;
  dropped: number;
  paid: number;
}) {
  const chartData = useMemo(
    () => [
      { name: 'Đang làm', value: active, color: 'hsl(var(--success))' },
      { name: 'Có thể nghỉ', value: dropped, color: 'hsl(var(--warning))' },
    ],
    [active, dropped],
  );
  const total = active + dropped;

  if (total === 0) {
    return (
      <div className="h-[180px] px-3">
        <EmptyState title="Chưa có dữ liệu" size="sm" showIllustration={false} className="h-full py-2" />
      </div>
    );
  }

  return (
    <div>
      <div className="relative h-[176px]" aria-label="Biểu đồ cơ cấu nhân sự">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart accessibilityLayer>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius={58}
              outerRadius={82}
              paddingAngle={3}
              dataKey="value"
              startAngle={90}
              endAngle={-270}
              stroke="none"
            >
              {chartData.map((entry) => (
                <Cell key={entry.name} fill={entry.color} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="console-value font-display text-2xl font-semibold leading-none text-foreground">
            {total.toLocaleString('vi-VN')}
          </span>
          <span className="console-kicker mt-1">Nhân viên</span>
        </div>
      </div>

      <div className="mt-2 grid grid-cols-2 gap-3 border-t border-border/70 pt-2">
        {chartData.map((item) => (
          <div key={item.name} className="min-w-0">
            <div className="flex items-center gap-1.5">
              <span
                className="h-1.5 w-1.5 shrink-0 rounded-full"
                style={{ backgroundColor: item.color }}
                aria-hidden="true"
              />
              <span className="console-kicker truncate">{item.name}</span>
            </div>
            <span className="console-value mt-0.5 block font-display text-xl font-semibold text-foreground">
              {item.value.toLocaleString('vi-VN')}
            </span>
          </div>
        ))}
      </div>

      <ConsoleStat
        className="mt-2"
        label="Đã thanh toán trong kỳ"
        value={paid.toLocaleString('vi-VN')}
        hint={cn(paid > 0 ? 'có phát sinh chi trả' : 'chưa có chi trả')}
      />
    </div>
  );
}
