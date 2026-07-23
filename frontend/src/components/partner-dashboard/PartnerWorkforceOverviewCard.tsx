import { useMemo } from 'react';
import {
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
} from 'recharts';
import { CircleDollarSign } from 'lucide-react';
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
    <section className={cn('rounded-2xl border border-border/60 bg-card p-5 shadow-soft', className)}>
      <div className="mb-2">
        <h2 className="text-base font-bold text-foreground">Cơ cấu nhân sự</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Phân loại theo trạng thái hoạt động
        </p>
      </div>
      {isLoading ? (
        <Skeleton className="h-[200px] w-full rounded-xl" />
      ) : (
        <PartnerWorkforceDonut active={active} dropped={dropped} paid={paid} />
      )}
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
      <div className="flex h-[180px] items-center justify-center text-[12px] text-muted-foreground">
        Chưa có dữ liệu
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
          <span className="text-3xl font-extrabold tabular-nums leading-none text-foreground">
            {total.toLocaleString('vi-VN')}
          </span>
          <span className="mt-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            Nhân viên
          </span>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {chartData.map((item) => (
          <div key={item.name} className="rounded-xl bg-muted/40 p-3">
            <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.color }} />
              <span>{item.name}</span>
            </div>
            <span className="mt-1 block text-lg font-bold tabular-nums text-foreground">
              {item.value.toLocaleString('vi-VN')}
            </span>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center justify-between gap-3 border-t border-border/60 pt-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <CircleDollarSign className="h-4 w-4" />
          </span>
          <span className="text-xs leading-snug text-muted-foreground">
            Đã thanh toán trong kỳ
          </span>
        </div>
        <span className="shrink-0 text-lg font-bold tabular-nums text-foreground">
          {paid.toLocaleString('vi-VN')}
        </span>
      </div>
    </div>
  );
}
