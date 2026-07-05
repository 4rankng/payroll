// Authentication related types

export interface User {
  id: number;
  email: string;
  username: string;
  fullname: string;
  role: "admin" | "partner" | "employee" | "adv_partner";
  cccd?: string;
  mobile?: string;
  last_login?: string;
  created_at: string;
  updated_at: string;
}

export interface BasicUserProfile {
  id: number;
  username: string;
  role: "admin" | "partner" | "employee" | "adv_partner";
}

export interface LoginCredentials {
  username: string;
  password: string;
  /** CAPTCHA challenge response (required after N failed logins). */
  captcha_id?: string;
  captcha_code?: string;
}

export interface LoginResponse {
  // Populated on a normal (non-OTP) login:
  user?: User;
  access_token?: string;
  token_type?: "Bearer";
  expires_in?: number;
  // Populated when the email-OTP second factor is required (admin/partner,
  // OTP_ENABLE=true). The client must NOT treat otp_session_id as an auth
  // token — it is a pending-session id consumed by /auth/login/verify.
  otp_required?: boolean;
  otp_session_id?: string;
}

export interface VerifyOTPRequest {
  otp_session_id: string;
  code: string;
}

export interface ResendOTPRequest {
  otp_session_id: string;
}

export interface ChangePasswordData {
  current_password: string;
  new_password: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface CompleteUserProfile extends User {
  // Add any additional profile fields here if needed
}

export interface PasswordStrengthResponse {
  score: number;
  feedback: string[];
  isStrong: boolean;
}

export type CheckInTargetStatus = "ready" | "unavailable" | "ambiguous" | "missing_gates";

export interface GeofenceGate {
  name: string;
  lat: number;
  lng: number;
}

export interface CheckInTarget {
  project_id: number;
  project_name: string;
  radius_meters: number;
  gates: GeofenceGate[];
}

// Employee-specific types
export interface EmployeeProfile {
  id: number;
  fullname: string;
  email?: string;
  username: string;
  mobile?: string;
  address?: string;
  date_of_birth?: string;
  bank?: {
    id: number;
    branch_name: string;
    branch_code: string;
  } | null;
  bank_account_number?: string;
  bank_account_name?: string;
  payment_schedule?: "weekly" | "monthly" | "flexible";
  check_in_enabled?: boolean;
  check_in_target_status?: CheckInTargetStatus;
  check_in_target?: CheckInTarget | null;
  check_in_geofence_radius_meters?: number;
  // Advisory shift window (for the check-in button readiness gate). Null when
  // no shift is configured. The server's validateCheckInWindow is authoritative.
  shift_start?: string;
  shift_end?: string;
  check_in_window_start?: string;
  check_in_window_end?: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateEmployeeProfileData {
  fullname?: string;
  email?: string;
  username?: string;
}

export interface EmployeeTimesheetEntry {
  id: number;
  date: string;
  project: {
    id: number;
    name: string;
    code: string;
    client_name: string;
  };
  hours_worked: number;
  amount: number;
  timesheet_status: "pending" | "approved" | "rejected";
  payment_status: "unpaid" | "paid" | "pending";
  paid_amount: number;
  payment_date: string | null;
  approved_at: string | null;
  approved_by: {
    id: number;
    fullname: string;
  } | null;
  force_payroll?: boolean;
  created_at: string;
}

export interface EmployeeTimesheetFilters {
  page?: number;
  pageSize?: number;
  sortBy?: "date" | "amount" | "payment_date";
  sortOrder?: "asc" | "desc";
  fromDate?: string;
  toDate?: string;
  status?: "pending" | "approved" | "rejected";
  payment_status?: "unpaid" | "paid";
}

export interface EmployeeTimesheetResponse {
  status: string;
  data: EmployeeTimesheetEntry[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface EmployeeSummary {
  weekly_salary: number;
  weekly_clocked_hours: number;
  weekly_paid_amount: number;
  total_approved_timesheets: number;
  total_approved_amount: number;
  total_paid_timesheets: number;
  total_paid_amount: number;
  total_pending_timesheets: number;
  total_pending_amount: number;
  calculation_period: {
    weeks: number;
    from_date: string;
    to_date: string;
  };
  last_payment_date: string | null;
  current_projects: Array<{
    project_id: number;
    project_name: string;
    project_code: string;
    position: string;
    start_date: string;
  }>;
}

export interface EmployeeSummaryResponse {
  status: string;
  data: EmployeeSummary;
  message: string;
}
