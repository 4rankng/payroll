import type { Employee } from '@/types/api/employee.types';
import { PREFERRED_DAY_TYPES } from '@/constants/timesheetConstants';
import { normalizeVietnamese } from '@/utils/vietnamese';
import { generateId, getEmployeePosition } from '@/utils/timesheetTransformers';
import type { TimesheetEntry } from '../../types/multi-timesheet.types';

interface ApiEmployeeTimesheet {
  employee_id: number;
  entries: Array<{
    date: string;
    [dayType: string]: number | Record<string, number> | string;
  }>;
}

interface MapApiEntriesParams {
  apiEntries: ApiEmployeeTimesheet[];
  projectId: number;
  availableEmployees: Employee[];
  getDayTypesForPosition: (position: string) => string[];
  getHourTypesForEntry: (position: string, dayType: string) => string[];
}

const resolveDayType = (
  rawDayType: string,
  position: string,
  getDayTypesForPosition: (position: string) => string[]
) => {
  const availableDayTypes = getDayTypesForPosition(position);

  if (availableDayTypes.length > 0) {
    const match = availableDayTypes.find(dt =>
      normalizeVietnamese(dt) === normalizeVietnamese(rawDayType)
    );
    if (match) {
      return match;
    }
  }

  const fallbackMatch = PREFERRED_DAY_TYPES.find(dt =>
    normalizeVietnamese(dt) === normalizeVietnamese(rawDayType)
  );

  return fallbackMatch || rawDayType;
};

export const mapApiEntriesToTimesheetEntries = ({
  apiEntries,
  projectId,
  availableEmployees,
  getDayTypesForPosition,
  getHourTypesForEntry
}: MapApiEntriesParams): TimesheetEntry[] => {
  const newEntries: TimesheetEntry[] = [];

  apiEntries.forEach(employeeRecord => {
    const employee = availableEmployees.find(e => e.id === employeeRecord.employee_id);
    if (!employee) return;

    const position = getEmployeePosition(employee);

    employeeRecord.entries.forEach(dailyEntry => {
      const { date, ...dayTypes } = dailyEntry;

      Object.entries(dayTypes).forEach(([rawDayType, hourData]) => {
        if (typeof hourData !== 'object' || !hourData) return;

        const resolvedDayType = resolveDayType(rawDayType, position, getDayTypesForPosition);

        const entry: TimesheetEntry = {
          id: generateId(),
          employeeId: employee.id,
          employee,
          projectId,
          date,
          position,
          dayType: resolvedDayType,
          hours: {},

        };

        const availableHourTypes = getHourTypesForEntry(position, resolvedDayType);

        Object.entries(hourData as Record<string, number>).forEach(([rawHourType, hours]) => {
          if (typeof hours === 'number' && hours > 0) {
            let hourType = rawHourType;
            if (availableHourTypes && availableHourTypes.length > 0) {
              const match = availableHourTypes.find(ht =>
                normalizeVietnamese(ht) === normalizeVietnamese(rawHourType)
              );
              if (match) {
                hourType = match;
              }
            }

            entry.hours[hourType] = hours;
          }
        });

        if (Object.keys(entry.hours).length > 0) {
          entry.originalValues = {
            hours: { ...entry.hours },
            dayType: entry.dayType
          };
          newEntries.push(entry);
        }
      });
    });
  });

  return newEntries;
};

interface MergeWithGapsParams {
  existingEntries: TimesheetEntry[];
  newEntries: TimesheetEntry[];
  startDate: string;
  endDate: string;
  projectId: number;
  createEntry: (employee: Employee, selectedProjectId?: number, date?: string) => TimesheetEntry;
  allProjectEmployees?: Employee[];
}

export const mergeEntriesWithGaps = ({
  existingEntries,
  newEntries,
  startDate,
  endDate,
  projectId,
  createEntry,
  allProjectEmployees = []
}: MergeWithGapsParams): TimesheetEntry[] => {
  const mergedEntries = [...existingEntries];

  newEntries.forEach(newEntry => {
    // First try exact match: same employee + date + dayType
    const exactIndex = mergedEntries.findIndex(e =>
      e.employeeId === newEntry.employeeId &&
      e.date === newEntry.date &&
      normalizeVietnamese(e.dayType || '') === normalizeVietnamese(newEntry.dayType || '')
    );

    if (exactIndex >= 0) {
      // Merge hours into the existing entry
      mergedEntries[exactIndex] = {
        ...mergedEntries[exactIndex],
        hours: { ...mergedEntries[exactIndex].hours, ...newEntry.hours },
        originalValues: newEntry.originalValues || mergedEntries[exactIndex].originalValues
      };
      return;
    }

    // No exact match — check if there's a blank placeholder for this employee+date
    // (created by auto-populate with no hours and no originalValues)
    const blankIndex = mergedEntries.findIndex(e =>
      e.employeeId === newEntry.employeeId &&
      e.date === newEntry.date &&
      !e.originalValues &&
      Object.keys(e.hours).length === 0
    );

    if (blankIndex >= 0) {
      // Replace the blank placeholder with the real API entry
      mergedEntries[blankIndex] = newEntry;
      return;
    }

    // No placeholder — just add the API entry
    mergedEntries.push(newEntry);
  });

  const start = new Date(startDate);
  const end = new Date(endDate);
  const dates: string[] = [];
  for (let date = new Date(start); date <= end; date.setDate(date.getDate() + 1)) {
    dates.push(date.toISOString().split('T')[0]);
  }

  // Build employee list from: existing entries + API entries + all project employees
  const employeeMap = new Map<number, Employee>();
  mergedEntries.forEach(entry => {
    if (entry.employeeId && entry.employee) employeeMap.set(entry.employeeId, entry.employee);
  });
  allProjectEmployees.forEach(emp => {
    if (emp && emp.id) employeeMap.set(emp.id, emp);
  });
  const uniqueEmployees = Array.from(employeeMap.values());

  const finalEntries = [...mergedEntries];

  dates.forEach(date => {
    uniqueEmployees.forEach(employee => {
      if (!employee) return;

      const exists = finalEntries.some(
        entry => entry.employeeId === employee.id && entry.date === date
      );

      if (!exists) {
        const newEntry = createEntry(employee, projectId, date);
        finalEntries.push(newEntry);
      }
    });
  });

  finalEntries.sort((a, b) => {
    if (a.employeeId !== b.employeeId) {
      return a.employeeId - b.employeeId;
    }
    return a.date.localeCompare(b.date);
  });

  return finalEntries;
};
