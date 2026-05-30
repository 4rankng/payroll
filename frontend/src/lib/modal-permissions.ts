import { AbilityBuilder, createMongoAbility } from '@casl/ability';
import { authManager } from '@/lib/auth';
import type { ModalAction, ModalSubject } from '@/types/modal-config.types';

/**
 * Modal Permission System using CASL
 * Defines fine-grained permissions for modal access and operations
 */

// Note: Types moved to /types/modal-config.types.ts for consistency

export type AppAbility = ReturnType<typeof createMongoAbility>;

/**
 * Define abilities based on user role
 */
const defineAbilityFor = (role: string): AppAbility => {
  const { can, cannot, build } = new AbilityBuilder(createMongoAbility);

  // Admin permissions - full access
  if (role === 'admin') {
    can('manage', 'all');
  }

  // Partner permissions - limited access
  else if (role === 'partner') {
    // Read access to most resources
    can('read', ['Project', 'Employee', 'Timesheet', 'Report', 'PayRate', 'Transaction', 'Notification']);

    // Full project management permissions for Partners
    can(['create', 'update', 'delete'], 'Project');

    // Create/update access for specific resources
    can(['create', 'update'], 'Employee');
    can(['create', 'update'], 'Timesheet');
    can(['create', 'update'], 'PayRate');
    can(['create', 'update'], 'Transaction');

    // Import/export permissions
    can(['import', 'export'], ['Employee', 'Timesheet']);

    // Allow password updates for own account
    can('update', 'User'); // Allow partners to update user data (for change password)

    // Cannot manage users or settings (except password updates)
    cannot(['create', 'delete'], 'User'); // Partners can't create/delete users
    cannot('manage', 'Settings');
    cannot('manage', 'PayrollData');

    // Cannot approve timesheets (Admin-only function)
    cannot('approve', 'Timesheet');
  }

  // Default - no permissions for unknown roles
  else {
    cannot('manage', 'all');
  }

  return build();
};

/**
 * Get current user's ability
 */
export const getCurrentAbility = (): AppAbility => {
  const role = authManager.getUserRole();

  if (!role) {
    throw new Error('No user role found');
  }

  return defineAbilityFor(role);
};

/**
 * Check if user can perform action on subject
 */
export const canAccess = (action: ModalAction, subject: ModalSubject): boolean => {
  try {
    const ability = getCurrentAbility();
    return ability.can(action, subject);
  } catch {
    return false;
  }
};

/**
 * Modal permission registry
 * Manual registration for better reliability
 */
export interface ModalPermission {
  modalId: string;
  action: ModalAction;
  subject: ModalSubject;
  roles: string[];
  additionalChecks?: (params?: Record<string, unknown>) => boolean;
}

/**
 * Manual modal permissions registry
 * Simple and reliable approach
 */
