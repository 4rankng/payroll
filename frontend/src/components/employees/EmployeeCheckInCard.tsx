import { useState } from "react";
import { AlertCircle, BadgeCheck, BriefcaseBusiness, DoorOpen, Loader2, MapPin, WalletCards } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useTodayAttendance,
  useCheckIn,
  useCheckOut,
} from "@/hooks/api/useAttendance";
import { format } from "date-fns";
import { toast } from "@/components/ui/sonner";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";
import { formatCurrency } from "@/utils/formatters";
import { isSalaryRecorded } from "@/utils/attendanceHelpers";

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
  const salaryRecorded = attendance ? isSalaryRecorded(attendance) : false;

  return (
    <div className={`p-4 ${className}`} style={style}>
      {attendance?.status === "completed" ? (
        <div className="space-y-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-start gap-3">
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
                <BadgeCheck className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[18px] font-bold leading-6 text-slate-950">Ca hôm nay đã xong</p>
                <div className="mt-2 grid grid-cols-2 gap-2 text-[16px] font-semibold text-slate-600">
                  <span className="rounded-md bg-slate-50 px-2.5 py-2">
                    Vào {safeFormatTime(attendance.check_in_time)}
                  </span>
                  <span className="rounded-md bg-slate-50 px-2.5 py-2">
                    Tan {safeFormatTime(attendance.check_out_time)}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div
            className={`rounded-xl border p-4 ${
              salaryRecorded
                ? "border-emerald-200 bg-emerald-50 text-emerald-950"
                : "border-amber-200 bg-amber-50 text-amber-950"
            }`}
          >
            <div className="flex items-start gap-3">
              <div
                className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white ${
                  salaryRecorded ? "text-emerald-700" : "text-amber-700"
                }`}
              >
                {salaryRecorded ? <WalletCards className="h-5 w-5" /> : <AlertCircle className="h-5 w-5" />}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[18px] font-bold leading-6">
                  {salaryRecorded ? `Lương ca: ${formatCurrency(attendance.earning_amount)}` : "Ca này chưa tính lương"}
                </p>
                <p className="mt-1 text-[16px] font-medium leading-6 opacity-85">
                  {attendance.salary_reject_reason ||
                    attendance.salary_message ||
                    (salaryRecorded
                      ? "Bạn có thể yêu cầu ứng lương nếu còn hạn mức."
                      : "Vui lòng kiểm tra khung giờ ca làm hoặc liên hệ quản lý.")}
                </p>
              </div>
            </div>
          </div>
        </div>
      ) : attendance?.status === "checked_in" ? (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-start gap-3">
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-sky-50 text-sky-700">
                <BriefcaseBusiness className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[18px] font-bold leading-6 text-slate-950">Bạn đang làm việc</p>
                <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                  Vào làm lúc {safeFormatTime(attendance.check_in_time)}
                </p>
                <p className="mt-2 text-[16px] font-medium leading-6 text-sky-700">
                  Khi hết ca, bấm Tan ca để hệ thống ghi nhận lương.
                </p>
              </div>
            </div>
          </div>
          <Button
            size="lg"
            className="h-14 w-full rounded-xl bg-slate-950 text-[18px] font-bold text-white shadow-lg shadow-slate-900/15 hover:bg-slate-800"
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
        <div className="rounded-xl border border-red-200 bg-red-50 p-4">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-red-600">
              <AlertCircle className="h-5 w-5" />
            </div>
            <div>
              <p className="text-[18px] font-bold leading-6 text-red-950">Ca này cần quản lý kiểm tra</p>
              <p className="mt-1 text-[16px] font-medium leading-6 text-red-700">
                Bạn chưa bấm Tan ca cho ca trước. Hãy báo quản lý để kiểm tra lại lương.
              </p>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-start gap-3">
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-employee">
                <MapPin className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[18px] font-bold leading-6 text-slate-950">Sẵn sàng vào làm</p>
                <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                  Khi đã tới cổng nhà máy, bấm Vào làm để bắt đầu ca.
                </p>
              </div>
            </div>
          </div>
          <Button
            size="lg"
            className="h-14 w-full rounded-xl bg-employee text-[18px] font-bold text-white shadow-lg hover:bg-employee-600"
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
