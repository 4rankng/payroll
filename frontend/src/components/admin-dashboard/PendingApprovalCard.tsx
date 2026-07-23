import { Clock } from 'lucide-react';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { Skeleton } from '@/components/ui/skeleton';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { cn } from '@/lib/utils';

interface PendingApprovalCardProps {
  onClick: () => void;
  className?: string;
}

// A single KpiHeroCard with onClick. KpiHeroCard supplies role/tabIndex/keyboard
// handling, fixing the prior clickable-<div> a11y gap. Emerald when caught up,
// amber when work is still pending.
export const PendingApprovalCard = ({ onClick, className }: PendingApprovalCardProps) => {
  const { data: pendingData, isLoading } = useTimesheets({
    status: 'pending_approval',
    page: 1,
    pageSize: 20,
  });

  const totalRecords = pendingData?.pagination?.totalRecords || 0;

  if (isLoading) {
    return (
      <div className={cn('rounded-xl border border-border/40 bg-card p-4 lg:p-6 shadow-soft', className)}>
        <div className="space-y-2">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-8 w-16" />
        </div>
      </div>
    );
  }

  const isEmpty = totalRecords === 0;

  return (
    <KpiHeroCard
      label="Công chờ duyệt"
      value={totalRecords}
      icon={Clock}
      color={isEmpty ? 'emerald' : 'amber'}
      sublabel={isEmpty ? 'Tất cả đã được xử lý' : undefined}
      onClick={onClick}
      className={cn('h-full', className)}
    />
  );
};
