import { apiClient } from "./api/client";
import { API_ENDPOINTS } from "@/config/api.config";

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

  checkIn: async (payload: { lat: number; lng: number }) => {
    return apiClient.post<AttendanceRecord>(
      API_ENDPOINTS.attendance.mobile.checkIn,
      payload
    );
  },

  checkOut: async (payload: { lat: number; lng: number; confirm_no_salary?: boolean }) => {
    return apiClient.post<AttendanceRecord>(
      API_ENDPOINTS.attendance.mobile.checkOut,
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
  }
};
