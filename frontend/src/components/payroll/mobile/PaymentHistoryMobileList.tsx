import { useMemo } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Receipt, Banknote, User, Briefcase, Calendar } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { PaymentHistory } from '@/types/api/payroll.types';

interface PaymentHistoryMobileListProps {
  data: PaymentHistory[];
  isLoading?: boolean;
}

export function PaymentHistoryMobileList({ data, isLoading }: PaymentHistoryMobileListProps) {
  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'dd/MM/yyyy HH:mm', { locale: vi });
    } catch {
      return dateString;
    }
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency: 'VND',
    }).format(amount);
  };

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
      <div className="text-center py-12">
        <Receipt className="mx-auto h-12 w-12 text-muted-foreground/50" />
        <h3 className="mt-4 typography-title-large">Không tìm thấy lịch sử thanh toán</h3>
        <p className="mt-2 typography-body-medium text-muted-foreground">
          Chưa có lịch sử thanh toán nào trong khoảng thời gian này.
        </p>
      </div>
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
            <div className="flex items-center justify-between gap-3 pt-2 border-t">
              {/* Amount with icon */}
              <div className="flex items-center gap-1.5">
                <Banknote className="w-4 h-4 shrink-0 text-muted-foreground" />
                <span className="typography-currency typography-data-medium font-semibold text-emerald-600">
                  {formatCurrency(payment.total_paid_amount)}
                </span>
              </div>

              {/* Date */}
              <div className="typography-body-small text-muted-foreground flex items-center gap-1.5">
                <Calendar className="w-3.5 h-3.5 shrink-0" />
                <span>{formatDate(payment.paid_date)}</span>
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
