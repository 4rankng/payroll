import { useMemo } from 'react';
import { useDashboardFinancialOverview } from './useDashboardFinancialOverview';
import { useEmployeeSummary } from './useEmployeeSummary';
import { useTimesheetSummary } from './useTimesheetSummary';

// Synthetic analytics data structure
export interface DashboardAnalyticsMetric {
  count: number;
  change_percentage: number;
  change_from_previous: number;
}

export interface DashboardAnalyticsAmountMetric {
  amount_vnd: number;
  change_percentage: number;
  change_from_previous_vnd: number;
}

export interface DashboardAnalyticsData {
  current_month: string;
  total_employees: DashboardAnalyticsMetric;
  active_projects: DashboardAnalyticsMetric;
  pending_approvals: DashboardAnalyticsMetric;
  current_month_salary: DashboardAnalyticsAmountMetric;
}

export const useDashboardAnalytics = () => {
  const financialQuery = useDashboardFinancialOverview({ period: 'month', months: 2 });
  const employeeSummaryQuery = useEmployeeSummary();
  const timesheetSummaryQuery = useTimesheetSummary();

  const data: DashboardAnalyticsData | undefined = useMemo(() => {
    if (!financialQuery.data) {
      return undefined;
    }

    const financial = financialQuery.data;
    const employeeSummary = employeeSummaryQuery.data;
    const timesheetSummary = timesheetSummaryQuery.data;

    // Calculate synthetic metrics from available data
    const currentMonth = financial.current_month.month;

    // Calculate revenue change
    const revenueChange = financial.growth.revenue_change_percentage;
    const revenueChangePrevious = financial.current_month.total_revenue_vnd - financial.previous_month.total_revenue_vnd;

    return {
      current_month: currentMonth,
      total_employees: {
        count: employeeSummary?.total_employees || 0,
        change_percentage: 0, // TODO: Calculate from historical data when available
        change_from_previous: 0,
      },
      active_projects: {
        count: 0, // Removed project status
        change_percentage: 0,
        change_from_previous: 0,
      },
      pending_approvals: {
        count: timesheetSummary?.pendingApproval || 0,
        change_percentage: 0, // TODO: Calculate from historical data when available
        change_from_previous: 0,
      },
      current_month_salary: {
        amount_vnd: financial.current_month.total_expenses_vnd,
        change_percentage: revenueChange,
        change_from_previous_vnd: revenueChangePrevious,
      },
    };
  }, [financialQuery.data, employeeSummaryQuery.data, timesheetSummaryQuery.data]);

  return {
    data,
    isLoading: financialQuery.isLoading || employeeSummaryQuery.isLoading || timesheetSummaryQuery.isLoading,
    error: financialQuery.error || employeeSummaryQuery.error || timesheetSummaryQuery.error,
    refetch: () => {
      financialQuery.refetch();
      employeeSummaryQuery.refetch();
      timesheetSummaryQuery.refetch();
    }
  };
};
