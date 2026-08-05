import { apiClient } from "./api/client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  AdminCheckInShiftResponse,
  AdminCreateCheckInData,
  AdminAttendanceResponse,
} from "@/types/api/attendance.types";

export interface AttendanceRecord {
  id: number;
  project_id: number;
  employee_id: number;
  date: string;
  check_in_time: string;
  check_in_gate: string;
  check_out_time?: string;
  check_out_gate?: string;
  earning_amount?: number;
  salary_reject_reason?: string;
  salary_status?: "pending" | "recorded" | "not_recorded";
  salary_message?: string;
  status: "checked_in" | "completed" | "orphaned" | "rejected";
}

export interface AttendanceHistoryParams {
  limit?: number;
  offset?: number;
  from_date?: string;
  to_date?: string;
}

interface AttendanceLocationPayload {
  lat: number;
  lng: number;
  accuracy?: number;
  gps_at?: number;
  gps_sample_count?: number;
  gps_best_accuracy?: number;
  gps_elapsed_ms?: number;
}

export type AttendanceAttemptType = "check_in" | "check_out";
export type AttendanceDeviceGpsStatus = "denied" | "timeout" | "unavailable" | "unsupported";

export interface AttendanceDeviceAttemptPayload {
  attempt_type: AttendanceAttemptType;
  gps_status: AttendanceDeviceGpsStatus;
}

export const attendanceService = {
  getToday: async () => {
    return apiClient.get<AttendanceRecord | null>(
      API_ENDPOINTS.attendance.mobile.today
    );
  },

  getHistory: async (params?: AttendanceHistoryParams) => {
    return apiClient.get<AttendanceRecord[]>(
      API_ENDPOINTS.attendance.mobile.history,
      { params: { limit: 20, ...params } }
    );
  },

  checkIn: async (payload: AttendanceLocationPayload) => {
    return apiClient.post<AttendanceRecord>(
      API_ENDPOINTS.attendance.mobile.checkIn,
      payload
    );
  },

  checkOut: async (payload: AttendanceLocationPayload & { confirm_no_salary?: boolean }) => {
    return apiClient.post<AttendanceRecord>(
      API_ENDPOINTS.attendance.mobile.checkOut,
      payload
    );
  },

  cancelCurrent: async () => {
    return apiClient.post<AttendanceRecord>(
      API_ENDPOINTS.attendance.mobile.cancelCurrent
    );
  },

  logDeviceAttempt: async (payload: AttendanceDeviceAttemptPayload) => {
    return apiClient.post<null>(
      API_ENDPOINTS.attendance.mobile.attemptLog,
      payload
    );
  },

  // Admin APIs
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  adminList: async (params?: Record<string, any>) => {
    const { data } = await apiClient.get(API_ENDPOINTS.attendance.admin.list, { params });
    return data;
  },

  adminGetById: async (id: number) => {
    const { data } = await apiClient.get(API_ENDPOINTS.attendance.admin.byId(id));
    return data;
  },

  adminApprove: async (id: number, note?: string) => {
    const { data } = await apiClient.post(API_ENDPOINTS.attendance.admin.approve(id), { note: note ?? "" });
    return data;
  },

  adminReject: async (id: number, note: string) => {
    const { data } = await apiClient.post(API_ENDPOINTS.attendance.admin.reject(id), { note });
    return data;
  },

  adminGetCheckInShifts: async (employeeId: number, projectId: number, date: string) => {
    const { data } = await apiClient.get<AdminCheckInShiftResponse>(
      API_ENDPOINTS.attendance.admin.shifts,
      { params: { employee_id: employeeId, project_id: projectId, date } },
    );
    return data;
  },

  adminCreateCheckIn: async (payload: AdminCreateCheckInData) => {
    const { data } = await apiClient.post<AdminAttendanceResponse>(
      API_ENDPOINTS.attendance.admin.create,
      payload,
    );
    return data;
  }
};
