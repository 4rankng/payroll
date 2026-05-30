import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';

/**
 * Format Vietnamese currency (đ)
 * @param amount Amount in VND (integer)
 * @param options Formatting options
 */
export const formatVietnameseCurrency = (
  amount: number,
  options: {
    compact?: boolean;
    showSymbol?: boolean;
  } = {}
): string => {
  const { compact = false, showSymbol = true } = options;

  const formatter = new Intl.NumberFormat('vi-VN', {
    style: 'decimal',
    notation: compact ? 'compact' : 'standard',
    compactDisplay: 'short',
  });

  const formattedAmount = formatter.format(amount);
  return showSymbol ? `${formattedAmount} d` : formattedAmount;
};

/**
 * Format Vietnamese date
 * @param dateString ISO date string or Date object
 * @param formatPattern Date format pattern
 */
export const formatVietnameseDate = (
  dateString: string | Date,
  formatPattern: string = 'dd/MM/yyyy'
): string => {
  try {
    const date = typeof dateString === 'string' ? parseISO(dateString) : dateString;
    return format(date, formatPattern, { locale: vi });
  } catch (error) {
    console.error('Error formatting Vietnamese date:', error);
    return 'Ngày không hợp lệ';
  }
};

/**
 * Format Vietnamese datetime
 * @param dateString ISO date string or Date object
 * @param includeTime Whether to include time
 */
export const formatVietnameseDateTime = (
  dateString: string | Date,
  includeTime: boolean = true
): string => {
  const pattern = includeTime ? 'dd/MM/yyyy HH:mm' : 'dd/MM/yyyy';
  return formatVietnameseDate(dateString, pattern);
};

/**
 * Format relative time in Vietnamese
 * @param dateString ISO date string or Date object
 */
export const formatVietnameseRelativeTime = (dateString: string | Date): string => {
  try {
    const date = typeof dateString === 'string' ? parseISO(dateString) : dateString;
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();

    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    const months = Math.floor(days / 30);
    const years = Math.floor(days / 365);

    if (years > 0) {
      return `${years} năm trước`;
    } else if (months > 0) {
      return `${months} tháng trước`;
    } else if (days > 0) {
      return `${days} ngày trước`;
    } else if (hours > 0) {
      return `${hours} giờ trước`;
    } else if (minutes > 0) {
      return `${minutes} phút trước`;
    } else {
      return 'Vài giây trước';
    }
  } catch (error) {
    console.error('Error formatting Vietnamese relative time:', error);
    return 'Thời gian không xác định';
  }
};

/**
 * Format month year in Vietnamese
 * @param dateString ISO date string or Date object
 */
export const formatVietnameseMonthYear = (dateString: string | Date): string => {
  return formatVietnameseDate(dateString, 'MM/yyyy');
};

/**
 * Format percentage with Vietnamese locale
 * @param value Percentage value (e.g., 12.5 for 12.5%)
 * @param decimals Number of decimal places
 */
export const formatVietnamesePercentage = (
  value: number,
  decimals: number = 1
): string => {
  return new Intl.NumberFormat('vi-VN', {
    style: 'percent',
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(value / 100);
};

/**
 * Format number with Vietnamese locale
 * @param value Number to format
 * @param decimals Number of decimal places
 */
export const formatVietnameseNumber = (
  value: number,
  decimals: number = 0
): string => {
  return new Intl.NumberFormat('vi-VN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(value);
};

/**
 * Get Vietnamese month name
 * @param monthIndex 0-based month index (0 = January)
 */
export const getVietnameseMonthName = (monthIndex: number): string => {
  const months = [
    'Tháng 1', 'Tháng 2', 'Tháng 3', 'Tháng 4',
    'Tháng 5', 'Tháng 6', 'Tháng 7', 'Tháng 8',
    'Tháng 9', 'Tháng 10', 'Tháng 11', 'Tháng 12'
  ];
  return months[monthIndex] || 'Tháng không hợp lệ';
};

/**
 * Get Vietnamese day name
 * @param dayIndex 0-based day index (0 = Sunday)
 */
export const getVietnameseDayName = (dayIndex: number): string => {
  const days = [
    'Chủ nhật', 'Thứ hai', 'Thứ ba', 'Thứ tư',
    'Thứ năm', 'Thứ sáu', 'Thứ bảy'
  ];
  return days[dayIndex] || 'Ngày không hợp lệ';
};

/**
 * Format chart month label (e.g., "2024-01" -> "T1/2024")
 * @param monthString Month string in YYYY-MM format
 */
export const formatChartMonthLabel = (monthString: string): string => {
  try {
    if (!monthString || typeof monthString !== 'string') {
      return monthString || '';
    }
    const [year, month] = monthString.split('-');
    const monthNum = parseInt(month, 10);
    return `T${monthNum}/${year}`;
  } catch (error) {
    console.error('Error formatting chart month label:', error);
    return monthString || '';
  }
};

/**
 * Format activity icon mapping for Vietnamese context
 */
export const getVietnameseActivityIcon = (activityType: string): string => {
  const iconMap: Record<string, string> = {
    'timesheet_approved': 'check-circle',
    'timesheet_rejected': 'x-circle',
    'employee_created': 'user-plus',
    'employee_updated': 'user-edit',
    'project_created': 'folder-plus',
    'project_status_changed': 'git-branch',
    'payroll_processed': 'dollar-sign',
    'user_login': 'log-in',
    'import_completed': 'upload',
  };

  return iconMap[activityType] || 'activity';
};

/**
 * Format notification type for Vietnamese display
 */
export const getVietnameseNotificationIcon = (notificationType: string): string => {
  const iconMap: Record<string, string> = {
    'error': 'alert-triangle',
    'warning': 'alert-circle',
    'info': 'info',
    'success': 'check-circle',
    'approval_needed': 'clock',
    'reminder': 'bell',
  };

  return iconMap[notificationType] || 'bell';
};

/**
 * Get Vietnamese project status label
 * @param status Project status value
 */
export const getVietnameseProjectStatus = (status: 'draft' | 'active' | 'paused' | 'completed' | 'cancelled'): string => {
  const statusMap: Record<string, string> = {
    'draft': 'Bản nháp',
    'active': 'Đang dùng',
    'paused': 'Tạm dừng',
    'completed': 'Hoàn thành',
    'cancelled': 'Đã hủy',
  };

  return statusMap[status] || status;
};

/**
 * Get project status badge variant
 * @param status Project status value
 */
export const getProjectStatusVariant = (status: string): 'default' | 'secondary' | 'outline' | 'destructive' => {
  switch (status?.toLowerCase()) {
    case 'active':
      return 'default';
    case 'paused':
      return 'outline';
    case 'completed':
      return 'secondary';
    case 'draft':
      return 'outline';
    case 'cancelled':
      return 'destructive';
    default:
      return 'secondary';
  }
};

const COMBINING_RE = /[\u0300-\u036f]/g;

/** Strip Vietnamese diacritics for search. "Lù Văn Hà" → "lu van ha" */
export function normalizeVietnamese(str: string): string {
  return str
    .normalize('NFD')
    .replace(COMBINING_RE, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toLowerCase();
}
