import { useMemo } from 'react';
import { 
  FolderOpen, 
  TrendingUp, 
  TrendingDown, 
  Clock, 
  AlertCircle 
} from 'lucide-react';
import { useProjectsSummary } from '@/hooks/api/useProjects';
import { formatBudgetDisplay } from '@/utils/projectHelpers';
import type { StatCardConfig } from '@/components/shared/SummaryStatsCards';

export const useProjectStatsConfig = () => {
  const { data: summary, isLoading, error } = useProjectsSummary();

  const statsConfig: StatCardConfig[] = useMemo(() => {
    if (!summary) return [];

    return [
      {
        title: 'Dự án đang hoạt động',
        value: summary.total_active_projects,
        icon: FolderOpen
      },
      {
        title: 'Tổng đã nhận',
        value: formatBudgetDisplay(summary.total_received_vnd),
        icon: TrendingUp
      },
      {
        title: 'Tổng đã chi trả',
        value: formatBudgetDisplay(summary.total_payout_vnd),
        icon: TrendingDown
      },
      {
        title: 'Chờ chi trả',
        value: formatBudgetDisplay(summary.total_pending_payable_vnd),
        icon: Clock
      },
      {
        title: 'Chờ thu',
        value: formatBudgetDisplay(summary.total_pending_receivable_vnd),
        icon: AlertCircle
      }
    ];
  }, [summary]);

  return {
    statsConfig,
    isLoading,
    error: !!error
  };
};