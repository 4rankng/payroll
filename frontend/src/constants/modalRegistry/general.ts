import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * General Domain Modal Registry
 * Contains all general-purpose modals with schemas and metadata
 */

export const GENERAL_MODAL_IDS = {
  FILE_UPLOAD: 'file_upload',
  NOTIFICATION_SHEET: 'notification_sheet',
  LEDGER_ENTRY_DETAILS: 'ledger_entry_details',
  APPROVAL_DETAILS: 'approval_details',
} as const;

export type GeneralModalId = typeof GENERAL_MODAL_IDS[keyof typeof GENERAL_MODAL_IDS];

// Zod schemas for general modals
export const generalModalSchemas = {
  [GENERAL_MODAL_IDS.FILE_UPLOAD]: z.object({}),
  [GENERAL_MODAL_IDS.NOTIFICATION_SHEET]: z.object({}),
  [GENERAL_MODAL_IDS.LEDGER_ENTRY_DETAILS]: z.object({
    id: z.string().min(1, 'Entry ID is required'),
    tab: z.enum(['details', 'entries', 'adjustments']).optional().default('details'),
  }),
  [GENERAL_MODAL_IDS.APPROVAL_DETAILS]: z.object({
    id: z.string().min(1, 'Approval ID is required'),
    type: z.enum(['timesheet', 'project', 'employee']).optional(),
  }),
} as const;

// Modal metadata with permissions
interface GeneralModalMetadata {
  id: GeneralModalId;
  name: string;
  description: string;
  category: 'general' | 'report';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof generalModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'all' | 'Report' | 'PayrollData';
  };
}

export const generalModals: Record<GeneralModalId, GeneralModalMetadata> = {
  [GENERAL_MODAL_IDS.FILE_UPLOAD]: {
    id: GENERAL_MODAL_IDS.FILE_UPLOAD,
    name: 'Tải lên tập tin',
    description: 'Tải lên tập tin',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'FileUploadModal',
    hasParams: false,
    permissions: {
      action: 'create',
      subject: 'all',
    },
  },
  [GENERAL_MODAL_IDS.NOTIFICATION_SHEET]: {
    id: GENERAL_MODAL_IDS.NOTIFICATION_SHEET,
    name: 'Thông báo',
    description: 'Xem thông báo',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'NotificationSheet',
    hasParams: false,
    permissions: {
      action: 'read',
      subject: 'all',
    },
  },
  [GENERAL_MODAL_IDS.LEDGER_ENTRY_DETAILS]: {
    id: GENERAL_MODAL_IDS.LEDGER_ENTRY_DETAILS,
    name: 'Chi tiết bút toán',
    description: 'Xem chi tiết bút toán',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'LedgerEntryDetailsSheet',
    hasParams: true,
    schema: GENERAL_MODAL_IDS.LEDGER_ENTRY_DETAILS,
    permissions: {
      action: 'read',
      subject: 'PayrollData',
    },
  },
  [GENERAL_MODAL_IDS.APPROVAL_DETAILS]: {
    id: GENERAL_MODAL_IDS.APPROVAL_DETAILS,
    name: 'Chi tiết phê duyệt',
    description: 'Xem chi tiết phê duyệt',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ApprovalDetailsModal',
    hasParams: true,
    schema: GENERAL_MODAL_IDS.APPROVAL_DETAILS,
    permissions: {
      action: 'read',
      subject: 'Report',
    },
  },
} as const;
