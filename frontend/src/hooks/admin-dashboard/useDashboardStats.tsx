import { useMemo } from 'react';

// Types
import { StatItem } from '@/components/shared/GroupedStatCard';

interface DashboardSummary {
  total_employees: number;
  total_working_employees: number;
  employees_hired_this_month: number;
  pending_salary_this_month: number;
  paid_salary_this_month: number;
  avg_weekly_salary: number;
  avg_monthly_salary: number;
  total_weekly_salary_employees: number;
  total_monthly_salary_employees: number;
  total_profit_this_month: number;
  total_profit: number;
  total_paid_salary: number;
  total_revenue: number;
  total_revenue_this_month: number;
}

interface DashboardNavigation {
  navigateToActiveEmployees: () => void;
  navigateToNewEmployees: () => void;
  navigateToSalaryLedger: () => void;
  navigateToPendingApprovals: () => void;
}

interface UseDashboardStatsParams {
  dashboardSummary: DashboardSummary | null;
  totalPendingApprovals: number;
  dashboardNav: DashboardNavigation;
  activityStats?: { active_weekly: number; active_monthly: number; active_flexible: number; active_total: number } | null;
  onActivityClick?: (schedule: 'weekly' | 'monthly' | 'flexible') => void;
}

interface UseDashboardStatsReturn {
  employeeStats: StatItem[];
  salaryStats: StatItem[];
  profitStats: StatItem[];
  activityStats: StatItem[];
}

export const useDashboardStats = ({
  dashboardSummary,
  totalPendingApprovals,
  dashboardNav,
  activityStats,
  onActivityClick,
}: UseDashboardStatsParams): UseDashboardStatsReturn => {
  // Employee statistics
  const employeeStats = useMemo(() => {
    if (!dashboardSummary) return [];
    return [
      {
        label: 'Tổng',
        value: dashboardSummary.total_employees,
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Đang làm việc',
        value: dashboardSummary.total_working_employees,
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Mới',
        value: dashboardSummary.employees_hired_this_month,
        onClick: dashboardNav.navigateToNewEmployees,
      },
    ] as StatItem[];
  }, [dashboardSummary, dashboardNav]);

  // Combined salary statistics
  const salaryStats = useMemo(() => {
    const stats: StatItem[] = [];

    if (dashboardSummary) {
      stats.push(
        {
          label: 'Chờ trả',
          value: dashboardSummary.pending_salary_this_month,
          unit: ' đ',
          onClick: dashboardNav.navigateToSalaryLedger,
        },
        {
          label: 'Đã trả',
          value: dashboardSummary.paid_salary_this_month,
          unit: ' đ',
          onClick: dashboardNav.navigateToSalaryLedger,
        },
        {
          label: 'Tổng đã trả',
          value: dashboardSummary.total_paid_salary,
          unit: ' đ',
          onClick: dashboardNav.navigateToSalaryLedger,
        }
      );
    }

    stats.push({
      label: 'Chờ duyệt',
      value: totalPendingApprovals,
      onClick: dashboardNav.navigateToPendingApprovals,
    });

    return stats;
  }, [dashboardSummary, totalPendingApprovals, dashboardNav]);

  // Profit & Revenue statistics
  const profitStats = useMemo(() => {
    if (!dashboardSummary) return [];
    return [
      {
        label: 'Tiền ứng',
        value: dashboardSummary.total_revenue_this_month,
        unit: ' đ',
      },
      {
        label: 'Lợi nhuận',
        value: dashboardSummary.total_profit_this_month,
        unit: ' đ',
      },
      {
        label: 'Tổng ứng',
        value: dashboardSummary.total_revenue,
        unit: ' đ',
      },
      {
        label: 'Tổng lãi',
        value: dashboardSummary.total_profit,
        unit: ' đ',
      },
    ] as StatItem[];
  }, [dashboardSummary]);

  // Employee activity stats (logins by schedule type for selected month)
  const activityStatItems = useMemo(() => {
    if (!activityStats) return [];
    return [
      {
        label: 'Lương tuần',
        value: activityStats.active_weekly,
        onClick: onActivityClick ? () => onActivityClick('weekly') : undefined,
      },
      {
        label: 'Lương tháng',
        value: activityStats.active_monthly,
        onClick: onActivityClick ? () => onActivityClick('monthly') : undefined,
      },
      {
        label: 'Linh hoạt',
        value: activityStats.active_flexible,
        onClick: onActivityClick ? () => onActivityClick('flexible') : undefined,
      },
    ] as StatItem[];
  }, [activityStats, onActivityClick]);

  return {
    employeeStats,
    salaryStats,
    profitStats,
    activityStats: activityStatItems,
  };
};
