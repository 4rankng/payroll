import type { Employee } from '@/types/api/employee.types';
import { findPreferredDayType } from '@/utils/vietnamese';
import { PREFERRED_DAY_TYPES } from '@/constants/timesheetConstants';
import { generateId, getEmployeePosition } from '@/utils/timesheetTransformers';
import { isOffDay } from '@/components/projects/OffDaysPicker';
import type { TimesheetEntry } from '../../types/multi-timesheet.types';



interface EntryFactoryOptions {
  getDayTypesForPosition: (position: string) => string[];
  getFallbackProjectId: () => number;
  preferredDayTypes?: readonly string[];
  getProjectOffDays?: () => number;
}

export const createEntryFactory = ({
  getDayTypesForPosition,
  getFallbackProjectId,
  preferredDayTypes = PREFERRED_DAY_TYPES,
  getProjectOffDays
}: EntryFactoryOptions) => {
  return (employee?: Employee, selectedProjectId?: number, initialDate?: string): TimesheetEntry => {
    const today = initialDate || new Date().toISOString().split('T')[0];

    let projectId = selectedProjectId || getFallbackProjectId() || 0;

    if (employee && employee.current_projects && employee.current_projects.length > 0) {
      if (!selectedProjectId) {
        projectId = employee.current_projects[0].project_id;
      }
    }

    const position = getEmployeePosition(employee);
    let defaultDayType = '';

    // Check if this date is a configured off day for the project
    const offDays = getProjectOffDays ? getProjectOffDays() : 0;
    if (offDays && initialDate) {
      const dateObj = new Date(initialDate + 'T00:00:00');
      const dow = dateObj.getDay(); // 0=Sun ... 6=Sat
      if (isOffDay(offDays, dow)) {
        // Find "ngày nghỉ" in available day types for this position
        if (position) {
          const availableDayTypes = getDayTypesForPosition(position);
          const ngayNghi = availableDayTypes.find(dt =>
            dt.toLowerCase().includes('ngh')
          );
          if (ngayNghi) {
            defaultDayType = ngayNghi;
          }
        }
      }
    }

    // Fall back to preferred day type if not set by off-day logic
    if (!defaultDayType && position) {
      const availableDayTypes = getDayTypesForPosition(position);
      if (availableDayTypes.length > 0) {
        const preferred = [...preferredDayTypes];
        defaultDayType = findPreferredDayType(availableDayTypes, preferred);
      }
    }

    return {
      id: generateId(),
      employeeId: employee?.id || 0,
      employee,
      projectId,
      date: today,
      hours: {},
      position,
      dayType: defaultDayType as string
    };
  };
};
