import { useState, useCallback, useEffect } from "react";
import { useEmployeeTimesheet, useEmployeeSummary } from "@/hooks/api/useEmployees";
import type { EmployeeTimesheetFilters } from "@/types/api/employee.types";
import { useProjects } from "@/hooks/api/useProjects";
import { useRemoveEmployeesFromProject } from "@/hooks/api/useProjectEmployees";
import { SortingState } from "@tanstack/react-table";
import type { Employee } from "@/types/api/employee.types";

interface UseEmployeeDetailsProps {
  employee: Employee | null;
  onRefetch?: () => void;
}

export const useEmployeeDetails = ({ employee, onRefetch }: UseEmployeeDetailsProps) => {
  const [showRemoveModal, setShowRemoveModal] = useState(false);
  const [removeResponseMessage, setRemoveResponseMessage] = useState<string | undefined>();

  // Initialize filters with a stable default - simplified state management
  const [timesheetFilters, setTimesheetFilters] = useState<EmployeeTimesheetFilters>(() => ({
    page: 1,
    pageSize: 10,
    sortBy: 'date',
    sortOrder: 'desc',
    fromDate: undefined,
    toDate: undefined,
  }));

  const [timesheetSorting, setTimesheetSorting] = useState<SortingState>([]);

  // Sync sorting state to API filters
  useEffect(() => {
    if (timesheetSorting.length === 0) return;
    const col = timesheetSorting[0];
    setTimesheetFilters(prev => ({
      ...prev,
      sortBy: col.id,
      sortOrder: (col.desc ? 'desc' : 'asc') as 'desc' | 'asc',
      page: 1,
    }));
  }, [timesheetSorting]);

  // Fetch employee timesheet data
  const { data: timesheetData, isLoading: timesheetLoading } = useEmployeeTimesheet(
    employee?.id || 0,
    timesheetFilters,
    !!employee?.id
  );

  // Fetch employee summary data
  const { data: summaryData, isLoading: summaryLoading } = useEmployeeSummary(
    employee?.id || 0,
    !!employee?.id
  );

  // Fetch projects for assignment
  const { data: projectsData } = useProjects();
  const projects = projectsData?.data || [];

  // Project assignment mutations
  const { mutateAsync: removeEmployees, isPending: isRemoving, error: removeError } = useRemoveEmployeesFromProject();

  const handleRemoveFromProject = useCallback(async (employeeId: number, projectId: number, lastDate?: string) => {
    try {
      const response = await removeEmployees({
        projectId,
        data: [{
          employee_id: employeeId,
          ...(lastDate && { last_date: lastDate }),
        }]
      });
      setRemoveResponseMessage(response.message);
      setShowRemoveModal(false);

      // Trigger immediate refetch of employee data to update current_projects
      if (onRefetch) {
        onRefetch();
      }
    } catch (error) {
      console.error('Failed to remove employee:', error);
      setRemoveResponseMessage(undefined);
    }
  }, [removeEmployees, onRefetch]);

  // Timesheet pagination handlers with memoization
  const handlePageChange = useCallback((page: number) => {
    setTimesheetFilters(prev => ({ ...prev, page }));
  }, []);

  const handleDateRangeChange = useCallback((fromDate?: string, toDate?: string) => {
    setTimesheetFilters(prev => ({ ...prev, fromDate, toDate, page: 1 }));
  }, []);

  return {
    // Data
    timesheetData,
    timesheetLoading,
    summaryData,
    summaryLoading,
    projects,

    // Project assignment state
    showRemoveModal,
    setShowRemoveModal,
    removeResponseMessage,
    setRemoveResponseMessage,

    // Actions
    handleRemoveFromProject,

    // Timesheet controls
    timesheetFilters,
    timesheetSorting,
    setTimesheetSorting,
    handlePageChange,
    handleDateRangeChange,

    // Loading states
    removeEmployee: { isPending: isRemoving, error: removeError },
  };
};
