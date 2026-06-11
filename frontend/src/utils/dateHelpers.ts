import { formatDate as formatDateUtil } from '@/utils/formatters';
import { DayType } from '@/types/api/payrate.types';

/**
 * Vietnam public holidays for the current year
 * Note: This is a simplified version. In production, you might want to:
 * - Fetch from an API
 * - Store in database with admin management
 * - Handle lunar calendar holidays properly
 */
const VIETNAM_PUBLIC_HOLIDAYS_2024 = [
  '2024-01-01', // New Year's Day
  '2024-02-08', // Vietnamese New Year (Tet) - Day 1
  '2024-02-09', // Vietnamese New Year (Tet) - Day 2  
  '2024-02-10', // Vietnamese New Year (Tet) - Day 3
  '2024-02-11', // Vietnamese New Year (Tet) - Day 4
  '2024-02-12', // Vietnamese New Year (Tet) - Day 5
  '2024-02-13', // Vietnamese New Year (Tet) - Day 6
  '2024-02-14', // Vietnamese New Year (Tet) - Day 7
  '2024-04-18', // Hung Kings' Commemoration Day
  '2024-04-30', // Liberation Day
  '2024-05-01', // International Workers' Day
  '2024-09-02', // Independence Day
];

// For future years, you would need to update or fetch these dates
const VIETNAM_PUBLIC_HOLIDAYS_2025 = [
  '2025-01-01', // New Year's Day
  '2025-01-29', // Vietnamese New Year (Tet) - Day 1 (approximate)
  '2025-01-30', // Vietnamese New Year (Tet) - Day 2
  '2025-01-31', // Vietnamese New Year (Tet) - Day 3
  '2025-02-01', // Vietnamese New Year (Tet) - Day 4
  '2025-02-02', // Vietnamese New Year (Tet) - Day 5
  '2025-02-03', // Vietnamese New Year (Tet) - Day 6
  '2025-02-04', // Vietnamese New Year (Tet) - Day 7
  '2025-04-07', // Hung Kings' Commemoration Day (approximate)
  '2025-04-30', // Liberation Day
  '2025-05-01', // International Workers' Day
  '2025-09-02', // Independence Day
];

const ALL_VIETNAM_PUBLIC_HOLIDAYS = [
  ...VIETNAM_PUBLIC_HOLIDAYS_2024,
  ...VIETNAM_PUBLIC_HOLIDAYS_2025,
];

/**
 * Check if a given date is a Vietnam public holiday
 */
export function isVietnamesePublicHoliday(date: string | Date): boolean {
  const dateStr = typeof date === 'string' ? date : date.toISOString().split('T')[0];
  return ALL_VIETNAM_PUBLIC_HOLIDAYS.includes(dateStr);
}

/**
 * Determine the day type based on date
 * @param date - Date string in YYYY-MM-DD format or Date object
 * @returns DayType for the given date
 */
export function getDayType(date: string | Date): DayType {
  const dateObj = typeof date === 'string' ? new Date(date + 'T00:00:00') : date;
  const dayOfWeek = dateObj.getDay(); // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
  
  // Check if it's a public holiday first
  if (isVietnamesePublicHoliday(date)) {
    return 'ngày lễ';
  }
  
  // Sunday is always "ngày nghỉ"
  if (dayOfWeek === 0) {
    return 'ngày nghỉ';
  }
  
  // Monday to Friday are "ngày thường"
  if (dayOfWeek >= 1 && dayOfWeek <= 5) {
    return 'ngày thường';
  }
  
  // Saturday requires user input - we return null to indicate this
  // The calling code should handle Saturday selection separately
  throw new Error('Saturday day type must be specified by user');
}

/**
 * Check if a date is Saturday and requires user input for day type
 */
export function isSaturdayRequiringDayType(date: string | Date): boolean {
  const dateObj = typeof date === 'string' ? new Date(date + 'T00:00:00') : date;
  const dayOfWeek = dateObj.getDay();
  
  // If it's Saturday and not a public holiday, it requires user input
  return dayOfWeek === 6 && !isVietnamesePublicHoliday(date);
}

/**
 * Get day type with Saturday handling
 * Returns the day type or null if Saturday requires user selection
 */
export function getDayTypeOrNull(date: string | Date): DayType | null {
  try {
    return getDayType(date);
  } catch (error) {
    // This happens for Saturdays that require user input
    if (isSaturdayRequiringDayType(date)) {
      return null;
    }
    throw error;
  }
}

/**
 * Format date for display
 */
export function formatDateForDisplay(date: string | Date): string {
  return formatDateUtil(date);
}

/**
 * Get day of week in Vietnamese
 */
export function getDayOfWeekVietnamese(date: string | Date): string {
  const dateObj = typeof date === 'string' ? new Date(date + 'T00:00:00') : date;
  const dayOfWeek = dateObj.getDay();
  
  const dayNames = [
    'Chủ nhật',
    'Thứ 2',
    'Thứ 3', 
    'Thứ 4',
    'Thứ 5',
    'Thứ 6',
    'Thứ 7'
  ];
  
  return dayNames[dayOfWeek];
}

/**
 * Check if date string is in correct format (YYYY-MM-DD)
 */
export function isValidDateString(dateString: string): boolean {
  const regex = /^\d{4}-\d{2}-\d{2}$/;
  if (!regex.test(dateString)) {
    return false;
  }
  
  const date = new Date(dateString + 'T00:00:00');
  return !isNaN(date.getTime());
}

/**
 * Convert Date object to YYYY-MM-DD string
 * Uses local timezone to avoid UTC conversion issues
 */
export function dateToString(date: Date): string {
  const year = date.getFullYear();
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Get the day type description in Vietnamese
 */
export function getDayTypeDescription(dayType: DayType): string {
  const descriptions = {
    'ngày thường': 'Ngày làm việc bình thường (Thứ 2 - Thứ 6)',
    'ngày nghỉ': 'Ngày nghỉ (Chủ nhật hoặc Thứ 7 nghỉ)',
    'ngày lễ': 'Ngày lễ quốc gia'
  };

  return descriptions[dayType];
}

/**
 * Generate month options for dropdowns
 * @param count - Number of months to generate (default: 12)
 * @returns Array of month options with value (YYYY-MM) and label
 */
export function generateMonthOptions(count: number = 12): Array<{ value: string; label: string }> {
  const months = [];
  const today = new Date();

  for (let i = 0; i < count; i++) {
    const date = new Date(today.getFullYear(), today.getMonth() - i, 1);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const value = `${year}-${month}`;
    const label = `Tháng ${month}/${year}`;
    months.push({ value, label });
  }

  return months;
}

/**
 * Check if an ISO date string occurs today (local time)
 */
export function isISODateToday(dateStr: string): boolean {
  const dt = new Date(dateStr);
  if (isNaN(dt.getTime())) return false;
  const now = new Date();
  return (
    dt.getFullYear() === now.getFullYear() &&
    dt.getMonth() === now.getMonth() &&
    dt.getDate() === now.getDate()
  );
}
