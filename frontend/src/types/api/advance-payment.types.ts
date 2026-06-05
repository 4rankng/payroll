// Advance Payment domain types derived from backend API specs

// ---------------------------------------------------------------------------
// Status Types
// ---------------------------------------------------------------------------

export type AdvancePaymentStatus =
  | "PENDING"
  | "APPROVED"
  | "COMPLETED"
  | "FAILED"
  | "CANCELLED";

export type AdvancePaymentRequestStatus =
  | "pending"
  | "approved"
  | "completed"
  | "failed"
  | "cancelled";

export type RequestMethod = "app" | "admin_batch";

// ---------------------------------------------------------------------------
// Employee Types
// ---------------------------------------------------------------------------

/**
 * Employee's advance payment info for current month
 */
export interface AdvancePaymentQuota {
  forMonth: string;
  maxAdvanceAmount: number;
  completedAmount: number;
  pendingAmount: number;
  remainingAmount: number;
}

export interface AdvancePaymentInfo {
  forMonth: string; // YYYY-MM format
  maxAdvanceAmount: number;
  completedAmount: number;
  pendingAmount: number;
  remainingAmount: number;
  canRequest: boolean;
  canRequestTitle?: string;
  canRequestReason?: string;
  feePercentage: number; // e.g., 2 for 2%
  minFee: number; // e.g., 10000 VND
  hasFlexible: boolean;
  quotas: AdvancePaymentQuota[];
  /** Minimum single-transfer amount enforced by the disbursement provider (VND). 0 = no limit. */
  providerMinTransferAmount?: number;
  /** Maximum single-transfer amount enforced by the disbursement provider (VND). 0 = no limit. */
  providerMaxTransferAmount?: number;
}

/**
 * Request to create advance payment
 */
export interface CreateAdvancePaymentRequest {
  amount: number; // Minimum: 10,000 VND
  forMonth: string;
}

/**
 * Fee calculation request
 */
export interface CalculateFeeRequest {
  amount: number;
}

/**
 * Fee calculation response
 */
export interface CalculateFeeResponse {
  fee: number;
  netAmount: number;
}

/**
 * Employee in advance payment history
 */
export interface AdvancePaymentEmployee {
  id: number;
  fullname: string;
}

/**
 * History item for employee
 */
export interface AdvancePaymentHistoryItem {
  id: number;
  requestAmount: number;
  fee: number;
  netAmount: number;
  status: AdvancePaymentStatus;
  requestMethod?: RequestMethod;
  forMonth?: string;
  createdAt: string;
  completedAt?: string;
  employee?: AdvancePaymentEmployee;
  projectName?: string;
  projectCode?: string;
}

/**
 * History response with pagination
 */
