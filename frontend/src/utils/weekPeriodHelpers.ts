
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
 * Week 1: Days 1-7, Week 2: Days 8-14, Week 3: Days 15-21, Week 4: Days 22 through end of month
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
      endDay = getLastDayOfMonth(year, month);
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
  } else {
    // Day 22 through month end: show Week 3 & 4 of current, default = Week 3.
    // Week 4 (day 22 to month end) stays selectable for early preparation of
    // the day-1 payment, but the default never lands on an in-progress week.
    availablePeriods = [
      createWeekPeriod(currentYear, currentMonth, 3, 'current'),
      createWeekPeriod(currentYear, currentMonth, 4, 'current')
    ];
    defaultPeriod = availablePeriods[0];
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
    // Range 2: 22 through end of current month
    const range22_eom = getWeekPeriodDates(currentYear, currentMonth + 1, 4);
    const lastDay = getLastDayOfMonth(currentYear, currentMonth + 1);

    return [
        { from: range15_21.from, to: range15_21.to, label: `15 - 21 Tháng ${monthNumber}` },
        { from: range22_eom.from, to: range22_eom.to, label: `22 - ${lastDay} Tháng ${monthNumber}` }
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

    // Range 1: 22 through end of previous month
    const rangePrev22_eom = getWeekPeriodDates(prevYear, prevMonthIndex + 1, 4);
    const prevLastDay = getLastDayOfMonth(prevYear, prevMonthIndex + 1);

    // Range 2: 01-07 Current Month
    const currentMonthDateStart = new Date(currentYear, currentMonth, 1);
    const currentMonthDateEnd = new Date(currentYear, currentMonth, 7);

    return [
      { from: rangePrev22_eom.from, to: rangePrev22_eom.to, label: `22 - ${prevLastDay} Tháng ${prevMonthNumber}` },
      {
        from: dateToString(currentMonthDateStart),
        to: dateToString(currentMonthDateEnd),
        label: `01 - 07 Tháng ${currentMonthNumber}`
      }
    ];
  }

  return [];
}

/**
 * The month the weekly payroll screens open on.
 *
 * Weekly timesheets arrive a week at a time (weeks of 1-7, 8-14, 15-21,
 * 22-month end) and are approved after the fact, so in the first days of a new
 * month the unapproved batch is still the previous month's final week. Defaulting to
 * the calendar month opened the screen on an empty window with that batch
 * sitting right there — on 2026-10-01 the tiles read zero while 1,573 timesheets
 * dated 2026-09-22..28 were pending.
 *
 * From the 4th the current month is the live one and is the right default.
 *
 * This is the WEEKLY rule. It is deliberately not the FlexPay advance period,
 * which rolls on the 20th: the two subsystems use "period" for different things
 * and borrowing one calendar from the other shows the wrong month.
 */
export function getWeeklyPayrollDefaultMonth(currentDate: Date = new Date()): string {
  const year = currentDate.getFullYear();
  const monthIndex = currentDate.getMonth();
  const resolved = currentDate.getDate() <= 3 ? new Date(year, monthIndex - 1, 1) : new Date(year, monthIndex, 1);
  return `${resolved.getFullYear()}-${String(resolved.getMonth() + 1).padStart(2, '0')}`;
}
