import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CheckInTarget } from "@/types/api/auth.types";
import { getCheckInGeofenceGuidance } from "@/utils/checkInGeofenceGuidance";
import {
  CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS,
  EMPLOYEE_ATTENDANCE_REQUIRED_ACCURACY_METERS,
  createGeolocationError,
  createInaccurateGeolocationError,
  getLocationPermissionState,
  isSampleFresh,
  watchContinuousLocation,
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
  fatalError: GeolocationPositionError | null;
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
  /** Clear state and restart the watch (recovery CTA, or after OS settings change). */
  retry: () => void;
}

const DEFAULT_SUBMIT_TIMEOUT_MS = 30000;
const PERMISSION_DENIED = 1;

function createPermissionDeniedError(): GeolocationPositionError {
  return createGeolocationError(
    PERMISSION_DENIED,
    "Quyền truy cập vị trí đang bị chặn"
  ) as GeolocationPositionError;
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
  const [fatalError, setFatalError] = useState<GeolocationPositionError | null>(null);
  const [isWatching, setIsWatching] = useState(false);
  const [permissionState, setPermissionState] = useState<LocationPermissionState>("unknown");
  const [hasRequestedPermission, setHasRequestedPermission] = useState(false);
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
  const fatalErrorRef = useRef<GeolocationPositionError | null>(null);
  const isSubmitReadyRef = useRef(false);
  const isWatchingRef = useRef(false);
  const retainSampleOnWatchStopRef = useRef(false);
  const pendingAwaitersRef = useRef<Set<PendingAwaiter>>(new Set());
  const pendingFreshSampleAwaitersRef = useRef<Set<PendingAwaiter>>(new Set());
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
      } else if (nextPermissionState === "denied") {
        const deniedError = createPermissionDeniedError();
        fatalErrorRef.current = deniedError;
        setFatalError(deniedError);
        setHasRequestedPermission(false);
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
    setHasRequestedPermission(false);
    setRestartEpoch((n) => n + 1);
  }, [rejectAwaiters]);

  const requestPermission = useCallback(() => {
    if (!navigator.geolocation) return;
    fatalErrorRef.current = null;
    setFatalError(null);
    setHasRequestedPermission(true);
    setWatchPausedForAccuracy(false);
    setRestartEpoch((n) => n + 1);
  }, []);

  return {
    sample,
    progress,
    isSubmitReady,
    isWatching,
    permissionState,
    needsPermission,
    fatalError,
    requestPermission,
    awaitSubmitReady,
    awaitAccurateSample,
    retry,
  };
}
