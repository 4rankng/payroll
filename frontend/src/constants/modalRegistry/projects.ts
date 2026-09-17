import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * Project Domain Modal Registry
 * Contains all project-related modals with schemas and metadata
 */

export const PROJECT_MODAL_IDS = {
  ADD_PROJECT: 'add_project',
  PROJECT_DETAILS: 'project_details',
  PROJECT_CREATE: 'project_create',
  PROJECT_EDIT: 'project_edit',
  PROJECT_ASSIGNMENT: 'project_assignment',
  ADD_EMPLOYEE_TO_PROJECT: 'add_employee_to_project',
  REMOVE_EMPLOYEE: 'remove_employee',
} as const;

export type ProjectModalId = typeof PROJECT_MODAL_IDS[keyof typeof PROJECT_MODAL_IDS];

// Zod schemas for project modals
export const projectModalSchemas = {
  [PROJECT_MODAL_IDS.ADD_PROJECT]: z.object({}),
  [PROJECT_MODAL_IDS.PROJECT_DETAILS]: z.object({
    id: z.string().min(1, 'Project ID is required'),
    tab: z.enum(['details', 'employees', 'timesheet', 'payrates', 'settings']).optional().default('details'),
  }),
  [PROJECT_MODAL_IDS.PROJECT_CREATE]: z.object({}),
  [PROJECT_MODAL_IDS.PROJECT_EDIT]: z.object({
    id: z.string().min(1, 'Project ID is required'),
  }),
  [PROJECT_MODAL_IDS.PROJECT_ASSIGNMENT]: z.object({
    projectId: z.string().min(1, 'Project ID is required'),
    employeeId: z.string().optional(),
  }),
  [PROJECT_MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT]: z.object({
    projectId: z.string().min(1, 'Project ID is required'),
  }),
  [PROJECT_MODAL_IDS.REMOVE_EMPLOYEE]: z.object({
    projectId: z.string().min(1, 'Project ID is required'),
    employeeId: z.string().min(1, 'Employee ID is required'),
  }),
} as const;

// Modal metadata with permissions
interface ProjectModalMetadata {
  id: ProjectModalId;
  name: string;
  description: string;
  category: 'project';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof projectModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'Project';
  };
}

export const projectModals: Record<ProjectModalId, ProjectModalMetadata> = {
  [PROJECT_MODAL_IDS.ADD_PROJECT]: {
    id: PROJECT_MODAL_IDS.ADD_PROJECT,
    name: 'Thêm dự án',
    description: 'Thêm dự án mới',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'AddProjectSheet',
    hasParams: false,
    permissions: {
      action: 'create',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.PROJECT_DETAILS]: {
    id: PROJECT_MODAL_IDS.PROJECT_DETAILS,
    name: 'Chi tiết dự án',
    description: 'Xem thông tin chi tiết dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ProjectDetailsSheet',
    hasParams: true,
    schema: PROJECT_MODAL_IDS.PROJECT_DETAILS,
    permissions: {
      action: 'read',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.PROJECT_CREATE]: {
    id: PROJECT_MODAL_IDS.PROJECT_CREATE,
    name: 'Tạo dự án',
    description: 'Tạo dự án mới',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ProjectCreateModal',
    hasParams: false,
    permissions: {
      action: 'create',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.PROJECT_EDIT]: {
    id: PROJECT_MODAL_IDS.PROJECT_EDIT,
    name: 'Chỉnh sửa dự án',
    description: 'Chỉnh sửa thông tin dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ProjectEditSheet',
    hasParams: true,
    schema: PROJECT_MODAL_IDS.PROJECT_EDIT,
    permissions: {
      action: 'update',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.PROJECT_ASSIGNMENT]: {
    id: PROJECT_MODAL_IDS.PROJECT_ASSIGNMENT,
    name: 'Phân công dự án',
    description: 'Phân công nhân viên vào dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ProjectAssignmentSheet',
    hasParams: true,
    schema: PROJECT_MODAL_IDS.PROJECT_ASSIGNMENT,
    permissions: {
      action: 'update',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT]: {
    id: PROJECT_MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT,
    name: 'Thêm nhân viên vào dự án',
    description: 'Thêm nhân viên vào dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'AddEmployeeToProjectSheetContainer',
    hasParams: true,
    schema: PROJECT_MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT,
    permissions: {
      action: 'update',
      subject: 'Project',
    },
  },
  [PROJECT_MODAL_IDS.REMOVE_EMPLOYEE]: {
    id: PROJECT_MODAL_IDS.REMOVE_EMPLOYEE,
    name: 'Xóa nhân viên khỏi dự án',
    description: 'Xóa nhân viên khỏi dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'RemoveEmployeeModal',
    hasParams: true,
    schema: PROJECT_MODAL_IDS.REMOVE_EMPLOYEE,
    permissions: {
      action: 'delete',
      subject: 'Project',
    },
  },
} as const;
