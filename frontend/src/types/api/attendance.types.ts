export interface AdminAttendanceResponse {
  id: number;
  project_id: number;
  project_name: string;
  employee_id: number;
  employee_name: string;
  date: string;
  check_in_time: string;
  check_in_lat: number;
  check_in_lng: number;
  check_in_accuracy?: number | null;
  check_in_gps_at?: string;
  check_in_gate: string;
  check_out_time?: string;
  check_out_lat?: number | null;
  check_out_lng?: number | null;
  check_out_accuracy?: number | null;
  check_out_gps_at?: string;
  check_out_gate?: string;
  earning_amount?: number;
  salary_reject_reason?: string;
  rejected_at?: string;
  // Admin review audit (backend migration 083). Null/undefined when never reviewed.
  review_action?: "approved" | "rejected" | null;
  review_note?: string | null;
  reviewed_by?: number | null;
  reviewed_at?: string | null;
  nearest_checkpoint_name?: string | null;
  nearest_checkpoint_lat?: number | null;
  nearest_checkpoint_lng?: number | null;
  nearest_checkpoint_distance_meters?: number | null;
  check_out_nearest_checkpoint_name?: string | null;
  check_out_nearest_checkpoint_lat?: number | null;
  check_out_nearest_checkpoint_lng?: number | null;
  check_out_nearest_checkpoint_distance_meters?: number | null;
  geofence_radius_meters?: number | null;
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
