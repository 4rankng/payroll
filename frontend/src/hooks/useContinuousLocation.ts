import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CheckInTarget } from "@/types/api/auth.types";
import { getCheckInGeofenceGuidance } from "@/utils/checkInGeofenceGuidance";
import {
  CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS,
  EMPLOYEE_ATTENDANCE_REQUIRED_ACCURACY_METERS,
  createGeolocationError,
  createInaccurateGeolocationError,
  forgetLocationGrant,
  getLocationPermissionState,
  hasRememberedLocationGrant,
  isSampleFresh,
  watchContinuousLocation,
  rememberLocationGrant,
  type ContinuousLocationHandle,
  type LocationAcquisitionProgress,
  type LocationPermissionState,
  type LocationSample,
} from "@/utils/geolocation";

export interface UseContinuousLocationOptions {
  /** The gate target. When null, the hook stays idle. */
  target: CheckInTarget | null | undefined;
  /** True when a check-in/out action is conceptually possible. The watch keeps
   *  running while enabled, INCLUDING during a tap's submit-await — that is what
   *  makes the tap instant. */
  enabled: boolean;
  /** Hard cap for awaitSubmitReady. Default 30s (matches the cold-acquire budget). */
  submitTimeoutMs?: number;
}

export interface UseContinuousLocationResult {
  /** Latest fresh sample (current position). Use for the map preview and as the
   *  submit payload. Null until the first fresh fix. */
  sample: LocationSample | null;
  /** Rolling acquisition progress for the converging-accuracy UX. */
  progress: LocationAcquisitionProgress | null;
  /** True iff `sample` is fresh, accurate to the attendance GPS threshold,
   *  and its guidance against the target gate is "inside". A warm tap
   *  with this true submits instantly. */
  isSubmitReady: boolean;
  /** True while the underlying watchPosition is active. */
  isWatching: boolean;
  /** Browser-reported permission state when the Permissions API is available. */
  permissionState: LocationPermissionState;
  /** True when the employee needs to explicitly ask the browser for location access. */
  needsPermission: boolean;
  /** A fatal geolocation error from the watch (permission denied). The card
   *  surfaces this via its existing recovery banner + failed-attempt log. Null
   *  while the watch is healthy. */
  fatalError: Pick<GeolocationPositionError, "code" | "message"> | null;
  /** Start GPS acquisition from an employee action, so the browser can display
   *  its native location-permission prompt in a clear user context. */
  requestPermission: () => void;
  /** Resolve with a submit-ready sample, or reject with a geolocation error on
   *  timeout / fatal error / abort. */
  awaitSubmitReady: (timeoutMs?: number) => Promise<LocationSample>;
  /** Resolve with the next fresh GPS sample within the attendance GPS accuracy
   *  threshold, regardless of geofence status. The server remains the
   *  geofence authority. */
  awaitAccurateSample: (timeoutMs?: number) => Promise<LocationSample>;
  /** True once the browser has definitively denied location for this page.
   *
   *  A PERMISSION_DENIED (code 1) means the decision is already made: no
   *  browser re-prompts after a denial, and iOS Safari additionally caches the
   *  per-page answer for the lifetime of the document. So an in-page retry can
   *  never recover — the only way to re-read the OS grant is a reload. Derived
   *  from `fatalError` rather than tracked separately, because a separate flag
   *  could desync and strand the worker on a button that does nothing. */
  requiresPageReload: boolean;
  /** Reload the document so the browser re-reads the OS-level location grant. */
  reload: () => void;
  /** Clear state and restart the watch (recovery CTA, or after OS settings change). */
  retry: () => void;
}

const DEFAULT_SUBMIT_TIMEOUT_MS = 30000;
const PERMISSION_DENIED = 1;

function createPermissionDeniedError(): Pick<GeolocationPositionError, "code" | "message"> {
  return createGeolocationError(
    PERMISSION_DENIED,
    "Quyền truy cập vị trí đang bị chặn"
  );
}

