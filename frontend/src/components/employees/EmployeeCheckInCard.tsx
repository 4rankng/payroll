import { useEffect, useRef, useState } from "react";
import { AlertCircle, BadgeCheck, BriefcaseBusiness, DoorOpen, Loader2, MapPin, RotateCcw, Settings, WalletCards } from "lucide-react";
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
import {
  createGeolocationError,
  getLocationPermissionIssue,
  getLocationPermissionState,
  isGeolocationError,
  requestCurrentLocation,
  type LocationPermissionIssue,
} from "@/utils/geolocation";

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
  const [locationIssue, setLocationIssue] = useState<LocationPermissionIssue | null>(null);
  // Synchronous in-flight guard. The button's `disabled` only takes effect after
  // the next render, so a rapid double-tap (common on mobile) can fire handleAction
  // twice before `isLocating`/`isPending` flips — sending a second request that the
  // backend rejects with 400 ("Bạn đã vào làm/tan ca rồi"), producing a duplicate
  // toast. This ref is set in the same call stack, closing that race.
  const submittingRef = useRef(false);

  const attendance = attendanceResponse?.data;

  // Clear a stale location-recovery banner when the user returns to the tab.
  // useTodayAttendance refetches on window focus, so if the user fixed the OS
  // location permission elsewhere and comes back, the banner — which is only
  // ever set from a failed handleAction — should not linger. The next attempt
  // (or a successful refetch) re-establishes correct state.
  useEffect(() => {
    const clearOnReturn = () => setLocationIssue(null);
    window.addEventListener("focus", clearOnReturn);
    return () => window.removeEventListener("focus", clearOnReturn);
  }, []);

  const handleAction = async (type: "check_in" | "check_out") => {
    if (submittingRef.current) return;
    submittingRef.current = true;
    // Start each attempt with a clean slate so a stale banner from a previous
    // failed attempt doesn't persist while the new one is in flight.
    setLocationIssue(null);
    setIsLocating(true);
    try {
      const permissionState = await getLocationPermissionState();
      if (permissionState === "denied") {
        throw createGeolocationError(1, "Quyền vị trí đã bị chặn");
      }

      const position = await requestCurrentLocation();
      const payload = {
        lat: position.coords.latitude,
        lng: position.coords.longitude,
      };

      if (type === "check_in") {
        await checkInMutation.mutateAsync(payload);
      } else {
        await checkOutMutation.mutateAsync(payload);
      }
      setLocationIssue(null);
    } catch (error: unknown) {
      if (!isGeolocationError(error)) {
        setLocationIssue(null);
        return;
      }

      const issue = getLocationPermissionIssue(error);
      setLocationIssue(issue);
      // The persistent amber recovery panel below renders the full title +
      // description + retry, so the toast only carries the headline (no
      // description) to avoid showing the same text twice.
      toast({ title: issue.title, variant: "destructive" });
    } finally {
      setIsLocating(false);
      submittingRef.current = false;
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
  // The next action is fully determined by the attendance status — there is no
  // need to track the last-tapped action separately (doing so defaulted it to
  // "check_in" and could mislabel the recovery CTA on a fresh check_out).
  const actionType = attendance?.status === "checked_in" ? "check_out" : "check_in";
  const locationRecoveryText =
    actionType === "check_out" ? "Thử cấp quyền lại để tan ca" : "Thử cấp quyền lại để vào làm";
  const showLocationRecovery = locationIssue && (attendance?.status === "checked_in" || !attendance);

  return (
    <div className={`p-4 ${className}`} style={style}>
      {showLocationRecovery ? (
        <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-amber-950">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-amber-700">
              {locationIssue.requiresSettings ? <Settings className="h-5 w-5" /> : <MapPin className="h-5 w-5" />}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-[17px] font-bold leading-6">{locationIssue.title}</p>
              <p className="mt-1 text-[15px] font-medium leading-6 text-amber-800">{locationIssue.description}</p>
              {locationIssue.requiresSettings ? (
                <p className="mt-2 text-[14px] font-semibold leading-5 text-amber-900">
                  iPhone Safari: vào Cài đặt &gt; Quyền riêng tư &amp; Bảo mật &gt; Dịch vụ định vị &gt; Safari Websites, chọn Hỏi lần sau hoặc Khi dùng ứng dụng, rồi tải lại trang.
                </p>
              ) : null}
              {locationIssue.canRetry ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="mt-3 h-11 rounded-lg border-amber-300 bg-white text-[15px] font-bold text-amber-950 hover:bg-amber-100"
                  disabled={isPending}
                  onClick={() => handleAction(actionType)}
                >
                  {isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <RotateCcw className="mr-2 h-4 w-4" />
                  )}
                  {isLocating ? "Đang lấy vị trí..." : locationRecoveryText}
                </Button>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
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
      ) : attendance?.status === "rejected" ? (
        <div className="rounded-xl border border-orange-200 bg-orange-50 p-4">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-orange-600">
              <AlertCircle className="h-5 w-5" />
            </div>
            <div>
              <p className="text-[18px] font-bold leading-6 text-orange-950">Ca làm việc đã bị từ chối</p>
              <p className="mt-1 text-[16px] font-medium leading-6 text-orange-700">
                {attendance.salary_reject_reason ||
                  "Ca này đã hết hạn tan ca và bị từ chối tự động. Vui lòng liên hệ quản lý."}
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
