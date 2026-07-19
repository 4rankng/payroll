import { ApprovalPriority, ApprovalStatus, ApprovalType } from '@/types/approval';

export const getPriorityColor = (priority: ApprovalPriority): string => {
  const colors = {
    low: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
    high: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300',
    urgent: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300'
  };
  return colors[priority];
};

export const getStatusColor = (status: ApprovalStatus): string => {
  const colors = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
    approved: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    rejected: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
    reviewing: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300'
  };
  return colors[status];
};

export const getTypeLabel = (type: ApprovalType): string => {
  const labels = {
    timesheet: 'Bảng công',
    payrate: 'Mức lương',
    payroll: 'Bảng lương',
    employee: 'Nhân viên'
  };
  return labels[type];
};

export const getTypeColor = (type: ApprovalType): string => {
  const colors = {
    timesheet: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
    payrate: 'bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-300',
    payroll: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    employee: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
  };
  return colors[type];
};

export const formatTimeAgo = (dateString: string): string => {
  // Simple time ago formatting - can be enhanced
  return dateString;
};

export const getPriorityWeight = (priority: ApprovalPriority): number => {
  const weights = { low: 1, medium: 2, high: 3, urgent: 4 };
  return weights[priority];
};

export const getInitials = (name: string) => {
  return name
    .split(' ')
    .slice(-2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};