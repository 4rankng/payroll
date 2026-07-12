import { lazy, Suspense, useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import { AlertCircle, BadgeCheck, BriefcaseBusiness, CalendarClock, ChevronDown, Clock, DoorOpen, Loader2, MapPin, RotateCcw, Settings, WalletCards } from "lucide-react";
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
import {
  getTimingErrorGuidance,
  hasLocationErrorGuidance,
  scrollToLocationGuidance,
  type TimingErrorGuidance,
} from "@/utils/attendance-error-guidance";
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
import { getCheckInGeofenceGuidance, type CheckInGeofenceGuidance } from "@/utils/checkInGeofenceGuidance";
import { formatDistanceMeters } from "@/utils/geoDistance";
import { parseEpochMs } from "@/utils/vn-time";
import type { AttendanceScheduleWindow, CheckInTarget } from "@/types/api/auth.types";
import {
  EmployeeAttendanceActionDock,
  type AttendanceDockAction,
} from "./EmployeeAttendanceActionDock";

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
  /** Advisory shift window (from the profile DTO). Null/absent = no timing gate. */
  shiftStart?: string;
  shiftEnd?: string;
  checkInWindowStart?: string;
  checkInWindowEnd?: string;
  checkOutWindowStart?: string;
  checkOutWindowEnd?: string;
  scheduleWindows?: AttendanceScheduleWindow[];
  activeScheduleWindow?: AttendanceScheduleWindow | null;
  onAdvanceRequest: () => void;
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
          <p className="employee-type-pill whitespace-nowrap uppercase text-slate-500">{detail.label}</p>
          <p className="employee-type-body mt-0.5 font-semibold">{detail.value}</p>
        </div>
      ))}
    </div>
  );
}

function MapFallback() {
  return (
    <div className="employee-type-body-sm rounded-xl border border-sky-200 bg-white px-3 py-3 font-medium text-slate-600">
      Đang tải bản đồ vị trí...
    </div>
  );
}

function formatAccuracy(accuracy: number | undefined): string | null {
  if (typeof accuracy !== "number") return null;
  return `${Math.round(accuracy)}m`;
}

function getOutsideGeofenceInstruction(guidance: CheckInGeofenceGuidance): string | null {
  if (guidance.status !== "outside") return null;
  const gateName = guidance.nearestGate?.name || "cổng chấm công gần nhất";
  const distance = formatDistanceMeters(guidance.distanceMeters);
  return `Hãy di chuyển gần hơn tới ${gateName}. Cách ${distance}.`;
}

