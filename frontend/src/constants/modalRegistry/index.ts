import type { UserRole } from '@/types/modal-config.types';
import { userModals, USER_MODAL_IDS, userModalSchemas } from './users';
import { projectModals, PROJECT_MODAL_IDS, projectModalSchemas } from './projects';
import { employeeModals, EMPLOYEE_MODAL_IDS, employeeModalSchemas } from './employees';
import { timesheetModals, TIMESHEET_MODAL_IDS, timesheetModalSchemas } from './timesheets';
import { generalModals, GENERAL_MODAL_IDS, generalModalSchemas } from './general';

/**
 * Unified Modal Registry
 * Merges all domain-specific registries into a single source of truth
 */

// Unified Modal IDs - strongly typed
export const MODAL_IDS = {
  ...USER_MODAL_IDS,
  ...PROJECT_MODAL_IDS,
  ...EMPLOYEE_MODAL_IDS,
  ...TIMESHEET_MODAL_IDS,
  ...GENERAL_MODAL_IDS,
} as const;

// Extract the union type of all modal IDs
export type ModalId = typeof MODAL_IDS[keyof typeof MODAL_IDS];

// Unified Modal Registry - complete metadata for all modals
export const MODAL_REGISTRY = {
  ...userModals,
  ...projectModals,
  ...employeeModals,
  ...timesheetModals,
  ...generalModals,
} as const;

// Unified Modal Schemas - all Zod schemas in one place
export const MODAL_SCHEMAS = {
  ...userModalSchemas,
  ...projectModalSchemas,
  ...employeeModalSchemas,
  ...timesheetModalSchemas,
  ...generalModalSchemas,
} as const;

// Export schema keys type
export type ModalSchemaKey = keyof typeof MODAL_SCHEMAS;

// Utility type to get parameters for a specific modal
export type ModalParams<T extends ModalSchemaKey> = T extends keyof typeof MODAL_SCHEMAS 
  ? typeof MODAL_SCHEMAS[T] extends { parse: (input: unknown) => infer P }
    ? P
    : never
  : never;

// Utility functions
export const getModalMetadata = (modalId: ModalId) => {
  return MODAL_REGISTRY[modalId] || null;
};

export const getModalSchema = (schemaKey: ModalSchemaKey) => {
  return MODAL_SCHEMAS[schemaKey] || null;
};


export const isValidModalId = (id: string): id is ModalId => {
  return Object.values(MODAL_IDS).includes(id as ModalId);
};

export const getModalsByCategory = (category: string) => {
  return Object.values(MODAL_REGISTRY).filter(modal => modal.category === category);
};

export const getModalsByRole = (role: UserRole) => {
  return Object.values(MODAL_REGISTRY).filter(modal => modal.roles.includes(role));
};

// Development helpers
export const getAllModalIds = (): ModalId[] => {
  return Object.values(MODAL_IDS);
};

export const getRegistryStats = () => {
  const modals = Object.values(MODAL_REGISTRY);
  return {
    total: modals.length,
    byCategory: modals.reduce((acc, modal) => {
      acc[modal.category] = (acc[modal.category] || 0) + 1;
      return acc;
    }, {} as Record<string, number>),
    byRole: modals.reduce((acc, modal) => {
      modal.roles.forEach(role => {
        acc[role] = (acc[role] || 0) + 1;
      });
      return acc;
    }, {} as Record<string, number>),
    withParams: modals.filter(m => m.hasParams).length,
  };
};

// Re-export domain-specific IDs for convenience
export {
  USER_MODAL_IDS,
  PROJECT_MODAL_IDS,
  EMPLOYEE_MODAL_IDS,
  TIMESHEET_MODAL_IDS,
  GENERAL_MODAL_IDS,
};

// Re-export types for external use
export type { UserModalId } from './users';
export type { ProjectModalId } from './projects';
export type { EmployeeModalId } from './employees';
export type { TimesheetModalId } from './timesheets';
export type { GeneralModalId } from './general';