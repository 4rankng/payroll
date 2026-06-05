/**
 * Centralized color configuration for timesheet status indicators
 *
 * This file serves as the single source of truth for all status-related colors
 * across timesheet components to ensure consistency.
 */

import { TimesheetStatus, PaymentStatus } from '@/types/api/timesheet.types';

export type AggregatedStatus = TimesheetStatus | PaymentStatus | 'mixed' | 'none';

/**
 * Color configuration for different status types
 */
export interface StatusColors {
  /** Background color for calendar cells and badges */
  bg: string;
  /** Text color for badges */
  text: string;
  /** Border color */
  border: string;
  /** Hover background color */
  hover: string;
}

/**
 * Strip/indicator color configuration (used in table rows, legends)
 */
export interface StripColors {
  /** Background color for status strips */
  bg: string;
  /** Tailwind class that targets the pseudo-element indicator */
  indicator: string;
}

/**
 * Timesheet approval status colors
 *
 * Color scheme:
 * - Draft: Gray (darker) - Work in progress
 * - Pending Approval: Gray (light) - Awaiting review
 * - Approved: Green - Successfully reviewed
 * - Rejected: Red - Needs revision
 */
export const TIMESHEET_STATUS_COLORS: Partial<Record<TimesheetStatus, StatusColors>> = {
  draft: {
    bg: 'bg-muted',
    text: 'text-foreground',
    border: 'border-border',
    hover: 'hover:bg-gray-200/70',
  },
  pending_approval: {
    bg: 'bg-yellow-50',
    text: 'text-amber-800',
    border: 'border-yellow-200',
    hover: 'hover:bg-yellow-100/70',
  },
  approved: {
    bg: 'bg-blue-50',
    text: 'text-blue-800',
    border: 'border-blue-200',
    hover: 'hover:bg-blue-100/70',
  },
  rejected: {
    bg: 'bg-red-50',
    text: 'text-red-700',
    border: 'border-red-200',
    hover: 'hover:bg-red-100/70',
  },
};

/**
 * Payment status colors
 *
 * Color scheme:
 * - Pending: Orange - Payment processing
 * - Paid: Amber/Gold - Payment completed
 * - Failed: Red - Payment error
 * - Cancelled: Gray - Payment cancelled
 */
export const PAYMENT_STATUS_COLORS: Partial<Record<PaymentStatus, StatusColors>> = {
  pending: {
    bg: 'bg-orange-50',
    text: 'text-orange-700',
    border: 'border-orange-200',
    hover: 'hover:bg-orange-100/70',
  },
  paid: {
    bg: 'bg-green-50',
    text: 'text-green-800',
    border: 'border-green-200',
    hover: 'hover:bg-green-100/70',
  },
  failed: {
    bg: 'bg-red-50',
    text: 'text-red-700',
    border: 'border-red-200',
    hover: 'hover:bg-red-100/70',
  },
  cancelled: {
    bg: 'bg-muted/50',
    text: 'text-foreground',
    border: 'border-border',
    hover: 'hover:bg-muted/70',
  },
};

/**
 * Mixed status colors (for calendar cells with multiple statuses)
 */
export const MIXED_STATUS_COLORS: StatusColors = {
  bg: 'bg-purple-100',
  text: 'text-purple-700',
  border: 'border-purple-300',
  hover: 'hover:bg-purple-200/70',
};

/**
 * Strip/indicator colors for table rows and legends
 */
export const TIMESHEET_STRIP_COLORS: Partial<Record<TimesheetStatus, StripColors>> = {
  draft: {
    bg: 'bg-muted/500',
    indicator: 'before:bg-muted/500',
  },
  pending_approval: {
    bg: 'bg-yellow-500',
    indicator: 'before:bg-yellow-500',
  },
  approved: {
    bg: 'bg-blue-500',
    indicator: 'before:bg-blue-500',
  },
  rejected: {
    bg: 'bg-red-600',
    indicator: 'before:bg-red-600',
  },
};

export const PAYMENT_STRIP_COLORS: Partial<Record<PaymentStatus, StripColors>> = {
  pending: {
    bg: 'bg-orange-500',
    indicator: 'before:bg-orange-500',
  },
  paid: {
    bg: 'bg-green-500',
    indicator: 'before:bg-green-500',
  },
  failed: {
    bg: 'bg-red-500',
    indicator: 'before:bg-red-500',
  },
  cancelled: {
    bg: 'bg-muted/500',
    indicator: 'before:bg-muted/500',
  },
};

/**
 * Get combined color classes for a status badge
 */
export function getStatusBadgeClasses(status: TimesheetStatus | PaymentStatus): string {
  const colors = TIMESHEET_STATUS_COLORS[status as TimesheetStatus] || PAYMENT_STATUS_COLORS[status as PaymentStatus];
  if (!colors) return '';
  return `${colors.bg} ${colors.text} ${colors.border}`;
}

/**
 * Get combined color classes for calendar cells
 */
export function getCalendarCellClasses(status: AggregatedStatus): string {
  if (status === 'mixed') {
    const colors = MIXED_STATUS_COLORS;
    return `${colors.bg} ${colors.border} ${colors.hover}`;
  }

  const colors =
    TIMESHEET_STATUS_COLORS[status as TimesheetStatus] ||
    PAYMENT_STATUS_COLORS[status as PaymentStatus];

  if (!colors) return '';
  return `${colors.bg} ${colors.border} ${colors.hover}`;
}

/**
 * Get strip background class for table rows
 */
export function getStripBackgroundClass(status: TimesheetStatus | PaymentStatus): string {
  const stripColor =
    TIMESHEET_STRIP_COLORS[status as TimesheetStatus] ||
    PAYMENT_STRIP_COLORS[status as PaymentStatus];

  return stripColor?.bg || '';
}
