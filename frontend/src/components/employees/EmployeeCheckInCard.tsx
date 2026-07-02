import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { AlertCircle, BadgeCheck, BriefcaseBusiness, Clock, DoorOpen, Loader2, MapPin, RotateCcw, Settings, WalletCards } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useQueryClient } from "@tanstack/react-query";
import {
  ATTENDANCE_QUERY_KEYS,
  useTodayAttendance,
  useCheckIn,
  useCheckOut,
  useLogAttendanceDeviceAttempt,
} from "@/hooks/api/useAttendance";
import { getErrorMessage } from "@/utils/error-handler";
import { format } from "date-fns";
import { toast } from "@/components/ui/sonner";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";
import { formatCurrency } from "@/utils/formatters";
import {
  getAttendanceIssueSummary,
  getCheckoutWindowSummary,
  isSalaryRecorded,
  type AttendanceIssueDetail,
} from "@/utils/attendanceHelpers";
import {
  createPoorAccuracyLocationIssue,
  getLocationPermissionIssue,
  isGeolocationError,
  isPoorLocationAccuracyMessage,
  requestBestCurrentLocation,
  type LocationAcquisitionProgress,
  type LocationAcquisitionResult,
  type LocationSample,
  type LocationPermissionIssue,
} from "@/utils/geolocation";
import type { CheckInTarget } from "@/types/api/auth.types";

const EmployeeLocationMap = lazy(() =>
  import("./EmployeeLocationMap").then((module) => ({ default: module.EmployeeLocationMap }))
);

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
  checkInTarget?: CheckInTarget | null;
  checkInGeofenceRadiusMeters?: number | null;
  style?: React.CSSProperties;
}

// Returns true only for the two checkout-WINDOW violations (too early / too late),
// which the backend lets the employee override via confirm_no_salary. The orphaned
// ("Ca làm việc đã quá hạn tan ca") and auto-rejected ("...tự động từ chối...")
// guards run BEFORE the override branch in CheckOut, so they are intentionally NOT
// matched here — offering the dialog for them would dead-end (the confirmed call is
// re-rejected with the same message).
function canConfirmNoSalaryCheckout(message: string): boolean {
  return (
    message.includes("Chỉ có thể tan ca từ") ||
    message.includes("Bạn chỉ được tan ca từ")
  );
}

const CONFIRMED_NO_SALARY_MARKER = "Nhân viên đã xác nhận tan ca không ghi nhận tiền lương";

function isConfirmedNoSalaryAttendance(attendance: { salary_reject_reason?: string | null } | null | undefined): boolean {
  return Boolean(attendance?.salary_reject_reason?.includes(CONFIRMED_NO_SALARY_MARKER));
}

function IssueDetailChips({ details, tone }: { details: AttendanceIssueDetail[]; tone: "amber" | "orange" }) {
  if (details.length === 0) return null;

  const toneClass =
    tone === "orange"
      ? "border-orange-200 bg-white/80 text-orange-950"
      : "border-amber-200 bg-white/80 text-amber-950";

  return (
    <div className="mt-3 grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1.18fr)] gap-2">
      {details.map((detail) => (
        <div key={detail.label} className={`rounded-lg border px-2.5 py-2 ${toneClass}`}>
          <p className="whitespace-nowrap text-[11px] font-semibold uppercase leading-4 text-slate-500">{detail.label}</p>
          <p className="mt-0.5 text-[15px] font-bold leading-5">{detail.value}</p>
        </div>
      ))}
    </div>
  );
}

function MapFallback() {
  return (
    <div className="mt-3 rounded-xl border border-sky-200 bg-white px-3 py-3 text-[13px] font-medium leading-5 text-slate-600">
      Đang tải bản đồ vị trí...
    </div>
  );
}

function formatAccuracy(accuracy: number | undefined): string | null {
  if (typeof accuracy !== "number") return null;
  return `${Math.round(accuracy)}m`;
}

