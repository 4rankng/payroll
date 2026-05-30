import { useCallback, Dispatch, SetStateAction, MutableRefObject } from 'react';
import type { Employee } from '@/types/api/employee.types';
import { getEmployeePosition } from '@/utils/timesheetTransformers';
import { findPreferredDayType } from '@/utils/vietnameseHelpers';
import { PREFERRED_DAY_TYPES } from '@/constants/timesheetConstants';
import { generateId } from '@/utils/timesheetTransformers';
import type { TimesheetEntry, MultiTimesheetFormData } from '../types/multi-timesheet.types';

interface UseTimesheetFormActionsProps {
  formData: MultiTimesheetFormData;
  setFormData: Dispatch<SetStateAction<MultiTimesheetFormData>>;
  availableEmployees: Employee[];
  availableEmployeesRef: MutableRefObject<Employee[]>;
  createNewEntry: (employee?: Employee, project?: number, date?: string) => TimesheetEntry;
  getDayTypesForPosition: (position: string) => string[];
}

export const useTimesheetFormActions = ({
  formData,
  setFormData,
  availableEmployees,
  availableEmployeesRef,
  createNewEntry,
  getDayTypesForPosition
}: UseTimesheetFormActionsProps) => {

  const handleProjectChange = useCallback((newProjectId: number) => {
    setFormData(prev => {
      if (newProjectId > 0) {
        // Check if there are available employees for this project
        // Note: availableEmployees will be updated after this change via the useActiveProjectEmployees hook
        // So we create a blank entry first, then auto-select employee in useEffect
        const newEntry: TimesheetEntry = {
          id: generateId(),
          employeeId: 0,
          employee: undefined,
          projectId: newProjectId,
          date: new Date().toISOString().split('T')[0],
          hours: {},
          position: '',
          dayType: '' as string,

        };

        return {
          ...prev,
          projectId: newProjectId,
          entries: [newEntry] // Add blank entry when project selected
        };
      } else {
        return {
          ...prev,
          projectId: newProjectId,
          entries: [] // Clear entries when no project selected
        };
      }
    });
  }, [setFormData]);

  const handleAddEmployee = useCallback((employee: Employee) => {
    const newEntry = createNewEntry(employee);

    setFormData(prev => ({
      ...prev,
      entries: [...prev.entries, newEntry]
    }));
  }, [createNewEntry, setFormData]);

  const handleAddAllEmployees = useCallback(() => {
    const unselectedEmployees = availableEmployees.filter(emp =>
      !formData.entries.some(entry => entry.employeeId === emp.id)
    );

    const newEntries = unselectedEmployees.map(employee => createNewEntry(employee));

    setFormData(prev => ({
      ...prev,
      entries: [...prev.entries, ...newEntries]
    }));
  }, [availableEmployees, formData.entries, createNewEntry, setFormData]);

  const handleAddEntry = useCallback((employee?: Employee, initialDate?: string) => {
    const newEntry = createNewEntry(employee, undefined, initialDate);

    setFormData(prev => ({
      ...prev,
      entries: [...prev.entries, newEntry]
    }));
  }, [createNewEntry, setFormData]);

  const handleRemoveEntry = useCallback((index: number) => {
    setFormData(prev => {
      const entry = prev.entries[index];
      // New entry (no originalValues) → just remove from UI
      if (!entry?.originalValues) {
        return { ...prev, entries: prev.entries.filter((_, i) => i !== index) };
      }
      // Existing backend entry → zero all hours so transformEntriesToAPI sends hoursWorked:0
      const newEntries = [...prev.entries];
      newEntries[index] = { ...entry, hours: {} };
      return { ...prev, entries: newEntries };
    });
  }, [setFormData]);

  const handleDuplicateEntry = useCallback((index: number) => {
    const entryToDuplicate = formData.entries[index];
    if (entryToDuplicate) {
      const duplicatedEntry = {
        ...entryToDuplicate,
        id: generateId()
      };

      setFormData(prev => ({
        ...prev,
        entries: [...prev.entries, duplicatedEntry]
      }));
    }
  }, [formData.entries, setFormData]);

  const handleEntryChange = useCallback((index: number, field: keyof TimesheetEntry, value: unknown) => {
    setFormData(prev => {
      const newEntries = [...prev.entries];
      const entry = { ...newEntries[index], [field]: value };

      // When employee changes, update the employee object and position
      if (field === 'employeeId') {
        const selectedEmployee = availableEmployeesRef.current.find(emp => emp.id === value);
        entry.employee = selectedEmployee;
        entry.position = getEmployeePosition(selectedEmployee);

        // If employee has only one project, auto-select it
        if (selectedEmployee && selectedEmployee.current_projects?.length === 1) {
          entry.projectId = selectedEmployee.current_projects[0].project_id;
        }

        // Reset day type when position changes - prefer NGAY THUONG
        if (entry.position) {
          const availableDayTypes = getDayTypesForPosition(entry.position);
          if (availableDayTypes.length > 0) {
            entry.dayType = findPreferredDayType(availableDayTypes, [...PREFERRED_DAY_TYPES]) as string;
            // Reset hours object when position changes
            entry.hours = {};
          }
        }
      }

      newEntries[index] = entry;

      return {
        ...prev,
        entries: newEntries
      };
    });
  }, [getDayTypesForPosition, availableEmployeesRef, setFormData]);

  // Generate entries for date range when user changes the date range
  const handleDateRangeChange = useCallback((startDate: string, endDate: string) => {
    if (!startDate || !endDate) return;

    // Generate date range array (limited to current month by date picker constraints)
    const start = new Date(startDate);
    const end = new Date(endDate);

    const dates: string[] = [];

    for (let date = new Date(start); date <= end; date.setDate(date.getDate() + 1)) {
      dates.push(date.toISOString().split('T')[0]); // Format as YYYY-MM-DD
    }

    // Get unique employees from current entries
    const uniqueEmployees = Array.from(
      new Map(formData.entries
        .filter(entry => entry.employeeId && entry.employee)
        .map(entry => [entry.employeeId, entry.employee]))
        .values()
    );

    // Create a Set of valid date strings for quick lookup
    const validDates = new Set(dates);

    // Process existing entries: keep only those within the new date range
    const filteredEntries = formData.entries.filter(entry => {
      // Keep blank entries (no employee assigned)
      if (!entry.employeeId) return true;

      // Keep entries for employees that are still assigned
      const employeeExists = uniqueEmployees.some(emp => emp.id === entry.employeeId);
      if (!employeeExists) return false;

      // Keep entries within the new date range
      return validDates.has(entry.date);
    });

    // Generate new entries for missing date-employee combinations
    const newEntries: TimesheetEntry[] = [];
    dates.forEach(date => {
      uniqueEmployees.forEach(employee => {
        const existingEntry = filteredEntries.find(
          entry => entry.employeeId === employee.id && entry.date === date
        );

        if (!existingEntry) {
          const newEntry = createNewEntry(employee, undefined, date);
          newEntries.push(newEntry);
        }
      });
    });

    // Combine filtered existing entries with new entries
    const finalEntries = [...filteredEntries, ...newEntries];

    // Sort entries by employee ID first, then by date ascending
    const sortedEntries = finalEntries.sort((a, b) => {
      // Sort by employee ID (0/blank entries come first)
      if (a.employeeId !== b.employeeId) {
        return a.employeeId - b.employeeId;
      }

      // Then sort by date ascending
      return a.date.localeCompare(b.date);
    });

    setFormData(prev => ({
      ...prev,
      entries: sortedEntries
    }));
  }, [formData.entries, createNewEntry, setFormData]);

  const resetForm = useCallback((projectId?: number) => {
    setFormData({
      projectId: projectId || 0,
      entries: []
    });
  }, [setFormData]);

  return {
    handleProjectChange,
    handleAddEmployee,
    handleAddAllEmployees,
    handleAddEntry,
    handleRemoveEntry,
    handleDuplicateEntry,
    handleEntryChange,
    handleDateRangeChange,
    resetForm
  };
};
