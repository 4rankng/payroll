// Audit & Security API Types

// Base audit types
export interface AuditSummary {
  totalAuditEntries: number;
  uniqueUsers: number;
  topActions: {
    action: string;
    count: number;
    percentage: number;
  }[];
  entityDistribution: {
    entityType: string;
    count: number;
    percentage: number;
  }[];
  activityByHour: {
    hour: number;
    count: number;
  }[];
  lastAuditEntry: string;
}

export interface AuditLogEntry {
  id: number;
  userId: number;
  userName: string;
  userRole: 'admin' | 'partner';
  userEmail?: string;
  action: AuditAction;
  entityType: EntityType;
  entityId: number;
  entityDescription?: string;
  oldValues: Record<string, unknown> | null;
  newValues: Record<string, unknown> | null;
  changes?: AuditChange[];
  ipAddress: string;
  userAgent: string;
  location?: {
    country: string;
    city: string;
    region?: string;
    timezone: string;
  };
  createdAt: string;
}

export interface AuditChange {
  field: string;
  oldValue: unknown;
  newValue: unknown;
}

export interface UserActivity {
  userId: number;
  userName: string;
  userRole: 'admin' | 'partner';
  activitySummary: {
    totalActions: number;
    lastLogin: string;
    mostActiveDay: string;
    topActions: {
      action: string;
      count: number;
    }[];
  };
  recentActivity: {
    action: string;
    entityType: string;
    entityId: number;
    description: string;
    ipAddress: string;
    createdAt: string;
  }[];
}

export interface BlacklistedToken {
  id: number;
  tokenJti: string;
  userId: number;
  userName: string;
  expiresAt: string;
  blacklistedAt: string;
  reason: TokenBlacklistReason;
  isExpired: boolean;
}

export interface SecurityReport {
  loginAnalysis: {
    totalLogins: number;
    uniqueUsers: number;
    failedLogins: number;
    suspiciousIPs: {
      ipAddress: string;
      failedAttempts: number;
      lastAttempt: string;
      blocked: boolean;
    }[];
    unusualLoginTimes: {
      userId: number;
      userName: string;
      loginTime: string;
      ipAddress: string;
    }[];
  };
  dataModifications: {
    totalModifications: number;
    criticalChanges: number;
    userWithMostChanges: {
      userId: number;
      userName: string;
      changeCount: number;
    };
  };
  securityIncidents: SecurityIncident[];
  recommendations: string[];
  reportGeneratedAt: string;
}

export interface SecurityIncident {
  type: SecurityIncidentType;
  severity: 'low' | 'medium' | 'high' | 'critical';
  userId?: number;
  details: string;
  timestamp: string;
  resolved: boolean;
}

export interface LoginHistoryEntry {
  id: number;
  userId: number;
  userName: string;
  loginStatus: 'success' | 'failed';
  ipAddress: string;
  userAgent: string;
  deviceInfo: {
    browser: string;
    version: string;
    os: string;
    device: string;
  };
  location: {
    country: string;
    city: string;
    region: string;
    timezone: string;
  };
  sessionDuration?: number;
  logoutType?: 'manual' | 'timeout' | 'forced';
  createdAt: string;
  loggedOutAt?: string;
}

export interface ComplianceReport {
  report_id: string;
  report_type: ComplianceReportType;
  period: {
    start_date: string;
    end_date: string;
  };
  compliance_summary: {
    total_transactions: number;
    financial_entries: number;
    employee_records: number;
    payroll_batches: number;
    audit_coverage: number;
    data_integrity_score: number;
  };
  regulatory_checks: {
    vietnam_labor_law: ComplianceStatus;
    tax_reporting: ComplianceStatus;
    data_protection: ComplianceStatus;
    financial_records: ComplianceStatus;
  };
  findings: ComplianceFinding[];
  download_url: string;
  expires_at: string;
  generated_at: string;
}

export interface ComplianceFinding {
  category: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  description: string;
  recommendation: string;
}

export interface DataLineage {
  entity_info: {
    type: EntityType;
    id: number;
    current_status: string;
    created_date: string;
  };
  creation_trail: {
    created_by: {
      user_id: number;
      user_name: string;
      method: string;
    };
    creation_context: {
      ip_address: string;
      user_agent: string;
      session_id: string;
    };
  };
  modification_history: {
    audit_id: number;
    action: string;
    timestamp: string;
    user_id: number;
    user_name: string;
    changes?: Record<string, { from: unknown; to: unknown }>;
    approval_note?: string;
  }[];
  related_records: {
    payroll_entries?: {
      payroll_id: number;
      batch_code: string;
      amount_vnd: number;
      payment_status: string;
    }[];
    financial_entries?: {
      ledger_id: number;
      account: string;
      debit?: number;
      credit?: number;
      date: string;
    }[];
  };
  compliance_notes: string[];
}

