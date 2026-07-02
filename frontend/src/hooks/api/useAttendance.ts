import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  attendanceService,
  type AttendanceDeviceAttemptPayload,
  type AttendanceHistoryParams,
} from "@/services/attendance";
import { QueryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

export const ATTENDANCE_QUERY_KEYS = {
  all: ["attendance"] as const,
  today: () => [...ATTENDANCE_QUERY_KEYS.all, "today"] as const,
};

export function useTodayAttendance() {
  return useQuery({
    queryKey: ATTENDANCE_QUERY_KEYS.today(),
    queryFn: async () => {
      const data = await attendanceService.getToday();
      return data;
    },
    retry: false,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });
}

export function useCheckIn() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: attendanceService.checkIn,
    onSuccess: (response) => {
      toast.success("Vào làm thành công");
      queryClient.setQueryData(ATTENDANCE_QUERY_KEYS.today(), response);
      queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.all });
    },
    onError: (error: unknown) => {
      const e = error as { response?: { data?: { message?: string } }; message?: string };
      toast.error(e?.response?.data?.message || "Không thể vào làm");
    },
  });
}

export function useCheckOut() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: attendanceService.checkOut,
    onSuccess: (response) => {
      const salaryMessage = response.data?.salary_message;
      const salaryRecorded = response.data?.salary_status === "recorded";
      toast[salaryRecorded ? "success" : "warning"](
        salaryRecorded ? "Đã ghi nhận lương" : "Tan ca thành công",
        {
          description: salaryMessage || "Vui lòng kiểm tra thông tin lương ca làm.",
          duration: 6000,
        }
      );
      queryClient.setQueryData(ATTENDANCE_QUERY_KEYS.today(), response);
      queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.all });
      // Invalidate advance payment info to refresh quota. Check-in-enabled employees
      // use the dedicated check-in-advance key (70% cap); refresh both so the just-earned
      // quota is visible without waiting for the pending-request poll.
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.employee.info });
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.employee.checkInAdvanceInfo });
    },
    // onError is intentionally a no-op: EmployeeCheckInCard owns all checkout
    // error UI — the no-salary confirm dialog for window violations, and a toast
    // for everything else. Keeping onError defined suppresses the global
    // MutationCache error toast (App.tsx skips notifications when a mutation has
    // its own onError handler), which avoids a duplicate toast.
    onError: () => {},
  });
}

export function useLogAttendanceDeviceAttempt() {
  return useMutation({
    mutationFn: (payload: AttendanceDeviceAttemptPayload) =>
      attendanceService.logDeviceAttempt(payload),
    onError: () => {},
  });
}

export function useAttendanceHistory(params?: AttendanceHistoryParams) {
  return useQuery({
    queryKey: [...ATTENDANCE_QUERY_KEYS.all, "history", params],
    queryFn: async () => {
      const data = await attendanceService.getHistory(params);
      return data;
    },
  });
}
