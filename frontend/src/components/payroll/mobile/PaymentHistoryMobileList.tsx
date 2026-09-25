import { useMemo } from 'react';
import { Banknote, User, Briefcase, Calendar } from 'lucide-react';
import { EmptyState } from '@/components/shared/EmptyState';
import { formatDateTime, formatCurrency } from '@/utils/formatters';
import { cn } from '@/lib/utils';
import type { PaymentHistory } from '@/types/api/payroll.types';

interface PaymentHistoryMobileListProps {
  data: PaymentHistory[];
  isLoading?: boolean;
}

export function PaymentHistoryMobileList({ data, isLoading }: PaymentHistoryMobileListProps) {
  // Loading skeleton
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="bg-card rounded-xl border p-4 animate-pulse">
            <div className="space-y-3">
              <div className="h-5 bg-muted rounded w-3/4" />
              <div className="h-4 bg-muted rounded w-1/2" />
              <div className="flex gap-4">
                <div className="h-4 bg-muted rounded w-16" />
                <div className="h-4 bg-muted rounded w-24" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  // Empty state
  if (data.length === 0) {
    return (
      <EmptyState
        title="Không tìm thấy lịch sử thanh toán"
        description="Chưa có lịch sử thanh toán nào trong khoảng thời gian này."
        size="sm"
      />
    );
  }

  return (
    <div className="space-y-3">
      {data.map((payment, index) => (
        <div
          key={`${payment.employee_id}-${payment.project_id}-${payment.paid_date}-${index}`}
          className={cn(
            "border border-border rounded-xl bg-card p-4 shadow-sm"
          )}
        >
          <div className="space-y-3">
            {/* Employee Name - Bold and prominent */}
            <div className="typography-body-large font-semibold text-foreground">
              {payment.employee_name}
            </div>

            {/* CCCD */}
            <div className="typography-body-medium text-muted-foreground flex items-center gap-2">
              <User className="w-4 h-4 shrink-0" />
              <span>CCCD: {payment.employee_cccd}</span>
            </div>

            {/* Project and Position */}
            <div className="flex items-start gap-3 flex-wrap">
              <div className="typography-body-medium text-muted-foreground flex items-center gap-1.5 flex-1 min-w-0">
                <Briefcase className="w-4 h-4 shrink-0" />
                <span className="truncate">{payment.project_name}</span>
              </div>
              <div className="typography-body-small text-muted-foreground px-2 py-0.5 bg-muted/50 rounded">
                {payment.position}
              </div>
            </div>

            {/* Amount and Date Row */}
            <div className="grid grid-cols-1 gap-2 border-t pt-2 min-[380px]:flex min-[380px]:items-center min-[380px]:justify-between min-[380px]:gap-3">
              {/* Amount with icon */}
              <div className="flex min-w-0 items-center gap-1.5">
                <Banknote className="w-4 h-4 shrink-0 text-muted-foreground" />
                <span className="typography-currency typography-data-medium break-words font-semibold text-emerald-700">
                  {formatCurrency(payment.total_paid_amount)}
                </span>
              </div>

              {/* Date */}
              <div className="typography-body-small flex min-w-0 items-center gap-1.5 text-muted-foreground min-[380px]:justify-end">
                <Calendar className="w-3.5 h-3.5 shrink-0" />
                <span className="break-words">{formatDateTime(payment.paid_date)}</span>
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
