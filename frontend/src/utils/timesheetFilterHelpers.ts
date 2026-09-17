import type { TimesheetFilters } from '@/types/api/timesheet.types';

const TIMESHEET_STATUS_FILTERS = [
  'all',
  'pending_approval',
  'pending_payment',
  'approved',
  'rejected',
  'paid',
  'failed',
  'cancelled',
] as const;

export type TimesheetStatusFilter = (typeof TIMESHEET_STATUS_FILTERS)[number];

export function parseTimesheetStatusFilter(value: string | null): TimesheetStatusFilter | undefined {
  return TIMESHEET_STATUS_FILTERS.find((status) => status === value);
}

export function buildTimesheetStatusFilters(
  statusFilter: TimesheetStatusFilter,
): Pick<TimesheetFilters, 'status'> {
  if (statusFilter === 'all') return {};

  if (statusFilter === 'pending_payment') {
    return { status: 'pending_payment' };
  }

  if (statusFilter === 'paid') {
    return { status: 'paid' };
  }

  return { status: statusFilter };
}
