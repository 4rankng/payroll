import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import type { Employee } from '@/types/api/employee.types';
import type { ProjectEmployeeListResponse } from '@/types/api/project-employee.types';
import type {
  TimesheetEntry,
  TimesheetValidation
} from '../../types/multi-timesheet.types';

interface ValidationParams {
  entries: TimesheetEntry[];
  availableEmployees: Employee[];
  projectEmployeesData?: ProjectEmployeeListResponse;
}

export const validateTimesheetEntries = ({
  entries,
  availableEmployees,
  projectEmployeesData
}: ValidationParams): TimesheetValidation => {
  const entryValidations: Record<string, { valid: boolean; errors: string[] }> = {};
  let hasValidEntries = false;
  const errors: string[] = [];

  const employeeDateProjectMap = new Map<string, { projectId: number; entryId: string }>();

  entries.forEach((entry) => {
    const entryErrors: string[] = [];
    const hasNonZeroHours =
      entry.hours && Object.values(entry.hours).some(hours => hours > 0);

    if (!hasNonZeroHours) {
      entryValidations[entry.id] = {
        valid: true,
        errors: []
      };
      return;
    }

    if (!entry.employeeId && availableEmployees.length > 0) {
      entryErrors.push('Vui lòng chọn nhân viên');
    }

    if (!entry.projectId) {
      entryErrors.push('Vui lòng chọn dự án');
    }

    if (!entry.date) {
      entryErrors.push('Vui lòng chọn ngày');
    }

    if (!entry.position) {
      entryErrors.push('Vị trí nhân viên không có sẵn');
    }

    if (entry.employeeId && entry.date && entry.projectId) {
      const key = `${entry.employeeId}-${entry.date}`;
      const existing = employeeDateProjectMap.get(key);

      if (existing && existing.projectId !== entry.projectId) {
        entryErrors.push('Nhân viên chỉ có thể làm việc cho một dự án trong một ngày');
      } else if (!existing) {
        employeeDateProjectMap.set(key, { projectId: entry.projectId, entryId: entry.id });
      }
    }

    if (entry.employeeId && entry.date && projectEmployeesData?.data) {
      const assignment = projectEmployeesData.data.find(
        emp => emp.employee_id === entry.employeeId
      );

      if (assignment?.start_date) {
        const entryDate = new Date(entry.date);
        const startDate = new Date(assignment.start_date);

        if (entryDate < startDate) {
          const formattedStartDate = format(startDate, 'dd/MM/yyyy', { locale: vi });
          entryErrors.push(
            `Ngày chấm công không thể trước ngày bắt đầu làm việc (${formattedStartDate})`
          );
        }
      }
    }

    const isValid = entryErrors.length === 0;
    if (isValid) {
      hasValidEntries = true;
    }

    entryValidations[entry.id] = {
      valid: isValid,
      errors: entryErrors
    };
  });

  const finalValid = errors.length === 0 &&
    hasValidEntries &&
    Object.values(entryValidations).every(v => v.valid);

  return {
    valid: finalValid,
    errors,
    entryValidations
  };
};
