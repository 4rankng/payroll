import { useMemo } from 'react';
import type { LedgerEntry } from '@/types/api/financial.types';
import { ledgerService } from '@/services/api/ledger.service';

// Define account types for correct balance calculation
const ACCOUNT_TYPES: Record<string, 'debit' | 'credit'> = {
  asset: 'debit',
  expense: 'debit',
  salary: 'debit',
  liability: 'credit',
  equity: 'credit',
  revenue: 'credit',
  income: 'credit',
  cash: 'debit',
  'accounts-receivable': 'debit',
  'accounts-payable': 'credit',
  loan: 'credit',
};

export interface AccountBalance {
  debit: number;
  credit: number;
  balance: number;
  normalBalance: 'debit' | 'credit';
}

export function useLedgerAnalytics(entries: LedgerEntry[]) {
  const accountOptions = useMemo(() => ledgerService.getAccountOptions(), []);

  const getAccountName = (accountKey: string) => {
    const option = accountOptions.find(opt => opt.value === accountKey);
    return option ? option.label : accountKey;
  };

  const { accountBalances, totalDebits, totalCredits } = useMemo(() => {
    const balances = (entries || []).reduce(
      (acc, entry) => {
        if (!entry.account) return acc;
        if (!acc[entry.account]) {
          acc[entry.account] = {
            debit: 0,
            credit: 0,
            balance: 0,
            normalBalance: ACCOUNT_TYPES[entry.account] || 'debit',
          };
        }
        acc[entry.account].debit += entry.debit || 0;
        acc[entry.account].credit += entry.credit || 0;
        return acc;
      },
      {} as Record<string, AccountBalance>
    );

    let totalDebits = 0;
    let totalCredits = 0;

    for (const account in balances) {
      const data = balances[account];
      if (data.normalBalance === 'debit') {
        data.balance = data.debit - data.credit;
      } else {
        data.balance = data.credit - data.debit;
      }
      totalDebits += data.debit;
      totalCredits += data.credit;
    }

    return { accountBalances: balances, totalDebits, totalCredits };
  }, [entries]);

  const pnlSummary = useMemo(() => {
    const revenue =
      (accountBalances.revenue?.credit || 0) - (accountBalances.revenue?.debit || 0) +
      ((accountBalances.income?.credit || 0) - (accountBalances.income?.debit || 0));

    const expenses =
      (accountBalances.expense?.debit || 0) - (accountBalances.expense?.credit || 0) +
      ((accountBalances.salary?.debit || 0) - (accountBalances.salary?.credit || 0));

    const netProfit = revenue - expenses;

    return { revenue, expenses, netProfit };
  }, [accountBalances]);

  return {
    accountBalances,
    totalDebits,
    totalCredits,
    pnlSummary,
    getAccountName,
    accountOptions,
  };
}
