import { EmptyState } from '@/components/shared/EmptyState';
import { formatDateTime, formatCurrency } from '@/utils/formatters';
import type { PaymentHistory } from '@/types/api/payroll.types';

// Dense ledger rows (UU bridge tokens): identity + amount on the first line,
// metadata on the second — no cards, no icon chips, no wasted vertical space.

interface PaymentHistoryMobileListProps {
  data: PaymentHistory[];
  isLoading?: boolean;
}

export function PaymentHistoryMobileList({ data, isLoading }: PaymentHistoryMobileListProps) {
  if (isLoading) {
    return (
      <div className="divide-y divide-utility-gray-300" aria-label="Đang tải lịch sử thanh toán" aria-busy="true">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="h-14 w-full animate-pulse bg-utility-gray-100" />
        ))}
      </div>
    );
  }

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
    <div className="divide-y divide-utility-gray-300" role="list" aria-label="Lịch sử thanh toán">
      {data.map((payment, index) => (
        <div
          key={`${payment.employee_id}-${payment.project_id}-${payment.paid_date}-${index}`}
          role="listitem"
          className="px-1 py-2"
        >
          <div className="flex items-baseline justify-between gap-3">
            <p className="min-w-0 break-words font-display text-[13px] font-bold leading-tight text-fg-primary">
              {payment.employee_name}
            </p>
            <p className="shrink-0 whitespace-nowrap font-financial text-[15px] font-bold tabular-nums leading-tight text-emerald-700">
              {formatCurrency(payment.total_paid_amount)}
            </p>
          </div>

          <p className="mt-0.5 break-words text-[11px] leading-snug text-fg-tertiary">
            CCCD {payment.employee_cccd || '—'}
            {payment.project_name && <> · {payment.project_name}</>}
            {payment.position && <> · {payment.position}</>}
            {' · '}
            <time dateTime={payment.paid_date}>{formatDateTime(payment.paid_date)}</time>
          </p>
        </div>
      ))}
    </div>
  );
}
