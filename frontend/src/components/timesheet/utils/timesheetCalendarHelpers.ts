import { type EmployeeTimesheetEntry } from '@/types/api/timesheet.types';
import {
  startOfMonth,
  endOfMonth,
  eachDayOfInterval,
  format,
  isSameDay,
  getDay,
  addDays,
  subDays
} from 'date-fns';
import { vi } from 'date-fns/locale';

/** Narrowed timesheet type with only the fields the calendar actually reads. */
export interface CalendarTimesheetEntry {
  id: number;
  project_id: number;
  employee_id: number;
  date: string;
  hours_worked: number;
  amount: number;
  paid_amount?: number;
  status: 'approved' | 'pending_approval' | 'rejected' | 'draft' | 'paid';
  payment_status?: 'pending' | 'paid' | 'failed' | 'cancelled';
  force_payroll?: boolean;
  projectName: string;
  projectCode?: string;
  employeeName: string;
  employeeCode: string;
}

export interface CalendarDay {
  date: Date;
  dayNumber: number;
  isCurrentMonth: boolean;
  isToday: boolean;
  entries: CalendarTimesheetEntry[];
  totalHours: number;
  totalAmount: number;
  totalPaidAmount: number;
  aggregatedStatus: 'approved' | 'pending_approval' | 'rejected' | 'paid' | 'mixed' | 'none';
  hasForcePayroll?: boolean;
}

export interface ProjectCalendarData {
  projectId: number;
  projectName: string;
  projectCode?: string;
  calendarDays: CalendarDay[][];
  monthlyStats: {
    totalHours: number;
    totalAmount: number;
    workingDays: number;
    statusBreakdown: Record<string, number>;
  };
}

export interface EmployeeCalendarData {
  employeeId: number;
  employeeName: string;
  employeeCode: string;
  projects: ProjectCalendarData[];
  overallStats: {
    totalHours: number;
    totalAmount: number;
    workingDays: number;
    totalProjects: number;
  };
}

/**
 * Groups timesheets by project for calendar display
 */
export function groupTimesheetsByProject(timesheets: CalendarTimesheetEntry[]): Record<number, CalendarTimesheetEntry[]> {
  return timesheets.reduce((acc, timesheet) => {
    const projectId = timesheet.project_id;
    if (!acc[projectId]) {
      acc[projectId] = [];
    }
    acc[projectId].push(timesheet);
    return acc;
  }, {} as Record<number, CalendarTimesheetEntry[]>);
}

/**
 * Determines the aggregated status for multiple entries on the same day
 * Priority: paid > approval status
 */
export function getAggregatedStatus(entries: CalendarTimesheetEntry[]): CalendarDay['aggregatedStatus'] {
  if (entries.length === 0) return 'none';

  // Priority 1: Check payment status first (paid takes highest priority)
  const paymentStatuses = entries.map(entry => entry.payment_status);
  const allPaid = paymentStatuses.every(status => status === 'paid');
  const somePaid = paymentStatuses.some(status => status === 'paid');

  if (allPaid) {
    return 'paid';
  }

  // Priority 2: If some are paid but not all, it's mixed
  if (somePaid) {
    return 'mixed';
  }

  // Priority 3: Check approval status
  const statuses = entries.map(entry => entry.status);
  const uniqueStatuses = Array.from(new Set(statuses));

  if (uniqueStatuses.length === 1) {
    return uniqueStatuses[0] as CalendarDay['aggregatedStatus'];
  }

  return 'mixed';
}


/**
 * Creates calendar grid for a specific month and project
 */
export function createCalendarGrid(
  year: number,
  month: number,
  projectTimesheets: CalendarTimesheetEntry[]
): CalendarDay[][] {
  const monthStart = startOfMonth(new Date(year, month - 1));
  const monthEnd = endOfMonth(monthStart);

  // Get first day of calendar grid (might be from previous month)
  const calendarStart = subDays(monthStart, getDay(monthStart));

  // Get last day of calendar grid (might be from next month)
  const calendarEnd = addDays(monthEnd, 6 - getDay(monthEnd));

  // Get all days in the calendar grid
  const allDays = eachDayOfInterval({ start: calendarStart, end: calendarEnd });

  // Group timesheets by date for quick lookup
  const timesheetsByDate = projectTimesheets.reduce((acc, timesheet) => {
    const dateKey = format(new Date(timesheet.date), 'yyyy-MM-dd');
    if (!acc[dateKey]) {
      acc[dateKey] = [];
    }
    acc[dateKey].push(timesheet);
    return acc;
  }, {} as Record<string, CalendarTimesheetEntry[]>);

  // Create calendar days
  const calendarDays = allDays.map(date => {
    const dateKey = format(date, 'yyyy-MM-dd');
    const entries = timesheetsByDate[dateKey] || [];
    const totalHours = entries.reduce((sum, entry) => sum + entry.hours_worked, 0);
    const totalAmount = entries.reduce((sum, entry) => sum + entry.amount, 0);
    const totalPaidAmount = entries.reduce((sum, entry) => sum + (entry.paid_amount || 0), 0);
    const hasForcePayroll = entries.some(entry => entry.force_payroll);

    return {
      date,
      dayNumber: parseInt(format(date, 'd')),
      isCurrentMonth: date >= monthStart && date <= monthEnd,
      isToday: isSameDay(date, new Date()),
      entries,
      totalHours,
      totalAmount,
      totalPaidAmount,
      aggregatedStatus: getAggregatedStatus(entries),
      hasForcePayroll,
    } as CalendarDay;
  });

  // Split into weeks (7 days each)
  const weeks: CalendarDay[][] = [];
  for (let i = 0; i < calendarDays.length; i += 7) {
    weeks.push(calendarDays.slice(i, i + 7));
  }

  return weeks;
}

