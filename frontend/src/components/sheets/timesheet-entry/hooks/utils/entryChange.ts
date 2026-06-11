import type { TimesheetEntry } from '../../types/multi-timesheet.types';
import { normalizeVietnamese } from '@/utils/vietnamese';

/**
 * Determine whether a timesheet entry differs from its original values.
 */
export const hasEntryChangedFromOriginal = (entry: TimesheetEntry): boolean => {
  if (!entry.originalValues) {
    return true;
  }

  const dayTypeChanged = normalizeVietnamese(entry.dayType || '') !==
    normalizeVietnamese(entry.originalValues.dayType || '');

  // Compare hours as integers, not as JSON strings
  const currentHours = entry.hours || {};
  const originalHours = entry.originalValues.hours || {};

  // Get all unique hour type keys from both objects
  const allHourTypes = new Set([
    ...Object.keys(currentHours),
    ...Object.keys(originalHours)
  ]);

  // Compare each hour type numerically
  const hoursChanged = Array.from(allHourTypes).some(hourType => {
    const currentValue = currentHours[hourType] || 0;
    const originalValue = originalHours[hourType] || 0;
    return currentValue !== originalValue;
  });

  return dayTypeChanged || hoursChanged;
};
