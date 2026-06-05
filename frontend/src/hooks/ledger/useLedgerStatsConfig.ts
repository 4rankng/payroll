import { useMemo } from 'react';
import {
  Wallet,
  Building,
  TrendingUp,
  TrendingDown,
  Banknote,
  PiggyBank,
  ShoppingCart,
  Users
} from 'lucide-react';
import { ledgerService } from '@/services/api/ledger.service';
import { useLedgerMetadata } from '@/hooks/ledger/useLedgerMetadata';
import type { LedgerSummary } from '@/types/api/financial.types';
import type { StatCardConfig } from '@/components/shared/SummaryStatsCards';

interface UseLedgerStatsConfigProps {
  summary?: LedgerSummary;
  isLoading?: boolean;
}

export const useLedgerStatsConfig = ({ summary, isLoading }: UseLedgerStatsConfigProps) => {
  const { data: accountMetadata, isLoading: isLoadingMetadata } = useLedgerMetadata();

  const statsConfig: StatCardConfig[] = useMemo(() => {
    if (isLoading || !summary || isLoadingMetadata || !accountMetadata || accountMetadata.length === 0) {
      return [];
    }

    // Main financial totals (top row)
    const mainStats: StatCardConfig[] = [
      {
        title: 'Số Dư Đầu Kỳ',
        value: ledgerService.formatCurrencyShort(summary.totals.opening_balance),
        icon: Wallet,
      },
      {
        title: 'Số Dư Cuối Kỳ',
        value: ledgerService.formatCurrencyShort(summary.totals.closing_balance),
        icon: Building,
      },
      {
        title: 'Dòng Tiền Ròng',
        value: ledgerService.formatCurrencyShort(summary.totals.net_cashflow),
        icon: summary.totals.net_cashflow >= 0 ? TrendingUp : TrendingDown,
      },
    ];

    // Account breakdown stats (second row)
    const accountStats: StatCardConfig[] = Object.entries(summary.by_account).map(([accountKey, accountData]) => {
      const getAccountIcon = (key: string) => {
        const iconMap = {
          cash: Banknote,
          payable: ShoppingCart,
          receivable: PiggyBank,
          capital: PiggyBank,
          revenue: TrendingUp,
          expense: TrendingDown,
          salary: Users,
        };
        return iconMap[key as keyof typeof iconMap] || Wallet;
      };

      // Get intuitive title for each account - use backend label only, no fallback
      const getAccountTitle = (key: string) => {
        const meta = accountMetadata?.find(m => m.value === key);
        if (!meta) {
          console.error(`Missing metadata for account: ${key}`);
          return `ERROR: ${key}`;
        }
        return meta.label;
      };

      // Format value with intuitive display
      // All values now use the intuitive GetUserBalance calculation
      // Positive means: more cash, more spending, more income, owe more, owed more
      const getValueWithSign = () => {
        return ledgerService.formatCurrencyShort(accountData.net_amount);
      };

      return {
        title: getAccountTitle(accountKey),
        value: getValueWithSign(),
        icon: getAccountIcon(accountKey),
      };
    });

    return [...mainStats, ...accountStats];
  }, [summary, isLoading, accountMetadata, isLoadingMetadata]);

  const mainStatsConfig = statsConfig.slice(0, 3);
  const accountStatsConfig = statsConfig.slice(3);

  return {
    statsConfig,
    mainStatsConfig,
    accountStatsConfig,
    isLoading: !!isLoading,
    hasData: !!summary,
  };
};