export interface IntegrityCheck {
  integrity_check_id: string;
  check_type: string;
  period_checked: {
    start_date: string;
    end_date: string;
  };
  summary: {
    total_logs_checked: number;
    checksum_failures: number;
    sequence_gaps: number;
    timestamp_anomalies: number;
    integrity_score: number;
  };
  verification_results: {
    cryptographic_hashes: 'valid' | 'invalid';
    sequence_continuity: 'valid' | 'invalid';
    timestamp_consistency: 'valid' | 'invalid';
    user_session_validity: 'valid' | 'invalid';
  };
  anomalies: unknown[];
  recommendations: string[];
  compliance_status: 'fully_compliant' | 'warnings' | 'non_compliant';
  checked_at: string;
}

// Enums and constants
export type AuditAction =
  | 'CREATE' | 'UPDATE' | 'DELETE'
  | 'APPROVE' | 'REJECT'
  | 'BULK_APPROVE' | 'BULK_REJECT' | 'BULK_RESET' | 'BULK_CREATE'
  | 'LOGIN' | 'LOGOUT' | 'CHANGE_PASSWORD'
  | 'VIEW' | 'EXPORT' | 'IMPORT' | 'SETTLE'
  | 'FAILED_LOGIN' | 'SUSPICIOUS_ACTIVITY'
  | 'TOKEN_REVOKED' | 'PERMISSION_DENIED'
  | 'DATA_EXPORT' | 'BULK_OPERATION';

export type EntityType =
  | 'user' | 'project' | 'employee'
  | 'project_employee' | 'project_user' | 'employee_user'
  | 'bank' | 'payrate'
  | 'timesheet' | 'payroll' | 'ledger_entry'
  | 'settings' | 'asset' | 'transaction' | 'loan' | 'lender'
  | 'financial' | 'import_batch' | 'export_batch';

export type SecurityIncidentType =
  | 'MULTIPLE_FAILED_LOGINS' | 'SUSPICIOUS_LOCATION'
  | 'UNUSUAL_ACCESS_PATTERN' | 'DATA_BREACH_ATTEMPT'
  | 'UNAUTHORIZED_ACCESS' | 'PRIVILEGE_ESCALATION';

export type TokenBlacklistReason =
  | 'logout' | 'security' | 'expired' | 'admin_revoke';

export type ComplianceReportType =
  | 'financial' | 'employment' | 'data_protection' | 'full';

export type ComplianceStatus =
  | 'compliant' | 'non_compliant' | 'partial' | 'pending';

// Actual backend AuditLog shape (matches domain.AuditLog + user_fullname join)
export interface BackendAuditLog {
  id: number;
  user_id: number;
  user_fullname: string;
  user_username: string;
  user_role: string;
  action: string;
  entity_type: string;
  entity_id: number | null;
  message: string;
  ip_address: string | null;
  browser: string | null;
  platform: string | null;
  metadata: string | null; // raw JSON string
  created_at: string;
}

