import { cn } from '@/lib/utils';
import { transactionService } from '@/services/api/transaction.service';

interface SettlementProgressBarProps {
  totalAmount: number;
  settledAmount?: number;
  remainingAmount?: number;
  status: string;
  compact?: boolean;
}

export function SettlementProgressBar({
  totalAmount,
  settledAmount = 0,
  remainingAmount,
  status,
  compact = false,
}: SettlementProgressBarProps) {
  // Calculate percentage
  const settled = settledAmount || 0;
  const remaining = remainingAmount ?? totalAmount;
  const percentage = totalAmount > 0 ? (settled / totalAmount) * 100 : 0;

  // Determine color based on status
  const getProgressColor = () => {
    if (status === 'settled' || percentage === 100) {
      return 'bg-emerald-500';
    }
    if (percentage > 0) {
      return 'bg-yellow-500';
    }
    return 'bg-gray-300';
  };

  // Determine text color
  const getTextColor = () => {
    if (status === 'settled' || percentage === 100) {
      return 'text-emerald-700';
    }
    if (percentage > 0) {
      return 'text-yellow-700';
    }
    return 'text-muted-foreground';
  };

  const progressColor = getProgressColor();
  const textColor = getTextColor();

  // Format display text
  const getDisplayText = () => {
    if (status === 'settled' || percentage === 100) {
      return compact
        ? 'Đã thanh toán'
        : `Đã thanh toán ${transactionService.formatCurrency(settled)}`;
    }
    if (percentage > 0) {
      return compact
        ? `${percentage.toFixed(0)}%`
        : `Đã thanh toán ${transactionService.formatCurrency(settled)} / ${transactionService.formatCurrency(totalAmount)}`;
    }
    return compact
      ? 'Chưa thanh toán'
      : `Còn lại ${transactionService.formatCurrency(remaining)}`;
  };

  return (
    <div className="flex flex-col gap-1.5 w-full">
      <div className="flex items-center justify-between gap-2">
        <span className={cn('text-xs font-medium', textColor)}>
          {getDisplayText()}
        </span>
        {!compact && (
          <span className={cn('text-xs font-semibold', textColor)}>
            {percentage.toFixed(0)}%
          </span>
        )}
      </div>
      <div className="relative h-2 w-full overflow-hidden rounded-full bg-gray-200">
        <div
          className={cn('h-full transition-all', progressColor)}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
