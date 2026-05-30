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
  status: "checked_in" | "completed" | "orphaned";
}

export const attendanceService = {
  getToday: async () => {
    const { data } = await apiClient.get<{ data: AttendanceRecord | null }>(
      API_ENDPOINTS.attendance.mobile.today
    );
    return data;
  },

  getHistory: async (params?: { limit?: number; offset?: number }) => {
    const { data } = await apiClient.get<{ data: AttendanceRecord[] }>(
      API_ENDPOINTS.attendance.mobile.history,
      { params: { limit: 20, ...params } }
    );
    return data;
  },

  checkIn: async (payload: { lat: number; lng: number }) => {
    const { data } = await apiClient.post<{ data: AttendanceRecord }>(
      API_ENDPOINTS.attendance.mobile.checkIn,
      payload
    );
    return data;
  },

  checkOut: async (payload: { lat: number; lng: number }) => {
    const { data } = await apiClient.post<{ data: AttendanceRecord }>(
      API_ENDPOINTS.attendance.mobile.checkOut,
      payload
    );
    return data;
  },

  // Admin APIs
  adminList: async (params?: Record<string, any>) => {
    const { data } = await apiClient.get<{ data: any }>(API_ENDPOINTS.attendance.admin.list, { params });
    return data;
  },

  adminGetById: async (id: number) => {
    const { data } = await apiClient.get<{ data: any }>(API_ENDPOINTS.attendance.admin.byId(id));
    return data;
  }
};
