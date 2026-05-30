import { useMemo } from 'react';
import { SettlementCard } from './SettlementCard';
import { useUsersByIds } from '@/hooks/api/useUsers';
import { transactionService, getTransactionRemainingAmount } from '@/services/api/transaction.service';
import type { Transaction } from '@/services/api/transaction.service';
import { Receipt, TrendingUp, TrendingDown } from 'lucide-react';

interface SettlementHistorySectionProps {
  transaction: Transaction;
}

export function SettlementHistorySection({ transaction }: SettlementHistorySectionProps) {
  const settlements = useMemo(() => transaction.settlements || [], [transaction.settlements]);

  // Batch fetch all users who created settlements
  const userIds = useMemo(() => {
    const ids: number[] = [];
    settlements.forEach(settlement => {
      if (settlement.created_by) {
        ids.push(settlement.created_by);
      }
    });
    return Array.from(new Set(ids)); // Remove duplicates
  }, [settlements]);

  const { data: userMap, isLoading: isLoadingUsers } = useUsersByIds(userIds, {
    enabled: userIds.length > 0,
  });

  // Calculate amounts
  const totalAmount = transaction.amount;
  const settledAmount = transaction.settled_amount || 0;
  const pendingAmount = getTransactionRemainingAmount(transaction);

  // Hide entire section if fully paid with no settlement history
  const isFullyPaidWithNoHistory = settlements.length === 0 && totalAmount === settledAmount;

  if (isFullyPaidWithNoHistory) {
    return null;
  }

  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="grid grid-cols-3 gap-3 p-3 bg-muted/30 rounded-xl">
        <div className="text-center">
          <p className="text-xs text-muted-foreground mb-1">Tổng</p>
          <p className="text-sm font-semibold">
            {transactionService.formatCurrency(totalAmount)}
          </p>
        </div>
        <div className="text-center border-l border-r">
          <p className="text-xs text-emerald-700 mb-1 flex items-center justify-center gap-1">
            <TrendingUp className="h-3 w-3" />
            Đã TT
          </p>
          <p className="text-sm font-semibold text-emerald-700">
            {transactionService.formatCurrency(settledAmount)}
          </p>
        </div>
        <div className="text-center">
          <p className="text-xs text-red-700 mb-1 flex items-center justify-center gap-1">
            <TrendingDown className="h-3 w-3" />
            Còn lại
          </p>
          <p className="text-sm font-semibold text-red-700">
            {transactionService.formatCurrency(pendingAmount)}
          </p>
        </div>
      </div>

      {/* Section Header */}
      <div className="flex items-center gap-2">
        <Receipt className="h-5 w-5 text-muted-foreground" />
        <h3 className="text-lg font-semibold">Lịch sử thanh toán</h3>
      </div>

      {/* Settlement List */}
      <div>
        {settlements.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Receipt className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">Chưa có thanh toán nào</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-3">
              {settlements.map((settlement) => (
                <SettlementCard
                  key={settlement.id}
                  settlement={settlement}
                  userMap={isLoadingUsers ? undefined : userMap}
                />
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
