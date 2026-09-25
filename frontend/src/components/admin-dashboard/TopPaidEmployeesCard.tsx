import { memo } from 'react';
import { Trophy, UserCheck, UserX } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { EmptyState } from '@/components/shared/EmptyState';
import { useTopPaidEmployees } from '@/hooks/api/useDashboard';
import type { TopPaidEmployeeItem } from '@/types/api/dashboard.types';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

interface TopPaidEmployeesCardProps {
  month?: string; // YYYY-MM; if undefined shows all-time
}

const RANK_COLORS = ['text-amber-700', 'text-slate-500', 'text-amber-700'];
const RANK_BG = ['bg-amber-50 border-amber-200', 'bg-slate-50 border-slate-300', 'bg-amber-50/60 border-amber-100'];

const EmployeeRow = memo(function EmployeeRow({
  item,
  maxPaid,
}: {
  item: TopPaidEmployeeItem;
  maxPaid: number;
}) {
  const pct = maxPaid > 0 ? (item.total_paid_vnd / maxPaid) * 100 : 0;
  const rankColor = RANK_COLORS[item.rank - 1] ?? 'text-muted-foreground';
  const rankBg = RANK_BG[item.rank - 1] ?? '';

  return (
    <div className="flex items-center gap-3 py-2.5 border-b border-border/40 last:border-0">
      {/* Rank */}
      <div
        className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold shrink-0 border ${rankBg || 'bg-muted/40 border-border/40'} ${rankColor}`}
      >
        {item.rank <= 3 ? (
          <Trophy className="w-3 h-3" />
        ) : (
          <span>{item.rank}</span>
        )}
      </div>

      {/* Avatar + name */}
      <div className="flex items-center gap-2 flex-1 min-w-0">
        <UserAvatar name={item.employee_name} size="sm" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-xs font-medium text-foreground truncate">{item.employee_name}</span>
            {item.is_active ? (
              <Badge
                variant="outline"
                className="text-xs px-1 py-0 h-3.5 border-emerald-300 text-emerald-700 bg-emerald-50 shrink-0"
              >
                <UserCheck className="w-2.5 h-2.5 mr-0.5" />
                Đang làm
              </Badge>
            ) : (
              <Badge
                variant="outline"
                className="text-xs px-1 py-0 h-3.5 border-rose-300 text-rose-600 bg-rose-50 shrink-0"
              >
                <UserX className="w-2.5 h-2.5 mr-0.5" />
                Nghỉ
              </Badge>
            )}
          </div>
          {/* Progress bar */}
          <div className="mt-1 h-1 rounded-full bg-muted/60 overflow-hidden">
            <div
              className="h-full rounded-full bg-primary/70 transition-all duration-500"
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      </div>

      {/* Amount */}
      <div className="text-right shrink-0">
        <span className="text-xs font-semibold tabular-nums text-foreground">
          {formatVND(item.total_paid_vnd)}
        </span>
      </div>
    </div>
  );
});

export const TopPaidEmployeesCard = memo(function TopPaidEmployeesCard({
  month,
}: TopPaidEmployeesCardProps) {
  const { data, isLoading } = useTopPaidEmployees({ month, limit: 10 });

  const employees = data?.employees ?? [];
  const maxPaid = employees[0]?.total_paid_vnd ?? 0;

  return (
    <Card className="shadow-none">
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="w-6 h-6 rounded-full" />
                <Skeleton className="w-8 h-8 rounded-full" />
                <div className="flex-1 space-y-1">
                  <Skeleton className="h-3 w-32" />
                  <Skeleton className="h-1.5 w-full rounded-full" />
                </div>
                <Skeleton className="h-3 w-16" />
              </div>
            ))}
          </div>
        ) : employees.length === 0 ? (
          <EmptyState title="Chưa có dữ liệu thanh toán" size="sm" className="py-4" />
        ) : (
          <div>
            {employees.map((item) => (
              <EmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
});
