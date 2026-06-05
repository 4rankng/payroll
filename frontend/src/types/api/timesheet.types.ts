// Timesheet related types for API integration
// Updated to support 3-level payrate structure

import type { DayType } from './payrate.types';

export interface Timesheet {
  id: number;
  project_id: number;
  employee_id: number;
  date: string;
  hours_worked: number;
  paytype: string;
  hour_type: string;
  day_type: string;
  payrate_id: number;
  payrate: number;
  amount: number;
  status: 'approved' | 'pending_approval' | 'rejected' | 'draft';
  payment_status?: 'pending' | 'paid' | 'failed' | 'cancelled';
  allowed_edit?: boolean;
  request_edit_id?: number | null;
  request_edit_status?: 'pending' | 'approved' | 'rejected';
  request_edit_requested_at?: string;
  request_edit_requested_by_id?: number;
  request_edit_requested_by_name?: string;
  paid_amount?: number;
  paid_at?: string | null;
  force_payroll?: boolean;
  created_by: number;
  approved_by?: number | null;
  approved_at?: string | null;
  rejection_reason?: string;
  created_at: string;
  updated_at: string;
  projectName: string;
  employeeName: string;
  employeeCode: string;
  notes?: string;
  employeeCCCD?: string;
  employeeEmail?: string;
  projectCode?: string;
  period?: string;
  weekRange?: string;
  totalHours?: number;
  regularHours?: number;
  overtimeHours?: number;
  totalAmount?: number;
  submittedAt?: string;
  workDays?: number;
  overtime?: number;
}

export interface TimesheetSummary {
  totalEntries: number;
  totalEmployees?: number;
  pendingApproval: number;
  pendingEmployees?: number;
  pendingPaymentAmount: number;
  approvedEntries: number;
  paidEntries: number;
  paidAmount?: number;
  paidEmployees?: number;
  rejectedEntries: number;
  lastUpdated?: string;
}

export interface CreateTimesheetData {
  projectId: number;
  employeeId: number;
  date: string;
  hoursWorked: number;
  hourType: string;
  dayType?: 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ'; // Default: Ngày thường
}

export interface UpdateTimesheetData {
  hours_worked?: number;
  paytype?: string;
}

export interface TimesheetFilters {
  page?: number;
  pageSize?: number;
  project_ids?: string; // Multiple project IDs (comma-separated: "1,2,3")
  employee_id?: number; // snake_case to match backend query param
  fromDate?: string;
  toDate?: string;
  status?: 'pending_approval' | 'approved' | 'rejected' | 'paid' | 'failed' | 'cancelled' | ('pending_approval' | 'approved' | 'rejected' | 'paid' | 'failed' | 'cancelled')[];
  payment_status?: string; // Backend: filter by payment_status (pending, paid, failed, cancelled)
  allowed_edit?: boolean | 0 | 1;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  search?: string;
  created_by?: number;
  has_request_edit?: boolean | 0 | 1;
  [key: string]: unknown;
}

export interface BulkApproveData {
  timesheet_ids: number[];
  approvalNote?: string;
}

export interface BulkApproveResult {
  approved: number;
  skipped: number;
  failed: number;
  results: Array<{
    id: number;
    status: string;
    error: string;
  }>;
}

export interface BulkRejectData {
  timesheet_ids: number[];
  rejection_reason: string;
  notify_partners?: boolean;
}

export interface BulkRejectResult {
  rejected_count: number;
  failed: number;
  results: Array<{
    id: number;
    status: string;
    rejection_reason?: string;
    error?: string;
  }>;
  notifications_sent?: number;
}

export interface BulkResetData {
  timesheet_ids: number[];
}

export interface BulkResetResult {
  approved: number;
  failed: number;
  results: Array<{
    id: number;
    status: string;
    error?: string;
  }>;
}

export interface RejectTimesheetData {
  rejection_reason: string;
}

export interface ImportTimesheetResult {
  batch_id: string;
  total_records: number;
  successful_records: number;
  failed_records: number;
  status: 'processing' | 'completed' | 'failed';
  requires_approval: boolean;
  error_summary?: string;
  imported_at?: string;
  errors?: Array<{
    row: number;
    field: string;
    message: string;
  }>;
}

