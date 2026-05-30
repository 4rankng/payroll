import type { UserRole } from '@/types/modal-config.types';

/**
 * Centralized Modal Registry with Type-Safe IDs
 * Single source of truth for all modals in the application
 */

// Modal ID constants - strongly typed
export const MODAL_IDS = {
  // User Management
  USER_DETAILS: 'user_details',
  ADD_USER: 'add_user',
  EDIT_USER: 'edit_user',
  RESET_PASSWORD: 'reset_password',

  // Employee Management
  EMPLOYEE_DETAILS: 'employee_details',
  ADD_EMPLOYEE: 'add_employee',
  EMPLOYEE_IMPORT: 'employee_import',
  EMPLOYEE_EXPORT: 'employee_export',

  // Project Management
  PROJECT_DETAILS: 'project_details',
  PROJECT_CREATE: 'project_create',
  PROJECT_EDIT: 'project_edit',
  ADD_PROJECT: 'add_project',
  PROJECT_ASSIGNMENT: 'project_assignment',
  ADD_EMPLOYEE_TO_PROJECT: 'add_employee_to_project',
  REMOVE_EMPLOYEE: 'remove_employee',

  // Timesheet Management
  TIMESHEET_DETAILS: 'timesheet_details',
  TIMESHEET_ENTRY: 'timesheet_entry',
  TIMESHEET_BULK_TRANSFER: 'timesheet_bulk_transfer',

  // Ledger & Reports
  LEDGER_ENTRY_DETAILS: 'ledger_entry_details',
  ADD_LEDGER_ENTRY: 'add_ledger_entry',
  APPROVAL_DETAILS: 'approval_details',

  // Transactions
  TRANSACTION_DETAILS: 'transaction_details',
  SETTLE_TRANSACTION: 'settle_transaction',
  REVERSE_TRANSACTION: 'reverse_transaction',

  // General Purpose
  FILE_UPLOAD: 'file_upload',
  NOTIFICATION_SHEET: 'notification_sheet',
  USER_PROFILE: 'user_profile',
  CHANGE_PASSWORD: 'change_password'
} as const;

// Extract modal ID type from constants
export type ModalId = typeof MODAL_IDS[keyof typeof MODAL_IDS];

// Modal metadata interface
export interface ModalMetadata {
  id: ModalId;
  name: string;
  description: string;
  category: 'user' | 'employee' | 'project' | 'timesheet' | 'report' | 'general';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  paramSchema?: string; // Reference to schema key
}