export const MODAL_PERMISSIONS: Record<string, ModalPermission> = {
  // User Management
  user_details: {
    modalId: 'user_details',
    action: 'read',
    subject: 'User',
    roles: ['admin', 'partner']
  },
  add_user: {
    modalId: 'add_user',
    action: 'create',
    subject: 'User',
    roles: ['admin', 'partner']
  },
  edit_user: {
    modalId: 'edit_user',
    action: 'update',
    subject: 'User',
    roles: ['admin', 'partner']
  },
  reset_password: {
    modalId: 'reset_password',
    action: 'update',
    subject: 'User',
    roles: ['admin', 'partner']
  },

  // Employee Management
  employee_details: {
    modalId: 'employee_details',
    action: 'read',
    subject: 'Employee',
    roles: ['admin', 'partner'],
    additionalChecks: (params) => {
      // Allow modal to open even without params (for initial state)
      if (!params || Object.keys(params).length === 0) return true;
      // If params exist, validate them
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  add_employee: {
    modalId: 'add_employee',
    action: 'create',
    subject: 'Employee',
    roles: ['admin', 'partner']
  },
  employee_import: {
    modalId: 'employee_import',
    action: 'import',
    subject: 'Employee',
    roles: ['admin', 'partner']
  },
  employee_export: {
    modalId: 'employee_export',
    action: 'export',
    subject: 'Employee',
    roles: ['admin', 'partner']
  },

  // Project Management
  project_details: {
    modalId: 'project_details',
    action: 'read',
    subject: 'Project',
    roles: ['admin', 'partner'],
    additionalChecks: (params) => {
      // Allow modal to open even without params (for initial state)
      if (!params || Object.keys(params).length === 0) return true;
      // If params exist, validate them
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  project_create: {
    modalId: 'project_create',
    action: 'create',
    subject: 'Project',
    roles: ['admin', 'partner']
  },
  add_project: {
    modalId: 'add_project',
    action: 'create',
    subject: 'Project',
    roles: ['admin', 'partner']
  },
  project_edit: {
    modalId: 'project_edit',
    action: 'update',
    subject: 'Project',
    roles: ['admin', 'partner']
  },
  project_assignment: {
    modalId: 'project_assignment',
    action: 'update',
    subject: 'Project',
    roles: ['admin', 'partner']
  },
  remove_employee: {
    modalId: 'remove_employee',
    action: 'update',
    subject: 'Project',
    roles: ['admin', 'partner']
  },

  // PayRate Management
  payrate_create: {
    modalId: 'payrate_create',
    action: 'create',
    subject: 'PayRate',
    roles: ['admin', 'partner']
  },
  payrate_edit: {
    modalId: 'payrate_edit',
    action: 'update',
    subject: 'PayRate',
    roles: ['admin', 'partner']
  },
  payrate_details: {
    modalId: 'payrate_details',
    action: 'read',
    subject: 'PayRate',
    roles: ['admin', 'partner']
  },

  // Timesheet Management
  timesheet_details: {
    modalId: 'timesheet_details',
    action: 'read',
    subject: 'Timesheet',
    roles: ['admin', 'partner']
  },
  timesheet_entry: {
    modalId: 'timesheet_entry',
    action: 'create',
    subject: 'Timesheet',
    roles: ['admin', 'partner']
  },
  timesheet_bulk_transfer: {
    modalId: 'timesheet_bulk_transfer',
    action: 'update',
    subject: 'Timesheet',
    roles: ['admin', 'partner']
  },
  saturday_day_type: {
    modalId: 'saturday_day_type',
    action: 'update',
    subject: 'Timesheet',
    roles: ['admin', 'partner']
  },

  // Reports & Ledger
  ledger_entry_details: {
    modalId: 'ledger_entry_details',
    action: 'read',
    subject: 'Report',
    roles: ['admin', 'partner']
  },
  add_ledger_entry: {
    modalId: 'add_ledger_entry',
    action: 'create',
    subject: 'Report',
    roles: ['admin', 'partner']
  },
  approval_details: {
    modalId: 'approval_details',
    action: 'read',
    subject: 'Report',
    roles: ['admin', 'partner']
  },

  // Transactions
  transaction_details: {
    modalId: 'transaction_details',
    action: 'read',
    subject: 'Transaction',
    roles: ['admin', 'partner']
  },
  settle_transaction: {
    modalId: 'settle_transaction',
    action: 'update',
    subject: 'Transaction',
    roles: ['admin', 'partner']
  },
  reverse_transaction: {
    modalId: 'reverse_transaction',
    action: 'update',
    subject: 'Transaction',
    roles: ['admin', 'partner']
  },

  // General Purpose
  file_upload: {
    modalId: 'file_upload',
    action: 'create',
    subject: 'all',
    roles: ['admin', 'partner']
  },
  notification_sheet: {
    modalId: 'notification_sheet',
    action: 'read',
    subject: 'Notification',
    roles: ['admin', 'partner']
  },
  user_profile: {
    modalId: 'user_profile',
    action: 'read',
    subject: 'User',
    roles: ['admin', 'partner']
  },
  change_password: {
    modalId: 'change_password',
    action: 'update',
    subject: 'User',
    roles: ['admin', 'partner']
  }
};

/**
 * Get modal permissions (now synchronous)
 */
export function getModalPermissions(): Record<string, ModalPermission> {
  return MODAL_PERMISSIONS;
}

/**
 * Synchronous permissions getter (now the main implementation)
 */
export function getModalPermissionsSync(): Record<string, ModalPermission> {
  return MODAL_PERMISSIONS;
}

/**
 * Validate modal access permission
 */
export const validateModalPermission = (modalId: string, params?: Record<string, unknown>): boolean => {
  try {
    const permissions = getModalPermissions();
    const permission = permissions[modalId];

    if (!permission) {
      console.warn(`No permission definition found for modal: ${modalId}`);
      return false;
    }

    // Check role-based access
    const userRole = authManager.getUserRole();
    if (!userRole || !permission.roles.includes(userRole)) {
      return false;
    }

    // Check CASL ability
    if (!canAccess(permission.action, permission.subject)) {
      return false;
    }

    // Run additional checks if defined
    if (permission.additionalChecks) {
      return permission.additionalChecks(params);
    }

    return true;
  } catch (error) {
    console.error('Permission validation failed:', error);
    return false;
  }
};

/**
 * Synchronous permission validation (same as main implementation now)
 */
export const validateModalPermissionSync = (modalId: string, params?: Record<string, unknown>): boolean => {
  return validateModalPermission(modalId, params);
};

/**
 * Get all accessible modal IDs for current user
 */
export const getAccessibleModals = (): string[] => {
  const userRole = authManager.getUserRole();
  if (!userRole) return [];

  return Object.values(getModalPermissions())
    .filter(permission =>
      permission.roles.includes(userRole) &&
      canAccess(permission.action, permission.subject)
    )
    .map(permission => permission.modalId);
};

/**
 * Permission error types
 */
export class PermissionError extends Error {
  constructor(
    message: string,
    public modalId: string,
    public requiredAction: ModalAction,
    public requiredSubject: ModalSubject
  ) {
    super(message);
    this.name = 'PermissionError';
  }
}

/**
 * Throw permission error with context
 */
export const throwPermissionError = (modalId: string): never => {
  const permissions = getModalPermissions();
  const permission = permissions[modalId];

  throw new PermissionError(
    `Access denied to modal: ${modalId}`,
    modalId,
    permission?.action || 'read',
    permission?.subject || 'all'
  );
};
