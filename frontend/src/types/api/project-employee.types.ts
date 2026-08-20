// Project-Employee Assignment API Types

// Core assignment types - Updated to match actual API response
export interface ProjectEmployeeAssignment {
  id: number;
  project_id: number;
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  employee_code: string;
  position: string;
  start_date: string;
  last_date?: string | null;
  status: AssignmentStatus;
  check_in_enabled?: boolean;
  // Deferred check-in activation: enable is pending until day 1 of next month.
  pending_check_in_enabled?: boolean | null;
  check_in_effective_from?: string | null;
  payment_schedule?: "weekly" | "monthly" | "flexible";
  pending_payment_schedule?: "weekly" | "monthly" | "flexible" | null;
  schedule_effective_from?: string | null;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface EmployeeInfo {
  id: number;
  fullname: string;
  cccd: string;
  email?: string;
  mobile?: string;
  date_of_birth?: string;
  status: "active" | "inactive";
}

export interface ProjectInfo {
  id: number;
  name: string;
  code: string;
  client_name?: string;
  status: "draft" | "active" | "completed" | "cancelled";
}

export interface TimesheetSummary {
  assignment_id: number;
  employee_id: number;
  project_id: number;
  period: {
    from_date: string;
    to_date: string;
  };
  total_hours: {
    normal: number;
    overtime: number;
    weekend: number;
    holiday: number;
  };
  total_amount_vnd: number;
  working_days: number;
  timesheet_entries: number;
}

// Assignment status enum
export type AssignmentStatus = "active" | "inactive" | "current" | "upcoming";

// Request types
export interface AssignEmployeeRequest {
  employee_id: number;
  employee_code?: string;
  position?: string;
  start_date?: string;
  payment_schedule?: "weekly" | "monthly" | "flexible";
}

export interface UpdateAssignmentRequest {
  employee_code?: string;
  end_date?: string;
}

export interface RemoveEmployeeRequest {
  employee_id: number;
  last_date?: string;
}

export interface EndAssignmentRequest {
  end_date: string;
  reason?: string;
}

export interface EndAssignmentResponse {
  data: ProjectEmployeeAssignment;
  message: string;
}

export interface ProjectEmployeeListParams {
  status?: AssignmentStatus;
  start_date?: string;
  end_date?: string;
  check_in_enabled?: boolean;
  search?: string;
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
}

export interface EmployeeProjectListParams {
  status?: AssignmentStatus;
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
}

export interface TimesheetSummaryParams {
  fromDate?: string;
  toDate?: string;
}

// Response wrapper types
export interface ProjectEmployeeListResponse {
  status: "success";
  data: ProjectEmployeeAssignment[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface EmployeeProjectListResponse {
  status: "success";
  data: ProjectEmployeeAssignment[];
  message: string;
}

export interface AssignEmployeeResponse {
  status: "success";
  data: ProjectEmployeeAssignment[];
  message: string;
}

export interface UpdateAssignmentResponse {
  status: "success";
  data: {
    id: number;
    project_id: number;
    employee_id: number;
    employee_code?: string;
    start_date: string;
    end_date?: string;
    status: AssignmentStatus;
    updated_at: string;
  };
  message: string;
}

export interface RemoveEmployeeResponse {
  status: "success";
  data: ProjectEmployeeAssignment[];
  message: string;
}

export interface TimesheetSummaryResponse {
  data: TimesheetSummary;
  message: string;
}

export interface DeleteAssignmentResponse {
  data: null;
  message: string;
}

// Error types specific to assignments
export interface AssignmentError {
  code:
    | "DUPLICATE_ASSIGNMENT"
    | "EMPLOYEE_INACTIVE"
    | "PROJECT_INACTIVE"
    | "INVALID_DATE_RANGE"
    | "OVERLAPPING_ASSIGNMENT"
    | "CANNOT_DELETE_WITH_TIMESHEETS";
  message: string;
  details?: {
    timesheet_count?: number;
    suggestion?: string;
    existing_assignment_id?: number;
    conflicting_dates?: {
      start: string;
      end: string;
    };
  };
}

// Assignment statistics
export interface AssignmentStats {
  total_assignments: number;
  active_assignments: number;
  ended_assignments: number;
  employees_assigned: number;
  projects_with_assignments: number;
  average_assignment_duration: number; // in days
}

// Assignment history tracking
export interface AssignmentHistoryEntry {
  id: number;
  assignment_id: number;
  action: "created" | "updated" | "ended" | "deleted";
  field_changed?: string;
  old_value?: unknown;
  new_value?: unknown;
  changed_by: number;
  changed_by_name: string;
  changed_at: string;
  reason?: string;
}

// Advanced filtering options
export interface AdvancedAssignmentFilters extends ProjectEmployeeListParams {
  employee_name?: string;
  factory_employee_code?: string;
  assigned_by?: number;
  date_range?: {
    start: string;
    end: string;
  };
  has_timesheets?: boolean;
  min_assignment_duration?: number; // in days
  max_assignment_duration?: number; // in days
}

// Assignment validation rules
export interface AssignmentValidationRules {
  allow_overlapping_assignments: boolean;
  require_factory_code: boolean;
  max_assignment_duration_days?: number;
  min_assignment_duration_days?: number;
  require_end_date: boolean;
  allow_backdated_assignments: boolean;
  max_future_assignment_days?: number;
}

// Payment schedule management types
export interface ChangePaymentScheduleRequest {
  new_schedule: "weekly" | "monthly" | "flexible";
  effective_date?: string;
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

// Vietnamese translations
export const VIETNAMESE_ASSIGNMENT_LABELS = {
  statuses: {
    active: "Đang dùng",
    ended: "Đã kết thúc",
    inactive: "Không hoạt động",
  },
  actions: {
    assign: "Giao nhiệm vụ",
    update: "Cập nhật",
    end: "Kết thúc",
    remove: "Gỡ bỏ",
  },
  fields: {
    employee_code: "Mã nhân viên",
    start_date: "Ngày bắt đầu",
    end_date: "Ngày kết thúc",
    assigned_by: "Được giao bởi",
    reason: "Lý do",
  },
  payment_schedule: {
    weekly: "lương tuần",
    monthly: "lương tháng",
    flexible: "linh động",
  },
} as const;
