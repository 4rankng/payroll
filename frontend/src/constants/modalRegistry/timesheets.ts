import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * Timesheet Domain Modal Registry
 * Contains all timesheet-related modals with schemas and metadata
 */

export const TIMESHEET_MODAL_IDS = {
  TIMESHEET_DETAILS: 'timesheet_details',
  TIMESHEET_ENTRY: 'timesheet_entry',
} as const;

export type TimesheetModalId = typeof TIMESHEET_MODAL_IDS[keyof typeof TIMESHEET_MODAL_IDS];

// Zod schemas for timesheet modals
export const timesheetModalSchemas = {
  [TIMESHEET_MODAL_IDS.TIMESHEET_DETAILS]: z.object({
    id: z.string().min(1, 'Timesheet ID is required'),
    tab: z.enum(['details', 'entries', 'history']).optional().default('details'),
  }),
  [TIMESHEET_MODAL_IDS.TIMESHEET_ENTRY]: z.object({
    projectId: z.string().optional(),
    employeeId: z.string().optional(),
    date: z.string().optional(), // ISO date string
    entryId: z.string().optional(), // For editing existing entry
  }),
} as const;

// Modal metadata with permissions
interface TimesheetModalMetadata {
  id: TimesheetModalId;
  name: string;
  description: string;
  category: 'timesheet';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof timesheetModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'Timesheet';
  };
}

export const timesheetModals: Record<TimesheetModalId, TimesheetModalMetadata> = {
  [TIMESHEET_MODAL_IDS.TIMESHEET_DETAILS]: {
    id: TIMESHEET_MODAL_IDS.TIMESHEET_DETAILS,
    name: 'Chi tiết bảng chấm công',
    description: 'Xem chi tiết bảng chấm công',
    category: 'timesheet',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'TimesheetDetailsModal',
    hasParams: true,
    schema: TIMESHEET_MODAL_IDS.TIMESHEET_DETAILS,
    permissions: {
      action: 'read',
      subject: 'Timesheet',
    },
  },
  [TIMESHEET_MODAL_IDS.TIMESHEET_ENTRY]: {
    id: TIMESHEET_MODAL_IDS.TIMESHEET_ENTRY,
    name: 'Nhập bảng chấm công',
    description: 'Tạo hoặc chỉnh sửa bảng chấm công',
    category: 'timesheet',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'TimesheetEntryModal',
    hasParams: true,
    schema: TIMESHEET_MODAL_IDS.TIMESHEET_ENTRY,
    permissions: {
      action: 'create',
      subject: 'Timesheet',
    },
  },
} as const;