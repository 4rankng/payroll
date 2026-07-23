import { useMemo, useState } from 'react';
import { useActivitiesPagination } from './useActivitiesPagination';
import { useDashboardNewEmployees } from './useDashboardNewEmployees';
import { useDashboardSummary } from '@/hooks/api/useDashboard';

export const useDashboardData = (month?: string) => {
  const [employeeWeeks, setEmployeeWeeks] = useState(1);

  const activitiesPaginationQuery = useActivitiesPagination({
    pageSize: 10,
    sortBy: 'created_at',
    sortOrder: 'desc',
  });
  const newEmployeesQuery = useDashboardNewEmployees({ pageSize: 20, days: employeeWeeks * 7 });
  const dashboardSummaryQuery = useDashboardSummary(month);

  const loading = activitiesPaginationQuery.isLoading ||
                 newEmployeesQuery.isLoading ||
                 dashboardSummaryQuery.isLoading;

  const recentEmployees = useMemo(() => {
    return newEmployeesQuery.data?.employees || [];
  }, [newEmployeesQuery.data]);

  const error = activitiesPaginationQuery.error ||
               newEmployeesQuery.error ||
               dashboardSummaryQuery.error;

  const recentActivities = useMemo(() => {
    return activitiesPaginationQuery.data?.pages?.flatMap(page => page.activities) || [];
  }, [activitiesPaginationQuery.data]);

  const employees = useMemo(() => {
    const totalEmployees = newEmployeesQuery.data?.pagination?.totalRecords || 0;

    return {
      employees: recentEmployees,
      isLoading: newEmployeesQuery.isLoading,
      isLoadingMore: newEmployeesQuery.isFetching && !newEmployeesQuery.isLoading,
      weeks: employeeWeeks,
      totalEmployees,
      loadMore: () => setEmployeeWeeks(prev => prev + 1),
      refresh: () => newEmployeesQuery.refetch()
    };
  }, [newEmployeesQuery, recentEmployees, employeeWeeks]);

  const activities = useMemo(() => {
    const hasNextPage = activitiesPaginationQuery.hasNextPage;
    const totalActivities = activitiesPaginationQuery.data?.pages?.[0]?.pagination?.totalRecords || 0;

    return {
      activities: recentActivities,
      isLoading: activitiesPaginationQuery.isLoading,
      isLoadingMore: activitiesPaginationQuery.isFetchingNextPage,
      hasMoreActivities: hasNextPage,
      totalActivities,
      loadMore: () => activitiesPaginationQuery.fetchNextPage(),
      refresh: () => activitiesPaginationQuery.refetch(),
      error: activitiesPaginationQuery.error?.message
    };
  }, [activitiesPaginationQuery, recentActivities]);

  return {
    data: {
      recentActivities: recentActivities,
      projects: null,
      dashboardSummary: dashboardSummaryQuery.data || null,
    },
    loading,
    error: error?.message || error,
    recentEmployees,
    employees, // New employees pagination object
    activities, // Expose activities pagination
    refetch: () => Promise.all([
      activitiesPaginationQuery.refetch(),
      newEmployeesQuery.refetch(),
      dashboardSummaryQuery.refetch(),
    ]),
    isRefreshing: activitiesPaginationQuery.isFetching ||
                  newEmployeesQuery.isFetching ||
                  dashboardSummaryQuery.isFetching
  };
};
