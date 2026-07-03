import { Badge } from '@/components/ui/badge';
import { Receipt } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { useMetadata } from '@/contexts';
import { transactionService } from '@/services/api/transaction.service';
import { MobilePagination } from '@/components/shared/MobilePagination';
import { cn } from '@/lib/utils';
import type { Transaction } from '@/services/api/transaction.service';

interface TransactionMobileListProps {
  transactions: Transaction[];
  isLoading?: boolean;
  onRowClick?: (transaction: Transaction) => void;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange?: (page: number) => void;
}

export function TransactionMobileList({
  transactions,
  isLoading = false,
  onRowClick,
  pagination,
  onPageChange,
}: TransactionMobileListProps) {
  const { transactionMetadata } = useMetadata();

  const fmtDate = (d: string) => {
    try { return format(new Date(d), 'dd/MM', { locale: vi }); } catch { return d; }
  };

  const fmtCurrency = (amount: number) => transactionService.formatCurrency(amount);

  const typeColors: Record<string, string> = {
    revenue: 'bg-emerald-100 text-emerald-800',
    expense: 'bg-red-100 text-red-800',
    capital: 'bg-purple-100 text-purple-800',
    write_off: 'bg-amber-100 text-amber-800',
  };

  const statusColors: Record<string, string> = {
    settled: 'bg-emerald-600 text-white',
    pending: 'border border-yellow-500 text-yellow-800 bg-yellow-50',
    partially_settled: 'border border-indigo-400 text-indigo-800 bg-indigo-50',
  };

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="bg-card rounded-xl border border-border p-4 animate-pulse">
            <div className="flex justify-between mb-2">
              <div className="h-4 bg-gray-200 rounded w-2/3" />
              <div className="h-4 bg-gray-200 rounded w-1/5" />
            </div>
            <div className="h-3 bg-muted rounded w-1/2" />
          </div>
        ))}
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <div className="text-center py-12">
        <Receipt className="mx-auto h-10 w-10 text-muted-foreground/40" />
        <p className="mt-3 text-sm font-medium text-muted-foreground">Không tìm thấy giao dịch</p>
        <p className="mt-1 text-xs text-muted-foreground">Thử điều chỉnh bộ lọc</p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-2">
        {transactions.map((tx) => {
          const isRevenue = tx.transaction_type === 'revenue';
          const typeLabel = transactionService.getTransactionTypeDisplay(tx.transaction_type, transactionMetadata);
          const statusLabel = transactionService.getStatusDisplay(tx.status, transactionMetadata);

          return (
            <div
              key={tx.id}
              onClick={() => onRowClick?.(tx)}
              className="bg-card border border-border rounded-xl p-3.5 shadow-sm card-lift transition-all cursor-pointer touch-manipulation"
              role="button"
              tabIndex={0}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') onRowClick?.(tx); }}
            >
              {/* Top row: description + amount */}
              <div className="flex items-start justify-between gap-3 mb-1.5">
                <p className="text-sm font-semibold text-foreground leading-snug line-clamp-1 flex-1">
                  {tx.description}
                </p>
                <p className={cn(
                  'text-sm font-bold tabular-nums shrink-0',
                  isRevenue ? 'text-emerald-700' : 'text-red-600'
                )}>
                  {isRevenue ? '+' : '-'}{fmtCurrency(tx.amount)}
                </p>
              </div>

              {/* Bottom row: party + date + badges */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-muted-foreground truncate max-w-[120px]">{tx.party}</span>
                <span className="text-xs text-muted-foreground">·</span>
                <span className="text-xs text-muted-foreground shrink-0">{fmtDate(tx.created_at)}</span>
                <span className={cn('text-[10px] font-medium px-1.5 py-0.5 rounded-full shrink-0', typeColors[tx.transaction_type] || 'bg-muted text-foreground')}>
                  {typeLabel}
                </span>
                <span className={cn('text-[10px] font-medium px-1.5 py-0.5 rounded-full shrink-0', statusColors[tx.status] || 'bg-muted text-foreground')}>
                  {statusLabel}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Pagination — restores desktop paging capability */}
      {pagination && pagination.totalPages > 1 && onPageChange && (
        <MobilePagination
          pagination={pagination}
          onPageChange={onPageChange}
          className="pt-4"
        />
      )}
    </>
  );
}
