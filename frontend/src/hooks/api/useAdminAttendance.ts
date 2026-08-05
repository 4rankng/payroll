import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { attendanceService } from "@/services/attendance";
import type {
  AdminCheckInShiftResponse,
  AdminCreateCheckInData,
  AttendanceFilters,
  PaginatedAttendanceResponse,
  AdminAttendanceResponse,
} from "@/types/api/attendance.types";

export const ADMIN_ATTENDANCE_QUERY_KEYS = {
  all: ["admin", "attendance"] as const,
  list: (filters: AttendanceFilters) => [...ADMIN_ATTENDANCE_QUERY_KEYS.all, "list", filters] as const,
  detail: (id: number) => [...ADMIN_ATTENDANCE_QUERY_KEYS.all, "detail", id] as const,
  checkInShifts: (employeeId: number, projectId: number, date: string) =>
    [...ADMIN_ATTENDANCE_QUERY_KEYS.all, "check-in-shifts", employeeId, projectId, date] as const,
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

export function useAdminCheckInShifts(
  employeeId: number | null,
  projectId: number | null,
  date: string,
  options?: { enabled?: boolean },
) {
  const enabled = Boolean(employeeId && projectId && date) && (options?.enabled ?? true);
  return useQuery({
    queryKey: ADMIN_ATTENDANCE_QUERY_KEYS.checkInShifts(employeeId ?? 0, projectId ?? 0, date),
    queryFn: () => attendanceService.adminGetCheckInShifts(employeeId!, projectId!, date),
    enabled,
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

/** useAdminCreateCheckIn — records only the check-in; checkout remains employee-owned. */
export function useAdminCreateCheckIn() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: AdminCreateCheckInData) => attendanceService.adminCreateCheckIn(payload),
    onSuccess: () => {
      invalidateAttendanceLists(queryClient);
      toast.success("Đã tạo check-in. Nhân viên cần tự tan ca để hoàn tất ca làm.");
    },
  });
}
