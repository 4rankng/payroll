import { createAppError } from "@/utils/error-handler";

export type AttendanceActionType = "check_in" | "check_out";

export interface TimingErrorGuidance {
  title: string;
  message: string;
  actionLabel: string;
  windowStart?: string;
  windowEnd?: string;
}

interface TimingGuidanceDetails {
  guidance_type?: unknown;
  action?: unknown;
  window_start?: unknown;
  window_end?: unknown;
}

interface LocationGuidanceDetails {
  guidance_type?: unknown;
}

function readTime(value: unknown): string | undefined {
  return typeof value === "string" && /^\d{2}:\d{2}$/.test(value) ? value : undefined;
}

function getTimes(message: string): string[] {
  return Array.from(message.matchAll(/\b\d{2}:\d{2}\b/g), (match) => match[0]);
}

export function getTimingErrorGuidance(
  error: unknown,
  action: AttendanceActionType
): TimingErrorGuidance | null {
  const appError = createAppError(error);
  const details = (appError.details ?? {}) as TimingGuidanceDetails;
  const message = appError.message;
  const isStructuredTiming = details.guidance_type === "timing" && details.action === action;
  const isCheckInTiming = action === "check_in" && message.includes("Giờ vào làm không hợp lệ");
  const isTerminalCheckOutTiming =
    action === "check_out" &&
    (message.includes("Đã quá giờ tan ca") || message.includes("quá hạn tan ca") || message.includes("tự động từ chối"));

  if (!isStructuredTiming && !isCheckInTiming && !isTerminalCheckOutTiming) {
    return null;
  }

  const times = getTimes(message);
  const windowStart = readTime(details.window_start) ?? (action === "check_in" ? times[0] : times.at(-2));
  const windowEnd = readTime(details.window_end) ?? (action === "check_in" ? times[1] : times.at(-1));

  return {
    title: action === "check_in" ? "Chưa đến giờ vào làm" : "Không thể tan ca lúc này",
    message,
    actionLabel: action === "check_in" ? "Khung giờ vào làm" : "Khung giờ tan ca",
    windowStart,
    windowEnd,
  };
}

export function hasLocationErrorGuidance(error: unknown): boolean {
  const appError = createAppError(error);
  const details = (appError.details ?? {}) as LocationGuidanceDetails;
  return details.guidance_type === "location";
}

/**
 * The map sits below the employee's current scroll position on long portal pages.
 * Scheduling the scroll for the next frame lets React first expand the map, then
 * brings the recovery guidance into view after a location-related rejection.
 */
export function scrollToLocationGuidance(element: HTMLElement | null): void {
  if (!element) return;

  const prefersReducedMotion =
    typeof window !== "undefined" && window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;

  const scroll = () => {
    element.scrollIntoView({ behavior: prefersReducedMotion ? "auto" : "smooth", block: "center" });
  };

  if (typeof window !== "undefined" && typeof window.requestAnimationFrame === "function") {
    window.requestAnimationFrame(scroll);
    return;
  }

  scroll();
}
