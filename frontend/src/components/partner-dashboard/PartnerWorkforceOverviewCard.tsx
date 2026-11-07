import { useMemo } from 'react';
import {
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
} from 'recharts';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/shared/EmptyState';

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
    <section className={cn('console-panel', className)}>
      <header className="console-panel-header">
        <div className="min-w-0">
          <h2 className="console-kicker">Cơ cấu nhân sự</h2>
          <p className="mt-1 text-[11px] text-muted-foreground">
            Phân loại theo trạng thái hoạt động
          </p>
        </div>
      </header>
      <div className="console-panel-body">
        {isLoading ? (
          <Skeleton className="h-[200px] w-full rounded-lg" />
        ) : (
          <PartnerWorkforceDonut active={active} dropped={dropped} paid={paid} />
        )}
      </div>
    </section>
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
        <EmptyState title="Chưa có dữ liệu" size="sm" className="h-full py-2" />
      </div>
    );
  }

  return (
    <div>
      <div className="relative h-[190px]" aria-label="Biểu đồ cơ cấu nhân sự">
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
        <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span className="console-value text-3xl font-extrabold leading-none text-foreground">
            {total.toLocaleString('vi-VN')}
          </span>
          <span className="mt-1 text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            Nhân viên
          </span>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {chartData.map((item) => (
          <div key={item.name} className="rounded-lg bg-muted/40 p-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.color }} />
              <span>{item.name}</span>
            </div>
            <span className="console-value mt-1 block text-lg font-bold text-foreground">
              {item.value.toLocaleString('vi-VN')}
            </span>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center justify-between gap-3 border-t border-border/60 pt-3">
        <span className="min-w-0 text-xs leading-snug text-muted-foreground">
          Đã thanh toán trong kỳ
        </span>
        <span className="console-value shrink-0 text-lg font-bold text-foreground">
          {paid.toLocaleString('vi-VN')}
        </span>
      </div>
    </div>
  );
}
