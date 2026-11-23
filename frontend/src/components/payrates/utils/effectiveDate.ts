import { dateToString } from '@/utils/dateHelpers';

type LatestPayrateLike = {
  fromDate: string;
  earliest_effective_from?: string;
} | null | undefined;

/**
 * Earliest start date the backend accepts for a NEW payrate config: strictly
 * after the latest config's start AND on/after the paid floor (the day after
 * the project's most recent paid timesheet work date, mirrored into every
 * payrate row as earliest_effective_from). Undefined when the project has no
 * config yet — a first-ever config has no predecessor or floor.
 */
export function earliestAllowedEffectiveFrom(latest?: LatestPayrateLike): string | undefined {
  if (!latest?.fromDate) return undefined;
  const next = new Date(latest.fromDate + 'T00:00:00');
  next.setDate(next.getDate() + 1);
  const afterLatest = dateToString(next);
  return latest.earliest_effective_from && latest.earliest_effective_from > afterLatest
    ? latest.earliest_effective_from
    : afterLatest;
}

/**
 * Default 'Từ ngày' for the create form.
 *
 * First-ever config (no latest): the first day of the current month, so a new
 * project's rates cover the whole payroll month instead of starting mid-month.
 *
 * Successor config: today, floored at the earliest backend-allowed date.
 * Defaulting to today keeps the new config forward-looking — a past default
 * (latest config + 1 day) reads as "re-price history from that date", which is
 * not what creating a new config does.
 */
export function defaultCreateEffectiveFrom(latest?: LatestPayrateLike, today: Date = new Date()): string {
  const floor = earliestAllowedEffectiveFrom(latest);
  const todayStr = dateToString(today);
  if (!floor) {
    return dateToString(new Date(today.getFullYear(), today.getMonth(), 1));
  }
  return floor > todayStr ? floor : todayStr;
}
