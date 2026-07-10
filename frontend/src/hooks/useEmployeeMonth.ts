import { useCallback, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import {
  addMonths,
  endOfMonth,
  format,
  isBefore,
  parseISO,
  startOfMonth,
  subMonths,
} from "date-fns";
import { vi } from "date-fns/locale";

const MONTH_PARAM = "month";
const YYYY_MM_REGEX = /^\d{4}-(0[1-9]|1[0-2])$/;
// Floor navigation ~24 months back so the chevron can't wander off into the
// distant past where the employee almost certainly has no payroll data.
const MAX_MONTHS_BACK = 24;

export interface EmployeeMonth {
  /** `yyyy-MM` string — the canonical value mirrored to the URL. */
  value: string;
  /** First day of the selected month as a Date. */
  date: Date;
  /** Human label, e.g. "Tháng 07, 2026" (Vietnamese locale). */
  label: string;
  /** `yyyy-MM-dd` — first day of the selected month. */
  fromDate: string;
  /** `yyyy-MM-dd` — last day of the selected month. */
  toDate: string;
  /** Short label, e.g. "07/2026" — used in compact/empty states. */
  shortLabel: string;
  /** Can the user navigate one month forward? (false at/after current month) */
  canGoNext: boolean;
  /** Can the user navigate one month backward? (false at the 24-month floor) */
  canGoPrev: boolean;
  /** Is the selected month the current calendar month? */
  isCurrentMonth: boolean;
  goPrev: () => void;
  goNext: () => void;
  goToday: () => void;
  /** Jump to an arbitrary `yyyy-MM` value (validated; invalid values ignored). */
  setValue: (next: string) => void;
}

function isValidMonth(value: string | null): value is string {
  return !!value && YYYY_MM_REGEX.test(value);
}

/**
 * URL-backed month selector for the employee portal. Mirrors the selected
 * month to `?month=YYYY-MM` so reload / Back / Forward preserve it, and every
 * month-dependent card on the page reads from the same value.
 *
 * Invalid / missing / future values fall back to the current month, so the URL
 * is always safe to share and never renders a blank screen.
 */
export function useEmployeeMonth(): EmployeeMonth {
  const [searchParams, setSearchParams] = useSearchParams();

  const now = useMemo(() => new Date(), []);
  const currentMonth = useMemo(() => startOfMonth(now), [now]);
  const floorMonth = useMemo(() => subMonths(currentMonth, MAX_MONTHS_BACK), [currentMonth]);

  const rawParam = searchParams.get(MONTH_PARAM);

  // Validate + clamp. Invalid/missing → current month. Future → current month.
  const resolvedDate = useMemo(() => {
    if (!isValidMonth(rawParam)) return currentMonth;
    const parsed = startOfMonth(parseISO(`${rawParam}-01`));
    if (isBefore(currentMonth, parsed)) return currentMonth; // no future
    if (isBefore(parsed, floorMonth)) return floorMonth; // no past beyond floor
    return parsed;
  }, [rawParam, currentMonth, floorMonth]);

  const writeMonth = useCallback(
    (next: Date) => {
      const value = format(startOfMonth(next), "yyyy-MM");
      const nextParams = new URLSearchParams(searchParams);
      if (format(currentMonth, "yyyy-MM") === value) {
        // Current month is the natural default — drop the param so the URL stays
        // clean for the common case (matches the TimesheetPage write-back style).
        nextParams.delete(MONTH_PARAM);
      } else {
        nextParams.set(MONTH_PARAM, value);
      }
      if (nextParams.toString() !== searchParams.toString()) {
        setSearchParams(nextParams, { replace: true });
      }
    },
    [searchParams, setSearchParams, currentMonth]
  );

  const goPrev = useCallback(() => writeMonth(subMonths(resolvedDate, 1)), [writeMonth, resolvedDate]);
  const goNext = useCallback(
    () => writeMonth(addMonths(resolvedDate, 1)),
    [writeMonth, resolvedDate]
  );
  const goToday = useCallback(() => writeMonth(currentMonth), [writeMonth, currentMonth]);
  const setValue = useCallback(
    (next: string) => {
      if (isValidMonth(next)) writeMonth(startOfMonth(parseISO(`${next}-01`)));
    },
    [writeMonth]
  );

  return useMemo(() => {
    const start = startOfMonth(resolvedDate);
    const end = endOfMonth(start);
    return {
      value: format(start, "yyyy-MM"),
      date: start,
      label: `Tháng ${format(start, "MM, yyyy")}`,
      fromDate: format(start, "yyyy-MM-dd"),
      toDate: format(end, "yyyy-MM-dd"),
      shortLabel: format(start, "MM/yyyy"),
      canGoPrev: isBefore(floorMonth, start),
      canGoNext: isBefore(start, currentMonth),
      isCurrentMonth: start.getTime() === currentMonth.getTime(),
      goPrev,
      goNext,
      goToday,
      setValue,
    };
  }, [resolvedDate, currentMonth, floorMonth, goPrev, goNext, goToday, setValue]);
}
