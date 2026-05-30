import {
  Users,
  FolderOpen,
  UserCheck,
  Clock,
  CheckSquare,
  UserPlus,
  FolderPlus,
  CalendarPlus,
  Download,
  Upload,
  Eye,
  Edit,
  Search
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

export interface Action {
  id: string;
  label: string;
  description?: string;
  icon: LucideIcon;
  href?: string;
  onClick?: () => void;
  category: 'navigation' | 'creation' | 'data' | 'management';
  permissions?: ('admin' | 'partner')[];
}

// Define all available actions
export const allActions: Action[] = [
  // Navigation Actions
  {
    id: 'nav-admin-dashboard',
    label: 'Bảng điều khiển',
    description: 'Xem tổng quan hệ thống',
    icon: FolderOpen,
    href: '/admin',
    category: 'navigation',
    permissions: ['admin']
  },
  {
    id: 'nav-partner-dashboard',
    label: 'Bảng điều khiển',
    description: 'Xem tổng quan dự án',
    icon: FolderOpen,
    href: '/partner/projects',
    category: 'navigation',
    permissions: ['partner']
  },
  {
    id: 'nav-users',
    label: 'Người dùng',
    description: 'Quản lý tài khoản người dùng',
    icon: Users,
    href: '/admin/users',
    category: 'navigation',
    permissions: ['admin']
  },
  {
    id: 'nav-projects',
    label: 'Dự án',
    description: 'Xem và Dự án',
    icon: FolderOpen,
    href: '/admin/projects',
    category: 'navigation',
    permissions: ['admin']
  },
  {
    id: 'nav-admin-employees',
    label: 'Nhân viên',
    description: 'Quản lý thông tin nhân viên',
    icon: UserCheck,
    href: '/admin/employees',
    category: 'navigation',
    permissions: ['admin']
  },
  {
    id: 'nav-partner-employees',
    label: 'Nhân viên',
    description: 'Nhân viên dự án',
    icon: UserCheck,
    href: '/partner/employees',
    category: 'navigation',
    permissions: ['partner']
  },
  {
    id: 'nav-admin-timesheet',
    label: 'Chấm công',
    description: 'Quản lý thời gian làm việc',
    icon: Clock,
    href: '/admin/timesheet',
    category: 'navigation',
    permissions: ['admin']
  },
  {
    id: 'nav-partner-timesheet',
    label: 'Chấm công',
    description: 'Bảng công cá nhân',
    icon: Clock,
    href: '/partner/timesheet',
    category: 'navigation',
    permissions: ['partner']
  },
  {
    id: 'nav-approvals',
    label: 'Hàng đợi duyệt',
    description: 'Xử lý các yêu cầu cần duyệt',
    icon: CheckSquare,
    href: '/admin/approvals',
    category: 'navigation',
    permissions: ['admin']
  },

  // Creation Actions
  {
    id: 'create-user',
    label: 'Tạo người dùng',
    description: 'Thêm người dùng mới',
    icon: UserPlus,
    href: '/admin/users?modal=add_user',
    category: 'creation',
    permissions: ['admin'],
  },
  {
    id: 'create-project',
    label: 'Tạo dự án',
    description: 'Thêm dự án mới',
    icon: FolderPlus,
    href: '/admin/projects?modal=project_create',
    category: 'creation',
    permissions: ['admin'],
  },
  {
    id: 'create-employee',
    label: 'Thêm nhân viên',
    description: 'Thêm nhân viên mới',
    icon: Users,
    href: '/admin/employees?modal=add_employee',
    category: 'creation',
    permissions: ['admin', 'partner'],
  },
  {
    id: 'create-timesheet',
    label: 'Tạo chấm công',
    description: 'Tạo bảng chấm công mới',
    icon: CalendarPlus,
    href: '/admin/timesheet?modal=timesheet_entry',
    category: 'creation',
    permissions: ['admin', 'partner'],
  },

  // Data Actions
  {
    id: 'export-employees',
    label: 'Xuất danh sách nhân viên',
    description: 'Tải xuống danh sách nhân viên',
    icon: Download,
    category: 'data',
    permissions: ['admin', 'partner']
  },
  {
    id: 'import-employees',
    label: 'Nhập nhân viên',
    description: 'Nhập danh sách nhân viên từ file',
    icon: Upload,
    category: 'data',
    permissions: ['admin', 'partner']
  },
  {
    id: 'export-timesheet',
    label: 'Xuất chấm công',
    description: 'Tải xuống dữ liệu chấm công',
    icon: Download,
    category: 'data',
    permissions: ['admin', 'partner']
  },

  // Management Actions
  {
    id: 'view-profile',
    label: 'Xem hồ sơ',
    description: 'Xem thông tin cá nhân',
    icon: Eye,
    category: 'management',
    permissions: ['admin', 'partner']
  },
  {
    id: 'edit-profile',
    label: 'Chỉnh sửa hồ sơ',
    description: 'Cập nhật thông tin cá nhân',
    icon: Edit,
    category: 'management',
    permissions: ['admin', 'partner']
  },
  {
    id: 'search',
    label: 'Tìm kiếm',
    description: 'Tìm kiếm trong hệ thống',
    icon: Search,
    category: 'management',
    permissions: ['admin', 'partner'],
  }
];

// Helper functions to get actions by role
export const getActionsByRole = (role: 'admin' | 'partner'): Action[] => {
  return allActions.filter(action =>
    action.permissions?.includes(role) || !action.permissions
  );
};

export const getActionsByCategory = (category: Action['category'], role?: 'admin' | 'partner'): Action[] => {
  let actions = allActions.filter(action => action.category === category);

  if (role) {
    actions = actions.filter(action =>
      action.permissions?.includes(role) || !action.permissions
    );
  }

  return actions;
};

export const getQuickActions = (role: 'admin' | 'partner', limit: number = 6): Action[] => {
  // Fixed list of priority quick actions: add new project, add new employee, add new timesheet entry
  const priorityActionIds = [
    'create-project',
    'create-employee',
    'create-timesheet'
  ];

  const actions = getActionsByRole(role);

  // Get priority actions first
  const priorityActions = priorityActionIds
    .map(id => actions.find(action => action.id === id))
    .filter((action): action is Action => action !== undefined);

  // Fill remaining slots with other creation and navigation actions if needed
  const remainingActions = actions.filter(
    action => !priorityActionIds.includes(action.id) &&
    ['creation', 'navigation'].includes(action.category)
  );

  const quickActions = [...priorityActions, ...remainingActions];

  return quickActions.slice(0, limit);
};

export const getActionById = (id: string): Action | undefined => {
  return allActions.find(action => action.id === id);
};

// Search actions
export const searchActions = (query: string, role?: 'admin' | 'partner'): Action[] => {
  const normalizedQuery = query.trim();

  const actions = role ? getActionsByRole(role) : allActions;

  return actions.filter(action =>
    vietnameseIncludes(action.label, normalizedQuery) ||
    (action.description && vietnameseIncludes(action.description, normalizedQuery)) ||
    vietnameseIncludes(action.id, normalizedQuery)
  );
};
