import { useQuery } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';

export const useOverallBalance = () => {
  return useQuery({
    queryKey: ['ledger-overall-balance'],
    queryFn: async () => {
      try {
        const result = await ledgerService.getOverallBalance();
        return result ?? 0; // Return 0 if result is null or undefined
      } catch (error) {
        console.error('Error fetching overall balance:', error);
        return 0; // Return default value on error
      }
    },
    retry: 3,
  });
};

export const useAccountBalance = (account: string, enabled = true) => {
  return useQuery({
    queryKey: ['ledger-account-balance', account],
    queryFn: () => ledgerService.getAccountBalance(account),
    enabled: enabled && !!account,
    retry: 3,
  });
};

export const useProjectBalance = (projectId: number, enabled = true) => {
  return useQuery({
    queryKey: ['ledger-project-balance', projectId],
    queryFn: () => ledgerService.getProjectBalance(projectId),
    enabled: enabled && projectId > 0,
    retry: 3,
  });
};

export const useCashFlowSummary = (fromDate: string, toDate: string, enabled = true) => {
  return useQuery({
    queryKey: ['ledger-cash-flow', fromDate, toDate],
    queryFn: async () => {
      if (!fromDate || !toDate) {
        throw new Error('fromDate and toDate are required');
      }

      try {
        const result = await ledgerService.getCashFlowSummary(fromDate, toDate);

        if (!result) {
          console.warn('Cash flow summary returned null/undefined, using default values');
        }

        return result ?? {
          start_date: fromDate,
          end_date: toDate,
          total_inflow: 0,
          total_outflow: 0,
          net_cash_flow: 0,
          opening_balance: 0,
          closing_balance: 0,
          by_account: {},
          by_project: {}
        };
      } catch (error) {
        console.error('Error fetching cash flow summary:', error);
        return {
          start_date: fromDate,
          end_date: toDate,
          total_inflow: 0,
          total_outflow: 0,
          net_cash_flow: 0,
          opening_balance: 0,
          closing_balance: 0,
          by_account: {},
          by_project: {}
        };
      }
    },
    enabled: enabled && !!fromDate && !!toDate,
    retry: 3,
  });
};

export const useLedgerSummary = (fromDate: string, toDate: string, projectId?: number, enabled = true) => {
  return useQuery({
    queryKey: ['ledger-summary', fromDate, toDate, projectId],
    queryFn: async () => {
      if (!fromDate || !toDate) {
        throw new Error('fromDate and toDate are required');
      }

      try {
        const result = await ledgerService.getLedgerSummary(fromDate, toDate, projectId);

        return result ?? {
          period: { from: fromDate, to: toDate },
          totals: {
            opening_balance: 0,
            closing_balance: 0,
            net_cashflow: 0,
          },
          by_account: {},
        };
      } catch (error) {
        console.error('Error fetching ledger summary:', error);
        return {
          period: { from: fromDate, to: toDate },
          totals: {
            opening_balance: 0,
            closing_balance: 0,
            net_cashflow: 0,
          },
          by_account: {},
        };
      }
    },
    enabled: enabled && !!fromDate && !!toDate,
    retry: 3,
  });
};
