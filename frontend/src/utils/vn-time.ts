/**
 * Vietnam-timezone helpers for the employee attendance UI.
 *
 * Why this exists: the server returns shift/check-in windows as absolute
 * instants (RFC 3339 with a +07:00 offset, e.g. "2026-07-12T19:00:00+07:00").
 * Comparing them against the device clock must NOT depend on the browser's
 * local timezone — a worker whose phone is set to UTC would otherwise see their
 * 20:00 shift as starting at 13:00 and be blocked from checking in. These
 * helpers read clock/offset-absolute values (Date.now(), Date.getTime()) and
 * format instants as Asia/Ho_Chi_Minh HH:mm for display via
 * Intl.DateTimeFormat, which is immune to the device timezone setting.
 *
 * The business timezone is hard-coded to Asia/Ho_Chi_Minh, matching the
 * backend's clock.DefaultLocation (internal/pkg/clock/clock.go).
 */

export const VN_TIMEZONE = "Asia/Ho_Chi_Minh";

const vnTimeFormatter = new Intl.DateTimeFormat("en-GB", {
  timeZone: VN_TIMEZONE,
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

const vnDayFormatter = new Intl.DateTimeFormat("en-CA", {
  // en-CA yields ISO-like YYYY-MM-DD, stable across engines
  timeZone: VN_TIMEZONE,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

/**
 * Parse an ISO timestamp to epoch milliseconds. Returns null for invalid
 * input so callers can treat a missing window the same as "no shift configured".
 */
export function parseEpochMs(iso: string | null | undefined): number | null {
  if (!iso) return null;
  const ms = new Date(iso).getTime();
  return Number.isFinite(ms) ? ms : null;
}

/**
 * Format an ISO instant as HH:mm in Vietnam time (e.g. "20:00"). Returns ""
 * for falsy/invalid input.
 */
export function formatVnTime(iso: string | null | undefined): string {
  const ms = parseEpochMs(iso);
  if (ms === null) return "";
  // Intl "2-digit" hour can render "24:xx" in some engines near midnight; normalize.
  const formatted = vnTimeFormatter.format(new Date(ms));
  return formatted.startsWith("24") ? `00${formatted.slice(2)}` : formatted;
}

/**
 * The Vietnam calendar-day (YYYY-MM-DD) of an ISO instant. Used to detect
 * cross-midnight shifts by comparing the start and end calendar days.
 */
function vnCalendarDay(iso: string | null | undefined): string {
  const ms = parseEpochMs(iso);
  if (ms === null) return "";
  return vnDayFormatter.format(new Date(ms));
}

/** Whether `endIso` falls on the Vietnam day after `startIso`. */
export function isVnNextDay(startIso: string | null | undefined, endIso: string | null | undefined): boolean {
  const start = vnCalendarDay(startIso);
  const end = vnCalendarDay(endIso);
  if (!start || !end) return false;
  return end !== start;
}
