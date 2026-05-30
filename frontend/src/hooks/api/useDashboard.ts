import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import { QueryKeys } from '@/lib/queryKeys';
import type { TopPaidEmployeesParams, PartnerDashboardParams, BankUsageParams, PartnerEmployeeListParams } from '@/types/api/dashboard.types';

// Get dashboard summary
export const useDashboardSummary = (month?: string) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.summary(month),
    queryFn: () => dashboardService.getDashboardSummary(month ? { month } : undefined),
  });
};

// Get employee activity stats (logins grouped by schedule for a given month)
export const useEmployeeActivityStats = (month?: string) => {
  return useQuery({
    queryKey: [...QueryKeys.dashboard.employeeActivity(), month],
    queryFn: () => dashboardService.getEmployeeActivityStats(month),
  });
};

// Get active employees by schedule type for a given month
export const useActiveEmployeesBySchedule = (schedule: 'weekly' | 'monthly' | 'flexible' | null, month?: string) => {
  return useQuery({
    queryKey: [...QueryKeys.dashboard.employeeActivityUsers(schedule ?? ''), month],
    queryFn: () => dashboardService.getActiveEmployeesBySchedule(schedule!, month),
    enabled: !!schedule,
  });
};

// Get project profitability ranking
export const useProjectProfitability = () => {
  return useQuery({
    queryKey: QueryKeys.dashboard.projectProfitability(),
    queryFn: () => dashboardService.getProjectProfitability(),
  });
};

// Get daily cumulative profit per project
export const useProjectWeeklyProfit = (days?: number) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.projectWeeklyProfit(days),
    queryFn: () => dashboardService.getProjectWeeklyProfit(days),
  });
};

// Get top paid employees (admin dashboard metric)
export const useTopPaidEmployees = (params?: TopPaidEmployeesParams) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.topPaidEmployees(params?.month, params?.limit),
    queryFn: () => dashboardService.getTopPaidEmployees(params),
  });
};

// Get partner dashboard overview
export const usePartnerDashboard = (params?: PartnerDashboardParams) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.partnerDashboard(params?.month),
    queryFn: () => dashboardService.getPartnerDashboard(params),
  });
};

// Get bank usage breakdown (overall or per project)
export const useBankUsage = (params?: BankUsageParams) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.bankUsage(params?.project_id),
    queryFn: () => dashboardService.getBankUsage(params),
  });
};

// Get bank usage broken down by all active projects
export const useBankUsageAllProjects = () => {
  return useQuery({
    queryKey: QueryKeys.dashboard.bankUsageAllProjects(),
    queryFn: () => dashboardService.getBankUsageAllProjects(),
  });
};

// Get partner employee list by category (active / dropped / paid)
export const usePartnerEmployeeList = (params: PartnerEmployeeListParams | null) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.partnerEmployeeList(params?.type ?? '', params?.month),
    queryFn: () => dashboardService.getPartnerEmployeeList(params!),
    enabled: !!params,
  });
};