function getLocationAcquisitionMessage(
  progress: LocationAcquisitionProgress | null,
  guidance: CheckInGeofenceGuidance
): string {
  const outsideInstruction = getOutsideGeofenceInstruction(guidance);
  if (outsideInstruction) return outsideInstruction;

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

// Schedule timestamps are API instants. Render them with the device-local
// HH:mm formatter used elsewhere in this card; dates and timezone suffixes are
// intentionally not shown in the compact employee reference.

export function AttendanceReference({
  checkInTarget,
  shiftStart,
  shiftEnd,
  checkInWindowStart,
  checkInWindowEnd,
  checkOutWindowStart,
  checkOutWindowEnd,
  scheduleWindows,
  activeScheduleWindow,
}: Pick<
  EmployeeCheckInCardProps,
  | "checkInTarget"
  | "shiftStart"
  | "shiftEnd"
  | "checkInWindowStart"
  | "checkInWindowEnd"
  | "checkOutWindowStart"
  | "checkOutWindowEnd"
  | "scheduleWindows"
  | "activeScheduleWindow"
>) {
  const legacySchedule =
    shiftStart && shiftEnd && checkInWindowStart && checkInWindowEnd && checkOutWindowStart && checkOutWindowEnd
      ? [{
          shift_start: shiftStart,
          shift_end: shiftEnd,
          check_in_window_start: checkInWindowStart,
          check_in_window_end: checkInWindowEnd,
          check_out_window_start: checkOutWindowStart,
          check_out_window_end: checkOutWindowEnd,
        }]
      : [];
  const schedules = scheduleWindows?.length ? scheduleWindows : legacySchedule;
  const gates = checkInTarget?.gates ?? [];
  const activeIndex = schedules.findIndex(
    (schedule) =>
      activeScheduleWindow?.shift_start === schedule.shift_start &&
      activeScheduleWindow.shift_end === schedule.shift_end
  );
  const [selectedShiftIndex, setSelectedShiftIndex] = useState(activeIndex >= 0 ? activeIndex : 0);
  const selectedSchedule = schedules[selectedShiftIndex] ?? schedules[0];
  const shiftSelectorId = useId();
  const selectedShiftTabId = `${shiftSelectorId}-tab-${selectedShiftIndex}`;
  const selectedShiftPanelId = `${shiftSelectorId}-panel`;
  const additionalGatesId = `${shiftSelectorId}-additional-gates`;
  const [showAdditionalGates, setShowAdditionalGates] = useState(false);

  if (schedules.length === 0 && gates.length === 0) return null;

  return (
    <section className="mt-6 space-y-6 border-t border-slate-200/80 pt-5" aria-label="Thông tin chấm công">
      {schedules.length > 0 ? (
        <div>
          <h3 className="flex items-center gap-2.5 text-xl font-bold tracking-tight text-slate-950">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
              <CalendarClock className="h-5 w-5" aria-hidden="true" />
            </span>
            Ca làm việc
          </h3>
          {schedules.length > 1 ? (
            <div className="mt-5 grid grid-cols-2 rounded-2xl border border-slate-200 bg-slate-100/80 p-1.5 shadow-inner shadow-slate-200/60" role="tablist" aria-label="Chọn ca làm">
              {schedules.map((schedule, index) => (
                <button
                  key={`${schedule.shift_start}-${schedule.shift_end}`}
                  type="button"
                  role="tab"
                  id={`${shiftSelectorId}-tab-${index}`}
                  aria-controls={selectedShiftPanelId}
                  aria-selected={selectedShiftIndex === index}
                  onClick={() => setSelectedShiftIndex(index)}
                  className={`employee-type-action min-h-12 rounded-xl font-semibold transition-all duration-200 ${selectedShiftIndex === index ? "bg-emerald-600 text-white shadow-[0_6px_16px_rgba(5,150,105,0.22)]" : "text-slate-600 hover:bg-white hover:text-slate-950"}`}
                >
                  Ca {index + 1}
                </button>
              ))}
            </div>
          ) : null}
          {selectedSchedule ? (
            <dl
              id={selectedShiftPanelId}
              aria-labelledby={selectedShiftTabId}
              className="mt-5 grid grid-cols-3 divide-x divide-slate-100 rounded-[22px] border border-slate-200/80 bg-white px-2 py-5 shadow-[0_8px_24px_rgba(15,23,42,0.06)]"
              role="tabpanel"
            >
              {[
                { label: "Ca làm", startIso: selectedSchedule.shift_start, endIso: selectedSchedule.shift_end },
                { label: "Vào làm", startIso: selectedSchedule.check_in_window_start, endIso: selectedSchedule.check_in_window_end },
                { label: "Tan ca", startIso: selectedSchedule.check_out_window_start, endIso: selectedSchedule.check_out_window_end },
              ].map((row) => {
                const start = safeFormatTime(row.startIso);
                const end = safeFormatTime(row.endIso);
                return (
                  <div key={row.label} className="min-w-0 px-2 text-center">
                    <dt className="employee-type-pill uppercase text-slate-500">{row.label}</dt>
                    <dd className="mt-2 text-lg font-bold tracking-tight text-slate-950">{start}</dd>
                    <dd className="employee-type-body-sm mt-1 text-slate-500">{end}</dd>
                  </div>
                );
              })}
            </dl>
          ) : null}
        </div>
      ) : null}
      {gates.length > 0 ? (
        <div>
          <h3 className="flex items-center gap-2.5 text-xl font-bold tracking-tight text-slate-950">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
              <MapPin className="h-5 w-5" aria-hidden="true" />
            </span>
            {gates.length} điểm chấm công
          </h3>
          <ul className="mt-5 grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_0.48fr] gap-2.5" aria-label="Các điểm chấm công">
            {gates.slice(0, 2).map((gate, index) => (
              <li
                key={`${gate.name}-${gate.lat}-${gate.lng}`}
                className="employee-type-body-sm flex h-14 items-center rounded-xl border border-slate-200/80 bg-white px-3 font-semibold leading-5 text-slate-900 shadow-[0_5px_14px_rgba(15,23,42,0.04)]"
              >
                {gate.name || `Điểm chấm công ${index + 1}`}
              </li>
            ))}
            {gates.length > 2 ? (
              <li>
                <button
                  type="button"
                  aria-label={`${showAdditionalGates ? "Ẩn" : "Hiển thị"} ${gates.length - 2} điểm chấm công khác`}
                  aria-controls={additionalGatesId}
                  aria-expanded={showAdditionalGates}
                  onClick={() => setShowAdditionalGates((visible) => !visible)}
                  className={`employee-type-action flex h-14 w-full items-center justify-center rounded-xl border px-3 text-center font-semibold transition-all duration-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-700 ${showAdditionalGates ? "border-slate-300 bg-slate-100 text-slate-950 shadow-inner shadow-slate-200/70" : "border-slate-200/80 bg-[#fffef9] text-slate-900 shadow-[0_5px_14px_rgba(15,23,42,0.04)] hover:border-slate-300 hover:bg-white"}`}
                >
                  <strong className="text-xl leading-none">+{gates.length - 2}</strong>
                </button>
              </li>
            ) : null}
          </ul>
          {gates.length > 2 && showAdditionalGates ? (
            <ul id={additionalGatesId} className="attendance-gates-reveal mt-3 grid gap-2 sm:grid-cols-2" aria-label="Các điểm chấm công khác">
              {gates.slice(2).map((gate, index) => (
                <li
                  key={`${gate.name}-${gate.lat}-${gate.lng}`}
                  className="employee-type-body-sm flex h-14 items-center rounded-xl border border-slate-200/80 bg-[#fffef9] px-4 font-semibold leading-5 text-slate-900 shadow-[0_5px_14px_rgba(15,23,42,0.04)]"
                >
                  {gate.name || `Điểm chấm công ${index + 3}`}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

export function EmployeeCheckInCard({
  className,
  checkInTarget,
  shiftStart,
  shiftEnd,
  checkInWindowStart,
  checkInWindowEnd,
  checkOutWindowStart,
  checkOutWindowEnd,
  scheduleWindows,
  activeScheduleWindow,
  onAdvanceRequest,
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
  const [timingGuidance, setTimingGuidance] = useState<TimingErrorGuidance | null>(null);
  const [noSalaryReason, setNoSalaryReason] = useState<string | null>(null);
  const [showNoSalaryConfirm, setShowNoSalaryConfirm] = useState(false);
  const [showCancelShiftConfirm, setShowCancelShiftConfirm] = useState(false);
  const [showLocationMap, setShowLocationMap] = useState(false);
  const [attendanceConfirmation, setAttendanceConfirmation] = useState<{ action: "check_in" | "check_out"; time: string } | null>(null);
  const [showWindowOpen, setShowWindowOpen] = useState(false);
  const locationMapRegionId = useId();
  const locationMapRef = useRef<HTMLDivElement>(null);
  // Synchronous in-flight guard. The button's `disabled` only takes effect after
  // the next render, so a rapid double-tap (common on mobile) can fire handleAction
  // twice before `isLocating`/`isPending` flips — sending a second request that the
  // backend rejects with 400 ("Bạn đã vào làm/tan ca rồi"), producing a duplicate
  // toast. This ref is set in the same call stack, closing that race.
  const submittingRef = useRef(false);
  // Brief lockout after a GPS/geofence checkout failure. It gives the phone's GPS
  // a few seconds to settle before the next attempt and stops a frantic burst of
  // retries (observed six identical check-out failures in 42 s on demo) from
  // spamming the backend. The backend dedups the same window; this is the UX half.
  const CHECKOUT_GPS_COOLDOWN_MS = 10_000;
  const [checkoutCooldownUntil, setCheckoutCooldownUntil] = useState<number | null>(null);
  const checkoutCooldownTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attendanceConfirmationTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const checkoutCoolingDown = checkoutCooldownUntil !== null && Date.now() < checkoutCooldownUntil;
  const armCheckoutCooldown = () => {
    setCheckoutCooldownUntil(Date.now() + CHECKOUT_GPS_COOLDOWN_MS);
    if (checkoutCooldownTimer.current) clearTimeout(checkoutCooldownTimer.current);
    checkoutCooldownTimer.current = setTimeout(() => setCheckoutCooldownUntil(null), CHECKOUT_GPS_COOLDOWN_MS);
  };
  useEffect(() => {
    const timer = checkoutCooldownTimer;
    return () => {
      if (timer.current) clearTimeout(timer.current);
      if (attendanceConfirmationTimer.current) clearTimeout(attendanceConfirmationTimer.current);
    };
  }, []);

  const attendance = attendanceResponse?.data;
  const canStartCorrectShift = attendance?.status === "completed" && isConfirmedNoSalaryAttendance(attendance);
  // The continuous GPS watch runs whenever an action is conceptually possible and
  // STAYS running during a tap's submit-await. A fix warmed before the tap is then
  // submitted instantly, instead of the tap cold-starting a 25-30s acquisition
  // that times out before the phone's first GNSS fix — the dominant on-site check-
  // in failure. (The old canPreviewCheckLocation gate toggled off during isLocating,
  // which defeated warm reuse.)
  // Deliberately NOT gated on `isLoading`: the card's isLoading early-return only
  // swaps in a skeleton, but this hook still runs (hooks run before any return),
  // so letting the watch warm during the brief load makes the first actionable
  // tap instant. It also prevents a useTodayAttendance refetch mid-cold-tap from
  // toggling the watch off and stalling the awaiter for the full 30s.
  const locationEnabled =
    Boolean(checkInTarget) &&
    (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);
  const location = useContinuousLocation({
    target: checkInTarget,
    enabled: locationEnabled,
  });
  // Aliases so the existing JSX (converging-accuracy banner, map preview) reads the
  // continuous watch's reactive state unchanged.
  const locationProgress: LocationAcquisitionProgress | null = location.progress;
  const checkInGuidance = useMemo(
    () => getCheckInGeofenceGuidance(checkInTarget, location.sample),
    [checkInTarget, location.sample]
  );
  const outsideGeofenceInstruction = getOutsideGeofenceInstruction(checkInGuidance);

  const showLocationGuidance = useCallback(
    (issue: LocationPermissionIssue) => {
      setLocationIssue(issue);
      if (checkInTarget?.gates.length) {
        setShowLocationMap(true);
        scrollToLocationGuidance(locationMapRef.current);
      }
    },
    [checkInTarget]
  );

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

  const showAttendanceConfirmation = (action: "check_in" | "check_out") => {
    setAttendanceConfirmation({ action, time: format(new Date(), "HH:mm") });
    if (attendanceConfirmationTimer.current) clearTimeout(attendanceConfirmationTimer.current);
    attendanceConfirmationTimer.current = setTimeout(() => setAttendanceConfirmation(null), 3500);
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
          showLocationGuidance(
            createPoorAccuracyLocationIssue(
              location.progress?.bestAccuracy,
              location.progress?.requiredAccuracyMeters
            )
          );
          toast({ title: "Chưa thể chấm công", variant: "destructive" });
          armCheckoutCooldown();
          return;
        }
        if (isGeofenceOutsideMessage(message)) {
          showLocationGuidance(createOutsideGeofenceLocationIssue());
          toast({ title: "Chưa thể chấm công", variant: "destructive" });
          armCheckoutCooldown();
          return;
        }
        if (!options?.confirmNoSalary && canConfirmNoSalaryCheckout(message)) {
          // First attempt outside the checkout window: show the real reason and
          // offer the confirmed no-salary path. The dialog is the UI — no toast.
          setNoSalaryReason(message);
          setShowNoSalaryConfirm(true);
        } else {
          if (hasLocationErrorGuidance(error)) {
            showLocationGuidance(createOutsideGeofenceLocationIssue());
            return;
          }
          const timing = getTimingErrorGuidance(error, type);
          if (timing) {
            setTimingGuidance(timing);
            return;
          }
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
          showLocationGuidance(
            createPoorAccuracyLocationIssue(
              location.progress?.bestAccuracy,
              location.progress?.requiredAccuracyMeters
            )
          );
        } else if (isGeofenceOutsideMessage(message)) {
          showLocationGuidance(createOutsideGeofenceLocationIssue());
        } else if (getTimingErrorGuidance(error, type)) {
          setTimingGuidance(getTimingErrorGuidance(error, type)!);
        } else if (hasLocationErrorGuidance(error)) {
          showLocationGuidance(createOutsideGeofenceLocationIssue());
        } else {
          toast({ title: message || "Không thể vào làm", variant: "destructive" });
        }
      }
      return;
    }

    // Geolocation error from awaitSubmitReady rejection (timeout / inaccurate) or
    // a fatal watch error (permission denied). Same recovery UX + failed-attempt
    // logging as before.
    const issue = getLocationPermissionIssue(error);
    showLocationGuidance(issue);
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
    if (type === "check_out" && checkoutCoolingDown) return;
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
      showAttendanceConfirmation(type);
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

    // Cold path: submit the first fresh sub-50m device fix to the backend even
    // when the client-side guidance says "outside". The server is the geofence
    // authority and records validation failures in attendance_failed_attempts;
    // waiting for an "inside" sample here would leave outside-geofence taps
    // stuck client-side with no backend audit row.
    setIsLocating(true);
    try {
      const sample = await location.awaitAccurateSample();
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
  // Window gate. The server sends absolute instants (RFC 3339 with a +07:00
  // offset), so we compare epoch milliseconds directly — this is independent of
  // the device timezone. A worker whose phone is set to UTC must still be able
  // to check in to a 20:00 Vietnam shift at 20:17 Vietnam time. The previous
  // implementation parsed HH:mm strings and compared getHours() (device-local),
  // which mis-gated anyone whose device timezone was not Asia/Ho_Chi_Minh.
  const withinWindow = useMemo(() => {
    const startMs = parseEpochMs(checkInWindowStart);
    const endMs = parseEpochMs(checkInWindowEnd);
    if (startMs === null || endMs === null) return true; // no shift configured
    const now = nowTick.current;
    return now >= startMs && now <= endMs;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [checkInWindowStart, checkInWindowEnd, nowTick.current]);

  // Seconds until the check-in window opens, for the outside-window hint
  // countdown. Recomputed on the minute tick (minute precision is sufficient;
  // showing seconds would require a 1s timer and is noisy on mobile). Computed
  // from the absolute window-start epoch so it is correct regardless of device
  // timezone.
  const secondsUntilWindow = useMemo(() => {
    const startMs = parseEpochMs(checkInWindowStart);
    if (startMs === null || withinWindow) return null;
    const now = nowTick.current;
    if (startMs <= now) return null; // window already passed
    return Math.round((startMs - now) / 1000);
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
  const previousWithinWindow = useRef<boolean | null>(null);
  useEffect(() => {
    if (previousWithinWindow.current === false && withinWindow) {
      setShowWindowOpen(true);
      const timer = setTimeout(() => setShowWindowOpen(false), 2600);
      previousWithinWindow.current = withinWindow;
      return () => clearTimeout(timer);
    }
    previousWithinWindow.current = withinWindow;
  }, [withinWindow]);

  if (isLoading) {
    return (
      <>
        <div
          className={`rounded-2xl border border-slate-200 bg-white p-4 shadow-sm ${className ?? ""}`}
          style={style}
        >
          <div className="animate-pulse flex flex-col items-center justify-center space-y-4 h-32">
            <div className="h-6 w-32 bg-gray-200 rounded"></div>
            <div className="h-10 w-48 bg-gray-200 rounded-full"></div>
          </div>
        </div>
        <EmployeeAttendanceActionDock
          action="loading"
          actionLabel="Đang tải chấm công…"
          actionDisabled
          onAdvanceRequest={onAdvanceRequest}
        />
      </>
    );
  }

  const salaryRecorded = attendance ? isSalaryRecorded(attendance) : false;
  // The next action is fully determined by the attendance status — there is no
  // need to track the last-tapped action separately (doing so defaulted it to
  // "check_in" and could mislabel the recovery CTA on a fresh check_out).
  const actionType = attendance?.status === "checked_in" ? "check_out" : "check_in";
  const locationRecoveryText = "Thử lại";
  // Geofence failures already expand and scroll to the checkpoint map. Keeping
  // a second "Ngoài khu vực / Thử lại" banner above it duplicates the same
  // guidance, so reserve this recovery panel for device/GPS permission issues.
  const showLocationRecovery =
    locationIssue?.title !== "Ngoài khu vực" &&
    Boolean(locationIssue) &&
    (attendance?.status === "checked_in" || !attendance || canStartCorrectShift);
  const visibleLocationSample = location.sample;

  const locationPreview = checkInTarget && showLocationMap ? (
    <div
      id={locationMapRegionId}
      className="mt-2"
    >
      <Suspense fallback={<MapFallback />}>
        <EmployeeLocationMap target={checkInTarget} sample={visibleLocationSample} />
      </Suspense>
    </div>
  ) : null;
  const locationMapDisclosure = checkInTarget ? (
    <div ref={locationMapRef} className="mt-2 scroll-mt-24 border-t border-slate-100 pt-1" tabIndex={-1}>
      <Button
        type="button"
        variant="ghost"
        className="employee-type-action h-11 w-full justify-between rounded-lg px-2 text-sky-700 hover:bg-sky-50 hover:text-sky-800"
        aria-expanded={showLocationMap}
        aria-controls={locationMapRegionId}
        onClick={() => setShowLocationMap((isOpen) => !isOpen)}
      >
        <span className="inline-flex items-center gap-2">
          <MapPin className="h-4 w-4" aria-hidden="true" />
          {showLocationMap ? "Ẩn bản đồ" : "Xem bản đồ"}
        </span>
        <ChevronDown
          className={`h-4 w-4 transition-transform ${showLocationMap ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
      </Button>
      {locationPreview}
    </div>
  ) : null;
  const attendanceReference = (
    <AttendanceReference
      checkInTarget={checkInTarget}
      shiftStart={shiftStart}
      shiftEnd={shiftEnd}
      checkInWindowStart={checkInWindowStart}
      checkInWindowEnd={checkInWindowEnd}
      checkOutWindowStart={checkOutWindowStart}
      checkOutWindowEnd={checkOutWindowEnd}
      scheduleWindows={scheduleWindows}
      activeScheduleWindow={activeScheduleWindow}
    />
  );
  const showRecoveryDescription =
    Boolean(locationIssue) &&
    (!checkInTarget || locationIssue?.requiresSettings || locationIssue?.type !== "unknown");
  const noSalaryWindow = getCheckoutWindowSummary(noSalaryReason);
  const salaryIssue = getAttendanceIssueSummary(
    attendance?.salary_reject_reason || attendance?.salary_message,
    "Kiểm tra lại khung giờ ca làm."
  );
  const rejectedIssue = getAttendanceIssueSummary(attendance?.salary_reject_reason);
  let dockAction: AttendanceDockAction = "attention";
  let dockActionLabel = "Cần kiểm tra";
  let dockActionDisabled = true;
  let handleDockAttendanceAction: (() => void) | undefined;

  if (attendance?.status === "checked_in") {
    dockAction = checkoutCoolingDown || isLocating ? "loading" : "check_out";
    dockActionLabel = checkoutCoolingDown
      ? "Đang chờ GPS…"
      : isLocating
        ? "Đang lấy vị trí…"
        : "Tan ca";
    dockActionDisabled = isPending || checkoutCoolingDown;
    handleDockAttendanceAction = () => handleAction("check_out");
  } else if (!attendance || canStartCorrectShift) {
    if (!withinWindow) {
      dockActionLabel = "Chưa đến giờ";
    } else if (gpsAcquiring || isLocating) {
      dockAction = "loading";
      dockActionLabel = "Đang kiểm tra GPS…";
    } else if (location.fatalError) {
      dockActionLabel = "Kiểm tra vị trí";
    } else {
      dockAction = "check_in";
      dockActionLabel = "Vào làm";
      dockActionDisabled = isPending;
      handleDockAttendanceAction = () => handleAction("check_in");
    }
  } else if (attendance?.status === "completed") {
    dockAction = "completed";
    dockActionLabel = "Đã tan ca";
  }
  if (attendanceConfirmation) {
    dockAction = "completed";
    dockActionLabel = `${attendanceConfirmation.action === "check_in" ? "Đã vào làm" : "Đã tan ca"} ${attendanceConfirmation.time}`;
    dockActionDisabled = true;
    handleDockAttendanceAction = undefined;
  }

  return (
    <div className={className} style={style}>
      {showWindowOpen ? (
        <div className="attendance-window-open mb-3 flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-emerald-950" role="status">
          <Clock className="h-4 w-4 shrink-0 text-emerald-700" aria-hidden="true" />
          <span className="employee-type-body-sm font-semibold">Đã đến giờ chấm công</span>
        </div>
      ) : null}
      {attendanceConfirmation ? (
        <div className="attendance-action-confirm mb-3 flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-emerald-950" role="status">
          <BadgeCheck className="h-5 w-5 shrink-0 text-emerald-700" aria-hidden="true" />
          <span className="employee-type-body-sm font-semibold">
            {attendanceConfirmation.action === "check_in" ? "Đã vào làm" : "Đã tan ca"} lúc {attendanceConfirmation.time}
          </span>
        </div>
      ) : null}
      <AlertDialog open={showNoSalaryConfirm} onOpenChange={setShowNoSalaryConfirm}>
        <AlertDialogContent className="max-w-[calc(100vw-32px)] border-employee-100 bg-white shadow-2xl shadow-employee-900/20 sm:max-w-md">
          <AlertDialogHeader className="bg-employee-900 px-5 pb-4 pt-5 text-left">
            <AlertDialogTitle className="employee-type-hero-title font-semibold text-white">
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
                      <p className="employee-type-strong mt-1 text-slate-950">
                        {noSalaryWindow.checkInTime}
                      </p>
                    </div>
                  ) : null}
                  <div className={noSalaryWindow.checkInTime ? "" : "col-span-2"}>
                    <p className="text-xs font-medium leading-4 text-employee-700">Tan ca</p>
                    <p className="employee-type-strong mt-1 text-slate-950">
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
                <p className="employee-type-strong">Ca này sẽ không tính lương</p>
                <p className="mt-1 text-sm font-medium leading-5 text-red-800">
                  Chỉ tiếp tục nếu bạn muốn hủy ca hiện tại.
                </p>
              </div>
            </div>
          </div>
          <AlertDialogFooter className="grid grid-cols-2 gap-3 px-5 pb-5 pt-3">
            <AlertDialogCancel
              disabled={isPending}
              className="employee-type-action mt-0 h-12 w-full rounded-lg border-employee-200 bg-white font-semibold text-employee-900 hover:bg-employee-50 hover:text-employee-900"
            >
              Quay lại
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              className="employee-type-action h-12 w-full rounded-lg bg-red-600 font-semibold text-white shadow-sm shadow-red-900/15 hover:bg-red-700"
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
      <AlertDialog
        open={Boolean(timingGuidance)}
        onOpenChange={(open) => {
          if (!open) setTimingGuidance(null);
        }}
      >
        <AlertDialogContent className="max-w-[calc(100vw-32px)] border-amber-200 bg-white shadow-2xl shadow-amber-900/10 sm:max-w-md">
          <AlertDialogHeader className="bg-amber-50 px-5 pb-4 pt-5 text-left">
            <AlertDialogTitle className="employee-type-hero-title text-amber-950">
              {timingGuidance?.title}
            </AlertDialogTitle>
            <AlertDialogDescription className="mt-1 text-sm leading-5 text-amber-900">
              {timingGuidance?.message}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {timingGuidance?.windowStart && timingGuidance.windowEnd ? (
            <div className="px-5 py-4">
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-950">
                <p className="employee-type-label-caps font-semibold text-amber-800">
                  {timingGuidance.actionLabel}
                </p>
                <p className="employee-type-strong mt-1 text-slate-950">
                  {timingGuidance.windowStart} - {timingGuidance.windowEnd}
                </p>
              </div>
            </div>
          ) : null}
          <AlertDialogFooter className="px-5 pb-5">
            <AlertDialogAction
              className="employee-type-action h-12 w-full rounded-lg bg-[#07883F] font-semibold text-white hover:bg-[#067647]"
              onClick={() => setTimingGuidance(null)}
            >
              Đã hiểu
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={showCancelShiftConfirm} onOpenChange={setShowCancelShiftConfirm}>
        <AlertDialogContent className="max-w-[calc(100vw-32px)] border-red-100 bg-white shadow-2xl shadow-red-900/20 sm:max-w-md">
          <AlertDialogHeader className="bg-red-600 px-5 pb-4 pt-5 text-left">
            <AlertDialogTitle className="employee-type-hero-title font-semibold text-white">
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
                <p className="employee-type-strong">Ca này sẽ không tính lương</p>
                <p className="mt-1 text-sm font-medium leading-5 text-red-800">
                  Hệ thống sẽ ghi nhận ca đã hủy và mở lại nút Vào làm.
                </p>
              </div>
            </div>
          </div>
          <AlertDialogFooter className="grid grid-cols-2 gap-3 px-5 pb-5 pt-3">
            <AlertDialogCancel
              disabled={isPending}
              className="employee-type-action mt-0 h-12 w-full rounded-lg border-slate-200 bg-white font-semibold text-slate-900 hover:bg-slate-50"
            >
              Quay lại
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              className="employee-type-action h-12 w-full rounded-lg bg-red-600 font-semibold text-white shadow-sm shadow-red-900/15 hover:bg-red-700"
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
        <div className="mb-3 rounded-xl border border-sky-200 bg-sky-50 p-3 text-sky-950">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-white text-sky-700">
              <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="employee-type-card-title font-semibold">Đang kiểm tra vị trí...</p>
              <p className="employee-type-body-sm mt-0.5 font-medium text-sky-800">
                {getLocationAcquisitionMessage(locationProgress, checkInGuidance)}
              </p>
            </div>
          </div>
          {locationProgress?.sampleCount ? (
            <div className="mt-2 grid grid-cols-3 gap-2">
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="employee-type-label-caps font-semibold text-sky-700">Sai số</p>
                <p className="employee-type-body mt-0.5 truncate font-semibold text-sky-950">
                  {formatAccuracy(locationProgress.bestAccuracy) || "--"}
                </p>
              </div>
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="employee-type-label-caps font-semibold text-sky-700">Cần</p>
                <p className="employee-type-body mt-0.5 truncate font-semibold text-sky-950">
                  {"<"}
                  {formatAccuracy(locationProgress.requiredAccuracyMeters) || "50m"}
                </p>
              </div>
              <div className="min-w-0 rounded-lg bg-white/80 px-2.5 py-2">
                <p className="employee-type-label-caps font-semibold text-sky-700">Lần đo</p>
                <p className="employee-type-body mt-0.5 truncate font-semibold text-sky-950">
                  {locationProgress.sampleCount}
                </p>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
      {showLocationRecovery ? (
        <div className="mb-3 rounded-xl border border-amber-200 bg-amber-50 p-3 text-amber-950">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white text-amber-700">
              {locationIssue.requiresSettings ? <Settings className="h-5 w-5" /> : <MapPin className="h-5 w-5" />}
            </div>
            <div className="min-w-0 flex-1">
              <p className="employee-type-card-title truncate font-semibold">{locationIssue.title}</p>
              {showRecoveryDescription ? (
                <p className="employee-type-body-sm mt-0.5 font-medium text-amber-800">{locationIssue.description}</p>
              ) : null}
            </div>
              {locationIssue.canRetry ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="employee-type-action h-11 shrink-0 gap-2 rounded-lg border-amber-300 bg-white px-3 font-semibold text-amber-950 hover:bg-amber-100"
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
        <div className="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
          <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-700">
                <BadgeCheck className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="employee-type-card-title font-semibold text-slate-950">Ca hôm nay đã xong</p>
                <p className="employee-type-body-sm mt-0.5 font-semibold text-slate-600">
                  {safeFormatTime(attendance.check_in_time)} — {safeFormatTime(attendance.check_out_time)}
                </p>
              </div>
            </div>
          <div
            className={`mt-3 rounded-lg border px-3 py-2.5 ${
              salaryRecorded
                ? "border-emerald-200 bg-emerald-50 text-emerald-950"
                : "border-amber-200 bg-amber-50 text-amber-950"
            }`}
          >
            <div className="flex items-start gap-3">
              <div
                className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white ${
                  salaryRecorded ? "text-emerald-700" : "text-amber-700"
                }`}
              >
                {salaryRecorded ? <WalletCards className="h-4 w-4" /> : <AlertCircle className="h-4 w-4" />}
              </div>
              <div className="min-w-0 flex-1">
                <p className="employee-type-body font-semibold">
                  {salaryRecorded ? `Lương ca: ${formatCurrency(attendance.earning_amount)}` : salaryIssue.title}
                </p>
                <p className="employee-type-body-sm mt-0.5 font-medium opacity-85">
                  {salaryRecorded ? "Bạn có thể yêu cầu ứng lương nếu còn hạn mức." : salaryIssue.description}
                </p>
                {!salaryRecorded ? <IssueDetailChips details={salaryIssue.details} tone="amber" /> : null}
              </div>
            </div>
          </div>

          {canStartCorrectShift ? (
            <>
              {attendanceReference}
              {locationMapDisclosure}
              <Button
                size="lg"
                className="employee-type-action mt-2 hidden h-12 w-full rounded-xl bg-employee font-semibold text-white shadow-md hover:bg-employee-600 lg:inline-flex"
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
        <div className="rounded-2xl border border-[#B7E5C7] bg-white p-4 shadow-[0_2px_8px_rgba(16,24,40,0.06)]">
            <div className="flex items-start gap-3">
              <div className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#ECFDF3] text-[#067647]">
                <BriefcaseBusiness className="h-5 w-5" aria-hidden="true" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="employee-type-label-caps text-[#067647]">Đang trong ca</p>
                <p className="employee-type-hero-title mt-1 text-[#101828]">Đang làm việc</p>
                <p className="employee-type-body mt-1 text-[#475467]">
                  {safeFormatTime(attendance.check_in_time)} — hiện tại · Tan ca để ghi nhận lương
                </p>
              </div>
            </div>
          {attendanceReference}
          {locationMapDisclosure}
          <div className="mt-4 grid grid-cols-1 gap-2 lg:grid-cols-[0.8fr_1.2fr]">
            <Button
              type="button"
              size="lg"
              variant="outline"
              className="employee-type-action h-12 justify-start rounded-xl border border-transparent bg-transparent font-semibold text-[#B42318] hover:bg-[#FEF3F2] hover:text-[#B42318] lg:justify-center"
              disabled={isPending}
              onClick={() => setShowCancelShiftConfirm(true)}
            >
              <AlertCircle className="mr-2 h-5 w-5 shrink-0" />
              Hủy ca
            </Button>
            <Button
              size="lg"
              className="employee-type-action hidden h-12 rounded-xl bg-[#07883F] font-semibold text-white shadow-[0_8px_16px_-10px_rgba(6,118,71,0.7)] hover:bg-[#067647] lg:inline-flex"
              disabled={isPending || checkoutCoolingDown}
              onClick={() => handleAction("check_out")}
            >
              {isPending && !cancelCurrentAttendanceMutation.isPending ? (
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
              ) : checkoutCoolingDown ? (
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
              ) : (
                <DoorOpen className="w-5 h-5 mr-2" />
              )}
              {checkoutCoolingDown ? "Đang chờ GPS..." : isLocating ? "Đang lấy vị trí..." : "Tan ca"}
            </Button>
          </div>
        </div>
      ) : attendance?.status === "orphaned" ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-3 shadow-sm">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-white text-red-600">
              <AlertCircle className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <p className="employee-type-card-title font-semibold text-red-950">Ca cần kiểm tra</p>
              <p className="employee-type-body-sm mt-0.5 font-medium text-red-700">
                Bạn chưa bấm Tan ca cho ca trước. Hãy báo quản lý để kiểm tra lại lương.
              </p>
            </div>
          </div>
        </div>
      ) : attendance?.status === "rejected" ? (
        <div className="rounded-2xl border border-orange-200 bg-orange-50 p-3 shadow-sm">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-white text-orange-600">
              <AlertCircle className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="employee-type-card-title font-semibold text-orange-950">
                {rejectedIssue.title === "Cần quản lý kiểm tra" ? "Ca đã bị từ chối" : rejectedIssue.title}
              </p>
              <p className="employee-type-body-sm mt-0.5 font-medium text-orange-700">
                {rejectedIssue.description}
              </p>
              <IssueDetailChips details={rejectedIssue.details} tone="orange" />
            </div>
          </div>
        </div>
      ) : (
        <div>
          {/* Outside-window hint: show shift time + countdown instead of the button */}
          {!withinWindow ? (
            <div className="check-in-hint-fade overflow-hidden rounded-[28px] border border-amber-200/90 bg-[#fffdf5] p-5 shadow-[0_18px_42px_rgba(146,64,14,0.10)]">
              <div className="flex items-start gap-4">
                <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-amber-200/80 bg-[#fff3ca] text-amber-700 shadow-sm">
                  <Clock className="h-6 w-6" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="employee-type-label-caps font-bold tracking-[0.14em] text-amber-700">Ca làm tiếp theo</p>
                  <p className="employee-type-card-title mt-1 font-bold text-slate-950">Chưa đến giờ vào làm</p>
                  <p className="employee-type-body-sm mt-1.5 leading-6 text-slate-600">
                    {shiftStart
                      ? `Ca làm việc bắt đầu lúc ${safeFormatTime(shiftStart)}. Giờ chấm công từ ${safeFormatTime(checkInWindowStart)} đến ${safeFormatTime(checkInWindowEnd)}.`
                      : "Chưa có ca làm việc được cấu hình."}
                  </p>
                  {secondsUntilWindow != null && (
                    <p className="employee-type-body-sm mt-2 inline-flex rounded-full border border-amber-200 bg-white px-3 py-1 font-semibold text-amber-800 shadow-sm">
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
              {attendanceReference}
              {locationMapDisclosure}
            </div>
          ) : gpsReady ? (
              <div className="rounded-2xl border border-emerald-200 bg-emerald-50 p-3 shadow-sm">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-employee">
                    <MapPin className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="employee-type-card-title font-semibold text-slate-950">Sẵn sàng vào làm</p>
                    <p className="employee-type-body-sm mt-0.5 font-medium text-slate-600">Vị trí đã xác định</p>
                  </div>
                </div>
              {attendanceReference}
              {locationMapDisclosure}
              <Button
                size="lg"
                className={`employee-type-action mt-2 hidden h-12 w-full rounded-xl bg-employee font-semibold text-white shadow-md hover:bg-employee-600 lg:inline-flex ${showReadyPop ? "check-in-ready-pop" : ""}`}
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
            </div>
          ) : gpsAcquiring ? (
              <div className="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-employee">
                    <MapPin className="h-5 w-5 animate-pulse" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="employee-type-card-title font-semibold text-slate-950">Đang xác định vị trí...</p>
                    <p className="employee-type-body-sm mt-0.5 font-medium text-slate-600">
                      {outsideGeofenceInstruction || (locationProgress?.status === "excellent"
                        ? "Tín hiệu rất tốt — sẵn sàng chấm công."
                        : locationProgress?.status === "acceptable"
                          ? "Tín hiệu khá — đang ổn định thêm."
                          : "Đang tìm GPS. Hãy đứng ngoài trời nếu cần.")}
                    </p>
                  </div>
                </div>
              {attendanceReference}
              {locationMapDisclosure}
              <Button
                size="lg"
                className="employee-type-action check-in-warming mt-2 hidden h-12 w-full rounded-xl bg-employee font-semibold text-white shadow-md lg:inline-flex"
                style={{ boxShadow: `0 4px 12px ${EMPLOYEE_BRAND_COLOR}20` }}
                disabled
              >
                <Loader2 className="w-5 h-5 animate-spin mr-2" />
                Đang lấy vị trí...
              </Button>
            </div>
          ) : location.fatalError ? (
            <div className="rounded-2xl border border-red-200 bg-red-50 p-3 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-red-100 text-red-600">
                  <MapPin className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="employee-type-card-title font-semibold text-slate-950">Không lấy được vị trí</p>
                  <p className="employee-type-body-sm mt-0.5 font-medium text-slate-600">
                    {location.fatalError.title}. {location.fatalError.description}
                  </p>
                </div>
              </div>
              {attendanceReference}
              {locationMapDisclosure}
            </div>
          ) : (
              <div className="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-employee">
                    <MapPin className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="employee-type-card-title font-semibold text-slate-950">Sẵn sàng vào làm</p>
                    <p className="employee-type-body-sm mt-0.5 font-medium text-slate-600">Bấm Vào làm khi đã tới cổng</p>
                  </div>
                </div>
              {attendanceReference}
              {locationMapDisclosure}
              <Button
                size="lg"
                className="employee-type-action mt-2 hidden h-12 w-full rounded-xl bg-employee font-semibold text-white shadow-md hover:bg-employee-600 lg:inline-flex"
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
            </div>
          )}
        </div>
      )}
      <EmployeeAttendanceActionDock
        action={dockAction}
        actionLabel={dockActionLabel}
        actionDisabled={dockActionDisabled}
        onAdvanceRequest={onAdvanceRequest}
        onAttendanceAction={handleDockAttendanceAction}
      />
    </div>
  );
}
