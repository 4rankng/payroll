import { useQuery } from "@tanstack/react-query";
import { attendanceService } from "@/services/attendance";
import type { AttendanceFilters, PaginatedAttendanceResponse, AdminAttendanceResponse } from "@/types/api/attendance.types";

export const ADMIN_ATTENDANCE_QUERY_KEYS = {
  all: ["admin", "attendance"] as const,
  list: (filters: AttendanceFilters) => [...ADMIN_ATTENDANCE_QUERY_KEYS.all, "list", filters] as const,
  detail: (id: number) => [...ADMIN_ATTENDANCE_QUERY_KEYS.all, "detail", id] as const,
};

export function useAdminAttendances(filters: AttendanceFilters, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ADMIN_ATTENDANCE_QUERY_KEYS.list(filters),
    queryFn: async () => {
      const data = await attendanceService.adminList(filters);
      return data as PaginatedAttendanceResponse;
    },
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
}

export function useAdminAttendanceDetail(id: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ADMIN_ATTENDANCE_QUERY_KEYS.detail(id),
    queryFn: async () => {
      const data = await attendanceService.adminGetById(id);
      return data as { data: AdminAttendanceResponse };
    },
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
}
