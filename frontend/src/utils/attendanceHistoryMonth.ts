import {
  addMonths,
  endOfMonth,
  format,
  isBefore,
  startOfMonth,
  subMonths,
} from "date-fns";

export interface AttendanceHistoryMonth {
  monthDate: Date;
  label: string;
  fromDate: string;
  toDate: string;
  canGoNext: boolean;
}

export function getAttendanceHistoryMonth(date: Date, now = new Date()): AttendanceHistoryMonth {
  const monthDate = startOfMonth(date);
  const currentMonth = startOfMonth(now);

  return {
    monthDate,
    label: format(monthDate, "MM/yyyy"),
    fromDate: format(monthDate, "yyyy-MM-dd"),
    toDate: format(endOfMonth(monthDate), "yyyy-MM-dd"),
    canGoNext: isBefore(monthDate, currentMonth),
  };
}

export function getPreviousAttendanceHistoryMonth(date: Date): Date {
  return subMonths(date, 1);
}

export function getNextAttendanceHistoryMonth(date: Date): Date {
  return addMonths(date, 1);
}
