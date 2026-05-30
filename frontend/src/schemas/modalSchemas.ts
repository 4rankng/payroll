import { z } from 'zod';

/**
 * Modal Parameter Schemas using Zod
 * Type-safe validation for modal URL parameters
 */

// Base common schemas
const idSchema = z.string().min(1, 'ID is required');
const optionalIdSchema = z.string().optional();
const tabSchema = z.string().optional();
const statusSchema = z.enum(['active', 'inactive', 'pending']).optional();

// User-related schemas
export const userDetailsSchema = z.object({
  id: idSchema,
  tab: z.enum(['details', 'activity', 'permissions']).optional().default('details')
});

export const resetPasswordSchema = z.object({
  userId: idSchema
});

// Employee-related schemas  
export const employeeDetailsSchema = z.object({
  id: idSchema,
  tab: z.enum(['details', 'projects', 'timesheet', 'payroll']).optional().default('details')
});

export const addEmployeeSchema = z.object({
  projectId: optionalIdSchema
});

// Project-related schemas
export const projectDetailsSchema = z.object({
  id: idSchema,
  tab: z.enum(['details', 'employees', 'timesheet', 'settings']).optional().default('details')
});

export const projectEditSchema = z.object({
  id: idSchema
});

export const projectAssignmentSchema = z.object({
  projectId: idSchema,
  employeeId: optionalIdSchema
});

export const removeEmployeeSchema = z.object({
  projectId: idSchema,
  employeeId: idSchema
});

// Timesheet-related schemas
export const timesheetDetailsSchema = z.object({
  id: idSchema,
  tab: z.enum(['details', 'entries', 'history']).optional().default('details')
});

export const timesheetEntrySchema = z.object({
  projectId: optionalIdSchema,
  employeeId: optionalIdSchema,
  date: z.string().optional(), // ISO date string
  entryId: optionalIdSchema // For editing existing entry
});

export const saturdayDayTypeSchema = z.object({
  date: z.string().min(1, 'Date is required'), // ISO date string
  projectId: optionalIdSchema
});

// Report & Ledger schemas
export const ledgerDetailsSchema = z.object({
  id: idSchema,
  tab: z.enum(['details', 'entries', 'adjustments']).optional().default('details')
});

export const approvalDetailsSchema = z.object({
  id: idSchema,
  type: z.enum(['timesheet', 'project', 'employee']).optional()
});

// Schema registry mapping
export const MODAL_SCHEMAS = {
  userDetails: userDetailsSchema,
  resetPassword: resetPasswordSchema,
  employeeDetails: employeeDetailsSchema,
  addEmployee: addEmployeeSchema,
  projectDetails: projectDetailsSchema,
  projectEdit: projectEditSchema,
  projectAssignment: projectAssignmentSchema,
  removeEmployee: removeEmployeeSchema,
  timesheetDetails: timesheetDetailsSchema,
  timesheetEntry: timesheetEntrySchema,
  saturdayDayType: saturdayDayTypeSchema,
  ledgerDetails: ledgerDetailsSchema,
  approvalDetails: approvalDetailsSchema
} as const;

// Extract schema keys for type safety
export type ModalSchemaKey = keyof typeof MODAL_SCHEMAS;

// Utility type to get parameters for a specific modal schema
export type ModalParams<T extends ModalSchemaKey> = z.infer<typeof MODAL_SCHEMAS[T]>;

// Generic modal params type (for modals without specific schemas)
export type GenericModalParams = Record<string, string | undefined>;

// Validation function
export function validateModalParams<T extends ModalSchemaKey>(
  schemaKey: T,
  params: unknown
): { success: true; data: ModalParams<T> } | { success: false; error: z.ZodError } {
  const schema = MODAL_SCHEMAS[schemaKey];
  
  if (!schema) {
    return {
      success: false,
      error: new z.ZodError([{
        code: 'custom',
        message: `Unknown schema key: ${schemaKey}`,
        path: []
      }])
    };
  }

  const result = schema.safeParse(params);
  
  if (result.success) {
    return { success: true, data: result.data as ModalParams<T> };
  } else {
    return { success: false, error: result.error };
  }
}

// Utility to convert URL search params to object
export function parseSearchParamsToObject(searchParams: URLSearchParams): Record<string, string> {
  const params: Record<string, string> = {};
  
  searchParams.forEach((value, key) => {
    // Skip the 'modal' param as it's handled separately
    if (key !== 'modal') {
      params[key] = value;
    }
  });
  
  return params;
}

// Utility to build search params from validated object
export function buildSearchParams<T extends ModalSchemaKey>(
  modalId: string,
  schemaKey: T,
  params: ModalParams<T>
): URLSearchParams {
  const searchParams = new URLSearchParams();
  
  // Always set the modal ID
  searchParams.set('modal', modalId);
  
  // Add validated parameters
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.set(key, String(value));
    }
  });
  
  return searchParams;
}

// Type guard to check if a modal has parameters
export function modalHasSchema<T extends ModalSchemaKey>(
  schemaKey: string
): schemaKey is T {
  return schemaKey in MODAL_SCHEMAS;
}

// Development helpers
export const getSchemaForModal = (schemaKey: string) => {
  return modalHasSchema(schemaKey) ? MODAL_SCHEMAS[schemaKey] : null;
};

export const getAllSchemaKeys = (): ModalSchemaKey[] => {
  return Object.keys(MODAL_SCHEMAS) as ModalSchemaKey[];
};

// Example usage and validation helpers
export const validateUserDetailsParams = (params: unknown) =>
  validateModalParams('userDetails', params);

export const validateProjectDetailsParams = (params: unknown) =>
  validateModalParams('projectDetails', params);

export const validateTimesheetDetailsParams = (params: unknown) =>
  validateModalParams('timesheetDetails', params);