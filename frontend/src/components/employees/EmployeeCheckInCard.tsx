import { useState, useEffect, memo } from "react";
import { Clock, MapPin, Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useTodayAttendance,
  useCheckIn,
  useCheckOut,
} from "@/hooks/api/useAttendance";
import { format } from "date-fns";
import { toast } from "@/components/ui/sonner";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";

// Isolated clock — only this subtree re-renders every second
const LiveClock = memo(function LiveClock() {
  const [time, setTime] = useState(new Date());
  useEffect(() => {
    const id = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(id);
  }, []);
  return (
    <div className="text-3xl font-bold tracking-tight text-slate-800 mb-6">
      {format(time, "HH:mm:ss")}
    </div>
  );
});

function safeFormatTime(time: string | undefined | null, fallback = "--:--"): string {
  if (!time) return fallback;
  try {
    return format(new Date(time), "HH:mm");
  } catch {
    return fallback;
  }
}

interface EmployeeCheckInCardProps {
  className?: string;
  style?: React.CSSProperties;
}

export function EmployeeCheckInCard({ className, style }: EmployeeCheckInCardProps) {
  const { data: attendanceResponse, isLoading } = useTodayAttendance();
  const checkInMutation = useCheckIn();
  const checkOutMutation = useCheckOut();
  const [isLocating, setIsLocating] = useState(false);

  const attendance = attendanceResponse?.data;

  const getLocation = (): Promise<GeolocationPosition> => {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error("Trình duyệt không hỗ trợ GPS"));
        return;
      }
      navigator.geolocation.getCurrentPosition(resolve, reject, {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 0,
      });
    });
  };

  const handleAction = async (type: "check_in" | "check_out") => {
    setIsLocating(true);
    try {
      const position = await getLocation();
      const payload = {
        lat: position.coords.latitude,
        lng: position.coords.longitude,
      };

      if (type === "check_in") {
        await checkInMutation.mutateAsync(payload);
      } else {
        await checkOutMutation.mutateAsync(payload);
      }
    } catch (error: any) {
      if (error.code === 1) { // PERMISSION_DENIED
        toast({ title: "Vui lòng cấp quyền truy cập vị trí để sử dụng tính năng này", variant: "destructive" });
      } else {
        toast({ title: error.message || "Không thể lấy vị trí hiện tại", variant: "destructive" });
      }
    } finally {
      setIsLocating(false);
    }
  };

  if (isLoading) {
    return (
      <div className={`p-4 ${className}`} style={style}>
        <div className="animate-pulse flex flex-col items-center justify-center space-y-4 h-32">
          <div className="h-6 w-32 bg-gray-200 rounded"></div>
          <div className="h-10 w-48 bg-gray-200 rounded-full"></div>
        </div>
      </div>
    );
  }

  const isPending = checkInMutation.isPending || checkOutMutation.isPending || isLocating;

  return (
    <div className={`p-5 flex flex-col items-center ${className}`} style={style}>
      <h2 className="text-sm font-medium text-slate-500 mb-1">Thời gian hiện tại</h2>
      <LiveClock />

      {attendance?.status === "completed" ? (
        <div className="flex flex-col items-center p-4 bg-green-50 rounded-xl w-full border border-green-100">
          <CheckCircle2 className="w-8 h-8 text-green-500 mb-2" />
          <p className="text-green-800 font-medium">Đã hoàn thành ca làm việc</p>
          <div className="mt-2 text-sm text-green-700 flex flex-col items-center">
            <span>Vào: {safeFormatTime(attendance.check_in_time)}</span>
            <span>Ra: {safeFormatTime(attendance.check_out_time)}</span>
            {attendance.earning_amount && (
              <span className="mt-1 font-semibold text-green-800">
                Lương: {attendance.earning_amount.toLocaleString("vi-VN")} đ
              </span>
            )}
          </div>
        </div>
      ) : attendance?.status === "checked_in" ? (
        <div className="w-full flex flex-col items-center">
          <div className="mb-4 text-sm text-blue-700 bg-blue-50 px-4 py-2 rounded-full flex items-center gap-2 border border-blue-100">
            <Clock className="w-4 h-4" />
            Đã vào ca lúc {safeFormatTime(attendance.check_in_time)}
          </div>
          <Button
            size="lg"
            className="w-full rounded-xl h-14 text-lg bg-orange-500 hover:bg-orange-600 shadow-md shadow-orange-500/20"
            disabled={isPending}
            onClick={() => handleAction("check_out")}
          >
            {isPending ? (
              <Loader2 className="w-5 h-5 animate-spin mr-2" />
            ) : (
              <MapPin className="w-5 h-5 mr-2" />
            )}
            {isLocating ? "Đang lấy vị trí..." : "Check Out"}
          </Button>
        </div>
      ) : attendance?.status === "orphaned" ? (
        <div className="flex flex-col items-center p-4 bg-red-50 rounded-xl w-full border border-red-100">
          <AlertCircle className="w-8 h-8 text-red-500 mb-2" />
          <p className="text-red-800 font-medium">Ca làm việc không hợp lệ</p>
          <p className="text-xs text-red-600 text-center mt-1">
            Bạn đã quên check-out trong ca làm việc này.
          </p>
        </div>
      ) : (
        <Button
          size="lg"
          className="w-full rounded-xl h-14 text-lg text-white shadow-md"
          style={{
            backgroundColor: EMPLOYEE_BRAND_COLOR,
            boxShadow: `0 4px 14px 0 ${EMPLOYEE_BRAND_COLOR}40`,
          }}
          disabled={isPending}
          onClick={() => handleAction("check_in")}
        >
          {isPending ? (
            <Loader2 className="w-5 h-5 animate-spin mr-2" />
          ) : (
            <MapPin className="w-5 h-5 mr-2" />
          )}
          {isLocating ? "Đang lấy vị trí..." : "Check In"}
        </Button>
      )}
    </div>
  );
}
