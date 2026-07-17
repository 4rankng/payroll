import type { TimesheetFilters } from '@/types/api/timesheet.types';

export type TimesheetStatusFilter =
  | 'all'
  | 'pending_approval'
  | 'pending_payment'
  | 'approved'
  | 'rejected'
  | 'paid'
  | 'failed'
  | 'cancelled';

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
