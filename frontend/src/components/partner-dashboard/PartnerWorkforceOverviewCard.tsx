import { useMemo } from 'react';
import {
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
} from 'recharts';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

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
    <div className={cn('rounded-2xl border border-border/60 bg-card p-5 shadow-soft', className)}>
      <div className="mb-4">
        <h3 className="text-sm font-bold text-foreground">Cơ cấu nhân sự</h3>
        <p className="text-[11px] text-muted-foreground mt-0.5">
          Phân loại theo trạng thái hoạt động
        </p>
      </div>
      {isLoading ? (
        <Skeleton className="h-[200px] w-full rounded-xl" />
      ) : (
        <PartnerWorkforceDonut active={active} dropped={dropped} paid={paid} />
      )}
    </div>
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
      { name: 'Đã thanh toán', value: paid, color: 'hsl(var(--primary))' },
      { name: 'Có thể nghỉ', value: dropped, color: 'hsl(var(--warning))' },
    ],
    [active, dropped, paid],
  );
  const total = active + dropped + paid;

  if (total === 0) {
    return (
      <div className="flex h-[180px] items-center justify-center text-[12px] text-muted-foreground">
        Chưa có dữ liệu
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="relative h-[180px]" aria-label="Biểu đồ cơ cấu nhân sự">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart accessibilityLayer>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius={55}
              outerRadius={80}
              paddingAngle={2}
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
        <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span className="text-2xl font-extrabold tabular-nums text-foreground leading-none">
            {(active + dropped).toLocaleString('vi-VN')}
          </span>
          <span className="text-[10px] font-semibold uppercase tracking-normal text-muted-foreground mt-1">
            Tổng nhân viên
          </span>
        </div>
      </div>
      <div className="space-y-1.5">
        {chartData.map((item) => (
          <div key={item.name} className="flex items-center justify-between text-[12px]">
            <div className="flex items-center gap-2">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.color }} />
              <span className="text-foreground">{item.name}</span>
            </div>
            <span className="font-semibold tabular-nums text-foreground">
              {item.value.toLocaleString('vi-VN')}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
