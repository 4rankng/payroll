
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { dateToString } from './dateHelpers';

export interface WeekPeriod {
  label: string;
  from: string;
  to: string;
  month: 'current' | 'previous';
  week: number;
}

export interface CustomDateRange {
  from: string;
  to: string;
  label: string;
}

export interface WeekPeriodsResult {
  availablePeriods: WeekPeriod[];
  defaultPeriod: WeekPeriod;
}

/**
 * Calculate the start and end dates for a specific week in a given month/year
 * Week 1: Days 1-7, Week 2: Days 8-14, Week 3: Days 15-21, Week 4: Days 22-28
 */
function getWeekPeriodDates(year: number, month: number, week: number): { from: string; to: string } {
  let startDay: number;
  let endDay: number;

  switch (week) {
    case 1:
      startDay = 1;
      endDay = 7;
      break;
    case 2:
      startDay = 8;
      endDay = 14;
      break;
    case 3:
      startDay = 15;
      endDay = 21;
      break;
    case 4:
      startDay = 22;
      endDay = 28;
      break;
    default:
      throw new Error('Week must be between 1 and 4');
  }

  const fromDate = new Date(year, month - 1, startDay);
  const toDate = new Date(year, month - 1, endDay);

  return {
    from: dateToString(fromDate),
    to: dateToString(toDate)
  };
}

/**
 * Create a WeekPeriod object for a specific week
 */
function createWeekPeriod(
  year: number,
  month: number,
  week: number,
  monthType: 'current' | 'previous'
): WeekPeriod {
  const { from, to } = getWeekPeriodDates(year, month, week);

  return {
    label: `Tháng ${month}`,
    from,
    to,
    month: monthType,
    week
  };
}

/**
 * Determine which week periods to show and which one should be default
 * based on the current date
 */
export function getAvailableWeekPeriods(currentDate: Date = new Date()): WeekPeriodsResult {
  const currentDay = currentDate.getDate();
  const currentMonth = currentDate.getMonth() + 1; // 1-based month
  const currentYear = currentDate.getFullYear();

  // Calculate previous month/year
  let previousMonth = currentMonth - 1;
  let previousYear = currentYear;
  if (previousMonth === 0) {
    previousMonth = 12;
    previousYear = currentYear - 1;
  }

  let availablePeriods: WeekPeriod[] = [];
  let defaultPeriod: WeekPeriod;

  // When current day is <= 10, show the last period of previous month and first period of current month
  // This ensures users can select timesheets spanning the month boundary during early month
  if (currentDay >= 1 && currentDay <= 10) {
    // Show Week 4 of previous month + Week 1 of current month, default = Week 1 current
    // Goal: Allow users to select both the end of previous month and start of current month
    availablePeriods = [
      createWeekPeriod(previousYear, previousMonth, 4, 'previous'),
      createWeekPeriod(currentYear, currentMonth, 1, 'current')
    ];
    defaultPeriod = availablePeriods[1];
  } else if (currentDay >= 11 && currentDay <= 14) {
    // Show Week 1 & 2 of current month, default = Week 2 current
    availablePeriods = [
      createWeekPeriod(currentYear, currentMonth, 1, 'current'),
      createWeekPeriod(currentYear, currentMonth, 2, 'current')
    ];
    defaultPeriod = availablePeriods[1];
  } else if (currentDay >= 15 && currentDay <= 21) {
    // Show Week 1 & 2 of current, default = Week 2 current
    availablePeriods = [
      createWeekPeriod(currentYear, currentMonth, 1, 'current'),
      createWeekPeriod(currentYear, currentMonth, 2, 'current')
    ];
    defaultPeriod = availablePeriods[1];
  } else if (currentDay >= 22 && currentDay < 28) {
    // Day 22-27: Show Week 2 & 3 of current, default = Week 3 current
    availablePeriods = [
      createWeekPeriod(currentYear, currentMonth, 2, 'current'),
      createWeekPeriod(currentYear, currentMonth, 3, 'current')
    ];
    defaultPeriod = availablePeriods[1];
  } else {
    // Day 28+: Show Week 3 & 4 of current, default = Week 4 current
    availablePeriods = [
      createWeekPeriod(currentYear, currentMonth, 3, 'current'),
      createWeekPeriod(currentYear, currentMonth, 4, 'current')
    ];
    defaultPeriod = availablePeriods[1];
  }

  return {
    availablePeriods,
    defaultPeriod
  };
}