/**
 * Marks a submit-wait as cancelled (watch paused/restarted, component going
 * away, or retry). The card treats this as a SILENT no-op: no recovery banner,
 * no failed-attempt log. Without it, a worker who abandons a cold tap by
 * backgrounding the app would still get a ghost failed-attempt row in
 * attendance_failed_attempts when the 30s timer fired — polluting the audit
 * trail for a device that never actually failed.
 */
export class AbortedSubmitError extends Error {
  readonly aborted = true;
  constructor() {
    super("submit wait aborted");
    this.name = "AbortedSubmitError";
  }
}

export function isAbortedSubmitError(error: unknown): boolean {
  return (
    error instanceof AbortedSubmitError ||
    (typeof error === "object" &&
      error !== null &&
      (error as { aborted?: boolean }).aborted === true)
  );
}

interface PendingAwaiter {
  resolve: (sample: LocationSample) => void;
  reject: (error: unknown) => void;
  timer: ReturnType<typeof setTimeout>;
}

export function useContinuousLocation({
  target,
  enabled,
  submitTimeoutMs = DEFAULT_SUBMIT_TIMEOUT_MS,
}: UseContinuousLocationOptions): UseContinuousLocationResult {
  const [sample, setSample] = useState<LocationSample | null>(null);
  const [progress, setProgress] = useState<LocationAcquisitionProgress | null>(null);
  const [fatalError, setFatalError] = useState<UseContinuousLocationResult["fatalError"]>(null);
  const [isWatching, setIsWatching] = useState(false);
  const [permissionState, setPermissionState] = useState<LocationPermissionState>("unknown");
  // Seeded from a previous real fix rather than from the Permissions API: on iOS
  // that API answers "prompt" on every fresh page load even for an employee who
  // granted access long ago (WebKit Bug 275268), so gating on it re-asks on
  // every visit. A stored grant only exists because a fix actually arrived, and
  // the watch below is still the authority — a revoked permission surfaces as a
  // real denial there, which clears the stored grant again.
  const [hasRequestedPermission, setHasRequestedPermission] = useState(() =>
    hasRememberedLocationGrant()
  );
  const [watchPausedForAccuracy, setWatchPausedForAccuracy] = useState(false);
  const [, setSampleFreshnessEpoch] = useState(0);
  // Bump to force a clean watch restart (retry, or visibility return).
  const [restartEpoch, setRestartEpoch] = useState(0);
  const [visible, setVisible] = useState(
    typeof document === "undefined" ? true : document.visibilityState !== "hidden"
  );

  // Refs mirror the latest reactive state for use inside stable callbacks
  // (awaitSubmitReady / awaiter drain) without stale closures or resubscribing.
  const sampleRef = useRef<LocationSample | null>(null);
  const progressRef = useRef<LocationAcquisitionProgress | null>(null);
  const fatalErrorRef = useRef<UseContinuousLocationResult["fatalError"]>(null);
  const isSubmitReadyRef = useRef(false);
  const isWatchingRef = useRef(false);
  const retainSampleOnWatchStopRef = useRef(false);
  const pendingAwaitersRef = useRef<Set<PendingAwaiter>>(new Set());
  const pendingFreshSampleAwaitersRef = useRef<Set<PendingAwaiter>>(new Set());
  // True once the worker has pressed a recovery CTA, so a stale Permissions API
  // answer cannot tear down the watch they just asked for.
  const userRecoveryAttemptedRef = useRef(false);
  // Every fatalError this hook raises is a permission denial, and a denial is
  // terminal in-page. Derive the reload requirement from it so the two can never
  // disagree — the separate flag this replaced could lag a denial behind, which
  // left the worker tapping a no-op retry instead of the reload that works.
  const requiresPageReload = fatalError !== null;
  sampleRef.current = sample;
  progressRef.current = progress;
  fatalErrorRef.current = fatalError;

  // A geofence's radius describes its permitted area, not the confidence of a
  // location reading. Using the radius here allowed a ±100 m fix to be sent to
  // a 150 m geofence, where the server could still reject it as ambiguous.
  // Keep acquiring until the fixed attendance threshold is reached instead.
  const requiredAccuracyMeters = EMPLOYEE_ATTENDANCE_REQUIRED_ACCURACY_METERS;

  const guidance = useMemo(
    () => getCheckInGeofenceGuidance(target, sample),
    [target, sample]
  );
  const sampleIsFresh = Boolean(sample) && isSampleFresh(sample, Date.now(), CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS);
  const isSubmitReady =
    sampleIsFresh &&
    sample?.accuracy <= requiredAccuracyMeters &&
    guidance.status === "inside" &&
    fatalError === null;
  isSubmitReadyRef.current = isSubmitReady;
  const canAcquireLocation = permissionState === "granted" || hasRequestedPermission;
  const needsPermission =
    !fatalError &&
    !isWatching &&
    !hasRequestedPermission &&
    (permissionState === "prompt" || permissionState === "unknown");

  // Do not trigger the browser's permission prompt merely because the employee
  // opened the attendance card. A browser permission that was already granted
  // starts warming immediately; otherwise the card gives the employee a clear
  // "Cho phép vị trí" action that starts the native prompt in user context.
  useEffect(() => {
    let cancelled = false;
    void getLocationPermissionState().then((nextPermissionState) => {
      if (cancelled) return;
      setPermissionState(nextPermissionState);
      if (nextPermissionState === "granted") {
        setHasRequestedPermission(true);
      } else if (nextPermissionState === "denied" && !userRecoveryAttemptedRef.current) {
        // iOS Safari serves the permission state captured at page load and does
        // not re-read it after the worker changes Location in Settings, so a
        // "denied" here can be hours stale. Honouring it would tear down the
        // watch the worker just asked for on retry, making the recovery CTA a
        // silent no-op. After an explicit user action the watch is the authority:
        // it either delivers a fix or reports a real denial.
        const deniedError = createPermissionDeniedError();
        fatalErrorRef.current = deniedError;
        setFatalError(deniedError);
        setHasRequestedPermission(false);
        // A stored grant only survives while the permission does, so a real
        // "denied" from the browser retires it instead of leaving the next load
        // to skip the prompt and collide with this denial again.
        forgetLocationGrant();
      }
    });
    return () => {
      cancelled = true;
    };
  }, [restartEpoch]);

  // A paused watch does not emit another location update to make the React tree
  // re-check freshness. Re-render exactly when the retained fix expires so the
  // card does not offer it as an instant check-in after the 15s trust window.
  useEffect(() => {
    if (!sample) return;
    const remainingMs = sample.timestamp + CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS - Date.now();
    if (remainingMs <= 0) return;
    const timer = setTimeout(() => setSampleFreshnessEpoch((epoch) => epoch + 1), remainingMs);
    return () => clearTimeout(timer);
  }, [sample]);

  // Resolve every pending awaiter with a submit-ready sample and clear timers.
  const resolveAwaiters = useCallback((s: LocationSample) => {
    pendingAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.resolve(s);
    });
    pendingAwaitersRef.current.clear();
  }, []);

  const resolveFreshSampleAwaiters = useCallback((s: LocationSample) => {
    pendingFreshSampleAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.resolve(s);
    });
    pendingFreshSampleAwaitersRef.current.clear();
  }, []);

  // Reject every pending awaiter (fatal error or silent abort) and clear timers.
  const rejectAwaiters = useCallback((err: unknown) => {
    pendingAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.reject(err);
    });
    pendingAwaitersRef.current.clear();
    pendingFreshSampleAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.reject(err);
    });
    pendingFreshSampleAwaitersRef.current.clear();
  }, []);

  // Pause on tab hidden, resume on visible. iOS suspends the PWA anyway; this
  // stops the watch promptly and requires a fresh post-resume fix (the worker may
  // have moved while backgrounded) before isSubmitReady can be true again.
  useEffect(() => {
    if (typeof document === "undefined") return;
    const onVisibility = () => setVisible(document.visibilityState !== "hidden");
    document.addEventListener("visibilitychange", onVisibility);
    return () => document.removeEventListener("visibilitychange", onVisibility);
  }, []);

  useEffect(() => {
    if (!enabled || !target || !visible || !canAcquireLocation) {
      retainSampleOnWatchStopRef.current = false;
      isWatchingRef.current = false;
      setIsWatching(false);
      setWatchPausedForAccuracy(false);
      setSample(null);
      sampleRef.current = null;
      rejectAwaiters(new AbortedSubmitError());
      return;
    }

    if (watchPausedForAccuracy) {
      isWatchingRef.current = false;
      setIsWatching(false);
      return;
    }

    let cancelled = false;
    isWatchingRef.current = true;
    setIsWatching(true);

    const handle: ContinuousLocationHandle = watchContinuousLocation({
      requiredAccuracyMeters,
      onUpdate: (p) => {
        if (cancelled) return;
        progressRef.current = p;
        setProgress(p);
        const next = p.bestFreshSample ?? null;
        // A delivered fix is the only trustworthy proof of a grant on iOS, where
        // the Permissions API cannot answer. Persist it so the next page load
        // warms the watch instead of asking again.
        rememberLocationGrant();
        setPermissionState("granted");
        setHasRequestedPermission(true);
        sampleRef.current = next;
        setSample(next);
        if (next && next.accuracy <= requiredAccuracyMeters) {
          resolveFreshSampleAwaiters(next);
        }
        if (next && next.accuracy <= requiredAccuracyMeters) {
          // The fix is precise enough for an attendance submission.
          // Preserve it for the short freshness window and stop high-accuracy
          // GPS; a stale future tap restarts the watch below.
          retainSampleOnWatchStopRef.current = true;
          isWatchingRef.current = false;
          setIsWatching(false);
          handle.unsubscribe();
          setWatchPausedForAccuracy(true);
        }
      },
      onError: (geoError) => {
        if (cancelled) return;
        if (geoError.code === PERMISSION_DENIED) {
          // Fatal: no further fixes possible until the worker grants permission.
          // Stop the watch, surface via fatalError, and drain awaiters with the
          // real error so the cold-tap path classifies it as "denied" (not a
          // generic timeout).
          fatalErrorRef.current = geoError;
          setFatalError(geoError);
          setPermissionState("denied");
          setHasRequestedPermission(false);
          // The permission is gone for real, so the stored proof of a grant is
          // stale — drop it, otherwise the next load would skip the prompt and
          // walk straight back into this denial.
          forgetLocationGrant();
          isWatchingRef.current = false;
          setIsWatching(false);
          handle.unsubscribe();
          rejectAwaiters(geoError);
        }
        // Transient errors (momentary signal loss) are non-fatal; the next fix
        // updates state. Ignored here on purpose.
      },
    });

    return () => {
      cancelled = true;
      handle.unsubscribe();
      isWatchingRef.current = false;
      setIsWatching(false);
      if (retainSampleOnWatchStopRef.current) {
        retainSampleOnWatchStopRef.current = false;
        return;
      }
      // A paused/stopped watch must not trust a sample acquired before the pause
      // (the worker may have moved while the watch was off — e.g. backgrounded the
      // app and travelled). Clear so isSubmitReady re-flips only on a fresh
      // post-restart fix. NB: on iOS PWA suspension, JS freezes before
      // visibilitychange can fire reliably; the 15s freshness window + the server
      // gate are the remaining backstops for that edge (accepted per plan).
      setSample(null);
      sampleRef.current = null;
      // Cancel any in-flight cold-tap awaiter silently — the worker abandoned the
      // attempt (navigated away / backgrounded), the device did not fail, so no
      // failed-attempt row should be written when the 30s timer would have fired.
      rejectAwaiters(new AbortedSubmitError());
    };
  }, [canAcquireLocation, enabled, target, visible, watchPausedForAccuracy, restartEpoch, requiredAccuracyMeters, rejectAwaiters, resolveFreshSampleAwaiters]);

  // Drain pending awaiters the moment submit-readiness flips true.
  useEffect(() => {
    if (!isSubmitReady || !sample) return;
    resolveAwaiters(sample);
  }, [isSubmitReady, sample, resolveAwaiters]);

  const awaitSubmitReady = useCallback(
    (timeoutMs: number = submitTimeoutMs): Promise<LocationSample> => {
      return new Promise<LocationSample>((resolve, reject) => {
        if (fatalErrorRef.current) {
          reject(fatalErrorRef.current);
          return;
        }
        const currentSample = sampleRef.current;
        if (
          isSubmitReadyRef.current &&
          currentSample &&
          isSampleFresh(currentSample, Date.now(), CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS)
        ) {
          resolve(currentSample);
          return;
        }

        if (!isWatchingRef.current) {
          setWatchPausedForAccuracy(false);
        }

        const awaiter: PendingAwaiter = {
          resolve,
          reject,
          timer: setTimeout(
            () => {
              pendingAwaitersRef.current.delete(awaiter);
              reject(
                createInaccurateGeolocationError(
                  progressRef.current?.bestAccuracy,
                  requiredAccuracyMeters
                )
              );
            },
            timeoutMs
          ),
        };
        pendingAwaitersRef.current.add(awaiter);
      });
    },
    [requiredAccuracyMeters, submitTimeoutMs]
  );

  const awaitAccurateSample = useCallback(
    (timeoutMs: number = submitTimeoutMs): Promise<LocationSample> => {
      return new Promise<LocationSample>((resolve, reject) => {
        if (fatalErrorRef.current) {
          reject(fatalErrorRef.current);
          return;
        }
        const currentSample = sampleRef.current;
        if (
          currentSample &&
          currentSample.accuracy <= requiredAccuracyMeters &&
          isSampleFresh(currentSample, Date.now(), CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS)
        ) {
          resolve(currentSample);
          return;
        }

        if (!isWatchingRef.current) {
          setWatchPausedForAccuracy(false);
        }

        const awaiter: PendingAwaiter = {
          resolve,
          reject,
          timer: setTimeout(
            () => {
              pendingFreshSampleAwaitersRef.current.delete(awaiter);
              reject(
                createInaccurateGeolocationError(
                  progressRef.current?.bestAccuracy,
                  requiredAccuracyMeters
                )
              );
            },
            timeoutMs
          ),
        };
        pendingFreshSampleAwaitersRef.current.add(awaiter);
      });
    },
    [requiredAccuracyMeters, submitTimeoutMs]
  );

  const retry = useCallback(() => {
    // Cancel any in-flight awaiter (silent abort), then restart the watch.
    rejectAwaiters(new AbortedSubmitError());
    fatalErrorRef.current = null;
    retainSampleOnWatchStopRef.current = false;
    isWatchingRef.current = false;
    progressRef.current = null;
    sampleRef.current = null;
    setFatalError(null);
    setWatchPausedForAccuracy(false);
    setProgress(null);
    setSample(null);
    // Keep the acquisition gate open: the worker is explicitly telling us they
    // enabled location, so the watch must re-probe the OS. Clearing this back to
    // false (as it used to) made retry() a no-op whenever the browser reported
    // "denied" — a guaranteed dead end on iOS after a Settings change.
    setHasRequestedPermission(true);
    userRecoveryAttemptedRef.current = true;
    setRestartEpoch((n) => n + 1);
  }, [rejectAwaiters]);

  const requestPermission = useCallback(() => {
    if (!navigator.geolocation) return;
    fatalErrorRef.current = null;
    setFatalError(null);
    setHasRequestedPermission(true);
    // Not a "recovery" attempt: the first denial after the initial grant prompt
    // still has a Settings explanation. Only a denial that survives an explicit
    // "I have turned it on" retry escalates to a page reload.
    setRestartEpoch((n) => n + 1);
  }, []);

  const reload = useCallback(() => {
    window.location.reload();
  }, []);

  return {
    sample,
    progress,
    isSubmitReady,
    isWatching,
    permissionState,
    needsPermission,
    fatalError,
    requiresPageReload,
    requestPermission,
    awaitSubmitReady,
    awaitAccurateSample,
    reload,
    retry,
  };
}
