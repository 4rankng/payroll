import { useEffect } from 'react';
import type { Employee } from '@/types/api/employee.types';
import { findPreferredDayType } from '@/utils/vietnameseHelpers';
import { PREFERRED_DAY_TYPES } from '@/constants/timesheetConstants';
import type { MultiTimesheetFormData } from '../types/multi-timesheet.types';
import { useTimesheetProjects } from '@/hooks/api/useProjects';
import { Dispatch, SetStateAction } from 'react';
import { isOffDay } from '@/components/projects/OffDaysPicker';

interface UseFormInitializationProps {
  isOpen: boolean;
  isClosing?: boolean;
  projectId?: number;
  employeeId?: number;
  formData: MultiTimesheetFormData;
  setFormData: Dispatch<SetStateAction<MultiTimesheetFormData>>;
  availableEmployees: Employee[];
  isPayRateReady: boolean;
  getDayTypesForPosition: (position: string) => string[];
  handleAddEmployee: (employee: Employee) => void;
  handleProjectChange: (projectId: number) => void;
  projectOffDays?: number;
}

/** Resolve the correct day type for a date given off-days config and available day types */
function resolveDefaultDayType(
  date: string,
  offDays: number,
  availableDayTypes: string[],
  preferredDayTypes: readonly string[]
): string {
  if (offDays && availableDayTypes.length > 0) {
    const dateObj = new Date(date + 'T00:00:00');
    const dow = dateObj.getDay();
    if (isOffDay(offDays, dow)) {
      const ngayNghi = availableDayTypes.find(dt => dt.toLowerCase().includes('ngh'));
      if (ngayNghi) return ngayNghi;
    }
  }
  return findPreferredDayType(availableDayTypes, [...preferredDayTypes]);
}

export const useFormInitialization = ({
  isOpen,
  isClosing = false,
  projectId,
  employeeId,
  formData,
  setFormData,
  availableEmployees,
  isPayRateReady,
  getDayTypesForPosition,
  handleAddEmployee,
  handleProjectChange,
  projectOffDays = 0
}: UseFormInitializationProps) => {
  const { data: assignableProjectsData } = useTimesheetProjects({ enabled: isOpen });

  // Effects for initialization
  useEffect(() => {
    if (isOpen && !formData.projectId && projectId && !isClosing) {
      setFormData(prev => ({ ...prev, projectId }));
    }
  }, [isOpen, projectId, formData.projectId, isClosing, setFormData]);

  useEffect(() => {
    if (isOpen && employeeId && availableEmployees.length > 0 && !isClosing) {
      const employee = availableEmployees.find(emp => emp.id === employeeId);
      if (employee && !formData.entries.some(entry => entry.employeeId === employeeId)) {
        handleAddEmployee(employee);
      }
    }
  }, [isOpen, employeeId, availableEmployees, formData.entries, handleAddEmployee, isClosing]);

  // Auto-populate project if not provided - always select first project when sheet opens
  useEffect(() => {
    if (isOpen && !projectId && assignableProjectsData?.data?.length > 0 && !isClosing) {
      // Only auto-select if no project is currently selected or if form was reset
      if (!formData.projectId) {
        const firstActiveProject = assignableProjectsData.data.find(p => p.status === 'active');
        if (firstActiveProject) {
          handleProjectChange(firstActiveProject.id);
        }
      }
    }
  }, [isOpen, projectId, assignableProjectsData, formData.projectId, handleProjectChange, isClosing]);

  // Ensure correct dayType is set once payrate data is available, and re-apply when off-days config changes
  useEffect(() => {
    if (!isOpen || !isPayRateReady || isClosing) {
      return;
    }

    setFormData(prev => {
      let hasChanges = false;

      const updatedEntries = prev.entries.map(entry => {
        if (!entry.employeeId || !entry.position) {
          return entry;
        }

        const availableDayTypes = getDayTypesForPosition(entry.position);
        if (!availableDayTypes.length) {
          return entry;
        }

        // Skip entries that already have data from the server (originalValues set)
        if (entry.originalValues) {
          return entry;
        }

        const correctDayType = resolveDefaultDayType(
          entry.date,
          projectOffDays,
          availableDayTypes,
          PREFERRED_DAY_TYPES
        );

        if (entry.dayType === correctDayType) {
          return entry;
        }

        hasChanges = true;
        return { ...entry, dayType: correctDayType as string };
      });

      if (!hasChanges) return prev;
      return { ...prev, entries: updatedEntries };
    });
  }, [isOpen, isPayRateReady, getDayTypesForPosition, projectOffDays, isClosing, setFormData]);
};
