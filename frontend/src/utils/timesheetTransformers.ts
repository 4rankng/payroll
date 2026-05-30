/**
 * Utilities for transforming timesheet entries between UI and API formats
 */

import type { Employee } from '@/types/api/employee.types';
import type { TimesheetEntry } from '../components/sheets/timesheet-entry/types/multi-timesheet.types';
import type { NewTimesheetEntry } from '@/types/api/timesheet.types';
import { normalizeVietnamese } from './vietnameseHelpers';

/**
 * Simple ID generator (using slice instead of deprecated substr)
 */
export const generateId = (): string => Math.random().toString(36).slice(2, 11);

/**
 * Generate a unique cache key for a timesheet entry
 * Uses sorted hour entries to ensure consistent keys
 */
export const generateEntryKey = (entry: TimesheetEntry): string => {
  const hoursKey = entry.hours
    ? Object.entries(entry.hours)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, v]) => `${k}:${v}`)
        .join('|')
    : '';

  return `${entry.employeeId}-${entry.projectId}-${entry.date}-${hoursKey}-${entry.dayType || ''}`;
};

/**
 * Transform UI timesheet entries to API format
 * Splits each entry into multiple API entries (one per hour type)
 *
 * For existing entries (with originalValues):
 * - If day type changed: delete all old entries (hoursWorked=0) and create new ones
 * - If day type same: only include hour types that changed (including 0 for deletion)
 *
 * For new entries (without originalValues):
 * - Only includes hour types with positive values
 */
export const transformEntriesToAPI = (entries: TimesheetEntry[]): NewTimesheetEntry[] => {
  const apiEntries: NewTimesheetEntry[] = [];

  entries.forEach(entry => {
    // Check if this is an existing entry (has originalValues)
    const hasOriginalValues = !!entry.originalValues;

    if (hasOriginalValues) {
      const originalHours = entry.originalValues!.hours || {};
      const currentHours = entry.hours || {};
      const originalDayType = entry.originalValues!.dayType || '';
      const currentDayType = entry.dayType || '';

      // Check if day type changed (normalized comparison)
      const dayTypeChanged = normalizeVietnamese(originalDayType) !== normalizeVietnamese(currentDayType);

      if (dayTypeChanged) {
        // Day type changed: delete old entries and create new ones

        // 1. Delete all old entries with original day type (send hoursWorked=0)
        Object.keys(originalHours).forEach(hourType => {
          if (originalHours[hourType] > 0) {
            apiEntries.push({
              projectId: entry.projectId,
              employeeId: entry.employeeId,
              date: entry.date,
              hoursWorked: 0, // Delete old entry
              hourType,
              dayType: originalDayType as 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ'
            });
          }
        });

        // 2. Create new entries with current day type (only non-zero hours)
        Object.entries(currentHours).forEach(([hourType, hoursWorked]) => {
          if (hoursWorked > 0) {
            apiEntries.push({
              projectId: entry.projectId,
              employeeId: entry.employeeId,
              date: entry.date,
              hoursWorked,
              hourType,
              dayType: currentDayType as 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ'
            });
          }
        });
      } else {
        // Day type same: only include hour types that changed
        const allHourTypes = new Set([
          ...Object.keys(originalHours),
          ...Object.keys(currentHours)
        ]);

        allHourTypes.forEach(hourType => {
          const originalValue = originalHours[hourType] || 0;
          const currentValue = currentHours[hourType] || 0;

          // Only include if value changed
          if (originalValue !== currentValue) {
            apiEntries.push({
              projectId: entry.projectId,
              employeeId: entry.employeeId,
              date: entry.date,
              hoursWorked: currentValue, // Can be 0 to delete
              hourType,
              dayType: currentDayType as 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ'
            });
          }
        });
      }
    } else {
      // For new entries: only include hour types with positive values
      if (!entry.hours || Object.keys(entry.hours).length === 0) return;

      Object.entries(entry.hours).forEach(([hourType, hoursWorked]) => {
        // Skip if hours is 0 or falsy for new entries
        if (!hoursWorked || hoursWorked <= 0) return;

        apiEntries.push({
          projectId: entry.projectId,
          employeeId: entry.employeeId,
          date: entry.date,
          hoursWorked,
          hourType,
          dayType: entry.dayType as 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ' | undefined
        });
      });
    }
  });

  return apiEntries;
};

/**
 * Get employee position safely with type guard
 */
export const getEmployeePosition = (employee?: Employee): string => {
  if (!employee) return '';

  if ('assignment' in employee && employee.assignment) {
    const assignment = employee.assignment as { position: string };
    return assignment.position || '';
  }

  return (employee.position as string) || '';
};

/**
 * Merge API entries to UI format (group by employee/date/project)
 * Combines multiple API entries (different hour types) into single UI entries
 */
export const mergeAPIEntriesToUI = (
  apiEntries: Array<{
    projectId: number;
    employeeId: number;
    date: string;
    hoursWorked: number;
    hourType: string;
    dayType?: string;
  }>,
  availableEmployees: Employee[]
): TimesheetEntry[] => {
  const grouped = new Map<string, TimesheetEntry>();

  apiEntries.forEach(apiEntry => {
    const key = `${apiEntry.employeeId}-${apiEntry.date}-${apiEntry.projectId}`;

    if (!grouped.has(key)) {
      // Find employee from available employees
      const employee = availableEmployees.find(emp => emp.id === apiEntry.employeeId);

      grouped.set(key, {
        id: generateId(),
        employeeId: apiEntry.employeeId,
        employee,
        projectId: apiEntry.projectId,
        date: apiEntry.date,
        hours: {},
        position: getEmployeePosition(employee),
        dayType: apiEntry.dayType as 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ' | undefined
      });
    }

    const entry = grouped.get(key)!;
    entry.hours[apiEntry.hourType] = apiEntry.hoursWorked;
  });

  return Array.from(grouped.values());
};

/**
 * Check if two entries are equal for validation purposes
 * Uses normalized Vietnamese comparison for day types
 */
export const entriesEqual = (entry1: TimesheetEntry, entry2: TimesheetEntry): boolean => {
  return (
    entry1.employeeId === entry2.employeeId &&
    entry1.projectId === entry2.projectId &&
    entry1.date === entry2.date &&
    JSON.stringify(entry1.hours || {}) === JSON.stringify(entry2.hours || {}) &&
    normalizeVietnamese(entry1.dayType || '') === normalizeVietnamese(entry2.dayType || '')
  );
};
