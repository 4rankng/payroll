import { useMemo, useEffect, useRef, useCallback } from 'react';
import type { Employee, CurrentProject } from '@/types/api/employee.types';
import { getEmployeeProjects } from '@/types/api/employee.types';
import { useActiveProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { useTimesheetProjects } from '@/hooks/api/useProjects';
import type { TimesheetEntry } from '../types/multi-timesheet.types';

export const useEmployeeData = (projectId: number, entries: TimesheetEntry[], isOpen = false) => {
  const { data: projectEmployeesData, isLoading: isLoadingProjectEmployees } = useActiveProjectEmployees(
    projectId || 0,
    Boolean(projectId)
  );
  const { data: assignableProjectsData } = useTimesheetProjects({ enabled: isOpen });

  const availableEmployeesRef = useRef<Employee[]>([]);

  // Derived data - Optimized with Map for O(1) lookup
  const projectDetailsMap = useMemo(() => {
    if (!assignableProjectsData?.data) return new Map();
    return new Map(assignableProjectsData.data.map(p => [p.id, p]));
  }, [assignableProjectsData?.data]);

  const availableEmployees = useMemo(() => {
    if (!projectEmployeesData?.data) return [];

    return projectEmployeesData.data.map(assignment => {
      // O(1) lookup instead of O(n) find
      const projectDetails = projectDetailsMap.get(assignment.project_id);

      // Create current project from assignment data
      const currentProject: CurrentProject = {
        project_id: assignment.project_id,
        name: projectDetails?.name || `Project ${assignment.project_id}`,
        code: projectDetails?.code || '',
        client_name: projectDetails?.client_name || '',
        position: assignment.position,
        start_date: assignment.start_date,
        last_date: assignment.last_date || null
      };

      return {
        id: assignment.employee_id,
        fullname: assignment.employee_name,
        cccd: assignment.employee_cccd,
        email: '',
        position: assignment.position,
        assignment,
        current_projects: [currentProject]
      };
    });
  }, [projectEmployeesData?.data, projectDetailsMap]);

  // Sync availableEmployees to ref
  useEffect(() => {
    availableEmployeesRef.current = availableEmployees;
  }, [availableEmployees]);

  const selectedEmployees = useMemo(() => {
    const selectedIds = new Set(entries.map(entry => entry.employeeId));
    return availableEmployees.filter(emp => selectedIds.has(emp.id));
  }, [availableEmployees, entries]);

  // Helper to get available projects for an employee
  const getAvailableProjectsForEmployee = useCallback((employee: Employee) => {
    return getEmployeeProjects(employee);
  }, []);

  return {
    projectEmployeesData,
    isLoadingProjectEmployees,
    availableEmployees,
    availableEmployeesRef,
    selectedEmployees,
    getAvailableProjectsForEmployee
  };
};
