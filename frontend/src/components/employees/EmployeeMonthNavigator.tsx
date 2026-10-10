import { useEffect, useMemo, useState } from "react";
import {
  format,
  isAfter,
  isBefore,
  startOfMonth,
  subMonths,
} from "date-fns";
import { vi } from "date-fns/locale";
import { Calendar, ChevronLeft, ChevronRight } from "@untitledui/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type { EmployeeMonth } from "@/hooks/useEmployeeMonth";
import { cn } from "@/lib/utils";

interface EmployeeMonthNavigatorProps {
  month: EmployeeMonth;
  className?: string;
  /**
   * "card" is the standalone white surface. "canopy" renders the same control
   * as translucent glass for use inside the emerald EmployeeCanopy.
   */
  variant?: "card" | "canopy";
}

const MONTHS_BACK = 24;

export function EmployeeMonthNavigator({
  month,
  className,
  variant = "card",
}: EmployeeMonthNavigatorProps) {
  const isCanopy = variant === "canopy";
  const [isOpen, setIsOpen] = useState(false);
  const [pickerYear, setPickerYear] = useState(month.date.getFullYear());

  const maxMonth = useMemo(() => startOfMonth(new Date()), []);
  const minMonth = useMemo(() => subMonths(maxMonth, MONTHS_BACK), [maxMonth]);
  const years = useMemo(() => {
    const values: number[] = [];
    for (let year = maxMonth.getFullYear(); year >= minMonth.getFullYear(); year -= 1) {
      values.push(year);
    }
    return values;
  }, [maxMonth, minMonth]);

  useEffect(() => {
    setPickerYear(month.date.getFullYear());
  }, [month.date]);

  const handleOpenChange = (nextOpen: boolean) => {
    setIsOpen(nextOpen);
    if (nextOpen) setPickerYear(month.date.getFullYear());
  };

  const handleMonthSelect = (monthIndex: number) => {
    const next = startOfMonth(new Date(pickerYear, monthIndex, 1));
    month.setValue(format(next, "yyyy-MM"));
    setIsOpen(false);
  };

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-2xl border",
        isCanopy
          ? "border-white/20 bg-white/15 backdrop-blur-md"
          : "border-[var(--employee-accent-border)] bg-[var(--employee-surface)] shadow-[var(--employee-shadow)]",
        className
      )}
      role="group"
      aria-label={`Kỳ lương tháng ${month.shortLabel}`}
    >
      <div className="grid grid-cols-[48px_minmax(0,1fr)_48px] items-center px-1 py-1">
        <button
          type="button"
          onClick={month.goPrev}
          disabled={!month.canGoPrev}
          aria-label="Xem tháng trước"
          title="Tháng trước"
          className={cn(
            "flex h-12 w-12 items-center justify-center justify-self-center rounded-xl transition-all duration-150 active:scale-95 disabled:opacity-30 disabled:hover:bg-transparent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current",
            isCanopy
              ? "text-white/90 hover:bg-white/15 hover:text-white disabled:hover:text-white/90"
              : "text-fg-tertiary hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)] disabled:hover:text-fg-tertiary"
          )}
        >
          <ChevronLeft className="h-5 w-5" aria-hidden="true" />
        </button>

        <Popover open={isOpen} onOpenChange={handleOpenChange}>
          <PopoverTrigger asChild>
            <button
              type="button"
              className={cn(
                "mx-auto flex min-w-0 flex-nowrap items-center gap-3 rounded-xl px-3 transition-all duration-150 active:scale-[0.98] max-[359px]:gap-1.5 max-[359px]:px-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current",
                isCanopy ? "h-12 hover:bg-white/15" : "h-14 hover:bg-[var(--employee-accent-soft)]/60"
              )}
              aria-label={`Kỳ lương tháng ${month.shortLabel}. Nhấn để chọn tháng và năm khác`}
            >
              <span
                className={cn(
                  "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                  isCanopy
                    ? "border border-white/20 bg-white/15 text-white"
                    : "bg-[var(--employee-accent)] text-white"
                )}
              >
                <Calendar className="h-[1.125rem] w-[1.125rem]" aria-hidden="true" />
              </span>
              <span className="min-w-0 text-left">
                <span
                  className={cn(
                    "employee-type-card-title block tabular-nums",
                    isCanopy ? "text-white" : "text-[var(--employee-text)]"
                  )}
                >
                  Tháng {month.shortLabel}
                </span>
                <span
                  className={cn(
                    "employee-type-label-caps block",
                    isCanopy ? "text-white/65" : "text-[var(--employee-accent-strong)]"
                  )}
                >
                  Kỳ bảng công
                </span>
              </span>
            </button>
          </PopoverTrigger>
          <PopoverContent data-employee-ui="" data-theme="employee" className="w-[min(340px,calc(100vw-32px))] rounded-2xl p-4" align="center">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--employee-border)] pb-3">
              <div>
                <p className="employee-type-card-title text-[var(--employee-text)]">Chọn tháng</p>
                <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">Xem lịch sử lương và yêu cầu</p>
              </div>
              <label className="text-[0.8125rem] font-medium text-fg-secondary">
                <span className="sr-only">Chọn năm</span>
                <select
                  value={pickerYear}
                  onChange={(event) => setPickerYear(Number(event.target.value))}
                  className="employee-type-action h-11 rounded-xl border border-[var(--employee-border-strong)] bg-[var(--employee-surface)] px-3 text-[var(--employee-text)] outline-none transition-colors focus:border-[var(--employee-accent)] focus:ring-2 focus:ring-[var(--employee-accent-ring)]"
                  aria-label="Chọn năm"
                >
                  {years.map((year) => (
                    <option key={year} value={year}>{year}</option>
                  ))}
                </select>
              </label>
            </div>
            <div className="mt-3 grid grid-cols-3 gap-2">
              {Array.from({ length: 12 }, (_, monthIndex) => {
                const candidate = startOfMonth(new Date(pickerYear, monthIndex, 1));
                const disabled = isBefore(candidate, minMonth) || isAfter(candidate, maxMonth);
                const selected = candidate.getTime() === month.date.getTime();
                return (
                  <button
                    key={monthIndex}
                    type="button"
                    disabled={disabled}
                    aria-pressed={selected}
                    onClick={() => handleMonthSelect(monthIndex)}
                    className={cn(
                      "employee-type-action min-h-11 rounded-xl px-1 py-2 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2",
                      selected
                        ? "bg-[var(--employee-accent)] text-white shadow-sm"
                        : "border border-[var(--employee-border-strong)] bg-[var(--employee-surface)] text-fg-secondary hover:border-[var(--employee-accent-border)] hover:bg-[var(--employee-accent-soft)]/50 hover:text-[var(--employee-accent)] active:scale-95",
                      "disabled:cursor-not-allowed disabled:border-[var(--employee-border)] disabled:bg-[var(--employee-page)] disabled:text-fg-disabled disabled:hover:border-[var(--employee-border)] disabled:hover:bg-[var(--employee-page)] disabled:hover:text-fg-disabled"
                    )}
                  >
                    {format(candidate, "MMM", { locale: vi })}
                  </button>
                );
              })}
            </div>
          </PopoverContent>
        </Popover>

        <button
          type="button"
          onClick={month.goNext}
          aria-label="Xem tháng sau"
          title="Tháng sau"
          disabled={!month.canGoNext}
          className={cn(
            "flex h-12 w-12 items-center justify-center justify-self-center rounded-xl transition-all duration-150 active:scale-95 disabled:opacity-30 disabled:hover:bg-transparent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current",
            isCanopy
              ? "text-white/90 hover:bg-white/15 hover:text-white disabled:hover:text-white/90"
              : "text-fg-tertiary hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)] disabled:hover:text-fg-tertiary"
          )}
        >
          <ChevronRight className="h-5 w-5" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
