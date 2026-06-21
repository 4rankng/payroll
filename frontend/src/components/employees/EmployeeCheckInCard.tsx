import { useState } from "react";
import { BriefcaseBusiness, DoorOpen, Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useTodayAttendance,
  useCheckIn,
  useCheckOut,
} from "@/hooks/api/useAttendance";
import { format } from "date-fns";
import { toast } from "@/components/ui/sonner";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";

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
    } catch (error: unknown) {
      const e = error as { code?: number; message?: string };
      if (e.code === 1) { // PERMISSION_DENIED
        toast({ title: "Vui lòng cấp quyền truy cập vị trí để sử dụng tính năng này", variant: "destructive" });
      } else {
        toast({ title: e.message || "Không thể lấy vị trí hiện tại", variant: "destructive" });
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
    <div className={`p-4 ${className}`} style={style}>
      {attendance?.status === "completed" ? (
        <div className="rounded-2xl border border-emerald-100 bg-emerald-50/80 p-4">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-white text-emerald-600 shadow-sm">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-bold text-slate-900">Đã hoàn thành ca làm việc</p>
              <p className="mt-1 text-xs font-medium text-emerald-700">
                Vào làm {safeFormatTime(attendance.check_in_time)} · Tan ca {safeFormatTime(attendance.check_out_time)}
              </p>
              {attendance.earning_amount != null && attendance.earning_amount > 0 && (
                <p className="mt-2 text-sm font-bold text-emerald-700">
                  Lương: {attendance.earning_amount.toLocaleString("vi-VN")} đ
                </p>
              )}
            </div>
          </div>
        </div>
      ) : attendance?.status === "checked_in" ? (
        <div className="space-y-4">
          <div className="flex items-start gap-3 rounded-2xl border border-sky-100 bg-sky-50/80 p-4">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-white text-sky-600 shadow-sm">
              <BriefcaseBusiness className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-bold text-slate-900">Đang làm việc</p>
              <p className="mt-1 text-xs font-medium text-sky-700">
                Vào làm lúc {safeFormatTime(attendance.check_in_time)}
              </p>
            </div>
          </div>
          <Button
            size="lg"
            className="h-14 w-full rounded-2xl bg-slate-900 text-base font-bold text-white shadow-lg shadow-slate-900/20 hover:bg-slate-800"
            disabled={isPending}
            onClick={() => handleAction("check_out")}
          >
            {isPending ? (
              <Loader2 className="w-5 h-5 animate-spin mr-2" />
            ) : (
              <DoorOpen className="w-5 h-5 mr-2" />
            )}
            {isLocating ? "Đang lấy vị trí..." : "Tan ca"}
          </Button>
        </div>
      ) : attendance?.status === "orphaned" ? (
        <div className="flex flex-col items-center p-4 bg-red-50 rounded-xl w-full border border-red-100">
          <AlertCircle className="w-8 h-8 text-red-500 mb-2" />
          <p className="text-red-800 font-medium">Ca làm việc không hợp lệ</p>
          <p className="text-xs text-red-600 text-center mt-1">
            Bạn đã quên tan ca trong ca làm việc này.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex items-start gap-3 rounded-2xl border border-emerald-100 bg-gradient-to-br from-emerald-50 to-white p-4">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-white text-employee shadow-sm">
              <BriefcaseBusiness className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-bold text-slate-900">Sẵn sàng vào làm</p>
              <p className="mt-1 text-xs font-medium leading-5 text-slate-500">
                Chạm nút bên dưới để xác nhận vị trí tại cổng dự án.
              </p>
            </div>
          </div>
          <Button
            size="lg"
            className="h-14 w-full rounded-2xl bg-employee text-base font-bold text-white shadow-lg hover:bg-employee-600"
            style={{
              boxShadow: `0 10px 24px ${EMPLOYEE_BRAND_COLOR}30`,
            }}
            disabled={isPending}
            onClick={() => handleAction("check_in")}
          >
            {isPending ? (
              <Loader2 className="w-5 h-5 animate-spin mr-2" />
            ) : (
              <BriefcaseBusiness className="w-5 h-5 mr-2" />
            )}
            {isLocating ? "Đang lấy vị trí..." : "Vào làm"}
          </Button>
        </div>
      )}
    </div>
  );
}
