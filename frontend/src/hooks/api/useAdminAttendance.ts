import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
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

/**
 * Invalidate every cached admin attendance list (across all filter variants)
 * after a review mutation. Partial in-place updates are fragile here because
 * the row's status/earning both change and other open filter queries exist, so
 * a broad invalidate is the honest, simplest correct choice.
 */
function invalidateAttendanceLists(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: ADMIN_ATTENDANCE_QUERY_KEYS.all });
}

/** useApproveAttendance — admin approves a disputed attendance; backend recomputes earning. */
export function useApproveAttendance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, note }: { id: number; note?: string }) =>
      attendanceService.adminApprove(id, note),
    onSuccess: (updated: AdminAttendanceResponse) => {
      queryClient.setQueryData(ADMIN_ATTENDANCE_QUERY_KEYS.detail(updated.id), { data: updated });
      invalidateAttendanceLists(queryClient);
      toast.success("Đã duyệt chấm công");
    },
  });
}

/** useRejectAttendance — admin rejects an attendance; backend zeroes earning + stores reason. */
export function useRejectAttendance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, note }: { id: number; note: string }) =>
      attendanceService.adminReject(id, note),
    onSuccess: (updated: AdminAttendanceResponse) => {
      queryClient.setQueryData(ADMIN_ATTENDANCE_QUERY_KEYS.detail(updated.id), { data: updated });
      invalidateAttendanceLists(queryClient);
      toast.success("Đã từ chối chấm công");
    },
  });
}
