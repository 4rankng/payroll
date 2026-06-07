import { useCallback, useMemo, useState } from "react";
import { toast } from "@/components/ui/sonner";
import {
  useProjectEmployees,
  useRemoveEmployeesFromProject,
} from "@/hooks/api/useProjectEmployees";
import { createErrorMessage } from "@/utils/error-handler";
import { SortingState } from "@tanstack/react-table";
import type { Project } from "@/types/api/project.types";
import type {
  ProjectEmployeeAssignment,
  ProjectEmployeeListParams,
} from "@/types/api/project-employee.types";

interface UseProjectEmployeeManagementOptions {
  project: Project;
  params?: ProjectEmployeeListParams;
  onEmployeeRemoved?: () => void;
}

interface AssignmentMeta {
  // No UI-facing status; only capabilities derived from dates
  canRemove: boolean; // Currently working and project active
  canCancel: boolean; // Not started yet and project active
}

/**
 * Business logic hook for project employee management
 * Separates business logic from UI components following clean architecture principles
 */
export function useProjectEmployeeManagement({
  project,
  params = { pageSize: 9999 },
  onEmployeeRemoved,
}: UseProjectEmployeeManagementOptions) {
  const [sorting, setSorting] = useState<SortingState>([]);

  const { sortBy, sortOrder } = useMemo(() => {
    if (sorting.length === 0) return { sortBy: 'start_date' as string | undefined, sortOrder: 'desc' as const };
    const col = sorting[0];
    return {
      sortBy: col.id,
      sortOrder: (col.desc ? 'desc' : 'asc') as 'desc' | 'asc',
    };
  }, [sorting]);

  // API hooks
  const {
    data: employeesData,
    isLoading,
    error: fetchError,
  } = useProjectEmployees(project.id, { ...params, sortBy, sortOrder });

  const {
    mutateAsync: removeEmployees,
    isPending: isRemoving,
    error: removeError,
  } = useRemoveEmployeesFromProject();

  // Memoized capability calculator (no status label)
  const getAssignmentMeta = useCallback(
    (assignment: ProjectEmployeeAssignment): AssignmentMeta => {
      const today = new Date();
      today.setHours(0, 0, 0, 0);

      const startDate = new Date(assignment.start_date);
      startDate.setHours(0, 0, 0, 0);

      const lastDate = assignment.last_date
        ? new Date(assignment.last_date)
        : null;
      if (lastDate) {
        lastDate.setHours(0, 0, 0, 0);
      }

      // Determine status
      // Determine capabilities based on dates and project state
      const isUpcoming = startDate > today;
      const isResigned = !!lastDate && lastDate < today;
      const isEmployed = !isUpcoming && !isResigned;
      const isProjectActive =
        project.status !== "cancelled" && project.status !== "completed";
      const canRemove = isEmployed && isProjectActive;
      const canCancel = isUpcoming && isProjectActive;

      return {
        canRemove,
        canCancel,
      };
    },
    [project.status],
  );

  // Remove employee handler with proper error handling
  const handleRemoveEmployee = useCallback(
    async (
      assignment: ProjectEmployeeAssignment,
      customLastDate?: string,
    ): Promise<{ success: boolean; message?: string }> => {
      try {
        const response = await removeEmployees({
          projectId: assignment.project_id,
          data: [
            {
              employee_id: assignment.employee_id,
              ...(customLastDate && { last_date: customLastDate }),
            },
          ],
        });

        const employeeName =
          assignment.employee_name || assignment.employee_code || "nhân viên";
        toast({
          title: "Nhân viên đã được gỡ khỏi dự án",
          description: `${employeeName} đã được gỡ khỏi dự án ${project.name}`,
        });

        onEmployeeRemoved?.();
        return { success: true, message: response.message };
      } catch (error) {
        const errorMessage = createErrorMessage(
          error,
          "PROJECT_EMPLOYEE",
          "Có lỗi xảy ra khi gỡ nhân viên khỏi dự án",
        );
        toast({
          title: "Lỗi khi gỡ nhân viên",
          description: errorMessage,
          variant: "destructive",
        });
        return { success: false };
      }
    },
    [removeEmployees, project.name, onEmployeeRemoved],
  );

  // Process and sort employees
  const processedEmployees = useMemo(() => {
    if (!employeesData?.data) return [];

    return employeesData.data.map((assignment) => ({
      ...assignment,
      meta: getAssignmentMeta(assignment),
    }));
  }, [employeesData?.data, getAssignmentMeta]);

  // Error state
  const error = fetchError || removeError;
  const errorMessage = error
    ? createErrorMessage(error, "PROJECT_EMPLOYEE")
    : null;

  return {
    // Data
    employees: processedEmployees,
    totalCount:
      employeesData?.pagination?.totalRecords ?? processedEmployees.length,
    totalPages:
      employeesData?.pagination?.totalPages ?? 0,

    // Sorting
    sorting,
    setSorting,

    // Loading states
    isLoading,
    isRemoving,

    // Error states
    error,
    errorMessage,

    // Actions
    removeEmployee: handleRemoveEmployee,

    // Utilities
    // Meta utilities
    getAssignmentMeta,

    // Project state
    canAddEmployees: project.status === "active",
    isEmpty: processedEmployees.length === 0,
  };
}