export interface BackendAuditLogsResponse {
  status: string;
  message: string;
  data: BackendAuditLog[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface BackendAuditLogDetailResponse {
  status: string;
  message: string;
  data: BackendAuditLog;
}

export interface AuditLogsQueryParams {
  page?: number;
  pageSize?: number;
  userId?: number;
  action?: string[];
  entityType?: string[];
  fromDate?: string;
  toDate?: string;
  ipAddress?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface AuditSummaryParams {
  fromDate?: string;
  toDate?: string;
  user_id?: number;
}

export interface AuditLogsParams {
  page?: number;
  pageSize?: number;
  user_id?: number;
  action?: AuditAction;
  entity_type?: EntityType;
  entity_id?: number;
  fromDate?: string;
  toDate?: string;
  ip_address?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface UserActivityParams {
  page?: number;
  pageSize?: number;
  action?: AuditAction;
  fromDate?: string;
  toDate?: string;
}

export interface AuditExportParams {
  format?: 'excel' | 'csv';
  fromDate?: string;
  toDate?: string;
  user_id?: number;
  action?: AuditAction;
  entity_type?: EntityType;
}

export interface BlacklistedTokensParams {
  page?: number;
  pageSize?: number;
  user_id?: number;
  reason?: TokenBlacklistReason;
}

export interface SecurityReportParams {
  fromDate?: string;
  toDate?: string;
}

export interface LoginHistoryParams {
  page?: number;
  pageSize?: number;
  user_id?: number;
  ip_address?: string;
  success_only?: boolean;
}

export interface RevokeTokensRequest {
  reason: TokenBlacklistReason;
  note?: string;
}

export interface TrackEventRequest {
  eventType: string;
  description: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  metadata?: Record<string, unknown>;
  entityType?: EntityType;
  entityId?: number;
}

export interface ComplianceReportRequest {
  report_type: ComplianceReportType;
  period_start: string;
  period_end: string;
  include_personal_data?: boolean;
  format?: 'pdf' | 'excel' | 'csv';
}

export interface DataRetentionCleanupRequest {
  cleanup_type: string;
  retention_days: number;
  dry_run: boolean;
  categories: string[];
}

export interface IntegrityCheckRequest {
  check_type: 'full' | 'partial' | 'quick';
  date_range: {
    start: string;
    end: string;
  };
  verify_checksums: boolean;
  check_sequence: boolean;
}

// Response wrapper types
export interface AuditSummaryResponse {
  data: AuditSummary;
}

export interface AuditLogsResponse {
  data: {
    logs: AuditLogEntry[];
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
  };
}

export interface AuditLogDetailsResponse {
  data: AuditLogEntry;
}

export interface UserActivityResponse {
  data: UserActivity & {
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
  };
}

export interface AuditExportResponse {
  data: {
    downloadUrl: string;
    filename: string;
    fileSize: number;
    totalRecords: number;
    expiresAt: string;
    createdAt: string;
  };
}

export interface BlacklistedTokensResponse {
  data: {
    tokens: BlacklistedToken[];
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
    summary: {
      totalBlacklisted: number;
      expiredTokens: number;
      activelyBlacklisted: number;
      reasonBreakdown: {
        reason: string;
        count: number;
      }[];
    };
  };
}

export interface RevokeTokensResponse {
  data: {
    userId: number;
    tokensRevoked: number;
    reason: string;
    revokedAt: string;
  };
}

export interface SecurityReportResponse {
  data: SecurityReport;
}

export interface LoginHistoryResponse {
  data: {
    loginHistory: LoginHistoryEntry[];
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
  };
}

export interface TrackEventResponse {
  data: {
    eventId: string;
    eventType: string;
    tracked: boolean;
    timestamp: string;
  };
}

export interface ComplianceReportResponse {
  data: ComplianceReport;
}

export interface DataLineageResponse {
  data: DataLineage;
}

export interface IntegrityCheckResponse {
  data: IntegrityCheck;
}

// Vietnamese translations
export const VIETNAMESE_AUDIT_LABELS = {
  actions: {
    CREATE: 'Tạo mới',
    UPDATE: 'Cập nhật',
    DELETE: 'Xóa',
    APPROVE: 'Phê duyệt',
    REJECT: 'Từ chối',
    BULK_APPROVE: 'Duyệt hàng loạt',
    BULK_REJECT: 'Từ chối hàng loạt',
    BULK_RESET: 'Reset hàng loạt',
    BULK_CREATE: 'Tạo hàng loạt',
    LOGIN: 'Đăng nhập',
    LOGOUT: 'Đăng xuất',
    CHANGE_PASSWORD: 'Đổi mật khẩu',
    VIEW: 'Xem',
    EXPORT: 'Xuất dữ liệu',
    IMPORT: 'Nhập dữ liệu',
    SETTLE: 'Thanh toán',
    FAILED_LOGIN: 'Đăng nhập thất bại',
    SUSPICIOUS_ACTIVITY: 'Hoạt động đáng nghi',
    TOKEN_REVOKED: 'Thu hồi token',
    PERMISSION_DENIED: 'Từ chối quyền truy cập',
    DATA_EXPORT: 'Xuất dữ liệu',
    BULK_OPERATION: 'Thao tác hàng loạt',
  },
  entities: {
    user: 'Người dùng',
    project: 'Dự án',
    employee: 'Nhân viên',
    project_employee: 'Phân công nhân viên',
    project_user: 'Chia sẻ dự án',
    employee_user: 'Chia sẻ nhân viên',
    bank: 'Ngân hàng',
    payrate: 'Mức lương',
    timesheet: 'Bảng chấm công',
    payroll: 'Bảng lương',
    ledger_entry: 'Bút toán',
    settings: 'Cài đặt',
    asset: 'Tệp tin',
    transaction: 'Giao dịch',
    loan: 'Khoản vay',
    lender: 'Bên cho vay',
    financial: 'Tài chính',
    import_batch: 'Lô nhập liệu',
    export_batch: 'Lô xuất liệu',
  }
} as const;
