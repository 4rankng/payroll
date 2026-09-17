import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * User Domain Modal Registry
 * Contains all user-related modals with schemas and metadata
 */

export const USER_MODAL_IDS = {
  ADD_USER: 'add_user',
  USER_DETAILS: 'user_details',
  RESET_PASSWORD: 'reset_password',
  USER_PROFILE: 'user_profile',
  CHANGE_PASSWORD: 'change_password',
} as const;

export type UserModalId = typeof USER_MODAL_IDS[keyof typeof USER_MODAL_IDS];

// Zod schemas for user modals
export const userModalSchemas = {
  [USER_MODAL_IDS.ADD_USER]: z.object({}),
  [USER_MODAL_IDS.USER_DETAILS]: z.object({
    id: z.string().min(1, 'User ID is required'),
    tab: z.enum(['details', 'activity', 'permissions']).optional().default('details'),
  }),
  [USER_MODAL_IDS.RESET_PASSWORD]: z.object({
    userId: z.string().min(1, 'User ID is required'),
  }),
  [USER_MODAL_IDS.USER_PROFILE]: z.object({}),
  [USER_MODAL_IDS.CHANGE_PASSWORD]: z.object({}),
} as const;

// Modal metadata with permissions
interface UserModalMetadata {
  id: UserModalId;
  name: string;
  description: string;
  category: 'user';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof userModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'User' | 'SelfAccount' | 'all';
  };
}

export const userModals: Record<UserModalId, UserModalMetadata> = {
  [USER_MODAL_IDS.ADD_USER]: {
    id: USER_MODAL_IDS.ADD_USER,
    name: 'Thêm người dùng',
    description: 'Tạo tài khoản người dùng mới',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'AddUserSheet',
    hasParams: false,
    permissions: {
      action: 'create',
      subject: 'User',
    },
  },
  [USER_MODAL_IDS.USER_DETAILS]: {
    id: USER_MODAL_IDS.USER_DETAILS,
    name: 'Chi tiết người dùng',
    description: 'Xem và chỉnh sửa thông tin người dùng',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'UserDetailsSheet',
    hasParams: true,
    schema: USER_MODAL_IDS.USER_DETAILS,
    permissions: {
      action: 'read',
      subject: 'User',
    },
  },
  [USER_MODAL_IDS.RESET_PASSWORD]: {
    id: USER_MODAL_IDS.RESET_PASSWORD,
    name: 'Đặt lại mật khẩu',
    description: 'Đặt lại mật khẩu cho người dùng',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'ResetPasswordModal',
    hasParams: true,
    schema: USER_MODAL_IDS.RESET_PASSWORD,
    permissions: {
      action: 'update',
      subject: 'User',
    },
  },
  [USER_MODAL_IDS.USER_PROFILE]: {
    id: USER_MODAL_IDS.USER_PROFILE,
    name: 'Hồ sơ người dùng',
    description: 'Xem và chỉnh sửa hồ sơ cá nhân',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner'],
    componentPath: 'UserProfileSheet',
    hasParams: false,
    permissions: {
      action: 'read',
      subject: 'SelfAccount',
    },
  },
  [USER_MODAL_IDS.CHANGE_PASSWORD]: {
    id: USER_MODAL_IDS.CHANGE_PASSWORD,
    name: 'Đổi mật khẩu',
    description: 'Đổi mật khẩu tài khoản',
    category: 'user',
    requiresAuth: true,
    roles: ['admin', 'partner', 'adv_partner', 'accountant', 'employee'],
    componentPath: 'ChangePasswordModal',
    hasParams: false,
    permissions: {
      action: 'update',
      subject: 'SelfAccount',
    },
  },
} as const;
