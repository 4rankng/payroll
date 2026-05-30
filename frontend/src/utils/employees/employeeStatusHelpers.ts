import { CheckCircle, XCircle, AlertCircle } from "lucide-react";

export const getEmployeeStatusLabel = (status: string): string => {
  const statusLabels = {
    active: "Đang dùng",
    inactive: "Không hoạt động"
  };
  return statusLabels[status as keyof typeof statusLabels] || status;
};

export const getEmployeeStatusColor = (status: string): string => {
  return status === 'active'
    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
    : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300';
};

export const getTimesheetStatusIcon = (status: string) => {
  const statusIcons = {
    approved: CheckCircle,
    pending_approval: AlertCircle,
    rejected: XCircle,
    draft: XCircle,
  };
  return statusIcons[status as keyof typeof statusIcons] || AlertCircle;
};

export const getTimesheetStatusLabel = (status: string): string => {
  const statusLabels = {
    approved: 'Đã duyệt',
    pending_approval: 'Chờ duyệt',
    rejected: 'Loại',
    draft: 'Bản nháp',
  };
  return statusLabels[status as keyof typeof statusLabels] || status;
};

export const getTimesheetStatusColor = (status: string): string => {
  const statusColors = {
    approved: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    pending_approval: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
    rejected: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
    draft: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
  };
  return statusColors[status as keyof typeof statusColors] || 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300';
};
