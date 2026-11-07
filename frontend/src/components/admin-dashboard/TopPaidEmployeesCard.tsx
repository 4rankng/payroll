import { memo } from 'react';
import { Trophy, UserCheck, UserX } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { ConsolePanel } from '@/components/shared/console-panel';
import { EmptyState } from '@/components/shared/EmptyState';
import { useTopPaidEmployees } from '@/hooks/api/useDashboard';
import type { TopPaidEmployeeItem } from '@/types/api/dashboard.types';
import { formatFullCurrency as formatVND } from '@/utils/formatters';
import { cn } from '@/lib/utils';

interface TopPaidEmployeesCardProps {
  month?: string; // YYYY-MM; if undefined shows all-time
  className?: string;
}

const EmployeeRow = memo(function EmployeeRow({
  item,
  maxPaid,
}: {
  item: TopPaidEmployeeItem;
  maxPaid: number;
}) {
  // Amber is rank-1 semantics only; every other rank is neutral.
  const rankStyle =
    item.rank === 1
      ? 'bg-warning/10 text-warning ring-warning/30'
      : 'bg-muted/50 text-muted-foreground ring-border/60';
  const width = maxPaid > 0 ? Math.max(4, (item.total_paid_vnd / maxPaid) * 100) : 0;

  return (
    <div className="flex items-center gap-2.5 py-1.5">
      <div
        className={cn(
          'flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px] font-bold ring-1 ring-inset',
          rankStyle,
        )}
      >
        {item.rank === 1 ? <Trophy className="h-3 w-3" aria-hidden="true" /> : <span>{item.rank}</span>}
      </div>

      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className="truncate text-[13px] font-medium text-foreground">
            {item.employee_name}
          </span>
          <span
            className={cn(
              'inline-flex shrink-0 items-center gap-1 text-[10px] font-medium',
              item.is_active ? 'text-success' : 'text-destructive',
            )}
          >
            {item.is_active ? (
              <UserCheck className="h-2.5 w-2.5" aria-hidden="true" />
            ) : (
              <UserX className="h-2.5 w-2.5" aria-hidden="true" />
            )}
            {item.is_active ? 'Đang làm' : 'Nghỉ'}
          </span>
        </div>
        {/* 56px rank bar, not a full-width progress bar. */}
        <div className="mt-1 h-1 w-14 overflow-hidden rounded-full bg-muted/60" aria-hidden="true">
          <div className="h-full rounded-full bg-primary/70" style={{ width: `${width}%` }} />
        </div>
      </div>

      <span className="console-value shrink-0 text-[11px] font-semibold text-foreground tabular-nums">
        {formatVND(item.total_paid_vnd)}
      </span>
    </div>
  );
});

export const TopPaidEmployeesCard = memo(function TopPaidEmployeesCard({
  month,
  className,
}: TopPaidEmployeesCardProps) {
  const { data, isLoading } = useTopPaidEmployees({ month, limit: 10 });

  const employees = data?.employees ?? [];
  const maxPaid = employees[0]?.total_paid_vnd ?? 0;

  return (
    <ConsolePanel
      kicker="Chi trả"
      title={`Theo nhân viên · ${month ?? 'tất cả'}`}
      isFlush
      className={className}
      bodyClassName="flex flex-1 flex-col justify-center"
    >
      {isLoading ? (
        <div className="space-y-2 p-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-2.5">
              <Skeleton className="h-6 w-6 rounded-full" />
              <Skeleton className="h-8 w-8 rounded-full" />
              <div className="flex-1 space-y-1">
                <Skeleton className="h-3 w-32" />
                <Skeleton className="h-1 w-14 rounded-full" />
              </div>
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      ) : employees.length === 0 ? (
        <EmptyState title="Chưa có dữ liệu thanh toán" size="sm" showIllustration={false} className="py-4" />
      ) : (
        <div className="divide-y divide-border/60 px-3">
          {employees.map((item) => (
            <EmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
          ))}
        </div>
      )}
    </ConsolePanel>
  );
});