/**
 * Get the last day of a specific month/year
 */
function getLastDayOfMonth(year: number, month: number): number {
  return new Date(year, month, 0).getDate();
}

/**
 * Format week period for display in DD - DD Tháng MM format
 */
export function formatWeekPeriodDisplay(period: WeekPeriod, mobile?: boolean): string {
  // period.from and period.to are in YYYY-MM-DD format.
  // We need to parse them respecting the local timezone, as they were created
  // using local timezone dates. Adding T00:00:00 ensures this.
  const fromDate = new Date(`${period.from}T00:00:00`);
  const toDate = new Date(`${period.to}T00:00:00`);

  // Format: "01 - 07 Tháng 11"
  const fromDay = format(fromDate, "dd", { locale: vi });
  const toDay = format(toDate, "dd", { locale: vi });
  const month = format(fromDate, "MM", { locale: vi });

  return `${fromDay} - ${toDay} Tháng ${month}`;
}

/**
 * Check if current date is >= 28th of the month
 */
export function isTodayAfter28th(currentDate: Date = new Date()): boolean {
  return currentDate.getDate() >= 28;
}

/**
 * Check if current date is <= 10th of the month
 */
export function isEarlyMonth(currentDate: Date = new Date()): boolean {
  return currentDate.getDate() <= 10;
}

/**
 * Get custom date ranges for when current day >= 28 OR <= 10
 */
export function getCustomDateRanges(currentDate: Date = new Date()): CustomDateRange[] {
  const currentYear = currentDate.getFullYear();
  const currentMonth = currentDate.getMonth(); // 0-based

  // Case 1: Late month (>= 28th)
  if (isTodayAfter28th(currentDate)) {
    const monthNumber = format(currentDate, 'MM', { locale: vi });

    // Range 1: 15-21 Current Month
    const range15_21 = getWeekPeriodDates(currentYear, currentMonth + 1, 3);
    // Range 2: 22-28 Current Month
    const range22_28 = getWeekPeriodDates(currentYear, currentMonth + 1, 4);

    return [
        { from: range15_21.from, to: range15_21.to, label: `15 - 21 Tháng ${monthNumber}` },
        { from: range22_28.from, to: range22_28.to, label: `22 - 28 Tháng ${monthNumber}` }
    ];
  }

  // Case 2: Early month (<= 10th)
  if (isEarlyMonth(currentDate)) {
    // Calculate Previous Month
    let prevMonthIndex = currentMonth - 1;
    let prevYear = currentYear;
    if (prevMonthIndex < 0) {
      prevMonthIndex = 11;
      prevYear = currentYear - 1;
    }

    const prevMonthDate = new Date(prevYear, prevMonthIndex, 1);
    const prevMonthNumber = format(prevMonthDate, 'MM', { locale: vi });
    const currentMonthNumber = format(currentDate, 'MM', { locale: vi });

    // Range 1: 22-28 Previous Month
    const rangePrev22_28 = getWeekPeriodDates(prevYear, prevMonthIndex + 1, 4);

    // Range 2: 01-07 Current Month
    const currentMonthDateStart = new Date(currentYear, currentMonth, 1);
    const currentMonthDateEnd = new Date(currentYear, currentMonth, 7);

    return [
      { from: rangePrev22_28.from, to: rangePrev22_28.to, label: `22 - 28 Tháng ${prevMonthNumber}` },
      {
        from: dateToString(currentMonthDateStart),
        to: dateToString(currentMonthDateEnd),
        label: `01 - 07 Tháng ${currentMonthNumber}`
      }
    ];
  }

  return [];
}
