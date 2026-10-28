// ZNS tracking API types — admin view over zns_messages, the single audit
// table for every ZNS the system sends (all features).

export interface ZnsMessage {
  id: number;
  created_at: string;
  feature: string;
  template_id: string;
  phone: string;
  status: string;
  provider_msg_id: string;
  provider_code: number;
  error_msg: string;
  duration_ms: number;
}

export interface ZnsMessagesQueryParams {
  page?: number;
  pageSize?: number;
  /** Multi-select status filter (sent/failed/pending/processing/suppressed). */
  status?: string[];
  /** Call-source filter (salary_notification, password_reset, ...). */
  feature?: string;
  templateId?: string;
  /** YYYY-MM-DD, inclusive day bounds. */
  fromDate?: string;
  toDate?: string;
  /** Substring search on phone or provider message id. */
  q?: string;
}

export interface BackendZnsMessagesResponse {
  status: string;
  message: string;
  data: ZnsMessage[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

/** All statuses a zns_messages row can carry (provider writes sent/failed;
 * backfilled rows retain the flexpay queue states). */
export const ZNS_STATUSES = ['sent', 'failed', 'pending', 'processing', 'suppressed'] as const;

/** Known call-source values for the feature filter. */
export const ZNS_FEATURES = [
  'salary_notification',
  'password_reset',
  'integration_reset',
  'self_checkin',
  'test_send',
] as const;

export const ZNS_STATUS_LABELS: Record<string, string> = {
  sent: 'Đã gửi',
  failed: 'Lỗi',
  pending: 'Chờ gửi',
  processing: 'Đang gửi',
  suppressed: 'Đã chặn',
};

export const ZNS_FEATURE_LABELS: Record<string, string> = {
  salary_notification: 'Thông báo lương',
  password_reset: 'Đặt lại mật khẩu',
  integration_reset: 'Reset qua tích hợp',
  self_checkin: 'Tự chấm công',
  test_send: 'Gửi thử',
};