/**
 * Calculates monthly statistics for a project
 */
export function calculateMonthlyStats(projectTimesheets: CalendarTimesheetEntry[]) {
  const totalHours = projectTimesheets.reduce((sum, t) => sum + t.hours_worked, 0);
  const totalAmount = projectTimesheets.reduce((sum, t) => sum + t.amount, 0);

  // Count unique working days
  const uniqueDates = new Set(projectTimesheets.map(t => t.date));
  const workingDays = uniqueDates.size;

  // Status breakdown
  const statusBreakdown = projectTimesheets.reduce((acc, t) => {
    acc[t.status] = (acc[t.status] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  return {
    totalHours,
    totalAmount,
    workingDays,
    statusBreakdown,
  };
}

/**
 * Creates complete calendar data for an employee
 */
export function createEmployeeCalendarData(
  timesheets: CalendarTimesheetEntry[],
  year: number,
  month: number,
  employee?: { id: number; name: string; subtitle?: string },
  availableProjects?: { id: number; name: string }[]
): EmployeeCalendarData | null {
  // If no timesheets but employee and projects provided, create empty calendar
  if (timesheets.length === 0) {
    if (employee && availableProjects) {
      return createEmptyEmployeeCalendarData(employee, availableProjects, year, month);
    }
    return null;
  }

  const firstTimesheet = timesheets[0];
  const groupedByProject = groupTimesheetsByProject(timesheets);

  const projects: ProjectCalendarData[] = Object.entries(groupedByProject).map(([projectIdStr, projectTimesheets]) => {
    const projectId = parseInt(projectIdStr);
    const projectName = projectTimesheets[0].projectName;
    const projectCode = projectTimesheets[0].projectCode;

    return {
      projectId,
      projectName,
      projectCode,
      calendarDays: createCalendarGrid(year, month, projectTimesheets),
      monthlyStats: calculateMonthlyStats(projectTimesheets),
    };
  });

  // Calculate overall stats
  const overallStats = {
    totalHours: timesheets.reduce((sum, t) => sum + t.hours_worked, 0),
    totalAmount: timesheets.reduce((sum, t) => sum + t.amount, 0),
    workingDays: new Set(timesheets.map(t => t.date)).size,
    totalProjects: projects.length,
  };

  return {
    employeeId: firstTimesheet.employee_id,
    employeeName: firstTimesheet.employeeName,
    employeeCode: firstTimesheet.employeeCode,
    projects,
    overallStats,
  };
}

/**
 * Creates empty calendar data for an employee with all available projects
 */
export function createEmptyEmployeeCalendarData(
  employee: { id: number; name: string; subtitle?: string },
  availableProjects: { id: number; name: string }[],
  year: number,
  month: number
): EmployeeCalendarData {
  const projects: ProjectCalendarData[] = availableProjects.map(project => ({
    projectId: project.id,
    projectName: project.name,
    projectCode: undefined,
    calendarDays: createCalendarGrid(year, month, []), // Empty timesheets array
    monthlyStats: {
      totalHours: 0,
      totalAmount: 0,
      workingDays: 0,
      statusBreakdown: {},
    },
  }));

  return {
    employeeId: employee.id,
    employeeName: employee.name,
    employeeCode: employee.subtitle || `EMP${employee.id.toString().padStart(3, '0')}`,
    projects,
    overallStats: {
      totalHours: 0,
      totalAmount: 0,
      workingDays: 0,
      totalProjects: projects.length,
    },
  };
}



/**
 * Formats currency for Vietnamese locale
 */
export function formatCurrency(amount: number): string {
  return amount.toLocaleString('vi-VN');
}

/**
 * Formats date for Vietnamese locale
 */
export function formatVietnameseDate(date: Date, formatStr: string = 'dd/MM/yyyy'): string {
  return format(date, formatStr, { locale: vi });
}

/**
 * Gets the current month and year
 */
export function getCurrentMonthYear(): { year: number; month: number } {
  const now = new Date();
  return {
    year: now.getFullYear(),
    month: now.getMonth() + 1
  };
}

/**
 * Creates calendar data from employee timesheet entries (for employee portal)
 * Simplified version that just creates calendar grid without full employee data
 * Only shows approved timesheets, with colors based on payment status
 */
export function createEmployeeCalendarDataFromTimesheets(
  timesheets: EmployeeTimesheetEntry[],
  year: number,
  month: number
): {
  calendarDays: CalendarDay[][];
  monthlyStats: ProjectCalendarData['monthlyStats'];
} {
  // Filter to only show approved timesheets
  const approvedTimesheets = timesheets.filter((entry) => entry.status === 'approved');

  // Convert EmployeeTimesheetEntry to CalendarTimesheetEntry for calendar compatibility
  // Map payment_status to status field for visual display
  const convertedTimesheets: CalendarTimesheetEntry[] = approvedTimesheets.map((entry) => ({
    id: entry.id,
    employee_id: 0, // Not needed for employee view
    employeeName: '',
    employeeCode: '',
    project_id: entry.project_id,
    projectName: entry.projectName || '',
    projectCode: (entry as any).projectCode || '',
    date: entry.date,
    hours_worked: entry.hours_worked,
    amount: entry.amount,
    // Use payment status as the visual status for employee view
    // 'paid' stays as 'paid', 'unpaid'/'pending' mapped to 'approved' (gray background)
    status: entry.payment_status === 'paid' ? 'paid' : 'approved',
    payment_status: entry.payment_status,
    paid_amount: entry.paid_amount || 0,
  }));

  const calendarDays = createCalendarGrid(year, month, convertedTimesheets);
  const monthlyStats = calculateMonthlyStats(convertedTimesheets);

  return {
    calendarDays,
    monthlyStats,
  };
}
