/**
 * Utility functions for handling payrate calculations in timesheet management
 */

import { isVietnamesePublicHoliday } from '@/utils/dateHelpers';

interface Employee {
  id: number;
  fullname: string;
  position?: string;
  employee_code?: string;
}

/**
 * Get payrate for a specific employee and work type from the flexible payrate structure
 */
export function getPayrateForEmployeeAndWorkType(
  employee: Employee,
  workTypeKey: string,
  payrates: Record<string, unknown>,
  date?: Date
): number {
  if (!payrates || !workTypeKey) return 0;

  // Find rate for employee's position or use 'tất cả' as fallback
  const position = employee.position || 'tất cả';
  const positionRates = payrates[position] || payrates['tất cả'];

  if (!positionRates) return 0;

  // Determine day type based on the date
  let dayType = 'ngày thường'; // default
  if (date) {
    const dayOfWeek = date.getDay();
    const isWeekend = dayOfWeek === 0 || dayOfWeek === 6;
    const isHoliday = isVietnamesePublicHoliday(date);

    if (isHoliday) {
      dayType = 'ngày lễ';
    } else if (isWeekend) {
      dayType = 'cuối tuần';
    } else {
      dayType = 'ngày thường';
    }
  }

  // Navigate through the nested structure to find the rate
  const dayTypeRates = positionRates[dayType];
  if (!dayTypeRates || typeof dayTypeRates !== 'object') return 0;

  const rate = dayTypeRates[workTypeKey];
  return typeof rate === 'number' ? rate : 0;
}

/**
 * Extract unique work types (third level keys) from payrate structure
 */
export function extractWorkTypesFromPayrates(payrates: Record<string, unknown>): Array<{
  key: string;
  label: string;
}> {
  const workTypesSet = new Set<string>();

  // Extract from first skill level to get structure
  const firstSkillLevel = Object.values(payrates)[0];
  if (!firstSkillLevel || typeof firstSkillLevel !== 'object') return [];

  // Iterate through all day types to collect unique work types
  Object.keys(firstSkillLevel).forEach(dayType => {
    const dayTypeRates = firstSkillLevel[dayType];
    if (typeof dayTypeRates === 'object' && dayTypeRates !== null) {
      Object.keys(dayTypeRates).forEach(workType => {
        if (typeof dayTypeRates[workType] === 'number') {
          workTypesSet.add(workType);
        }
      });
    }
  });

  // Convert set to array and create work type objects
  return Array.from(workTypesSet).map(workType => ({
    key: workType,
    label: workType
  }));
}

/**
 * Generate display labels for work types (no abbreviations, use full values)
 */
export function generateWorkTypeLabels(
  workTypes: Array<{ key: string; label: string }>
): Record<string, string> {
  const labels: Record<string, string> = {};

  // Just use the full label as-is
  workTypes.forEach(type => {
    labels[type.key] = type.label;
  });

  return labels;
}

