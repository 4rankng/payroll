import { format } from "date-fns";
import type { Notification } from '@/types/api/notification.types';

// ---- Notification type → style mapping ----

interface NotificationStyle {
  iconColor: string;
  bgColor: string;
}

const NOTIFICATION_STYLES: Record<string, NotificationStyle> = {
  timesheet_approval:   { iconColor: 'text-blue-600',   bgColor: 'bg-blue-100'   },
  timesheet_reminder:   { iconColor: 'text-cyan-600',   bgColor: 'bg-cyan-100'   },
  payroll_ready:        { iconColor: 'text-orange-600',  bgColor: 'bg-orange-100' },
  payroll_complete:     { iconColor: 'text-emerald-600', bgColor: 'bg-emerald-100' },
  payment_reminder:     { iconColor: 'text-yellow-600',  bgColor: 'bg-yellow-100' },
  system_alert:         { iconColor: 'text-red-600',     bgColor: 'bg-red-100'    },
  project_update:       { iconColor: 'text-green-600',   bgColor: 'bg-green-100'  },
  password_change:      { iconColor: 'text-red-600',     bgColor: 'bg-red-100'    },
  custom:               { iconColor: 'text-gray-500',    bgColor: 'bg-gray-100'   },
};

const DEFAULT_STYLE: NotificationStyle = { iconColor: 'text-muted-foreground', bgColor: 'bg-slate-100' };

export function getNotificationStyle(type: Notification['type']): NotificationStyle {
  return NOTIFICATION_STYLES[type] || DEFAULT_STYLE;
}

/** Accent dot color for NotificationItem (hex for inline style) */
const NOTIFICATION_ACCENT_COLORS: Record<string, string> = {
  timesheet_approval: '#3B82F6',
  timesheet_reminder: '#06B6D4',
  payroll_ready:      '#F97316',
  payroll_complete:   '#22C55E',
  payment_reminder:   '#F59E0B',
  system_alert:       '#EF4444',
  project_update:     '#22C55E',
  password_change:    '#EF4444',
  custom:             '#9CA3AF',
};

export function getNotificationAccentColor(type: Notification['type']): string {
  return NOTIFICATION_ACCENT_COLORS[type] || '#9CA3AF';
}

// ---- Notification date formatting ----

/**
 * Format notification timestamp for compact display
 * - < 24h: "14:30"
 * - < 7d: "T2 14:30"
 * - else: "26/04/2026"
 */
export function formatNotificationDate(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffHours = Math.abs(now.getTime() - date.getTime()) / (1000 * 60 * 60);

    if (diffHours < 24) {
      return date.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' });
    }
    if (diffHours < 24 * 7) {
      return format(date, 'dd/MM/yyyy');
    }
    return format(date, 'dd/MM/yyyy');
  } catch {
    return '—';
  }
}

/**
 * Format notification timestamp for detail view (full date/time)
 */
export function formatNotificationDetailDate(timestamp: string): string {
  try {
    return new Date(timestamp).toLocaleString('vi-VN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit',
    });
  } catch {
    return timestamp;
  }
}
