import type { TimesheetEditRequest } from '@/types/api/timesheet.types';

type EditRequestStatus = TimesheetEditRequest['status'];

const STATUS_LABEL_MAP: Record<EditRequestStatus, string> = {
  pending: 'Đang chờ xử lý',
  approved: 'Đã phê duyệt',
  rejected: 'Đã từ chối',
};

const STATUS_BADGE_MAP: Record<EditRequestStatus, 'info' | 'success' | 'destructive'> = {
  pending: 'info',
  approved: 'success',
  rejected: 'destructive',
};

export const EDIT_REQUEST_STATUS_OPTIONS: Array<{ value: EditRequestStatus | 'all'; label: string; }> = [
  { value: 'all', label: 'Tất cả trạng thái' },
  { value: 'pending', label: STATUS_LABEL_MAP.pending },
  { value: 'approved', label: STATUS_LABEL_MAP.approved },
  { value: 'rejected', label: STATUS_LABEL_MAP.rejected },
];

export const getEditRequestStatusLabel = (status: EditRequestStatus): string => {
  return STATUS_LABEL_MAP[status];
};

export const getEditRequestBadgeVariant = (status: EditRequestStatus) => {
  return STATUS_BADGE_MAP[status];
};

export const formatEditRequestDateTime = (isoString?: string | null): string => {
  if (!isoString) {
    return '-';
  }
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) {
    return '-';
  }
  return date.toLocaleString('vi-VN', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};
