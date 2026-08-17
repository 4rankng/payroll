// Employee related types
import type { Bank } from "./bank.types";

export interface EmployeeProject {
  id: number;
  name: string;
  code: string;
  client_name: string;
}

export interface CurrentProject {
  project_id: number;
  project_employee_id?: number;
  name: string;
  code: string;
  client_name: string;
  position?: string;
  start_date?: string;
  last_date?: string | null;
  payment_schedule?: "weekly" | "monthly" | "flexible";
  pending_payment_schedule?: "weekly" | "monthly" | "flexible" | null;
  schedule_effective_from?: string | null;
  is_flexible?: boolean;
  check_in_enabled?: boolean;
}

export interface TimesheetSummary {
  total_timesheets: number;
  total_hours_worked: number;
  pending_timesheets: number;
  approved_timesheets: number;
  rejected_timesheets: number;
  current_week_hours: number;
  last_timesheet_date: string;
}

export interface PayrollSummary {
  total_payroll_payments: number;
  total_earnings_vnd: number;
  last_payment_date: string;
  avg_weekly_earnings_vnd: number;
}

export interface Employee {
  id: number;
  username?: string;
  employee_code?: string;
  fullname: string;
  email?: string;
  cccd: string;
  address?: string;
  mobile?: string;
  position?: string;
  status?: "active" | "inactive";
  bank?: Bank | null;
  bank_account_number?: string;
  bank_account_name?: string;
  /**
   * Outcome of the last OnePay account verification.
   * - `valid`     : account confirmed good (default for pre-existing rows)
   * - `invalid`   : OnePay confirmed account bad or holder-name mismatch
   * - `unverified`: OnePay unreachable / timed out (not shown in warning list)
   */
  bank_account_status?: "valid" | "invalid" | "unverified";
  /** Vietnamese human-readable reason, set only when status === "invalid". */
  bank_account_invalid_reason?: string | null;
  /** RFC3339 timestamp of the last OnePay validation. */
  bank_account_validated_at?: string | null;
  date_of_birth?: string;
  can_delete?: boolean;
  current_projects: CurrentProject[];
  timesheet_summary?: TimesheetSummary;
  payroll_summary?: PayrollSummary;
  created_by?: number;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown; // Add index signature for table compatibility
}

export interface EmployeeSummary {
  total_employees: number;
  total_working_employees: number;
  employees_hired_this_month: number;
  // Month-to-date totals from /employees/summary
  salary_month_to_date: number;
  paid_month_to_date: number;
}

export interface CreateEmployeeData {
  fullname: string;
  email?: string;
  cccd: string;
  address?: string;
  mobile?: string;
  bank_id?: number;
  bank_account_number?: string;
  bank_account_name?: string;
  date_of_birth?: string;
}

export type UpdateEmployeeData = Partial<CreateEmployeeData>;

export interface EmployeeFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  status?: "working" | "unassigned";
  search?: string;
  projectId?: number;
  projectIds?: string;
  include_inactive?: boolean;
  month?: string;
  fromDate?: string;
  toDate?: string;
  paymentSchedule?: "weekly" | "monthly" | "flexible";
}

