import { Notification, NotificationType } from '@/types/api/notification.types';
import { authManager } from '@/lib/auth';

interface NavigationTarget {
  path: string;
  params?: Record<string, string>;
  highlight?: string; // Element to highlight after navigation
}

// Whitelist of valid navigation paths
const SAFE_ROUTES = {
  admin: [
    '/admin/timesheet',
    '/admin/approvals',
    '/admin/employees',
    '/admin/projects',
    '/admin/settings',
  ],
  partner: [
    '/partner/timesheet',
    '/partner/employees',
    '/partner/project',
  ],
  employee: [
    '/employee/timesheet',
  ],
  adv_partner: [
    '/adv-partner/advance-payments',
  ],
} as const;

export const getNotificationNavigation = (notification: Notification): NavigationTarget | null => {
  const userRole = authManager.getUserRole();
  if (!userRole) return null;

  const { type, metadata } = notification;
  const baseRoutes = SAFE_ROUTES[userRole as keyof typeof SAFE_ROUTES];

  switch (type) {
    case 'timesheet_reminder': {
      const path = userRole === 'admin' ? '/admin/timesheet' : '/partner/timesheet';
      return {
        path,
        params: metadata?.employee_id ? { employee: String(metadata.employee_id) } : {},
        highlight: metadata?.timesheet_id ? `timesheet-${metadata.timesheet_id}` : undefined
      };
    }

    case 'approval_request':
    case 'approval_result': {
      if (userRole === 'admin') {
        return {
          path: '/admin/approvals',
          params: metadata?.approval_id ? { approval: String(metadata.approval_id) } : {},
          highlight: metadata?.approval_id ? `approval-${metadata.approval_id}` : undefined
        };
      } else {
        // Partners see approval items in their timesheet
        return {
          path: '/partner/timesheet',
          params: metadata?.timesheet_id ? { timesheet: String(metadata.timesheet_id) } : {},
          highlight: metadata?.approval_id ? `approval-${metadata.approval_id}` : undefined
        };
      }
    }

    case 'payroll_complete': {
      // Redirect payroll notifications to timesheet since payroll pages are removed
      const path = userRole === 'admin' ? '/admin/timesheet' : '/partner/timesheet';
      return {
        path,
        params: {},
        highlight: undefined
      };
    }

    default:
      return null;
  }
};

export const validateNavigationPath = (path: string, userRole: string): boolean => {
  const allowedRoutes = SAFE_ROUTES[userRole as keyof typeof SAFE_ROUTES];
  if (!allowedRoutes) return false;
  return allowedRoutes.some(route => path.startsWith(route));
};

export const buildNavigationUrl = (target: NavigationTarget): string => {
  const url = new URL(target.path, window.location.origin);
  
  // Add query parameters
  if (target.params) {
    Object.entries(target.params).forEach(([key, value]) => {
      url.searchParams.set(key, value);
    });
  }

  // Add highlight parameter
  if (target.highlight) {
    url.searchParams.set('highlight', target.highlight);
  }

  return url.pathname + url.search;
};

export const getFallbackPath = (userRole: string): string => {
  switch (userRole) {
    case 'admin': return '/admin/notifications';
    case 'partner': return '/partner/notifications';
    case 'employee': return '/employee/timesheet';
    case 'adv_partner': return '/adv-partner/advance-payments';
    default: return '/';
  }
};