export interface ExportTimesheetResult {
  download_url: string;
  filename: string;
  file_size: number;
  total_records: number;
  expires_at: string;
}

export interface SettlementUploadResultResponse {
  status: string;
  message?: string;
  total_records?: number;
  total_settled?: number;
  failed_records?: number;
}

// For timesheet validation and rate preview
export interface TimesheetValidationRequest {
  employee_id: number;
  project_id: number;
  date: string;
  position: string;
  hour_type: string;
  hours_worked: number;
  exclude_id?: number; // For edit mode - exclude current record from validation
}

export interface TimesheetValidationResponse {
  valid: boolean;
  errors: string[];
  warnings?: string[];
  calculated_rate?: number;
  calculated_amount?: number;
  calculated_day_type?: DayType;
  duplicate_entry?: boolean;
}


// New simplified API format - array of individual timesheet entries
// UPSERT behavior: If same date and hourType exists, hours will be updated instead of creating new entry
export interface NewTimesheetEntry {
  projectId: number;
  employeeId: number;
  date: string; // YYYY-MM-DD format
  hoursWorked: number;
  hourType: string; // e.g., "ca ngày", "ca đêm", "tăng ca"
  dayType?: 'Ngày thường' | 'Ngày nghỉ' | 'Ngày lễ'; // Default: Ngày thường
}

export interface BulkCreateTimesheetResult {
  status: string;
  message: string;
  data: {
    created: Timesheet[];
    failed: Array<{
      employeeId: number;
      error: string;
      details?: string;
    }> | null;
    total_created: number;
    total_failed: number;
    total_success?: number;
  };
}


// Saturday day type selection for bulk entries
export interface SaturdayDayTypeSelection {
  date: string;
  day_type: 'ngày thường' | 'ngày nghỉ';
  confirmed: boolean;
}

// Employee timesheet summary for detail view
export interface EmployeeTimesheetSummary {
  employeeId: number;
  employeeName: string;
  employeeCode: string;
  employeeEmail?: string;
  totalHours: Record<string, number>; // hour type breakdown like "phổ thông.ngày nghỉ.ca ngày": 2
  totalAmount: number;
  averageHoursPerDay: number;
  workingDays: number;
  lastEntryDate: string;
  pendingEntries: number;
  paymentStatus?: 'pending' | 'processing' | 'paid' | 'overdue';
  approvalStats?: {
    approved: number;
    pending: number;
    rejected: number;
  };
}

// Bulk approval by project/employee data
export interface BulkApproveByProjectData {
  approvalNote?: string;
}

export interface BulkApproveByEmployeeData {
  timesheet_ids: number[];
  approvalNote?: string;
}

// Employee timesheet history API response
export interface EmployeeTimesheetEntry {
  id: number;
  project_id: number;
  employee_id: number;
  date: string;
  hours_worked: number;
  paytype: string;
  hour_type: string;
  day_type: string;
  payrate_id: number;
  payrate: number;
  amount: number;
  status: 'approved' | 'pending_approval' | 'rejected' | 'draft';
  payment_status?: 'pending' | 'paid' | 'failed' | 'cancelled';
  paid_amount?: number;
  paid_at?: string | null;
  created_by: number;
  approved_by: number | null;
  approved_at: string | null;
  rejection_reason: string;
  created_at: string;
  updated_at: string;
  projectName: string;
  employeeName: string;
  employeeCode: string;
  employeeEmail?: string;
}


// API Response Wrapper Types (matching actual API response structure)
export interface ApiResponse<T> {
  status: string;
  data: T;
  message: string;
}

