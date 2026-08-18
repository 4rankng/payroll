import { Timesheet } from '@/types/api/timesheet.types';
import { formatNumber } from '@/utils/formatters';

/** Formats displayed work hours without exposing floating-point precision noise. */
export function formatTimesheetHours(hours: number): string {
  return formatNumber(Number.isFinite(hours) ? hours : 0, 2);
}

export interface EmployeeGroupedTimesheet {
  groupKey: string; // employeeId_effectiveStatus
  employeeId: number;
  employeeName: string;
  employeeCode: string;
  entries: Timesheet[];
  totalHours: number;
  totalAmount: number;
  /** Effective status for the whole group — never 'mixed' */
  aggregatedStatus: 'approved' | 'pending_approval' | 'rejected' | 'draft' | 'paid';
  statusBreakdown: {
    approved: number;
    pending_approval: number;
    rejected: number;
    draft: number;
  };
  hasMultipleProjects: boolean;
  hasMultipleStatuses: boolean;
}

export interface ProjectTimesheetSection {
  projectId: number;
  projectName: string;
  projectCode?: string;
  positions: string[];
  entries: Timesheet[];
  totalHours: number;
  totalAmount: number;
}

export interface GroupedTimesheet {
  employeeId: number;
  employeeName: string;
  employeeCode: string;
  date: string;
  entries: Timesheet[];
  totalHours: number;
  totalAmount: number;
  aggregatedStatus: 'approved' | 'pending_approval' | 'mixed' | 'rejected' | 'draft';
  hasMultipleProjects: boolean;
  hasMultipleStatuses: boolean;
  statusBreakdown: {
    approved: number;
    pending_approval: number;
    rejected: number;
    draft: number;
  };
}

/**
 * Groups timesheets by employee and date combination
 */
export function groupTimesheetsByEmployeeDate(timesheets: Timesheet[]): GroupedTimesheet[] {
  const groupMap = new Map<string, GroupedTimesheet>();

  timesheets.forEach((timesheet) => {
    const key = `${timesheet.employee_id}_${timesheet.date}`;

    if (!groupMap.has(key)) {
      // Initialize new group
      groupMap.set(key, {
        employeeId: timesheet.employee_id,
        employeeName: timesheet.employeeName,
        employeeCode: timesheet.employeeCode,
        date: timesheet.date,
        entries: [],
        totalHours: 0,
        totalAmount: 0,
        aggregatedStatus: timesheet.status,
        hasMultipleProjects: false,
        hasMultipleStatuses: false,
        statusBreakdown: {
          approved: 0,
          pending_approval: 0,
          rejected: 0,
          draft: 0
        }
      });
    }

    const group = groupMap.get(key)!;
    group.entries.push(timesheet);
    group.totalHours += timesheet.hours_worked;
    group.totalAmount += timesheet.amount;

    // Update status breakdown
    group.statusBreakdown[timesheet.status]++;

    // Check if multiple projects exist
    const uniqueProjects = new Set(group.entries.map(e => e.project_id));
    group.hasMultipleProjects = uniqueProjects.size > 1;
  });

  // Calculate aggregated status and finalize groups
  return Array.from(groupMap.values()).map(group => ({
    ...group,
    aggregatedStatus: calculateAggregatedStatus(group.statusBreakdown),
    hasMultipleStatuses: Object.values(group.statusBreakdown).filter(count => count > 0).length > 1
  }));
}

/**
 * Determines the aggregated status based on individual entry statuses
 */
function calculateAggregatedStatus(statusBreakdown: GroupedTimesheet['statusBreakdown']): GroupedTimesheet['aggregatedStatus'] {
  const { approved, pending_approval, rejected, draft } = statusBreakdown;
  const totalEntries = approved + pending_approval + rejected + draft;

  // All same status
  if (approved === totalEntries) return 'approved';
  if (pending_approval === totalEntries) return 'pending_approval';
  if (rejected === totalEntries) return 'rejected';
  if (draft === totalEntries) return 'draft';

  // Mixed statuses
  return 'mixed';
}

/**
 * Sorts grouped timesheets by date (descending) then by employee name (ascending)
 */
export function sortGroupedTimesheets(groups: GroupedTimesheet[]): GroupedTimesheet[] {
  return groups.sort((a, b) => {
    // Sort by date descending first
    const dateComparison = new Date(b.date).getTime() - new Date(a.date).getTime();
    if (dateComparison !== 0) return dateComparison;

    // Then by employee name ascending
    return a.employeeName.localeCompare(b.employeeName, 'vi-VN');
  });
}

/**
 * Gets the status badge variant for mixed statuses
 */
export function getGroupedStatusBadgeVariant(status: GroupedTimesheet['aggregatedStatus']) {
  switch (status) {
    case 'approved':
      return { variant: 'default' as const, text: 'Đã duyệt', color: 'green' };
    case 'pending_approval':
      return { variant: 'secondary' as const, text: 'Chờ duyệt', color: 'yellow' };
    case 'rejected':
      return { variant: 'destructive' as const, text: 'Loại', color: 'red' };
    case 'draft':
      return { variant: 'outline' as const, text: 'Nháp', color: 'gray' };
    case 'mixed':
      return { variant: 'outline' as const, text: 'Hỗn hợp', color: 'orange' };
    default:
      return { variant: 'outline' as const, text: 'Không xác định', color: 'gray' };
  }
}

/**
 * Formats the status breakdown for mixed status tooltip
 */
