// Unified Notification Types
// All notification types live here — do NOT create separate notification type files

export type NotificationType =
  | 'timesheet_reminder'
  | 'timesheet_approval'
  | 'approval_request'
  | 'approval_result'
  | 'payroll_ready'
  | 'payroll_complete'
  | 'system_alert'
  | 'project_update'
  | 'payment_reminder'
  | 'password_change'
  | 'advance_quota_changed'
  | 'custom';

export type NotificationChannel = 'push' | 'email';

export type NotificationContentType = 'plain_text' | 'html' | 'markdown';

export interface Notification {
  id: number;
  type: NotificationType;
  channel: NotificationChannel;
  title: string;
  message: string;
  content_type?: NotificationContentType;
  read_at: string | null;
  created_at: string;
  metadata?: NotificationMetadata;
}

export interface NotificationMetadata {
  target_id?: string | number;
  target_type?: string;
  project_id?: string | number;
  employee_id?: string | number;
  timesheet_id?: string | number;
  payroll_id?: string | number;
  approval_id?: string | number;
  deep_link?: string;
}

// ---- Request/Response types ----

export interface NotificationFilters {
  page?: number;
  pageSize?: number;
  type?: NotificationType;
  channel?: NotificationChannel;
  unread_only?: boolean;
  [key: string]: unknown;
}

export interface PaginatedNotificationResponse {
  status: string;
  data: Notification[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface UnreadNotificationResponse {
  status: string;
  data: Notification[];
  count: number;
  message: string;
}

export interface SendNotificationRequest {
  recipient_ids: number[];
  title: string;
  message: string;
  content_type?: NotificationContentType;
  to_all_partners?: boolean;
  to_all_admins?: boolean;
  to_all_employees?: boolean;
}

export interface SendNotificationResponse {
  status: string;
  message: string;
  data?: Notification;
}