export interface ApiListResponse<T> {
  status: string;
  data: T[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

// Specific response types for timesheet endpoints
export interface TimesheetListResponse extends ApiListResponse<Timesheet> {}
export interface TimesheetSummaryResponse extends ApiResponse<TimesheetSummary> {}
export interface TimesheetResponse extends ApiResponse<Timesheet> {}
export interface PendingApprovalsResponse extends ApiResponse<Timesheet[]> {}
export interface BulkApproveResponse extends ApiResponse<BulkApproveResult> {}
export interface BulkRejectResponse extends ApiResponse<BulkRejectResult> {}
export interface EmployeeTimesheetSummaryResponse extends ApiResponse<EmployeeTimesheetSummary> {}

// Type alias for employee-specific timesheet filters
export type EmployeeTimesheetFilters = TimesheetFilters;

// Timesheet Edit Request Types
export interface TimesheetEditRequest {
  id: number;
  timesheet_id: number;
  status: 'pending' | 'approved' | 'rejected';
  requested_by: number;
  requested_by_name: string;
  approved_by?: number | null;
  approved_by_name?: string | null;
  rejected_by?: number | null;
  rejected_by_name?: string | null;
  created_at: string;
  updated_at: string;
  timesheet?: {
    id: number;
    date: string;
    end_date?: string;
    employee_id: number;
    employee_name: string;
    employee_code?: string;
    employee_cccd?: string;
    project_id: number;
    project_name: string;
    project_code?: string;
    hours_worked: number;
    overtime_hours?: number;
    amount: number;
    paytype?: string;
    status: string;
    payment_status: string;
  };
}

export interface TimesheetEditRequestFilters {
  page?: number;
  pageSize?: number;
  status?: 'pending' | 'approved' | 'rejected';
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface TimesheetEditRequestListResponse extends ApiListResponse<TimesheetEditRequest> {}
export interface TimesheetEditRequestResponse extends ApiResponse<TimesheetEditRequest> {}

// Upload Excel entries response
export interface UploadEntriesExcelResult {
  created_count: number;
  error_count: number;
  skipped_count: number;
  failed_entries: Array<{
    Index: number;
    Error: string;
    Request: {
      project_id: number;
      employee_id: number;
      date: string;
      hours_worked: number;
      hour_type: string;
      day_type: string;
    };
  }>;
}

export interface UploadEntriesExcelResponse extends ApiResponse<UploadEntriesExcelResult> {}

// Timesheet Entry Table API Types
export interface DailyEntry {
  date: string;
  [dayType: string]: string | number | Record<string, number>; // dayType -> hourType -> hours
}

export interface EmployeeEntryTable {
  employee_id: number;
  entries: Array<{
    date: string;
    [dayType: string]: Record<string, number> | string; // Dynamic keys for day types containing hour types
  }>;
}

export interface TimesheetEntryTableResponse extends ApiResponse<EmployeeEntryTable[]> {}

// Grouped Timesheet API Types (for partner view with server-side grouping)
export interface EmployeeGroupedTimesheet {
  employee_id: number;
  employee_name: string;
  employee_code: string;
  entries: TimesheetWithDetails[];
  total_hours: number;
  total_amount: number;
  aggregated_status: 'approved' | 'pending_approval' | 'mixed' | 'rejected' | 'draft';
}

export interface TimesheetWithDetails extends Timesheet {
  project_name: string;
  project_code?: string;
  employee_fullname: string;
  employee_code: string;
  employee_cccd?: string;
  created_user_fullname?: string;
  approved_user_fullname?: string;
  payroll_batch_code?: string;
}

export interface GroupedTimesheetPagination {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface ListGroupedTimesheetsResponse {
  status: string;
  data: {
    groups: EmployeeGroupedTimesheet[];
  };
  message: string;
  pagination?: GroupedTimesheetPagination;
}

// ─── BCC Partner Import ───────────────────────────────────────────────────

export interface PartnerImportFile {
  id: number;
  project_id: number;
  uploaded_by: number;
  original_name: string;
  for_month: string; // "2026-05"
  status: 'completed' | 'failed';
  total_rows: number;
  created_count: number;
  skipped_count: number;
  error_count: number;
  error_detail?: string | null; // JSON string of ImportError[]
  processed_at?: string | null;
  created_at: string;
}

export interface ImportError {
  row: number;
  employee: string;
  reason: string;
}

export interface PartnerImportListParams {
  project_id?: number;
  for_month?: string;
  page?: number;
  page_size?: number;
}

export type TimesheetStatus =
  | 'approved'
  | 'pending_approval'
  | 'rejected'
  | 'draft'
  | 'paid'
  | 'failed'
  | 'cancelled'
  | 'pending';

export type PaymentStatus = 'pending' | 'paid' | 'failed' | 'cancelled' | 'processing' | 'overdue';

// Re-export from employee.types to maintain backwards compatibility with existing imports
export type { EmployeeTimesheetResponse } from './employee.types';
