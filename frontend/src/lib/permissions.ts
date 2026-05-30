import { authManager } from '@/lib/auth';
import { Timesheet } from '@/types/api/timesheet.types';

/**
 * Permission utilities
 * Handles business rules for role-based actions
 */

export type UserRole = 'admin' | 'partner';

/**
 * Check if Partner can edit a timesheet
 * Rules:
 * - Admin can always edit timesheets
 * - Partner can edit timesheets EXCEPT when status is 'approved' or 'rejected'
 */
export const canPartnerEditTimesheet = (
  timesheet: Timesheet | { status: string },
  userRole?: UserRole | null
): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  // Admin has full access
  if (role === 'admin') return true;

  // Partner cannot edit approved or rejected timesheets
  if (role === 'partner') {
    return timesheet.status !== 'approved' && timesheet.status !== 'rejected';
  }

  return false;
};

/**
 * Check if user can edit a timesheet entry
 * Rules:
 * - Admin can edit approved timesheets if not paid
 * - Admin can edit any timesheet that is not paid
 * - Partner can only edit pending/draft timesheets
 */
export const canEditTimesheet = (
  timesheet: Timesheet | { status: string; payment_status?: string },
  userRole?: UserRole | null
): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  // Cannot edit if already paid
  const paymentStatus = 'payment_status' in timesheet ? timesheet.payment_status : undefined;
  if (paymentStatus === 'paid') return false;

  // Admin can edit if not paid
  if (role === 'admin') return true;

  // Partner cannot edit approved or rejected timesheets
  if (role === 'partner') {
    return timesheet.status !== 'approved' && timesheet.status !== 'rejected';
  }

  return false;
};

/**
 * Check if user can manage projects
 * Both Admin and Partner have full project management permissions
 */
export const canPartnerManageProject = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  return role === 'admin' || role === 'partner';
};

/**
 * Check if user can approve/reject timesheets
 * Only Admins can approve/reject timesheets
 */
export const canApproveTimesheet = (userRole?: UserRole | null): boolean => {
  const role = userRole || authManager.getUserRole();

  return role === 'admin';
};

/**
 * Check if user can delete timesheets
 * Rules:
 * - Admin can always delete
 * - Partner can delete if not approved or rejected
 */
export const canDeleteTimesheet = (
  timesheet: Timesheet | { status: string },
  userRole?: UserRole | null
): boolean => {
  const role = userRole || authManager.getUserRole();

  if (!role) return false;

  // Admin has full access
  if (role === 'admin') return true;

  // Partner cannot delete approved or rejected timesheets
  if (role === 'partner') {
    return timesheet.status !== 'approved' && timesheet.status !== 'rejected';
  }

  return false;
};