function getLocationAcquisitionMessage(progress: LocationAcquisitionProgress | null): string {
  const bestAccuracy = formatAccuracy(progress?.bestAccuracy);
  const requiredAccuracy = formatAccuracy(progress?.requiredAccuracyMeters);
  if (!progress || progress.sampleCount === 0 || !bestAccuracy) {
    return "Đang kiểm tra vị trí. Giữ điện thoại yên trong vài giây.";
  }

  if (progress.status === "excellent" || progress.status === "acceptable") {
    return "Vị trí đã sẵn sàng. Đang gửi chấm công.";
  }

  return `Chưa thể chấm công. Sai số hiện khoảng ${bestAccuracy}${
    requiredAccuracy ? `, cần trong vòng ${requiredAccuracy}` : ""
  }. Hãy đứng ở nơi thoáng hơn và giữ điện thoại yên vài giây.`;
}

function getDeviceGpsStatus(issue: LocationPermissionIssue): "denied" | "timeout" | "unavailable" | "unsupported" {
  switch (issue.type) {
    case "denied":
    case "timeout":
    case "unsupported":
      return issue.type;
    case "unavailable":
    default:
      return "unavailable";
  }
}

function isGeofenceOutsideMessage(message: string): boolean {
  return message.includes("ngoài khu vực chấm công");
}

function createOutsideGeofenceLocationIssue(): LocationPermissionIssue {
  return {
    type: "unknown",
    title: "Ngoài khu vực",
    description: "Di chuyển vào vùng xanh rồi thử lại.",
    canRetry: true,
    requiresSettings: false,
  };
}

