import { useMemo } from 'react';
import type { Transaction } from '@/services/api/transaction.service';

interface TransactionSummary {
  total_revenue: number;
  total_expense: number;
  net_balance: number;
  total_receivable: number;
  total_payable: number;
  period: {
    from: string;
    to: string;
  };
}

export function useTransactionSummary(
  transactions: Transaction[] | undefined,
  fromDate: string,
  toDate: string
): { summary: TransactionSummary | undefined; isLoading: boolean } {
  const summary = useMemo(() => {
    if (!transactions || transactions.length === 0) {
      return undefined;
    }

    const total_revenue = transactions
      .filter(t => t.transaction_type === 'revenue')
      .reduce((sum, t) => sum + t.amount, 0);

    const total_expense = transactions
      .filter(t => t.transaction_type === 'expense')
      .reduce((sum, t) => sum + t.amount, 0);

    const total_receivable = transactions
      .filter(t => t.transaction_type === 'revenue' && t.status === 'pending')
      .reduce((sum, t) => sum + t.amount, 0);

    const total_payable = transactions
      .filter(t => t.transaction_type === 'expense' && t.status === 'pending')
      .reduce((sum, t) => sum + t.amount, 0);

    const net_balance = total_revenue - total_expense;

    return {
      total_revenue,
      total_expense,
      net_balance,
      total_receivable,
      total_payable,
      period: {
        from: fromDate,
        to: toDate,
      },
    };
  }, [transactions, fromDate, toDate]);

  return {
    summary,
    isLoading: !transactions,
  };
}