export interface AdvancePaymentHistoryResponse {
  data: AdvancePaymentHistoryItem[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

// ---------------------------------------------------------------------------
// Admin Types
// ---------------------------------------------------------------------------

/**
 * Employee info in admin list item
 */
export interface AdvancePaymentEmployeeInfo {
  id: number;
  fullname: string;
  cccd: string;
}

/**
 * Project info in admin list item
 */
export interface AdvancePaymentProjectInfo {
  id: number;
  name: string;
  code?: string;
}

/**
 * Admin list item for advance payments (flat structure from API)
 */
export interface AdvancePaymentListItem {
  id: number;
  employeeId: number;
  employeeName: string;
  employeeCCCD: string;
  projectId: number;
  projectCode: string;
  projectName: string;
  requestAmount: number;
  fee: number;
  netAmount: number;
  status: AdvancePaymentStatus;
  createdAt: string;
  completedAt?: string; // alias for paidAt, kept for backward compat
  paidAt?: string;
  paymentReference?: string;
}

/**
 * Admin list filters
 */
export interface AdvancePaymentFilters {
  page?: number;
  pageSize?: number;
  status?: AdvancePaymentRequestStatus;
  projectId?: number;
  employeeId?: number;
  fromDate?: string;
  toDate?: string;
  forMonth?: string;
  search?: string;
  [key: string]: unknown;
}

/**
 * Admin summary statistics
 */
export interface AdvancePaymentSummary {
  totalRequests: number;
  totalPending: number;
  totalApproved: number;
  totalCancelled: number;
  totalFailed: number;
  totalPaid: number;
  totalAmount: number;
  totalPaidAmount: number;
  totalPendingAmount: number;
  totalFailedAmount: number;
  totalCancelledAmount: number;
  totalFee: number;
  totalFeeEarned: number;
  totalFeeEarnedAllTime: number;
  totalNet: number;
  totalProviderFee: number;
  totalProviderFeeAllTime: number;
  avgProcessingTimeSecs: number;
  completedUnder30s: number;
  feePercentage: number;
  avgFeePerRequest: number;
  avgFeePerEmployee: number;
  successRate: number;
  disbursementPercentage: number;
  fromDate: string;
  toDate: string;
}

// ---------------------------------------------------------------------------
// Flex Pay Employee List Types
// ---------------------------------------------------------------------------

/**
 * Available month for flex pay data
 */
export interface AvailableMonth {
  forMonth: string;
  employeeCount: number;
}

/**
 * Bank details for flex pay employee
 */
export interface FlexPayEmployeeBank {
  bankId: number | null;
  bankName: string | null;
  accountNumber: string;
  accountName: string;
}

/**
 * Project info for flex pay employee
 */
export interface FlexPayEmployeeProject {
  id: number;
  name: string;
  code: string;
  assignment_id?: number;
  check_in_enabled?: boolean;
}

/**
 * Flex pay employee list item
 */
export interface FlexPayEmployeeListItem {
  employeeId: number;
  fullname: string;
  email?: string;
  cccd: string;
  username: string;
  bank: FlexPayEmployeeBank | null;
  project: FlexPayEmployeeProject;
  forMonth: string;
  maxAdvanceAmount: number;
  utilizedAmount: number;
  availableAmount: number;
  totalFeeGenerated: number;
  pendingAmount: number;
  completedRequestsCount: number;
  pendingRequestsCount: number;
  createdAt?: string;
}

/**
 * Flex pay employee list filters
 */
export interface FlexPayEmployeeFilters {
  forMonth?: string;
  page?: number;
  pageSize?: number;
  search?: string;
  sortBy?: string;
  sortOrder?: "ASC" | "DESC";
  [key: string]: unknown;
}

/**
 * Flex pay employee list response meta
 */
export interface FlexPayEmployeeListMeta {
  forMonth: string;
  totalRecords: number;
  totalPages: number;
  currentPage: number;
  pageSize: number;
}

/**
 * Uploaded file history item (matches backend UploadedFileResponse)
 */
export interface AdvancePaymentFileHistoryItem {
  id: number;
  filename: string;
  uploadType: "flex_pay_import" | "advance_payment_result" | "advance_payment_export" | "advance_payment_sao_ke_export" | "advance_payment_sao_ke_result";
  createdAt: string;
  uploadedBy: string;
}

/**
 * Import flex template response
 */
export interface ImportFlexTemplateResponse {
  forMonth: string;
  totalRows: number;
  employeesCreated: number;
  employeesSkipped: number;
  projectsCreated: number;
  projectsSkipped: number;
  assignmentsCreated: number;
  assignmentsSkipped: number;
}

/**
 * Individual advance payment result item from bank transfer
 */
export interface AdvancePaymentResultItem {
  row: number;
  employee_bank: string;
  employee_account_number: string;
  employee_name: string;
  employee_cccd: string;
  amount: string;
  payment_status: 'paid' | 'failed';
  paid_at?: string;
}

/**
 * Upload bank result response
 */
export interface UploadBankResultResponse {
  items: AdvancePaymentResultItem[];
  total_txn: number;
  completed_txn: number;
  failed_txn: number;
}

// ---------------------------------------------------------------------------
// Async Import Job Types
// ---------------------------------------------------------------------------

/**
 * Import job status type
 */
export type ImportJobStatus = "pending" | "processing" | "completed" | "failed";

/**
 * Response when starting an import job
 */
export interface ImportJobResponse {
  id: number;
  status: ImportJobStatus;
  message: string;
  createdAt: string;
}

/**
 * Import job status response
 */
export interface ImportJobStatusResponse {
  id: number;
  status: ImportJobStatus;
  forMonth: string;
  filename: string;
  totalRows: number;
  processedRows: number;
  percentage: number;
  result?: ImportFlexTemplateResponse;
  error?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
}

/**
 * Polling options for import job status
 */
export interface ImportPollingOptions {
  maxWaitTime?: number; // Default: 300000 (5 minutes)
  pollInterval?: number; // Default: 3000 (3 seconds)
  onProgress?: (
    percentage: number,
    processedRows: number,
    totalRows: number,
  ) => void;
  onStatusChange?: (status: ImportJobStatus) => void;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const ADVANCE_PAYMENT_CONSTANTS = {
  MIN_AMOUNT: 10000, // 10,000 VND
  FEE_PERCENTAGE: 2, // 2%
  MIN_FEE: 10000, // 10,000 VND
  MAX_ADVANCE_PERCENTAGE: 60, // 60%
} as const;

// ---------------------------------------------------------------------------
// Status Labels (Vietnamese)
// ---------------------------------------------------------------------------

export const ADVANCE_PAYMENT_STATUS_LABELS: Record<
  AdvancePaymentStatus,
  string
> = {
  PENDING: "Chờ xử lý",
  APPROVED: "Đã duyệt",
  COMPLETED: "Hoàn tất",
  FAILED: "Thất bại",
  CANCELLED: "Đã hủy",
};

export const ADVANCE_PAYMENT_REQUEST_STATUS_LABELS: Record<
  AdvancePaymentRequestStatus,
  string
> = {
  pending: "Chờ xử lý",
  approved: "Đã duyệt",
  completed: "Hoàn tất",
  failed: "Thất bại",
  cancelled: "Đã hủy",
};

export const REQUEST_METHOD_LABELS: Record<RequestMethod, string> = {
  app: "Ứng dụng",
  admin_batch: "Nhập từ admin",
};

// ---------------------------------------------------------------------------
// Flexible Employee List Import Types
// ---------------------------------------------------------------------------

/**
 * Result of importing flexible employee list
 */
export interface ImportFlexibleEmployeeListResult {
  total_rows: number;
  employees_created: number;
  employees_skipped: number;
  projects_created: number;
  projects_skipped: number;
  assignments_created: number;
  assignments_skipped: number;
  errors?: ImportRowError[];
}

/**
 * Individual row error during import
 */
export interface ImportRowError {
  row: number;
  cccd: string;
  error: string;
}
