import { CheckCircle, XCircle, Clock, AlertCircle } from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';
import { TIMESHEET_STATUS_COLORS, TIMESHEET_STRIP_COLORS } from './timesheetStatusColors';

type TimesheetStatus = Timesheet['status'];

interface StatusColorConfig {
  bg: string;
  border: string;
  text: string;
  badge: string;
}

export const getStatusColor = (status: TimesheetStatus): StatusColorConfig => {
  const configs: Record<TimesheetStatus, StatusColorConfig> = {
    draft: {
      bg: 'bg-gray-400',
      border: 'border-border',
      text: 'text-muted-foreground',
      badge: 'bg-muted text-foreground border-border'
    },
    pending_approval: {
      bg: 'bg-yellow-500',
      border: 'border-yellow-200',
      text: 'text-amber-800',
      badge: 'bg-yellow-50 text-amber-800 border-yellow-200'
    },
    approved: {
      bg: 'bg-blue-500',
      border: 'border-blue-200',
      text: 'text-blue-800',
      badge: 'bg-blue-50 text-blue-800 border-blue-200'
    },
    rejected: {
      bg: 'bg-red-500',
      border: 'border-red-200',
      text: 'text-red-600',
      badge: 'bg-red-50 text-red-700 border-red-200'
    }
  };

  return configs[status] || configs.draft;
};

export const getStatusIcon = (status: TimesheetStatus) => {
  const icons = {
    draft: AlertCircle,
    pending_approval: Clock,
    approved: CheckCircle,
    rejected: XCircle
  };

  return icons[status] || AlertCircle;
};

export const getStatusLabel = (status: TimesheetStatus): string => {
  const labels: Record<TimesheetStatus, string> = {
    draft: 'Nháp',
    pending_approval: 'Chờ duyệt',
    approved: 'Đã duyệt',
    rejected: 'Loại'
  };

  return labels[status] || status;
};
