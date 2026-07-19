import { ResponsiveTable } from '@/components/ui/responsive-table';
import { ColumnDef } from '@tanstack/react-table';
import { useMetadata } from '@/contexts';
import { transactionService, getTransactionRemainingAmount } from '@/services/api/transaction.service';
import { formatDate, formatCurrency } from '@/utils/formatters';
import type { Transaction } from '@/services/api/transaction.service';
import type { MobileField, RowAction } from '@/components/ui/mobile-table';
import { useMemo, useCallback } from 'react';
import { cn } from '@/lib/utils';
import type { SortingState } from '@tanstack/react-table';

interface TransactionTableProps {
  transactions: Transaction[];
  onRowClick?: (transaction: Transaction) => void;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  sorting?: SortingState;
  onSortingChange?: (sorting: SortingState) => void;
}

// Badge styles keyed by settlement status — mirrors the strip/legend colors
const STATUS_BADGE: Record<string, { badge: string; remaining: string }> = {
  settled:           { badge: 'bg-emerald-50 text-emerald-700', remaining: 'text-emerald-600/60' },
  pending:           { badge: 'bg-amber-50 text-amber-700',     remaining: 'text-amber-600/60'   },
  partially_settled: { badge: 'bg-teal-50 text-teal-700',   remaining: 'text-teal-600/60'  },
};
const FALLBACK_BADGE = { badge: 'bg-slate-100 text-foreground', remaining: 'text-muted-foreground/60' };

// Color palette keyed by transaction type — chip only
const TYPE_COLORS: Record<string, string> = {
  revenue:   'bg-emerald-50 text-emerald-700',
  expense:   'bg-red-50 text-red-700',
  capital:   'bg-teal-50 text-teal-700',
  write_off: 'bg-amber-50 text-amber-700',
};

// Compact inline badge — no border-radius pill, just a tight label
function TypeChip({ type, label }: { type: string; label: string }) {
  return (
    <span className={cn(
      'inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium leading-none whitespace-nowrap',
      TYPE_COLORS[type] ?? 'bg-slate-100 text-muted-foreground',
    )}>
      {label}
    </span>
  );
}

// Strip color for transaction status — mirrors timesheet strip design
const TRANSACTION_STRIP_BASE = "relative before:absolute before:left-0 before:top-2 before:bottom-2 before:w-1 before:content-[''] before:rounded-r-full";
const TRANSACTION_STRIP_COLOR: Record<string, string> = {
  settled:           'before:bg-emerald-500',
  pending:           'before:bg-amber-400',
  partially_settled: 'before:bg-teal-500',
};

