import { useEmployeesWithMissingMobile, useEmployeesWithDuplicateMobiles } from '@/hooks/api/useEmployees';
import type { EmployeeFilters } from '@/types/api/employee.types';

export interface UseMissingMobileOptions {
  filters?: EmployeeFilters;
  enabled?: boolean;
}

export const useMissingMobile = (options: UseMissingMobileOptions = {}) => {
  const { filters, enabled = true } = options;

  const {
    data,
    isLoading,
    error,
    refetch
  } = useEmployeesWithMissingMobile(
    enabled ? filters : undefined
  );

  return {
    employees: data?.data || [],
    totalCount: data?.pagination?.totalRecords || 0,
    isLoading,
    error: error ? 'Không thể tải danh sách nhân viên thiếu số điện thoại' : null,
    refetch,
  };
};

export const useDuplicateMobiles = () => {
  const {
    data,
    isLoading,
    error,
    refetch
  } = useEmployeesWithDuplicateMobiles();

  return {
    groups: data?.groups || [],
    totalEmployees: data?.total_employees || 0,
    isLoading,
    error: error ? 'Không thể tải danh sách số điện thoại trùng' : null,
    refetch,
  };
};