export function EmployeeCheckInCard({
  className,
  checkInTarget,
  checkInGeofenceRadiusMeters,
  style,
}: EmployeeCheckInCardProps) {
  const { data: attendanceResponse, isLoading } = useTodayAttendance();
  const checkInMutation = useCheckIn();
  const checkOutMutation = useCheckOut();
  const logDeviceAttemptMutation = useLogAttendanceDeviceAttempt();
  const queryClient = useQueryClient();
  const [isLocating, setIsLocating] = useState(false);
  const [locationProgress, setLocationProgress] = useState<LocationAcquisitionProgress | null>(null);
  const [lastFreshSample, setLastFreshSample] = useState<LocationSample | null>(null);
  const [locationIssue, setLocationIssue] = useState<LocationPermissionIssue | null>(null);
  const [noSalaryReason, setNoSalaryReason] = useState<string | null>(null);
  const [showNoSalaryConfirm, setShowNoSalaryConfirm] = useState(false);
  // Synchronous in-flight guard. The button's `disabled` only takes effect after
  // the next render, so a rapid double-tap (common on mobile) can fire handleAction
  // twice before `isLocating`/`isPending` flips — sending a second request that the
  // backend rejects with 400 ("Bạn đã vào làm/tan ca rồi"), producing a duplicate
  // toast. This ref is set in the same call stack, closing that race.
  const submittingRef = useRef(false);

  const attendance = attendanceResponse?.data;
  const canStartCorrectShift = attendance?.status === "completed" && isConfirmedNoSalaryAttendance(attendance);
  const canPreviewCheckLocation =
    !isLoading &&
    Boolean(checkInTarget) &&
    !isLocating &&
    (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);

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

  useEffect(() => {
    if (!canPreviewCheckLocation) return;

    let cancelled = false;

    const locationOptions =
      typeof checkInTarget?.radius_meters === "number" && checkInTarget.radius_meters > 0
        ? {
            requiredAccuracyMeters: checkInTarget.radius_meters,
            timeoutMs: 12000,
            minimumWarmupMs: 1500,
            minimumAcceptableSamples: 1,
          }
        : {
            timeoutMs: 12000,
            minimumWarmupMs: 1500,
            minimumAcceptableSamples: 1,
          };

    requestBestCurrentLocation((progress) => {
      if (cancelled) return;
      if (progress.bestFreshSample) {
        setLastFreshSample(progress.bestFreshSample);
      }
    }, locationOptions)
      .then((result) => {
        if (cancelled) return;
        setLastFreshSample(result.bestFreshSample);
      })
      .catch(() => {
        // The always-on map can still show the configured gate/radius without a
        // preview GPS sample. The explicit submit action owns visible GPS errors.
      });

    return () => {
      cancelled = true;
    };
  }, [canPreviewCheckLocation, checkInTarget?.radius_meters]);

  const handleAction = async (type: "check_in" | "check_out", options?: { confirmNoSalary?: boolean }) => {
    if (submittingRef.current) return;
    submittingRef.current = true;
    // Start each attempt with a clean slate so a stale banner from a previous
    // failed attempt doesn't persist while the new one is in flight.
    setLocationIssue(null);
    setLocationProgress(null);
    setLastFreshSample(null);
    setIsLocating(true);
    let acquisition: LocationAcquisitionResult | null = null;
    try {
      // iOS Safari can keep navigator.permissions stale after the worker changes
      // Settings. Always ask for a fresh position; watchPosition is the
      // authoritative permission check and either resolves or returns the real
      // browser error for the recovery panel. We warm up GPS and use the best
      // fresh high-accuracy sample instead of trusting the first Wi-Fi/cell fix.
      const locationOptions =
        typeof checkInTarget?.radius_meters === "number" && checkInTarget.radius_meters > 0
          ? { requiredAccuracyMeters: checkInTarget.radius_meters }
          : typeof checkInGeofenceRadiusMeters === "number" && checkInGeofenceRadiusMeters > 0
            ? { requiredAccuracyMeters: checkInGeofenceRadiusMeters }
          : undefined;
      acquisition = await requestBestCurrentLocation((progress) => {
        setLocationProgress(progress);
        if (progress.bestFreshSample) {
          setLastFreshSample(progress.bestFreshSample);
        }
      }, locationOptions);
      setLastFreshSample(acquisition.bestFreshSample);
      const position = acquisition.position;
      const payload = {
        lat: position.coords.latitude,
        lng: position.coords.longitude,
        // Send GPS accuracy + fix time so the backend can reject unreliable
        // fixes and keep a forensic trail. Without accuracy the server trusts the
        // reported coordinate blindly, which lets an off-site check-in through
        // when the phone misreports (WiFi/cell positioning, stale fix, poor GNSS).
        accuracy: position.coords.accuracy,
        // GeolocationPosition.timestamp is epoch milliseconds.
        gps_at: position.timestamp,
        gps_sample_count: acquisition.sampleCount,
        gps_best_accuracy: acquisition.bestAccuracy,
        gps_elapsed_ms: acquisition.elapsedMs,
      };

      if (type === "check_in") {
        await checkInMutation.mutateAsync(payload);
      } else {
        await checkOutMutation.mutateAsync({
          ...payload,
          confirm_no_salary: options?.confirmNoSalary,
        });
      }
      setLocationIssue(null);
      setNoSalaryReason(null);
      setShowNoSalaryConfirm(false);
    } catch (error: unknown) {
      if (!isGeolocationError(error)) {
        setLocationIssue(null);
        if (type === "check_out") {
          // useCheckOut.onError is a no-op, so the card owns all checkout error UI.
          const message = getErrorMessage(error);
          if (isPoorLocationAccuracyMessage(message)) {
            setLocationIssue(
              createPoorAccuracyLocationIssue(
                acquisition?.bestAccuracy,
                acquisition?.requiredAccuracyMeters
              )
            );
            toast({ title: "Chưa thể chấm công", variant: "destructive" });
            return;
          }
          if (isGeofenceOutsideMessage(message)) {
            setLocationIssue(createOutsideGeofenceLocationIssue());
            toast({ title: "Chưa thể chấm công", variant: "destructive" });
            return;
          }
          if (!options?.confirmNoSalary && canConfirmNoSalaryCheckout(message)) {
            // First attempt outside the checkout window: show the real reason and
            // offer the confirmed no-salary path. The dialog is the UI — no toast.
            setNoSalaryReason(message);
            setShowNoSalaryConfirm(true);
          } else {
            // A confirmed no-salary call that failed (e.g. the shift was auto-
            // rejected while the dialog was open) or a non-overridable error:
            // close any stale dialog, surface the message once, and refetch today's
            // attendance so the card reflects the real (possibly rejected) state.
            setShowNoSalaryConfirm(false);
            setNoSalaryReason(null);
            toast({ title: message || "Không thể tan ca", variant: "destructive" });
            queryClient.invalidateQueries({ queryKey: ATTENDANCE_QUERY_KEYS.today() });
          }
        } else {
          const message = getErrorMessage(error);
          if (isPoorLocationAccuracyMessage(message)) {
            setLocationIssue(
              createPoorAccuracyLocationIssue(
                acquisition?.bestAccuracy,
                acquisition?.requiredAccuracyMeters
              )
            );
          } else if (isGeofenceOutsideMessage(message)) {
            setLocationIssue(createOutsideGeofenceLocationIssue());
          }
        }
        return;
      }

      const issue = getLocationPermissionIssue(error);
      setLocationIssue(issue);
      logDeviceAttemptMutation.mutate({
        attempt_type: type,
        gps_status: getDeviceGpsStatus(issue),
      });
      // The persistent amber recovery panel below renders the full title +
      // description + retry, so the toast only carries the headline (no
      // description) to avoid showing the same text twice.
      toast({ title: issue.title, variant: "destructive" });
    } finally {
      setIsLocating(false);
      setLocationProgress(null);
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
  const locationRecoveryText = "Thử lại";
  const showLocationRecovery = locationIssue && (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);
  const visibleLocationSample = locationProgress?.bestFreshSample ?? lastFreshSample;
  const locationPreview = checkInTarget ? (
    <Suspense fallback={<MapFallback />}>
      <EmployeeLocationMap target={checkInTarget} sample={visibleLocationSample} />
    </Suspense>
  ) : null;
  const showRecoveryDescription =
    Boolean(locationIssue) &&
    (!checkInTarget || locationIssue?.requiresSettings || locationIssue?.type !== "unknown");
  const noSalaryWindow = getCheckoutWindowSummary(noSalaryReason);
  const salaryIssue = getAttendanceIssueSummary(
    attendance?.salary_reject_reason || attendance?.salary_message,
    "Kiểm tra lại khung giờ ca làm."
  );
  const rejectedIssue = getAttendanceIssueSummary(attendance?.salary_reject_reason);

  return (
    <div className={`p-4 ${className}`} style={style}>
      <AlertDialog open={showNoSalaryConfirm} onOpenChange={setShowNoSalaryConfirm}>
        <AlertDialogContent className="max-w-[calc(100vw-32px)] border-employee-100 bg-white shadow-2xl shadow-employee-900/20 sm:max-w-md">
          <AlertDialogHeader className="bg-employee-900 px-5 pb-4 pt-5 text-left">
            <AlertDialogTitle className="text-lg font-bold leading-snug text-white">
              Tan ca ngoài giờ hợp lệ
            </AlertDialogTitle>
            <AlertDialogDescription className="mt-1 text-sm leading-5 text-employee-100">
              Tan ca đúng khung giờ để hệ thống ghi nhận lương.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-3 px-5 py-4">
            <div className="rounded-lg border border-employee-100 bg-employee-50/80 p-4">
              <div className="flex items-center gap-2 text-employee-800">
                <Clock className="h-5 w-5 shrink-0" />
                <p className="text-sm font-semibold leading-5">Giờ hợp lệ</p>
              </div>
              {noSalaryWindow ? (
                <div className="mt-3 grid grid-cols-2 gap-3">
                  {noSalaryWindow.checkInTime ? (
                    <div>
                      <p className="text-xs font-medium leading-4 text-employee-700">Vào làm</p>
                      <p className="mt-1 text-base font-bold leading-6 text-slate-950">
                        {noSalaryWindow.checkInTime}
                      </p>
                    </div>
                  ) : null}
                  <div className={noSalaryWindow.checkInTime ? "" : "col-span-2"}>
                    <p className="text-xs font-medium leading-4 text-employee-700">Tan ca</p>
                    <p className="mt-1 text-base font-bold leading-6 text-slate-950">
                      {noSalaryWindow.validStartTime} - {noSalaryWindow.validEndTime}
                    </p>
                  </div>
                </div>
              ) : (
                <p className="mt-2 text-sm font-medium leading-5 text-slate-800">
                  {noSalaryReason || "Thời gian tan ca không nằm trong khung giờ hợp lệ của ca này."}
                </p>
              )}
            </div>
            <div className="flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-red-950">
              <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600" aria-hidden="true" />
              <div>
                <p className="text-base font-bold leading-6">Ca này sẽ không tính lương</p>
                <p className="mt-1 text-sm font-medium leading-5 text-red-800">
                  Chỉ tiếp tục nếu bạn muốn hủy ca hiện tại.
                </p>
              </div>
            </div>
          </div>
          <AlertDialogFooter className="grid grid-cols-2 gap-3 px-5 pb-5 pt-3">
            <AlertDialogCancel
              disabled={isPending}
              className="mt-0 h-12 w-full rounded-lg border-employee-200 bg-white text-base font-bold text-employee-900 hover:bg-employee-50 hover:text-employee-900"
            >
              Quay lại
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              className="h-12 w-full rounded-lg bg-red-600 text-base font-bold text-white shadow-sm shadow-red-900/15 hover:bg-red-700"
              onClick={(event) => {
                event.preventDefault();
                handleAction("check_out", { confirmNoSalary: true });
              }}
            >
              {isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <DoorOpen className="mr-2 h-4 w-4" />}
              Hủy ca
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {isLocating ? (
        <div className="mb-4 rounded-xl border border-sky-200 bg-sky-50 p-4 text-sky-950">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-sky-700">
              <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-[17px] font-bold leading-6">Đang kiểm tra vị trí...</p>
              <p className="mt-1 text-[15px] font-medium leading-6 text-sky-800">
                {getLocationAcquisitionMessage(locationProgress)}
              </p>
            </div>
          </div>
          {locationProgress?.sampleCount ? (
            <div className="mt-3 grid grid-cols-3 gap-2">
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="text-[11px] font-bold uppercase leading-4 text-sky-700">Sai số</p>
                <p className="mt-0.5 truncate text-[15px] font-extrabold leading-5 text-sky-950">
                  {formatAccuracy(locationProgress.bestAccuracy) || "--"}
                </p>
              </div>
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="text-[11px] font-bold uppercase leading-4 text-sky-700">Cần</p>
                <p className="mt-0.5 truncate text-[15px] font-extrabold leading-5 text-sky-950">
                  {"<="}
                  {formatAccuracy(locationProgress.requiredAccuracyMeters) || "50m"}
                </p>
              </div>
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="text-[11px] font-bold uppercase leading-4 text-sky-700">Lần đo</p>
                <p className="mt-0.5 truncate text-[15px] font-extrabold leading-5 text-sky-950">
                  {locationProgress.sampleCount}
                </p>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
      {showLocationRecovery ? (
        <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-amber-950">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white text-amber-700">
              {locationIssue.requiresSettings ? <Settings className="h-5 w-5" /> : <MapPin className="h-5 w-5" />}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[16px] font-bold leading-6">{locationIssue.title}</p>
              {showRecoveryDescription ? (
                <p className="mt-0.5 text-[14px] font-medium leading-5 text-amber-800">{locationIssue.description}</p>
              ) : null}
            </div>
              {locationIssue.canRetry ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="h-10 shrink-0 gap-2 rounded-lg border-amber-300 bg-white px-3 text-[14px] font-bold leading-5 text-amber-950 hover:bg-amber-100"
                  disabled={isPending}
                  onClick={() => handleAction(actionType)}
                >
                  {isPending ? (
                    <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
                  ) : (
                    <RotateCcw className="h-4 w-4 shrink-0" />
                  )}
                  <span>{isLocating ? "Đang..." : locationRecoveryText}</span>
                </Button>
              ) : null}
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
                  {salaryRecorded ? `Lương ca: ${formatCurrency(attendance.earning_amount)}` : salaryIssue.title}
                </p>
                <p className="mt-1 text-[16px] font-medium leading-6 opacity-85">
                  {salaryRecorded ? "Bạn có thể yêu cầu ứng lương nếu còn hạn mức." : salaryIssue.description}
                </p>
                {!salaryRecorded ? <IssueDetailChips details={salaryIssue.details} tone="amber" /> : null}
              </div>
            </div>
          </div>

          {canStartCorrectShift ? (
            <>
              {locationPreview}
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
            </>
          ) : null}
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
          {locationPreview}
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
              <p className="text-[18px] font-bold leading-6 text-orange-950">
                {rejectedIssue.title === "Cần quản lý kiểm tra" ? "Ca đã bị từ chối" : rejectedIssue.title}
              </p>
              <p className="mt-1 text-[16px] font-medium leading-6 text-orange-700">
                {rejectedIssue.description}
              </p>
              <IssueDetailChips details={rejectedIssue.details} tone="orange" />
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
          {locationPreview}
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
