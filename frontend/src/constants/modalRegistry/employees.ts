import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * Employee Domain Modal Registry
 * Contains all employee-related modals with schemas and metadata
 */

export const EMPLOYEE_MODAL_IDS = {
  ADD_EMPLOYEE: 'add_employee',
  EMPLOYEE_DETAILS: 'employee_details',
} as const;

export type EmployeeModalId = typeof EMPLOYEE_MODAL_IDS[keyof typeof EMPLOYEE_MODAL_IDS];

// Zod schemas for employee modals
export const employeeModalSchemas = {
  [EMPLOYEE_MODAL_IDS.ADD_EMPLOYEE]: z.object({
    projectId: z.string().optional(),
  }),
  [EMPLOYEE_MODAL_IDS.EMPLOYEE_DETAILS]: z.object({
    id: z.string().min(1, 'Employee ID is required'),
    tab: z.enum(['details', 'projects', 'timesheet', 'payroll']).optional().default('details'),
  }),
} as const;

// Modal metadata with permissions
interface EmployeeModalMetadata {
  id: EmployeeModalId;
  name: string;
  description: string;
  category: 'employee';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof employeeModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'Employee';
  };
}

export const employeeModals: Record<EmployeeModalId, EmployeeModalMetadata> = {
  [EMPLOYEE_MODAL_IDS.ADD_EMPLOYEE]: {
    id: EMPLOYEE_MODAL_IDS.ADD_EMPLOYEE,
    name: 'Thêm nhân viên',
    description: 'Thêm nhân viên mới vào hệ thống',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'AddEmployeeSheet',
    hasParams: true,
    schema: EMPLOYEE_MODAL_IDS.ADD_EMPLOYEE,
    permissions: {
      action: 'create',
      subject: 'Employee',
    },
  },
  [EMPLOYEE_MODAL_IDS.EMPLOYEE_DETAILS]: {
    id: EMPLOYEE_MODAL_IDS.EMPLOYEE_DETAILS,
    name: 'Chi tiết nhân viên',
    description: 'Xem thông tin chi tiết nhân viên',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'EmployeeDetailsSheet',
    hasParams: true,
    schema: EMPLOYEE_MODAL_IDS.EMPLOYEE_DETAILS,
    permissions: {
      action: 'read',
      subject: 'Employee',
    },
  },
} as const;