// Complete modal registry configuration
export const MODAL_REGISTRY: Record<ModalId, ModalMetadata> = {
  // User Management Modals
  [MODAL_IDS.USER_DETAILS]: {
    id: MODAL_IDS.USER_DETAILS,
    name: 'Chi tiết người dùng',
    description: 'Xem và chỉnh sửa thông tin người dùng',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner', 'adv_partner'],
    componentPath: '/src/components/sheets/UserDetailsSheet',
    hasParams: true,
    paramSchema: 'userDetails'
  },

  [MODAL_IDS.ADD_USER]: {
    id: MODAL_IDS.ADD_USER,
    name: 'Thêm người dùng',
    description: 'Tạo tài khoản người dùng mới',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/AddUserSheet',
    hasParams: false
  },

  [MODAL_IDS.EDIT_USER]: {
    id: MODAL_IDS.EDIT_USER,
    name: 'Chỉnh sửa người dùng',
    description: 'Chỉnh sửa thông tin người dùng',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/EditUserSheet',
    hasParams: true,
    paramSchema: 'userEdit'
  },

  [MODAL_IDS.RESET_PASSWORD]: {
    id: MODAL_IDS.RESET_PASSWORD,
    name: 'Đặt lại mật khẩu',
    description: 'Đặt lại mật khẩu cho người dùng',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/ResetPasswordModal',
    hasParams: true,
    paramSchema: 'resetPassword'
  },

  // Employee Management Modals
  [MODAL_IDS.EMPLOYEE_DETAILS]: {
    id: MODAL_IDS.EMPLOYEE_DETAILS,
    name: 'Chi tiết nhân viên',
    description: 'Xem thông tin chi tiết nhân viên',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/EmployeeDetailsSheet',
    hasParams: true,
    paramSchema: 'employeeDetails'
  },


  [MODAL_IDS.ADD_EMPLOYEE]: {
    id: MODAL_IDS.ADD_EMPLOYEE,
    name: 'Thêm nhân viên',
    description: 'Thêm nhân viên mới vào hệ thống',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/AddEmployeeSheet',
    hasParams: false
  },

  [MODAL_IDS.EMPLOYEE_IMPORT]: {
    id: MODAL_IDS.EMPLOYEE_IMPORT,
    name: 'Import nhân viên',
    description: 'Import danh sách nhân viên từ file',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/EmployeeImportModal',
    hasParams: false
  },

  [MODAL_IDS.EMPLOYEE_EXPORT]: {
    id: MODAL_IDS.EMPLOYEE_EXPORT,
    name: 'Export nhân viên',
    description: 'Export danh sách nhân viên',
    category: 'employee',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/EmployeeExportModal',
    hasParams: false
  },

  // Project Management Modals
  [MODAL_IDS.PROJECT_DETAILS]: {
    id: MODAL_IDS.PROJECT_DETAILS,
    name: 'Chi tiết dự án',
    description: 'Xem thông tin chi tiết dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/ProjectDetailsSheet',
    hasParams: true,
    paramSchema: 'projectDetails'
  },


  [MODAL_IDS.PROJECT_CREATE]: {
    id: MODAL_IDS.PROJECT_CREATE,
    name: 'Tạo dự án',
    description: 'Tạo dự án mới',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/AddProjectSheet',
    hasParams: false
  },

  [MODAL_IDS.ADD_PROJECT]: {
    id: MODAL_IDS.ADD_PROJECT,
    name: 'Thêm dự án',
    description: 'Thêm dự án mới',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/AddProjectSheet',
    hasParams: false
  },

  [MODAL_IDS.PROJECT_EDIT]: {
    id: MODAL_IDS.PROJECT_EDIT,
    name: 'Chỉnh sửa dự án',
    description: 'Chỉnh sửa thông tin dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/ProjectEditSheet',
    hasParams: true,
    paramSchema: 'projectEdit'
  },



  [MODAL_IDS.PROJECT_ASSIGNMENT]: {
    id: MODAL_IDS.PROJECT_ASSIGNMENT,
    name: 'Phân công dự án',
    description: 'Phân công nhân viên vào dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/ProjectAssignmentSheet',
    hasParams: true,
    paramSchema: 'projectAssignment'
  },

  [MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT]: {
    id: MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT,
    name: 'Thêm nhân viên vào dự án',
    description: 'Thêm nhân viên vào dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/project-employees/AddEmployeeSheet',
    hasParams: true,
    paramSchema: 'addEmployeeToProject'
  },

  [MODAL_IDS.REMOVE_EMPLOYEE]: {
    id: MODAL_IDS.REMOVE_EMPLOYEE,
    name: 'Xóa nhân viên khỏi dự án',
    description: 'Xóa nhân viên khỏi dự án',
    category: 'project',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/RemoveEmployeeModal',
    hasParams: true,
    paramSchema: 'removeEmployee'
  },

  // Timesheet Management Modals
  [MODAL_IDS.TIMESHEET_DETAILS]: {
    id: MODAL_IDS.TIMESHEET_DETAILS,
    name: 'Chi tiết bảng chấm công',
    description: 'Xem chi tiết bảng chấm công',
    category: 'timesheet',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/TimesheetDetailsModal',
    hasParams: true,
    paramSchema: 'timesheetDetails'
  },

  [MODAL_IDS.TIMESHEET_ENTRY]: {
    id: MODAL_IDS.TIMESHEET_ENTRY,
    name: 'Nhập bảng chấm công',
    description: 'Tạo hoặc chỉnh sửa bảng chấm công',
    category: 'timesheet',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/TimesheetEntryModal',
    hasParams: false
  },

  [MODAL_IDS.TIMESHEET_BULK_TRANSFER]: {
    id: MODAL_IDS.TIMESHEET_BULK_TRANSFER,
    name: 'Chuyển giao hàng loạt',
    description: 'Chuyển giao timesheet hàng loạt',
    category: 'timesheet',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/TimesheetBulkTransferModal',
    hasParams: false
  },


  // Report & Ledger Modals
  [MODAL_IDS.LEDGER_ENTRY_DETAILS]: {
    id: MODAL_IDS.LEDGER_ENTRY_DETAILS,
    name: 'Chi tiết bút toán',
    description: 'Xem chi tiết bút toán',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/ledger/LedgerEntryDetailsSheet',
    hasParams: true,
    paramSchema: 'ledgerDetails'
  },

  [MODAL_IDS.ADD_LEDGER_ENTRY]: {
    id: MODAL_IDS.ADD_LEDGER_ENTRY,
    name: 'Thêm giao dịch',
    description: 'Thêm giao dịch mới',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/AddTransactionSheet',
    hasParams: false
  },

  [MODAL_IDS.APPROVAL_DETAILS]: {
    id: MODAL_IDS.APPROVAL_DETAILS,
    name: 'Chi tiết phê duyệt',
    description: 'Xem chi tiết phê duyệt',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/ApprovalDetailsModal',
    hasParams: true,
    paramSchema: 'approvalDetails'
  },

  // Transaction Modals
  [MODAL_IDS.TRANSACTION_DETAILS]: {
    id: MODAL_IDS.TRANSACTION_DETAILS,
    name: 'Chi tiết giao dịch',
    description: 'Xem chi tiết giao dịch',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/TransactionDetailsSheet',
    hasParams: true,
    paramSchema: 'transactionDetails'
  },

  [MODAL_IDS.SETTLE_TRANSACTION]: {
    id: MODAL_IDS.SETTLE_TRANSACTION,
    name: 'Thanh toán giao dịch',
    description: 'Thanh toán giao dịch với thông tin chi tiết',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/SettleTransactionModal',
    hasParams: true,
    paramSchema: 'settleTransaction'
  },

  [MODAL_IDS.REVERSE_TRANSACTION]: {
    id: MODAL_IDS.REVERSE_TRANSACTION,
    name: 'Đảo ngược giao dịch',
    description: 'Tạo bút toán đảo ngược để hủy giao dịch',
    category: 'report',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/ReverseTransactionModal',
    hasParams: true,
    paramSchema: 'reverseTransaction'
  },

  // General Purpose Modals
  [MODAL_IDS.FILE_UPLOAD]: {
    id: MODAL_IDS.FILE_UPLOAD,
    name: 'Tải lên tập tin',
    description: 'Tải lên tập tin',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/FileUploadModal',
    hasParams: false
  },

  [MODAL_IDS.NOTIFICATION_SHEET]: {
    id: MODAL_IDS.NOTIFICATION_SHEET,
    name: 'Thông báo',
    description: 'Xem thông báo',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/notifications/NotificationSheet',
    hasParams: false
  },

  [MODAL_IDS.USER_PROFILE]: {
    id: MODAL_IDS.USER_PROFILE,
    name: 'Hồ sơ người dùng',
    description: 'Xem và chỉnh sửa hồ sơ cá nhân',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/sheets/UserProfileSheet',
    hasParams: false
  },

  [MODAL_IDS.CHANGE_PASSWORD]: {
    id: MODAL_IDS.CHANGE_PASSWORD,
    name: 'Đổi mật khẩu',
    description: 'Đổi mật khẩu tài khoản',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: '/src/components/modals/ChangePasswordModal',
    hasParams: false
  }
};

// Utility functions
export const getModalMetadata = (modalId: ModalId): ModalMetadata | null => {
  return MODAL_REGISTRY[modalId] || null;
};

export const getModalsByCategory = (category: ModalMetadata['category']): ModalMetadata[] => {
  return Object.values(MODAL_REGISTRY).filter(modal => modal.category === category);
};

export const getModalsByRole = (role: UserRole): ModalMetadata[] => {
  return Object.values(MODAL_REGISTRY).filter(modal => modal.roles.includes(role));
};

export const isValidModalId = (id: string): id is ModalId => {
  return Object.values(MODAL_IDS).includes(id as ModalId);
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
    withParams: modals.filter(m => m.hasParams).length
  };
};
