import { Clock, ChevronRight } from 'lucide-react';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { cn } from '@/lib/utils';

interface PendingApprovalCardProps {
  onClick: () => void;
  className?: string;
}

export const PendingApprovalCard = ({
  onClick,
  className
}: PendingApprovalCardProps) => {
  const { data: pendingData, isLoading } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });

  const totalRecords = pendingData?.pagination?.totalRecords || 0;

  if (isLoading) {
    return (
      <div className={cn(
        "rounded-xl border border-border bg-card p-4 lg:p-6 shadow-sm",
        className
      )}>
        <div className="space-y-2">
          <div className="h-3 w-24 bg-gray-300 rounded animate-pulse" />
          <div className="h-8 w-16 bg-gray-300 rounded animate-pulse" />
        </div>
      </div>
    );
  }

  const isEmpty = totalRecords === 0;
  // Watermark tokens — green when caught up, amber when pending.
  const iconText = isEmpty ? 'text-emerald-600' : 'text-amber-600';
  const watermark = isEmpty ? 'text-emerald-500/15' : 'text-amber-500/15';

  return (
    <div
      className={cn(
        "group relative rounded-xl border border-border bg-card p-4 lg:p-6 shadow-sm overflow-hidden transition-colors",
        "cursor-pointer hover:bg-muted/40",
        className,
      )}
      onClick={onClick}
    >
      {/* Watermark — large faint icon decoration on the right side */}
      <Clock
        className={cn(
          "absolute right-3 top-1/2 -translate-y-1/2 h-16 w-16 pointer-events-none",
          "transition-transform duration-300 group-hover:scale-105",
          watermark,
        )}
        strokeWidth={1.5}
      />

      <div className="relative pr-16">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Clock className={cn("h-3.5 w-3.5 shrink-0", iconText)} strokeWidth={2.2} />
            <span className="text-xs font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight">
              Công chờ duyệt
            </span>
          </div>
          <ChevronRight className="h-4 w-4 text-gray-400 group-hover:text-muted-foreground transition-colors shrink-0" />
        </div>
        <p className="mt-1.5 text-2xl lg:text-3xl font-bold text-foreground tabular-nums leading-tight">
          {totalRecords.toLocaleString()}
        </p>
        {isEmpty && (
          <p className="mt-1 text-sm text-emerald-600">Tất cả đã được xử lý</p>
        )}
      </div>
    </div>
  );
};
