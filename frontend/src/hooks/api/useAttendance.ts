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
  });
}

export function useCheckIn() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: attendanceService.checkIn,
    onSuccess: () => {
      toast.success("Check-in thành công");
      queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.today() });
    },
    onError: (error: any) => {
      toast.error(error?.response?.data?.message || "Lỗi khi check-in");
    },
  });
}

export function useCheckOut() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: attendanceService.checkOut,
    onSuccess: () => {
      toast.success("Check-out thành công");
      queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.today() });
      // Invalidate advance payment info to refresh quota
      queryClient.invalidateQueries({ queryKey: ["advance-payment", "info"] });
    },
    onError: (error: any) => {
      toast.error(error?.response?.data?.message || "Lỗi khi check-out");
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