export function TransactionTable({
  transactions,
  onRowClick,
  pagination,
  onPageChange,
  onPageSizeChange,
  sorting,
  onSortingChange,
}: TransactionTableProps) {
  const { transactionMetadata } = useMetadata();

  const getTypeLabel = useCallback((type: string) =>
    transactionService.getTransactionTypeDisplay(type, transactionMetadata),
  [transactionMetadata]);

  const getRowClassName = useCallback((txn: Transaction) => {
    const color = TRANSACTION_STRIP_COLOR[txn.status] ?? 'before:bg-slate-300';
    return `${TRANSACTION_STRIP_BASE} ${color}`;
  }, []);

  const columns = useMemo<ColumnDef<Transaction>[]>(() => [
    {
      accessorKey: 'created_at',
      header: 'Ngày tạo',
      size: 90,
      maxSize: 90,
      cell: ({ row }) => (
        <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
    {
      accessorKey: 'description',
      header: 'Diễn giải',
      size: 220,
      cell: ({ row }) => (
        <span className="text-sm text-foreground line-clamp-2 leading-snug">
          {row.original.description}
        </span>
      ),
    },
    {
      accessorKey: 'transaction_type',
      header: 'Loại',
      size: 100,
      maxSize: 120,
      cell: ({ row }) => (
        <TypeChip type={row.original.transaction_type} label={getTypeLabel(row.original.transaction_type)} />
      ),
    },
    {
      accessorKey: 'party',
      header: 'Đối tượng',
      size: 130,
      maxSize: 180,
      cell: ({ row }) => (
        <span className="text-sm text-foreground">{row.original.party}</span>
      ),
    },
    {
      accessorKey: 'amount',
      header: () => <div className="text-right">Số tiền</div>,
      size: 140,
      maxSize: 160,
      cell: ({ row }) => {
        const remaining = getTransactionRemainingAmount(row.original);
        const { badge, remaining: remainingColor } = STATUS_BADGE[row.original.status] ?? FALLBACK_BADGE;

        return (
          <div className="flex flex-col items-end gap-0.5">
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-sm font-semibold tabular-nums', badge)}>
              {formatCurrency(row.original.amount)}
            </span>
            {row.original.status === 'partially_settled' && remaining > 0 && (
              <span className={cn('text-xs tabular-nums', remainingColor)}>
                Còn {formatCurrency(remaining)}
              </span>
            )}
          </div>
        );
      },
    },
  ], [getTypeLabel]);

  const mobileFields = useMemo<MobileField<Transaction>[]>(() => [
    {
      key: 'created_at',
      label: 'Ngày',
      priority: 1,
      render: (txn) => (
        <span className="text-xs text-muted-foreground tabular-nums">{formatDate(txn.created_at)}</span>
      ),
    },
    {
      key: 'description',
      label: 'Diễn giải',
      priority: 1,
      render: (txn) => (
        <span className="text-sm text-foreground line-clamp-2">{txn.description}</span>
      ),
    },
    {
      key: 'party',
      label: 'Đối tượng',
      priority: 2,
      render: (txn) => (
        <span className="text-sm text-muted-foreground">{txn.party}</span>
      ),
    },
    {
      key: 'transaction_type',
      label: 'Loại',
      priority: 2,
      render: (txn) => (
        <TypeChip type={txn.transaction_type} label={getTypeLabel(txn.transaction_type)} />
      ),
    },
    {
      key: 'amount',
      label: 'Số tiền',
      priority: 2,
      render: (txn) => {
        const remaining = getTransactionRemainingAmount(txn);
        const { badge, remaining: remainingColor } = STATUS_BADGE[txn.status] ?? FALLBACK_BADGE;
        return (
          <div className="flex flex-col items-end gap-0.5">
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-sm font-semibold tabular-nums', badge)}>
              {formatCurrency(txn.amount)}
            </span>
            {txn.status === 'partially_settled' && remaining > 0 && (
              <span className={cn('text-xs tabular-nums', remainingColor)}>
                Còn {formatCurrency(remaining)}
              </span>
            )}
          </div>
        );
      },
    },
  ], [getTypeLabel]);

  const mobileRowActions = useMemo<RowAction<Transaction>[]>(() => [
    { label: 'Xem chi tiết', onClick: (row) => onRowClick?.(row) },
  ], [onRowClick]);

  return (
    <div className="space-y-2">
      {/* Status legend */}
      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        {[
          { dot: 'bg-emerald-500', label: 'Đã thanh toán' },
          { dot: 'bg-amber-400',   label: 'Chờ thanh toán' },
          { dot: 'bg-teal-500',    label: 'Thanh toán thiếu' },
        ].map(({ dot, label }) => (
          <div key={label} className="flex items-center gap-1">
            <div className={cn('w-1.5 h-1.5 rounded-full shrink-0', dot)} />
            <span>{label}</span>
          </div>
        ))}
      </div>
      <ResponsiveTable
        data={transactions}
        columns={columns}
        mobileFields={mobileFields}
        rowActions={mobileRowActions}
        onRowClick={onRowClick}
        emptyState="Không có giao dịch nào"
        pagination={pagination ?? undefined}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        sorting={sorting}
        onSortingChange={onSortingChange}
        getRowClassName={getRowClassName}
      />
    </div>
  );
}
