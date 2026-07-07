import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
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
  useCancelCurrentAttendance,
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
  type LocationAcquisitionProgress,
  type LocationSample,
  type LocationPermissionIssue,
} from "@/utils/geolocation";
import { useContinuousLocation, isAbortedSubmitError } from "@/hooks/useContinuousLocation";
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
  /** Advisory shift window (from the profile DTO). Null/absent = no timing gate. */
  shiftStart?: string;
  shiftEnd?: string;
  checkInWindowStart?: string;
  checkInWindowEnd?: string;
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
    return "Đã có tín hiệu GPS. Đang gửi chấm công để hệ thống kiểm tra.";
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
  shiftStart,
  shiftEnd,
  checkInWindowStart,
  checkInWindowEnd,
  style,
}: EmployeeCheckInCardProps) {
  const { data: attendanceResponse, isLoading } = useTodayAttendance();
  const checkInMutation = useCheckIn();
  const checkOutMutation = useCheckOut();
  const cancelCurrentAttendanceMutation = useCancelCurrentAttendance();
  const logDeviceAttemptMutation = useLogAttendanceDeviceAttempt();
  const queryClient = useQueryClient();
  const [isLocating, setIsLocating] = useState(false);
  const [locationIssue, setLocationIssue] = useState<LocationPermissionIssue | null>(null);
  const [noSalaryReason, setNoSalaryReason] = useState<string | null>(null);
  const [showNoSalaryConfirm, setShowNoSalaryConfirm] = useState(false);
  const [showCancelShiftConfirm, setShowCancelShiftConfirm] = useState(false);
  // Synchronous in-flight guard. The button's `disabled` only takes effect after
  // the next render, so a rapid double-tap (common on mobile) can fire handleAction
  // twice before `isLocating`/`isPending` flips — sending a second request that the
  // backend rejects with 400 ("Bạn đã vào làm/tan ca rồi"), producing a duplicate
  // toast. This ref is set in the same call stack, closing that race.
  const submittingRef = useRef(false);

  const attendance = attendanceResponse?.data;
  const canStartCorrectShift = attendance?.status === "completed" && isConfirmedNoSalaryAttendance(attendance);
  // The continuous GPS watch runs whenever an action is conceptually possible and
  // STAYS running during a tap's submit-await. A fix warmed before the tap is then
  // submitted instantly, instead of the tap cold-starting a 25-30s acquisition
  // that times out before the phone's first GNSS fix — the dominant on-site check-
  // in failure. (The old canPreviewCheckLocation gate toggled off during isLocating,
  // which defeated warm reuse.) effectiveRadius keeps the legacy prop fallback.
  // Deliberately NOT gated on `isLoading`: the card's isLoading early-return only
  // swaps in a skeleton, but this hook still runs (hooks run before any return),
  // so letting the watch warm during the brief load makes the first actionable
  // tap instant. It also prevents a useTodayAttendance refetch mid-cold-tap from
  // toggling the watch off and stalling the awaiter for the full 30s.
  const locationEnabled =
    Boolean(checkInTarget) &&
    (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);
  const effectiveRadius =
    (typeof checkInTarget?.radius_meters === "number" && checkInTarget.radius_meters > 0
      ? checkInTarget.radius_meters
      : null) ??
    (typeof checkInGeofenceRadiusMeters === "number" && checkInGeofenceRadiusMeters > 0
      ? checkInGeofenceRadiusMeters
      : null) ??
    undefined;
  const location = useContinuousLocation({
    target: checkInTarget,
    enabled: locationEnabled,
    requiredAccuracyMeters: effectiveRadius,
  });
  // Aliases so the existing JSX (converging-accuracy banner, map preview) reads the
  // continuous watch's reactive state unchanged.
  const locationProgress: LocationAcquisitionProgress | null = location.progress;

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

  // The 12s preview effect that used to live here is gone: the continuous watch
  // (useContinuousLocation above) now warms the fix while the card is mounted, and
  // handleAction submits the warm sample instantly or awaits the watch's next
  // inside-gate sample. See plans/260704-1034-checkin-gps-continuous-warmup.

  const buildPayload = (sample: LocationSample) => {
    const p = location.progress;
    return {
      lat: sample.lat,
      lng: sample.lng,
      accuracy: sample.accuracy,
      gps_at: sample.timestamp,
      gps_sample_count: p?.sampleCount ?? 1,
      gps_best_accuracy: p?.bestAccuracy ?? sample.accuracy,
      gps_elapsed_ms: p?.elapsedMs ?? 0,
    };
  };

  // Centralized error classification shared by the warm and cold submit paths.
  // Geolocation errors (cold-tap timeout / inaccurate / fatal-permission) →
  // recovery banner + failed-attempt log. Backend errors (poor accuracy, outside
  // geofence, no-salary window) → existing classification and dialog flow. The
  // contract is identical to the previous inline handler; only the acquisition
  // source changed (continuous watch vs. one-shot requestBestCurrentLocation).
  const onActionError = async (
    error: unknown,
    type: "check_in" | "check_out",
    options?: { confirmNoSalary?: boolean }
  ) => {
    if (isAbortedSubmitError(error)) {
      // Watch was paused/restarted or the card unmounted during the cold-tap wait.
      // Silent cancellation: no banner, no failed-attempt log — the worker abandoned
      // the attempt; the device did not fail.
      return;
    }
    if (!isGeolocationError(error)) {
      // Backend rejection.
      if (type === "check_out") {
        // useCheckOut.onError is a no-op, so the card owns all checkout error UI.
        const message = getErrorMessage(error);
        if (isPoorLocationAccuracyMessage(message)) {
          setLocationIssue(
            createPoorAccuracyLocationIssue(
              location.progress?.bestAccuracy,
              location.progress?.requiredAccuracyMeters
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
              location.progress?.bestAccuracy,
              location.progress?.requiredAccuracyMeters
            )
          );
        } else if (isGeofenceOutsideMessage(message)) {
          setLocationIssue(createOutsideGeofenceLocationIssue());
        }
      }
      return;
    }

    // Geolocation error from awaitSubmitReady rejection (timeout / inaccurate) or
    // a fatal watch error (permission denied). Same recovery UX + failed-attempt
    // logging as before.
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
  };

  const handleAction = async (type: "check_in" | "check_out", options?: { confirmNoSalary?: boolean }) => {
    if (submittingRef.current) return;
    submittingRef.current = true;
    // Clear any stale recovery banner so it doesn't linger during the new attempt.
    setLocationIssue(null);

    // No configured gate: the watch can never produce an inside-gate sample, so
    // surface the configuration issue immediately instead of making the worker
    // wait 30s for a misleading "inaccurate" timeout. Mirrors the backend
    // validateGeofence "no gates configured" message.
    const hasValidGate =
      !!checkInTarget &&
      Array.isArray(checkInTarget.gates) &&
      checkInTarget.gates.length > 0 &&
      (checkInTarget.radius_meters ?? 0) > 0;
    if (!hasValidGate) {
      setLocationIssue({
        type: "unknown",
        title: "Chưa cấu hình vị trí chấm công",
        description: "Dự án chưa cấu hình cổng chấm công. Vui lòng báo quản lý.",
        canRetry: false,
        requiresSettings: false,
      });
      submittingRef.current = false;
      return;
    }

    const submit = async (sample: LocationSample) => {
      const payload = buildPayload(sample);
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
    };

    // Warm path: the continuous watch already has a fresh, gate-inside sample —
    // submit it instantly with no spinner. This is the seamless case (worker
    // opened the app at the gate, GPS warmed during mount, tap submits in <1s).
    if (location.isSubmitReady && location.sample) {
      try {
        await submit(location.sample);
      } catch (error: unknown) {
        await onActionError(error, type, options);
      } finally {
        submittingRef.current = false;
      }
      return;
    }

    // Cold path: submit the first fresh device fix to the backend even when the
    // client-side guidance says "outside". The server is the geofence authority
    // and records validation failures in attendance_failed_attempts; waiting for
    // an "inside" sample here would leave outside-geofence taps stuck client-side
    // with no backend audit row.
    setIsLocating(true);
    try {
      const sample = await location.awaitFreshSample();
      await submit(sample);
    } catch (error: unknown) {
      await onActionError(error, type, options);
    } finally {
      setIsLocating(false);
      submittingRef.current = false;
    }
  };

  const handleCancelCurrentShift = async () => {
    if (submittingRef.current) return;
    submittingRef.current = true;
    try {
      await cancelCurrentAttendanceMutation.mutateAsync();
      setShowCancelShiftConfirm(false);
      setLocationIssue(null);
    } catch {
      // Error toast is handled by useCancelCurrentAttendance.
    } finally {
      submittingRef.current = false;
    }
  };

  // --- Check-in readiness gate (GPS + geofence + timing) ---
  // Combines three signals: GPS stability (useContinuousLocation.isSubmitReady),
  // geofence position (from guidance), and the shift timing window (from the
  // profile DTO — Phase 1). The button shows only when all three are green.
  //
  // Re-evaluates at minute resolution: the original useMemo read Date.now() but
  // only depended on the window strings, so the result froze until those props
  // changed — a worker parked on the "outside-window" branch just before the
  // window opened would stay gated indefinitely. The minute tick forces a
  // recompute so the countdown advances and the ready pop fires on the crossing.
  // These hooks live BEFORE the isLoading early return (Rules of Hooks).
  const gpsReady = location.isSubmitReady;
  const gpsAcquiring = location.isWatching && !gpsReady && !location.fatalError;
  const nowTick = useRef(Date.now());
  const [, forceTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => {
      nowTick.current = Date.now();
      forceTick((n) => n + 1);
    }, 60_000);
    return () => clearInterval(id);
  }, []);
  const withinWindow = useMemo(() => {
    if (!checkInWindowStart || !checkInWindowEnd) return true; // no shift configured
    const now = new Date(nowTick.current);
    const nowMin = now.getHours() * 60 + now.getMinutes();
    const [sh, sm] = checkInWindowStart.split(":").map(Number);
    const [eh, em] = checkInWindowEnd.split(":").map(Number);
    if (!Number.isFinite(sh) || !Number.isFinite(eh)) return true;
    return nowMin >= sh * 60 + sm && nowMin <= eh * 60 + em;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [checkInWindowStart, checkInWindowEnd, nowTick.current]);

  // Seconds until the check-in window opens, for the outside-window hint
  // countdown. Recomputed on the minute tick (minute precision is sufficient;
  // showing seconds would require a 1s timer and is noisy on mobile).
  const secondsUntilWindow = useMemo(() => {
    if (!checkInWindowStart || withinWindow) return null;
    const [sh, sm] = checkInWindowStart.split(":").map(Number);
    if (!Number.isFinite(sh)) return null;
    const now = new Date(nowTick.current);
    const target = new Date(now);
    target.setHours(sh, sm ?? 0, 0, 0);
    // If the window already passed today, the gap is stale; treat as no
    // countdown (the static window text still shows).
    if (target.getTime() <= now.getTime()) return null;
    return Math.round((target.getTime() - now.getTime()) / 1000);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [checkInWindowStart, withinWindow, nowTick.current]);

  // Became-ready pop: fire a one-shot CSS class when transitioning to ready.
  // All hooks below run on every render (before the isLoading early return) to
  // satisfy the Rules of Hooks; isPending is derived from mutation/loading flags
  // that are all available pre-return.
  const isPending = checkInMutation.isPending || checkOutMutation.isPending || cancelCurrentAttendanceMutation.isPending || isLocating;
  const [showReadyPop, setShowReadyPop] = useState(false);
  const prevReadyRef = useRef(false);
  useEffect(() => {
    const ready = withinWindow && gpsReady && !isPending;
    if (ready && !prevReadyRef.current) {
      setShowReadyPop(true);
      const t = setTimeout(() => setShowReadyPop(false), 500);
      prevReadyRef.current = true;
      return () => clearTimeout(t);
    }
    if (!ready) prevReadyRef.current = false;
  }, [withinWindow, gpsReady, isPending]);

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

  const salaryRecorded = attendance ? isSalaryRecorded(attendance) : false;
  // The next action is fully determined by the attendance status — there is no
  // need to track the last-tapped action separately (doing so defaulted it to
  // "check_in" and could mislabel the recovery CTA on a fresh check_out).
  const actionType = attendance?.status === "checked_in" ? "check_out" : "check_in";
  const locationRecoveryText = "Thử lại";
  const showLocationRecovery = locationIssue && (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);
  const visibleLocationSample = location.sample;

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
      <AlertDialog open={showCancelShiftConfirm} onOpenChange={setShowCancelShiftConfirm}>
        <AlertDialogContent className="max-w-[calc(100vw-32px)] border-red-100 bg-white shadow-2xl shadow-red-900/20 sm:max-w-md">
          <AlertDialogHeader className="bg-red-600 px-5 pb-4 pt-5 text-left">
            <AlertDialogTitle className="text-lg font-bold leading-snug text-white">
              Hủy ca đang làm?
            </AlertDialogTitle>
            <AlertDialogDescription className="mt-1 text-sm leading-5 text-red-50">
              Dùng khi bạn đã vào nhầm ca và muốn vào làm lại đúng ca.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-3 px-5 py-4">
            <div className="flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-red-950">
              <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600" aria-hidden="true" />
              <div>
                <p className="text-base font-bold leading-6">Ca này sẽ không tính lương</p>
                <p className="mt-1 text-sm font-medium leading-5 text-red-800">
                  Hệ thống sẽ ghi nhận ca đã hủy và mở lại nút Vào làm.
                </p>
              </div>
            </div>
          </div>
          <AlertDialogFooter className="grid grid-cols-2 gap-3 px-5 pb-5 pt-3">
            <AlertDialogCancel
              disabled={isPending}
              className="mt-0 h-12 w-full rounded-lg border-slate-200 bg-white text-base font-bold text-slate-900 hover:bg-slate-50"
            >
              Quay lại
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              className="h-12 w-full rounded-lg bg-red-600 text-base font-bold text-white shadow-sm shadow-red-900/15 hover:bg-red-700"
              onClick={(event) => {
                event.preventDefault();
                handleCancelCurrentShift();
              }}
            >
              {cancelCurrentAttendanceMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <AlertCircle className="mr-2 h-4 w-4" />
              )}
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
            <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-3">
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
                  onClick={() => {
                    // Restart the watch (re-arms GPS, re-prompts permission if the
                    // failure was a denial) before re-attempting, so "Thử lại"
                    // recovers from a dead watch and not just transient errors.
                    location.retry();
                    setLocationIssue(null);
                    handleAction(actionType);
                  }}
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
          {locationPreview}
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
          <div className="grid grid-cols-[0.9fr_1.1fr] gap-3">
            <Button
              type="button"
              size="lg"
              variant="outline"
              className="h-14 rounded-xl border-red-200 bg-red-50 text-[17px] font-bold text-red-700 hover:bg-red-100 hover:text-red-800"
              disabled={isPending}
              onClick={() => setShowCancelShiftConfirm(true)}
            >
              <AlertCircle className="mr-2 h-5 w-5 shrink-0" />
              Hủy ca
            </Button>
            <Button
              size="lg"
              className="h-14 rounded-xl bg-slate-950 text-[18px] font-bold text-white shadow-lg shadow-slate-900/15 hover:bg-slate-800"
              disabled={isPending}
              onClick={() => handleAction("check_out")}
            >
              {isPending && !cancelCurrentAttendanceMutation.isPending ? (
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
              ) : (
                <DoorOpen className="w-5 h-5 mr-2" />
              )}
              {isLocating ? "Đang lấy vị trí..." : "Tan ca"}
            </Button>
          </div>
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
          {locationPreview}
          {/* Outside-window hint: show shift time + countdown instead of the button */}
          {!withinWindow ? (
            <div className="check-in-hint-fade rounded-xl border border-amber-200 bg-amber-50 p-4 shadow-sm">
              <div className="flex items-start gap-3">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-100 text-amber-600">
                  <Clock className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[18px] font-bold leading-6 text-slate-950">
                    Chưa đến giờ vào làm
                  </p>
                  <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                    {shiftStart
                      ? `Ca làm việc bắt đầu lúc ${shiftStart}. Giờ chấm công từ ${checkInWindowStart} đến ${checkInWindowEnd}.`
                      : "Chưa có ca làm việc được cấu hình."}
                  </p>
                  {secondsUntilWindow != null && (
                    <p className="mt-1 text-[15px] font-semibold leading-5 text-amber-700">
                      {(() => {
                        const m = Math.floor(secondsUntilWindow / 60);
                        const s = secondsUntilWindow % 60;
                        if (m >= 60) {
                          const h = Math.floor(m / 60);
                          return `Còn khoảng ${h} giờ ${m % 60} phút nữa.`;
                        }
                        if (m > 0) return `Còn khoảng ${m} phút nữa.`;
                        return `Còn khoảng ${s} giây nữa.`;
                      })()}
                    </p>
                  )}
                </div>
              </div>
            </div>
          ) : gpsReady ? (
            <>
              <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 shadow-sm">
                <div className="flex items-start gap-3">
                  <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-100 text-employee">
                    <MapPin className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-[18px] font-bold leading-6 text-slate-950">
                      Sẵn sàng vào làm
                    </p>
                    <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                      Vị trí đã xác định. Bấm Vào làm để bắt đầu ca.
                    </p>
                  </div>
                </div>
              </div>
              <Button
                size="lg"
                className={`h-14 w-full rounded-xl bg-employee text-[18px] font-bold text-white shadow-lg hover:bg-employee-600 ${showReadyPop ? "check-in-ready-pop" : ""}`}
                style={{ boxShadow: `0 10px 24px ${EMPLOYEE_BRAND_COLOR}30` }}
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
          ) : gpsAcquiring ? (
            <>
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                <div className="flex items-start gap-3">
                  <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-employee">
                    <MapPin className="h-5 w-5 animate-pulse" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-[18px] font-bold leading-6 text-slate-950">
                      Đang xác định vị trí...
                    </p>
                    <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                      {locationProgress?.status === "excellent"
                        ? "Tín hiệu rất tốt — sẵn sàng chấm công."
                        : locationProgress?.status === "acceptable"
                          ? "Tín hiệu khá — đang ổn định thêm."
                          : "Đang tìm GPS. Hãy đứng ngoài trời nếu cần."}
                    </p>
                  </div>
                </div>
              </div>
              <Button
                size="lg"
                className="check-in-warming h-14 w-full rounded-xl bg-employee text-[18px] font-bold text-white shadow-lg"
                style={{ boxShadow: `0 4px 12px ${EMPLOYEE_BRAND_COLOR}20` }}
                disabled
              >
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
                Đang lấy vị trí...
              </Button>
            </>
          ) : location.fatalError ? (
            <div className="rounded-xl border border-red-200 bg-red-50 p-4 shadow-sm">
              <div className="flex items-start gap-3">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-red-100 text-red-600">
                  <MapPin className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[18px] font-bold leading-6 text-slate-950">
                    Không lấy được vị trí
                  </p>
                  <p className="mt-1 text-[16px] font-medium leading-6 text-slate-600">
                    {location.fatalError.title}. {location.fatalError.description}
                  </p>
                </div>
              </div>
            </div>
          ) : (
            <>
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
                style={{ boxShadow: `0 10px 24px ${EMPLOYEE_BRAND_COLOR}30` }}
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
          )}
        </div>
      )}
    </div>
  );
}