export interface EmployeesResponse {
  status: "success";
  data: Employee[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

// Employee Payroll History Types
export interface EmployeePayrollRecord {
  id: number;
  timesheet_id: number;
  payroll_id: number | null;
  batch_code: string | null;
  project_id: number;
  project_name: string;
  date: string;
  hours_worked: number;
  paytype: string;
  payrate: number;
  amount: number;
  payment_status: "pending" | "paid" | "failed" | "cancelled";
  payment_reference: string | null;
  payment_date: string | null;
  period_start: string;
  period_end: string;
  created_at: string;
}

export interface EmployeePayrollFilters {
  page?: number;
  pageSize?: number;
  sort_by?: "created_at" | "date" | "amount" | "payment_status" | "pay_date";
  sort_order?: "asc" | "desc";
  status?: "pending" | "paid" | "failed" | "cancelled";
  project_id?: number;
  fromDate?: string;
  toDate?: string;
  [key: string]: unknown;
}

export interface EmployeePayrollResponse {
  status: "success";
  data: EmployeePayrollRecord[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

// Additional types based on API documentation

export interface EmployeeSummaryResponse {
  status: "success";
  data: EmployeeSummaryData;
  message: string;
}

export interface EmployeeSummaryData {
  total_payroll_payments: number;
  total_earnings_vnd: number;
  last_payment_date?: string;
  avg_weekly_earnings_vnd: number;
}

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
  status: "pending_approval" | "approved" | "rejected";
  payment_status: "pending" | "paid" | "failed" | "cancelled";
  force_payroll: boolean;
  paid_amount: number;
  paid_at: string | null;
  last_updated_by: number;
  last_approved_by: number;
  approved_at: string | null;
  rejection_reason: string | null;
  created_at: string;
  updated_at: string;
  projectName: string;
  employeeName: string;
  employeeCode: string;
  submittedBy: number;
  submittedByName: string;
  submittedAt: string;
}

export interface EmployeeTimesheetFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  project_id?: number;
  status?:
    | "draft"
    | "pending_approval"
    | "approved"
    | "rejected"
    | "pending"
    | "paid"
    | "failed"
    | "cancelled";
  fromDate?: string;
  toDate?: string;
  [key: string]: unknown;
}

export interface EmployeeTimesheetResponse {
  status: "success";
  data: EmployeeTimesheetEntry[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface EmployeeExportResponse {
  // Binary XLSX file response
  data: Blob;
}

export interface EmployeeListFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  search?: string;
  projectId?: number;
  status?: "working" | "unassigned";
  fromDate?: string;
  toDate?: string;
  include_inactive?: boolean;
}

// Employee Project Assignment Types
export interface EmployeeProjectAssignment {
  assignment_id: number;
  project_id: number;
  project_name: string;
  project_code: string;
  factory_employee_code: string;
  start_date: string;
  end_date?: string;
  status: "active" | "ended" | "inactive";
  total_hours_worked: number;
  total_earned_vnd: number;
}

export interface EmployeeProjectsResponse {
  data: EmployeeProjectAssignment[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

// Employee Individual Summary Types
export interface EmployeeIndividualSummary {
  total_payroll_payments: number;
  total_earnings_vnd: number;
  last_payment_date?: string;
  avg_weekly_earnings_vnd: number;
}

// Utility functions for employee projects
export const getEmployeeProjectCount = (employee: Employee): number => {
  return employee.current_projects?.length || 0;
};

export const getEmployeeProjects = (employee: Employee): CurrentProject[] => {
  return employee.current_projects || [];
};

export const isEmployeeAssignedToProject = (
  employee: Employee,
  projectId: number,
): boolean => {
  return (
    employee.current_projects?.some(
      (project) => project.project_id === projectId,
    ) || false
  );
};

export const hasCompleteBankDetails = (employee: Employee): boolean => {
  return !!(
    employee.bank &&
    employee.bank_account_number &&
    employee.bank_account_name
  );
};

// Employee Current Projects with Timesheets Types
export interface CurrentProjectTimesheet {
  id: number;
  date: string;
  hours_worked: number;
  paytype: string;
  hour_type: string;
  day_type: string;
  payrate: number;
  amount: number;
  status: "pending_approval" | "approved" | "rejected";
  payment_status?: "pending" | "paid" | "failed" | "cancelled";
  paid_amount?: number;
  paid_at?: string | null;
  approved_at: string | null;
  rejection_reason: string | null;
  created_at: string;
}

export interface CurrentProjectWithTimesheets {
  project_id: number;
  project_name: string;
  project_code: string;
  client_name: string;
  position: string;
  start_date: string;
  last_date: string | null;
  timesheets: CurrentProjectTimesheet[];
}

export interface EmployeeCurrentProjectsFilters {
  fromDate?: string;
  toDate?: string;
  [key: string]: unknown;
}

export interface EmployeeCurrentProjectsResponse {
  data: CurrentProjectWithTimesheets[];
}

// Employee User Access Types
export interface EmployeeUser {
  id: number;
  user_id: number;
  user_fullname: string;
  user_email: string;
  granted_by: number;
  granted_at: string;
}

export interface EmployeeUsersResponse {
  status: "success";
  data: EmployeeUser[];
  message: string;
}

export interface GrantEmployeeAccessData {
  user_id: number;
}

export interface GrantEmployeeAccessResponse {
  status: "success";
  message: string;
}

export interface RevokeEmployeeAccessResponse {
  status: "success";
  message: string;
}

// Employee Password Change Types
export interface ChangeEmployeePasswordRequest {
  new_password: string;
}

export interface ChangeEmployeePasswordResponse {
  status: "success";
  message: string;
}

// Payment Schedule Management Types
export interface ChangePaymentScheduleRequest {
  new_schedule: "weekly" | "monthly";
}

export interface ChangePaymentScheduleResponse {
  status: "success";
  data: {
    id: number;
    project_id: number;
    employee_id: number;
    payment_schedule: "weekly" | "monthly";
    pending_payment_schedule: "weekly" | "monthly" | null;
    schedule_effective_from: string | null;
    message: string;
  } | null;
  message: string;
}

export interface CancelScheduleChangeResponse {
  status: "success";
  data: {
    id: number;
    project_id: number;
    employee_id: number;
    payment_schedule: "weekly" | "monthly";
    pending_payment_schedule: null;
    schedule_effective_from: null;
  };
  message: string;
}

export interface PendingScheduleChange {
  id: number;
  project_id: number;
  project_name: string;
  employee_id: number;
  employee_name: string;
  current_schedule: "weekly" | "monthly";
  pending_schedule: "weekly" | "monthly";
  effective_from: string;
}

export interface PendingScheduleChangesResponse {
  status: "success";
  data: PendingScheduleChange[];
  message: string;
}

// Employee Import Types
export interface EmployeeImportStatus {
  import_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  total_rows: number;
  processed_rows: number;
  created_count: number;
  updated_count: number;
  error_count: number;
  errors: Array<{ row_number: number; cccd?: string; message: string }>;
  percentage: number;
  started_at: string;
  completed_at?: string;
}

export interface EmployeeImportResponse {
  import_id: string;
  status: string;
  message: string;
}

// Vietnamese translations
export const VIETNAMESE_EMPLOYEE_LABELS = {
  statuses: {
    active: "Đang dùng",
    inactive: "Không hoạt động",
    pending_approval: "Chờ duyệt",
    approved: "Đã duyệt",
    rejected: "Từ chối",
    pending: "Chờ thanh toán",
    paid: "Đã thanh toán",
    failed: "Thất bại",
    cancelled: "Đã hủy",
    working: "Đang làm việc",
    unassigned: "Chưa phân công",
  },
  actions: {
    create: "Tạo mới",
    update: "Cập nhật",
    delete: "Xóa",
    approve: "Duyệt",
    reject: "Từ chối",
    export: "Xuất Excel",
    change_password: "Đổi mật khẩu",
    assign_project: "Phân công dự án",
    grant_access: "Cấp quyền truy cập",
    revoke_access: "Thu hồi quyền truy cập",
    change_schedule: "Thay đổi chu kỳ",
    cancel_schedule_change: "Hủy thay đổi chu kỳ",
  },
  fields: {
    fullname: "Họ và tên",
    email: "Email",
    cccd: "CCCD",
    address: "Địa chỉ",
    mobile: "Số điện thoại",
    bank: "Ngân hàng",
    bank_account_number: "Số tài khoản",
    bank_account_name: "Tên tài khoản",
    date_of_birth: "Ngày sinh",
    position: "Vị trí",
    status: "Trạng thái",
    current_projects: "Dự án hiện tại",
    created_at: "Ngày tạo",
    updated_at: "Ngày cập nhật",
    project_name: "Tên dự án",
    project_code: "Mã dự án",
    client_name: "Tên khách hàng",
    start_date: "Ngày bắt đầu",
    last_date: "Ngày kết thúc",
    hours_worked: "Số giờ làm việc",
    paytype: "Loại lương",
    payrate: "Đơn giá",
    amount: "Thành tiền",
    date: "Ngày",
    hour_type: "Loại ca",
    day_type: "Loại ngày",
    payroll_id: "ID bảng lương",
    timesheet_id: "ID bảng công",
    batch_code: "Mã đợt thanh toán",
    payment_status: "Trạng thái thanh toán",
    payment_date: "Ngày thanh toán",
    payment_reference: "Mã tham chiếu",
    period_start: "Chu kỳ từ",
    period_end: "Chu kỳ đến",
    approved_at: "Ngày duyệt",
    rejection_reason: "Lý do từ chối",
    paid_amount: "Số tiền đã thanh toán",
    paid_at: "Ngày thanh toán",
    submitted_at: "Ngày nộp",
    employee_code: "Mã nhân viên",
    user_fullname: "Tên người dùng",
    user_email: "Email người dùng",
    granted_by: "Cấp bởi",
    granted_at: "Cấp lúc",
    schedule: "Chu kỳ thanh toán",
    current_schedule: "Chu kỳ hiện tại",
    pending_schedule: "Chu kỳ chờ áp dụng",
    effective_from: "Hiệu lực từ ngày",
    factory_employee_code: "Mã nhân viên nhà máy",
  },
  schedule: {
    weekly: "lương tuần",
    monthly: "lương tháng",
    flexible: "linh động",
  },
  messages: {
    employee_created: "Tạo nhân viên thành công",
    employee_updated: "Cập nhật nhân viên thành công",
    employee_deleted: "Xóa nhân viên thành công",
    password_changed: "Mật khẩu đã đổi thành công",
    access_granted: "Cấp quyền truy cập thành công",
    access_revoked: "Thu hồi quyền truy cập thành công",
    schedule_change_requested: "Yêu cầu thay đổi chu kỳ thanh toán thành công",
    schedule_change_cancelled: "Hủy thay đổi chu kỳ thanh toán thành công",
    export_success: "Xuất dữ liệu thành công",
    import_success: "Nhập dữ liệu thành công",
  },
} as const;
