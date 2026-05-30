import { useMemo, useCallback } from 'react';
import {
  Users,
  UserCheck,
  UserPlus,
  UserX
} from 'lucide-react';
import { useEmployeesSummary } from '@/hooks/api/useEmployees';
import type { StatCardConfig } from '@/components/shared/SummaryStatsCards';
import type { EmployeeFilters } from '@/types/api/employee.types';

interface EmployeeStatsConfigProps {
  formatCurrency?: (amount: number) => string;
  currentFilters?: Partial<EmployeeFilters>;
  updateFilters?: (filters: Partial<EmployeeFilters>) => void;
  clearFilters?: () => void;
}

export const useEmployeeStatsConfig = ({
  formatCurrency = (amount: number) => new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: 'VND',
    maximumFractionDigits: 0
  }).format(amount),
  currentFilters = {},
  updateFilters,
  clearFilters
}: EmployeeStatsConfigProps = {}) => {
  const { data: summary, isLoading, error } = useEmployeesSummary();

  // Get current month in YYYY-MM format
  const getCurrentMonth = useCallback(() => {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    return `${year}-${month}`;
  }, []);

  // Click handlers with toggle behavior
  const handleTotalClick = useCallback(() => {
    if (!clearFilters) return;
    // Clear status filter to show all employees
    if (currentFilters.status) {
      clearFilters();
    }
  }, [currentFilters.status, clearFilters]);

  const handleWorkingClick = useCallback(() => {
    if (!updateFilters && !clearFilters) return;
    // Toggle working status filter
    if (currentFilters.status === 'working') {
      clearFilters?.();
    } else {
      updateFilters?.({ status: 'working' });
    }
  }, [currentFilters.status, updateFilters, clearFilters]);

  const handleUnassignedClick = useCallback(() => {
    if (!updateFilters && !clearFilters) return;
    // Toggle unassigned status filter
    if (currentFilters.status === 'unassigned') {
      clearFilters?.();
    } else {
      updateFilters?.({ status: 'unassigned' });
    }
  }, [currentFilters.status, updateFilters, clearFilters]);

  const handleThisMonthClick = useCallback(() => {
    if (!updateFilters && !clearFilters) return;
    const currentMonth = getCurrentMonth();
    // Toggle current month filter without affecting status
    if (currentFilters.month === currentMonth) {
      // Clear month filter but preserve status filter if any
      if (currentFilters.status) {
        updateFilters?.({ month: undefined });
      } else {
        clearFilters?.();
      }
    } else {
      // Add month filter while preserving status filter if any
      updateFilters?.({
        month: currentMonth,
        ...(currentFilters.status && { status: currentFilters.status })
      });
    }
  }, [currentFilters.month, currentFilters.status, updateFilters, clearFilters, getCurrentMonth]);

  const statsConfig: StatCardConfig[] = useMemo(() => {
    if (!summary) return [];

    const currentMonth = getCurrentMonth();
    const isWorkingActive = currentFilters.status === 'working' && !currentFilters.month;
    const isUnassignedActive = currentFilters.status === 'unassigned' && !currentFilters.month;
    const isThisMonthActive = currentFilters.month === currentMonth;
    const isTotalActive = !currentFilters.status && !currentFilters.month;

    return [
      {
        title: 'Tổng nhân viên',
        value: summary.total_employees,
        icon: Users,
        onClick: handleTotalClick,
        isActive: isTotalActive
      },
      {
        title: 'Chưa phân công',
        value: summary.total_employees - summary.total_working_employees,
        icon: UserX,
        onClick: handleUnassignedClick,
        isActive: isUnassignedActive
      },
      {
        title: 'Đang làm việc',
        value: summary.total_working_employees,
        icon: UserCheck,
        onClick: handleWorkingClick,
        isActive: isWorkingActive
      },
      {
        title: 'NV tháng này',
        value: summary.employees_hired_this_month,
        icon: UserPlus,
        onClick: handleThisMonthClick,
        isActive: isThisMonthActive
      }
    ];
  }, [summary, currentFilters, handleTotalClick, handleWorkingClick, handleUnassignedClick, handleThisMonthClick, getCurrentMonth]);

  return {
    statsConfig,
    isLoading,
    error: !!error
  };
};
