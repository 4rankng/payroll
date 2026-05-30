// Project related types

export interface GeofenceGate {
  name: string;
  lat: number;
  lng: number;
}

export interface EmployeePosition {
  position: string;
  count: number;
}

export interface RecentEmployee {
  employee_id: number;
  fullname: string;
  position: string;
  start_date: string;
}

export interface EmployeeAssignments {
  total_employees: number;
  positions: EmployeePosition[];
  recent_employees: RecentEmployee[];
}

export interface Project {
  id: number;
  name: string;
  code: string;
  description?: string;
  client_name: string;
  start_date: string;
  end_date: string | null;
  employee_count: number;
  weekly_salary_employee_count: number;
  monthly_salary_employee_count: number;
  status: "draft" | "active" | "paused" | "completed" | "cancelled";
  is_weekly?: boolean;
  is_monthly?: boolean;
  salary_period_from?: number | null;
  salary_period_to?: number | null;
  off_days?: number; // bitmask: bit0=Sun, bit1=Mon, ..., bit6=Sat
  is_flexible?: boolean;
  geofence_gates?: GeofenceGate[] | null;
  geofence_radius_meters?: number;
  created_by: number;
  created_at: string;
  updated_at: string;
  startDate?: string;
  endDate?: string;
  employee_assignments?: EmployeeAssignments;
}

export interface ProjectSummary {
  total_active_projects: number;
  total_received_vnd: number;
  total_payout_vnd: number;
  total_pending_payable_vnd: number;
  total_pending_receivable_vnd: number;
}

export interface PartnerProjectSummary {
  total_projects: number;
  active_projects: number;
  completed_projects: number;
  total_employees: number;
}

export interface CreateProjectData {
  name: string;
  description?: string;
  client_name: string;
  code?: string;
  start_date: string;
  end_date: string;
  status?: "draft" | "active" | "paused" | "completed" | "cancelled";
  is_weekly?: boolean;
  is_monthly?: boolean;
  salary_period_from?: number | null;
  salary_period_to?: number | null;
  off_days?: number; // bitmask: bit0=Sun, bit1=Mon, ..., bit6=Sat
}

export interface UpdateProjectData {
  name?: string;
  code?: string;
  description?: string;
  client_name?: string;
  contact_person?: string;
  priority?: "high" | "medium" | "low";
  project_type?: string;
  start_date?: string;
  end_date?: string;
  status?: "draft" | "active" | "completed" | "cancelled";
  is_weekly?: boolean;
  is_monthly?: boolean;
  salary_period_from?: number | null;
  salary_period_to?: number | null;
  off_days?: number; // bitmask: bit0=Sun, bit1=Mon, ..., bit6=Sat
  geofence_gates?: GeofenceGate[];
  geofence_radius_meters?: number;
}

export interface UpdateProjectStatusData {
  status: "draft" | "active" | "paused" | "completed" | "cancelled";
}

export interface ProjectFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  status?:
    | "draft"
    | "active"
    | "paused"
    | "completed"
    | "cancelled"
    | Array<"draft" | "active" | "paused" | "completed" | "cancelled">;
  search?: string;
  month?: string; // Format: 'YYYY-MM'
  [key: string]: unknown;
}

export interface ProjectEmployee {
  id: number;
  project_id: number;
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  employee_code: string;
  position: string;
  start_date: string;
  last_date?: string | null;
  status: "active" | "inactive";
  payment_schedule?: "weekly" | "monthly" | "flexible";
  pending_payment_schedule?: "weekly" | "monthly" | "flexible" | null;
  schedule_effective_from?: string | null;
  check_in_enabled?: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
  // Nested employee for backwards compatibility
  employee?: {
    id: number;
    fullname: string;
    cccd: string;
    avatar?: string;
    position?: string;
  };
}

export interface AssignEmployeeData {
  employee_id: number;
  employee_code?: string; // optional, defaults to employee cccd
  position: string;
  start_date?: string; // optional, defaults to today
  end_date?: string;
  payment_schedule?: "weekly" | "monthly" | "flexible"; // optional, defaults to weekly
}

export interface ProjectAssignmentItem {
  id: number;
  project_id: number;
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  employee_code: string;
  position: string;
  start_date: string;
  last_date?: string | null;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface ProjectAssignmentResponse {
  data: ProjectAssignmentItem[];
}

export interface PayRate {
  normal: number;
  overtime: number;
  weekend: number;
  holiday: number;
}

export interface PayRateConfig {
  id: number;
  project_id: number;
  payrate: PayRate;
  from_date: string;
  to_date?: string;
  status: "active" | "inactive";
  last_updated_by?: number;
  last_approved_by?: number;
  created_at?: string;
  updated_at?: string;
}

export interface CreatePayRateData {
  payrate: PayRate;
  from_date: string;
  to_date?: string;
}

// Missing types for backwards compatibility
export interface ProjectFormData {
  name: string;
  description?: string;
  client_name: string;
  code?: string;
  start_date: string;
  end_date: string;
}

export interface PaginationData {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface ProjectsListResponse {
  data: Project[];
  pagination?: PaginationData;
}

export interface ProjectUser {
  id: number;
  user_id: number;
  user_fullname: string;
  user_email: string;
  granted_by: number;
  granted_at: string;
}

export interface ProjectUsersResponse {
  data: ProjectUser[];
}

export interface GrantProjectAccessData {
  user_id: number;
}

export interface ProjectTimesheetFilters {
  page?: number;
  pageSize?: number;
  employee_id?: number;
  date?: string; // Filter by specific date (YYYY-MM-DD)
  month?: string; // Filter by month (YYYY-MM)
  fromDate?: string; // Start date (YYYY-MM-DD)
  toDate?: string; // End date (YYYY-MM-DD)
  status?: "pending_approval" | "approved" | "rejected" | "paid" | "cancelled";
  paytype?: string;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
}
