import { dateToString } from './dateHelpers';

export interface MonthPeriod {
  label: string;
  from: string;
  to: string;
  month: 'current' | 'previous';
}

export interface MonthPeriodsResult {
  availablePeriods: MonthPeriod[];
  defaultPeriod: MonthPeriod;
}

/**
 * Calculate the monthly salary period dates (26th of previous month to 25th of current month)
 */
function getMonthlyPeriodDates(year: number, month: number): { from: string; to: string } {
  // Calculate previous month/year for "from" date
  let previousMonth = month - 1;
  let previousYear = year;
  if (previousMonth === 0) {
    previousMonth = 12;
    previousYear = year - 1;
  }

  // From: 26th of previous month
  const fromDate = new Date(previousYear, previousMonth - 1, 26);

  // To: 25th of current month
  const toDate = new Date(year, month - 1, 25);

  return {
    from: dateToString(fromDate),
    to: dateToString(toDate)
  };
}

/**
 * Create a MonthPeriod object for a specific month
 */
function createMonthPeriod(
  year: number,
  month: number,
  monthType: 'current' | 'previous'
): MonthPeriod {
  const { from, to } = getMonthlyPeriodDates(year, month);

  // Format month name in Vietnamese
  const monthNames = [
    'Tháng 1', 'Tháng 2', 'Tháng 3', 'Tháng 4', 'Tháng 5', 'Tháng 6',
    'Tháng 7', 'Tháng 8', 'Tháng 9', 'Tháng 10', 'Tháng 11', 'Tháng 12'
  ];

  const monthLabel = monthNames[month - 1];

  return {
    label: monthLabel,
    from,
    to,
    month: monthType
  };
}

/**
 * Get available monthly periods and default period
 * Shows current and previous month, with current month as default
 */
export function getAvailableMonthPeriods(currentDate: Date = new Date()): MonthPeriodsResult {
  const currentMonth = currentDate.getMonth() + 1; // 1-based month
  const currentYear = currentDate.getFullYear();

  // Calculate previous month/year
  let previousMonth = currentMonth - 1;
  let previousYear = currentYear;
  if (previousMonth === 0) {
    previousMonth = 12;
    previousYear = currentYear - 1;
  }

  const availablePeriods: MonthPeriod[] = [
    createMonthPeriod(previousYear, previousMonth, 'previous'),
    createMonthPeriod(currentYear, currentMonth, 'current')
  ];

  // Default to current month
  const defaultPeriod = availablePeriods[1];

  return {
    availablePeriods,
    defaultPeriod
  };
}

/**
 * Format month period for display in DD-MM -> DD-MM format
 */
export function formatMonthPeriodDisplay(period: MonthPeriod): string {
  // period.from and period.to are in YYYY-MM-DD format.
  // We need to parse them respecting the local timezone, as they were created
  // using local timezone dates. Adding T00:00:00 ensures this.
  const fromDate = new Date(`${period.from}T00:00:00`);
  const toDate = new Date(`${period.to}T00:00:00`);

  const fromDay = String(fromDate.getDate()).padStart(2, '0');
  const fromMonth = String(fromDate.getMonth() + 1).padStart(2, '0');

  const toDay = String(toDate.getDate()).padStart(2, '0');
  const toMonth = String(toDate.getMonth() + 1).padStart(2, '0');

  return `${fromDay}/${fromMonth} → ${toDay}/${toMonth}`;
}
