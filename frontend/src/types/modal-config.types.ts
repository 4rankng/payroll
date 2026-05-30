import type { ComponentType } from 'react';

/**
 * Modal Configuration Types for Auto-Registration System
 * Defines the structure for modal metadata and permissions
 */

export type ModalAction = 
  | 'read' 
  | 'create' 
  | 'update' 
  | 'delete' 
  | 'import' 
  | 'export'
  | 'approve'
  | 'manage';

export type ModalSubject =
  | 'User'
  | 'Project'
  | 'Employee'
  | 'Timesheet'
  | 'PayRate'
  | 'PayrollData'
  | 'Report'
  | 'Settings'
  | 'LedgerEntry'
  | 'Transaction'
  | 'Notification'
  | 'all';

export type UserRole = 'admin' | 'partner' | 'adv_partner';

/**
 * Deeplink configuration for modal
 */
export interface ModalDeeplink {
  /** Whether deeplinks are enabled for this modal */
  enabled: boolean;
  /** Expected URL parameters for this modal */
  params?: string[];
  /** Example deeplink URL for documentation */
  example?: string;
  /** Custom validation for deeplink parameters */
  validateParams?: (params: Record<string, unknown>) => boolean;
}

/**
 * Permission configuration for modal
 */
export interface ModalPermissions {
  /** CASL action required */
  action: ModalAction;
  /** CASL subject required */
  subject: ModalSubject;
  /** Required user roles */
  roles: UserRole[];
  /** Additional permission checks */
  additionalChecks?: (params?: Record<string, unknown>) => boolean;
}

/**
 * Complete modal configuration
 */
export interface ModalConfig {
  /** Unique modal identifier (used in URLs) */
  id: string;
  /** Human-readable name */
  name?: string;
  /** Modal description for documentation */
  description?: string;
  /** Permission requirements */
  permissions: ModalPermissions;
  /** Deeplink configuration */
  deeplink: ModalDeeplink;
  /** Category for grouping (optional) */
  category?: string;
  /** Whether to require authentication */
  requiresAuth?: boolean;
  /** Whether to encrypt modal data in URL */
  encryptData?: boolean;
}

/**
 * Modal module structure expected from glob imports
 */
export interface ModalModule {
  /** The modal component */
  default: ComponentType;
  /** Modal configuration */
  modalConfig: ModalConfig;
}

/**
 * Modal registry entry combining config and lazy loader
 */
export interface ModalRegistryEntry extends ModalConfig {
  /** Lazy loader function for the component */
  loader: () => Promise<ModalModule>;
  /** File path for debugging */
  filePath?: string;
}

/**
 * Complete modal registry
 */
export type ModalRegistry = Record<string, ModalRegistryEntry>;

/**
 * Type helper for modal parameters
 */
export type ModalParams = Record<string, string | number | boolean | undefined>;

/**
 * Modal ID type (will be generated from registry)
 */
export type ModalId = string;