export function formatStatusBreakdown(statusBreakdown: GroupedTimesheet['statusBreakdown']): string {
  const parts: string[] = [];

  if (statusBreakdown.approved > 0) {
    parts.push(`${statusBreakdown.approved} đã duyệt`);
  }
  if (statusBreakdown.pending_approval > 0) {
    parts.push(`${statusBreakdown.pending_approval} chờ duyệt`);
  }
  if (statusBreakdown.rejected > 0) {
    parts.push(`${statusBreakdown.rejected} loại`);
  }
  if (statusBreakdown.draft > 0) {
    parts.push(`${statusBreakdown.draft} nháp`);
  }

  return parts.join(', ');
}

/**
 * Gets all timesheet IDs from a group for bulk operations
 */
export function getGroupTimesheetIds(group: GroupedTimesheet): number[] {
  return group.entries.map(entry => entry.id);
}

/**
 * Gets only pending approval timesheet IDs from a group for bulk approve operations
 * This ensures only pending timesheets are sent to backend, excluding already-approved or rejected ones
 */
export function getPendingTimesheetIds(group: GroupedTimesheet): number[] {
  return group.entries
    .filter(entry => entry.status === 'pending_approval')
    .map(entry => entry.id);
}

/**
 * Checks if a group can be bulk approved (no rejected or already approved entries)
 */
export function canBulkApprove(group: GroupedTimesheet): boolean {
  return group.statusBreakdown.pending_approval > 0 &&
         group.statusBreakdown.rejected === 0 &&
         group.statusBreakdown.draft === 0;
}

/**
 * Checks if a group can be bulk rejected (has pending entries)
 */
export function canBulkReject(group: GroupedTimesheet): boolean {
  return group.statusBreakdown.pending_approval > 0;
}

/**
 * Returns the effective status key for a single timesheet entry.
 * Paid approved entries are treated as their own bucket so groups are always uniform.
 */
function getEffectiveStatus(timesheet: Timesheet): 'paid' | 'approved' | 'pending_approval' | 'rejected' | 'draft' {
  if (timesheet.status === 'approved' && timesheet.payment_status === 'paid') return 'paid';
  return timesheet.status as 'approved' | 'pending_approval' | 'rejected' | 'draft';
}

/**
 * Groups timesheets by employee × effective-status.
 *
 * Each group is guaranteed to have a single uniform status:
 *   - "pending_approval" → Chờ duyệt
 *   - "approved"         → Đã duyệt  (approved but not yet paid)
 *   - "paid"             → Đã thanh toán
 *   - "rejected"         → Loại
 *
 * There is NO "mixed" group — entries with different statuses for the same
 * employee appear as separate rows in the table.
 */
export function groupTimesheetsByEmployee(timesheets: Timesheet[]): EmployeeGroupedTimesheet[] {
  const groupMap = new Map<string, EmployeeGroupedTimesheet>();

  timesheets.forEach((timesheet) => {
    const effectiveStatus = getEffectiveStatus(timesheet);
    const key = `${timesheet.employee_id}_${effectiveStatus}`;

    if (!groupMap.has(key)) {
      groupMap.set(key, {
        groupKey: key,
        employeeId: timesheet.employee_id,
        employeeName: timesheet.employeeName,
        employeeCode: timesheet.employeeCode,
        entries: [],
        totalHours: 0,
        totalAmount: 0,
        aggregatedStatus: effectiveStatus,
        hasMultipleProjects: false,
        hasMultipleStatuses: false,
        statusBreakdown: {
          approved: 0,
          pending_approval: 0,
          rejected: 0,
          draft: 0
        }
      });
    }

    const group = groupMap.get(key)!;
    group.entries.push(timesheet);
    group.totalHours += timesheet.hours_worked || 0;
    group.totalAmount += timesheet.amount || 0;

    // statusBreakdown tracks underlying approval status (for display/tooltip)
    if (timesheet.status in group.statusBreakdown) {
      group.statusBreakdown[timesheet.status as keyof typeof group.statusBreakdown]++;
    }
  });

  return Array.from(groupMap.values()).map(group => ({
    ...group,
    hasMultipleStatuses: false, // always false — each group is uniform by design
    hasMultipleProjects: new Set(group.entries.map(e => e.project_id)).size > 1
  }));
}

function getPaytypePosition(paytype: string): string | null {
  const position = paytype.split('.')[0]?.trim();
  return position || null;
}

/**
 * Builds the display hierarchy used inside an expanded employee group.
 * Project facts and positions live on the section, while entries are sorted
 * by work date so the UI can suppress repeated dates on adjacent rows.
 */
export function groupEmployeeEntriesByProject(entries: Timesheet[]): ProjectTimesheetSection[] {
  const projects = new Map<number, ProjectTimesheetSection>();

  entries.forEach((entry) => {
    const existing = projects.get(entry.project_id);
    const position = getPaytypePosition(entry.paytype);

    if (existing) {
      existing.entries.push(entry);
      existing.totalHours += entry.hours_worked || 0;
      existing.totalAmount += entry.amount || 0;
      if (position && !existing.positions.includes(position)) existing.positions.push(position);
      return;
    }

    projects.set(entry.project_id, {
      projectId: entry.project_id,
      projectName: entry.projectName || 'Dự án chưa xác định',
      projectCode: entry.projectCode,
      positions: position ? [position] : [],
      entries: [entry],
      totalHours: entry.hours_worked || 0,
      totalAmount: entry.amount || 0,
    });
  });

  return Array.from(projects.values())
    .map((project) => ({
      ...project,
      entries: [...project.entries].sort(
        (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
      ),
    }))
    .sort((a, b) => a.projectName.localeCompare(b.projectName, 'vi-VN'));
}

/** Returns only the variable part of a pay type once position is shown above. */
export function getPaytypeDetail(paytype: string): string {
  const [, ...detailParts] = paytype.split('.').map((part) => part.trim()).filter(Boolean);
  return detailParts.join(' · ') || paytype;
}
