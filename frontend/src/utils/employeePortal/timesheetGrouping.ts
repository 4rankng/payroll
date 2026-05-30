import type { EmployeeTimesheetEntry } from '@/types/api/auth.types';

export type AggregatedPaymentStatus = 'paid' | 'unpaid' | 'partial' | 'pending';

export interface AggregatedDay {
  date: string; // YYYY-MM-DD
  totalHours: number;
  totalAmount: number;
  totalPaidAmount: number;
  paymentStatus: AggregatedPaymentStatus;
  hasForcePayroll?: boolean;
}

export interface ProjectTimesheetGroup {
  project: EmployeeTimesheetEntry['project'];
  entries: EmployeeTimesheetEntry[];
  totalHours: number;
  totalAmount: number;
  totalPaidAmount: number;
  uniqueDayCount: number;
  dailyAggregates: AggregatedDay[];
}

/**
 * Groups employee timesheet entries by project and aggregates totals.
 * Also computes the number of unique calendar days in each project group.
 */
export const groupTimesheetsByProject = (
  timesheets: EmployeeTimesheetEntry[]
): ProjectTimesheetGroup[] => {
  const groups: Record<number, ProjectTimesheetGroup> = {};

  for (const entry of timesheets) {
    const projectId = entry.project.id;
    if (!groups[projectId]) {
      groups[projectId] = {
        project: entry.project,
        entries: [],
        totalHours: 0,
        totalAmount: 0,
        totalPaidAmount: 0,
        uniqueDayCount: 0,
        dailyAggregates: [],
      };
    }

    const group = groups[projectId];
    group.entries.push(entry);
    group.totalHours += entry.hours_worked;
    group.totalAmount += entry.amount;
    group.totalPaidAmount += entry.paid_amount;
  }

  // Compute unique day counts and daily aggregates per group
  for (const group of Object.values(groups)) {
    const dailyMap: Record<string, AggregatedDay> = {};
    for (const entry of group.entries) {
      const day = normalizeToDate(entry.date);
      if (!dailyMap[day]) {
        dailyMap[day] = {
          date: day,
          totalHours: 0,
          totalAmount: 0,
          totalPaidAmount: 0,
        };
      }
      dailyMap[day].totalHours += entry.hours_worked;
      dailyMap[day].totalAmount += entry.amount;
      dailyMap[day].totalPaidAmount += entry.paid_amount;
    }
    group.uniqueDayCount = Object.keys(dailyMap).length;
    // Keep days sorted descending by date
    group.dailyAggregates = Object.values(dailyMap).sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : 0));
  }

  return Object.values(groups);
};

// Helper to get YYYY-MM-DD portion of various date strings
export const normalizeToDate = (value: string): string => {
  if (/^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10);
  if (value.includes('T')) return value.split('T')[0];
  const d = new Date(value);
  if (!isNaN(d.getTime())) {
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }
  return value;
};

/**
 * Aggregates all timesheet entries by unique day (across all projects).
 * Returns days sorted descending by date.
 */
export const groupTimesheetsByDay = (
  timesheets: EmployeeTimesheetEntry[]
): AggregatedDay[] => {
  type AggregatedDayInternal = AggregatedDay & { _paid: number; _unpaid: number; _pending: number; _total: number; _hasForcePayroll: boolean };
  const map: Record<string, AggregatedDayInternal> = {};
  for (const entry of timesheets) {
    const day = normalizeToDate(entry.date);
    if (!map[day]) {
      map[day] = {
        date: day,
        totalHours: 0,
        totalAmount: 0,
        totalPaidAmount: 0,
        paymentStatus: 'unpaid',
        hasForcePayroll: false,
        _paid: 0,
        _unpaid: 0,
        _pending: 0,
        _total: 0,
        _hasForcePayroll: false,
      };
    }
    map[day].totalHours += entry.hours_worked;
    map[day].totalAmount += entry.amount;
    map[day].totalPaidAmount += entry.paid_amount;
    map[day]._total += 1;
    if (entry.payment_status === 'paid') map[day]._paid += 1;
    else if (entry.payment_status === 'pending') map[day]._pending += 1;
    else map[day]._unpaid += 1;
    // Track if any entry has force_payroll flag
    if (entry.force_payroll) {
      map[day]._hasForcePayroll = true;
    }
  }
  const days = Object.values(map).map(d => {
    let paymentStatus: AggregatedPaymentStatus = 'unpaid';
    if (d._paid === d._total) paymentStatus = 'paid';
    else if (d._paid === 0 && d._pending > 0) paymentStatus = 'pending';
    else if (d._paid > 0) paymentStatus = 'partial';
    else paymentStatus = 'unpaid';
    const { _paid, _unpaid, _pending, _total, _hasForcePayroll, ...rest } = d;
    return { ...rest, paymentStatus, hasForcePayroll: _hasForcePayroll } as AggregatedDay;
  });
  return days.sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : 0));
};
