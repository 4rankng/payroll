import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { attendanceService } from "@/services/attendance";
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
      toast.success("Tan ca thành công");
      queryClient.setQueryData(ATTENDANCE_QUERY_KEYS.today(), response);
      queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.all });
      // Invalidate advance payment info to refresh quota
      queryClient.invalidateQueries({ queryKey: ["advance-payment", "info"] });
    },
    onError: (error: unknown) => {
      const e = error as { response?: { data?: { message?: string } }; message?: string };
      toast.error(e?.response?.data?.message || "Không thể tan ca");
    },
  });
}

export function useAttendanceHistory() {
  return useQuery({
    queryKey: [...ATTENDANCE_QUERY_KEYS.all, "history"],
    queryFn: async () => {
      const data = await attendanceService.getHistory();
      return data;
    },
  });
}
