export interface AdminAttendanceResponse {
  id: number;
  project_id: number;
  project_name: string;
  employee_id: number;
  employee_name: string;
  date: string;
  check_in_time: string;
  check_in_gate: string;
  check_out_time?: string;
  check_out_gate?: string;
  earning_amount?: number;
  salary_reject_reason?: string;
  status: string;
}

export interface AttendanceFilters {
  page?: number;
  pageSize?: number;
  limit?: number;
  employee_id?: number;
  project_id?: number;
  from_date?: string;
  to_date?: string;
  date_field?: "check_in_time";
  status?: string;
  successful_checkout?: boolean;
  zero_earning?: boolean;
}

export interface PaginatedAttendanceResponse {
  data: AdminAttendanceResponse[];
  total: number;
}
