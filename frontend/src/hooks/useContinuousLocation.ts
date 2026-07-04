import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CheckInTarget } from "@/types/api/auth.types";
import { getCheckInGeofenceGuidance } from "@/utils/checkInGeofenceGuidance";
import {
  createInaccurateGeolocationError,
  watchContinuousLocation,
  type ContinuousLocationHandle,
  type LocationAcquisitionProgress,
  type LocationSample,
} from "@/utils/geolocation";

export interface UseContinuousLocationOptions {
  /** The gate target. When null, the hook stays idle. */
  target: CheckInTarget | null | undefined;
  /** True when a check-in/out action is conceptually possible. The watch keeps
   *  running while enabled, INCLUDING during a tap's submit-await — that is what
   *  makes the tap instant. */
  enabled: boolean;
  /** Optional override for the accuracy threshold used only for status display. */
  requiredAccuracyMeters?: number;
  /** Hard cap for awaitSubmitReady. Default 30s (matches the cold-acquire budget). */
  submitTimeoutMs?: number;
}

export interface UseContinuousLocationResult {
  /** Latest fresh sample (current position). Use for the map preview and as the
   *  submit payload. Null until the first fresh fix. */
  sample: LocationSample | null;
  /** Rolling acquisition progress for the converging-accuracy UX. */
  progress: LocationAcquisitionProgress | null;
  /** True iff `sample` is non-null AND its guidance against the target gate is
   *  "inside" (dist + accuracy <= radius) — mirrors the backend validateGeofence
   *  certain-path. A warm tap with this true submits instantly. */
  isSubmitReady: boolean;
  /** True while the underlying watchPosition is active. */
  isWatching: boolean;
  /** A fatal geolocation error from the watch (permission denied). The card
   *  surfaces this via its existing recovery banner + failed-attempt log. Null
   *  while the watch is healthy. */
  fatalError: GeolocationPositionError | null;
  /** Resolve with a submit-ready sample, or reject with a geolocation error on
   *  timeout / fatal error / abort. */
  awaitSubmitReady: (timeoutMs?: number) => Promise<LocationSample>;
  /** Clear state and restart the watch (recovery CTA, or after OS settings change). */
  retry: () => void;
}

const DEFAULT_SUBMIT_TIMEOUT_MS = 30000;
const PERMISSION_DENIED = 1;

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
  requiredAccuracyMeters,
  submitTimeoutMs = DEFAULT_SUBMIT_TIMEOUT_MS,
}: UseContinuousLocationOptions): UseContinuousLocationResult {
  const [sample, setSample] = useState<LocationSample | null>(null);
  const [progress, setProgress] = useState<LocationAcquisitionProgress | null>(null);
  const [fatalError, setFatalError] = useState<GeolocationPositionError | null>(null);
  const [isWatching, setIsWatching] = useState(false);
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
  const pendingAwaitersRef = useRef<Set<PendingAwaiter>>(new Set());
  sampleRef.current = sample;
  progressRef.current = progress;
  fatalErrorRef.current = fatalError;

  const radius = requiredAccuracyMeters ?? target?.radius_meters ?? undefined;

  const guidance = useMemo(
    () => getCheckInGeofenceGuidance(target, sample),
    [target, sample]
  );
  const isSubmitReady = Boolean(sample) && guidance.status === "inside" && fatalError === null;
  isSubmitReadyRef.current = isSubmitReady;

  // Resolve every pending awaiter with a submit-ready sample and clear timers.
  const resolveAwaiters = useCallback((s: LocationSample) => {
    pendingAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.resolve(s);
    });
    pendingAwaitersRef.current.clear();
  }, []);

  // Reject every pending awaiter (fatal error or silent abort) and clear timers.
  const rejectAwaiters = useCallback((err: unknown) => {
    pendingAwaitersRef.current.forEach((a) => {
      clearTimeout(a.timer);
      a.reject(err);
    });
    pendingAwaitersRef.current.clear();
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
    if (!enabled || !target || !visible) {
      setIsWatching(false);
      return;
    }

    let cancelled = false;
    setIsWatching(true);

    const handle: ContinuousLocationHandle = watchContinuousLocation({
      requiredAccuracyMeters: radius,
      onUpdate: (p) => {
        if (cancelled) return;
        progressRef.current = p;
        setProgress(p);
        const next = p.bestFreshSample ?? null;
        sampleRef.current = next;
        setSample(next);
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
      setIsWatching(false);
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
  }, [enabled, target, visible, restartEpoch, radius, rejectAwaiters]);

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
        if (isSubmitReadyRef.current && sampleRef.current) {
          resolve(sampleRef.current);
          return;
        }

        const requiredAccuracy = radius ?? 50;
        const awaiter: PendingAwaiter = {
          resolve,
          reject,
          timer: setTimeout(
            () => {
              pendingAwaitersRef.current.delete(awaiter);
              reject(
                createInaccurateGeolocationError(
                  progressRef.current?.bestAccuracy,
                  requiredAccuracy
                )
              );
            },
            timeoutMs
          ),
        };
        pendingAwaitersRef.current.add(awaiter);
      });
    },
    [submitTimeoutMs, radius]
  );

  const retry = useCallback(() => {
    // Cancel any in-flight awaiter (silent abort), then restart the watch.
    rejectAwaiters(new AbortedSubmitError());
    fatalErrorRef.current = null;
    progressRef.current = null;
    sampleRef.current = null;
    setFatalError(null);
    setProgress(null);
    setSample(null);
    setRestartEpoch((n) => n + 1);
  }, [rejectAwaiters]);

  return {
    sample,
    progress,
    isSubmitReady,
    isWatching,
    fatalError,
    awaitSubmitReady,
    retry,
  };
}
