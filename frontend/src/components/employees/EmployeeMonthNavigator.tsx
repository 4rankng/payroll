import { useEffect, useMemo, useState } from "react";
import {
  format,
  isAfter,
  isBefore,
  startOfMonth,
  subMonths,
} from "date-fns";
import { vi } from "date-fns/locale";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type { EmployeeMonth } from "@/hooks/useEmployeeMonth";
import { cn } from "@/lib/utils";

interface EmployeeMonthNavigatorProps {
  month: EmployeeMonth;
  className?: string;
}

const MONTHS_BACK = 24;

export function EmployeeMonthNavigator({ month, className }: EmployeeMonthNavigatorProps) {
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
        "ct-card employee-surface-card grid min-h-16 grid-cols-[44px_minmax(0,1fr)_44px] items-center p-1.5",
        className
      )}
      role="group"
      aria-label={`Kỳ lương tháng ${month.shortLabel}`}
    >
      <button
        type="button"
        onClick={month.goPrev}
        disabled={!month.canGoPrev}
        aria-label="Xem tháng trước"
        title="Tháng trước"
        className="ct-btn ct-btn-ghost ct-btn-square h-11 min-h-11 w-11 rounded-xl text-base-content/70 shadow-none transition-colors active:bg-[var(--employee-accent-soft)] disabled:bg-transparent disabled:opacity-40"
      >
        <ChevronLeft className="h-5 w-5" aria-hidden="true" />
      </button>

      <Popover open={isOpen} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="ct-btn ct-btn-ghost mx-auto h-auto min-h-11 min-w-0 max-w-full gap-2.5 rounded-xl px-2 font-normal normal-case text-base-content shadow-none transition-colors active:bg-[var(--employee-accent-soft)]"
            aria-label={`Kỳ lương tháng ${month.shortLabel}. Nhấn để chọn tháng và năm khác`}
          >
            <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-primary">
              <CalendarDays className="h-4 w-4" aria-hidden="true" />
            </span>
            <span className="min-w-0 text-left">
              <span className="employee-type-month-title block truncate tabular-nums">Tháng {month.shortLabel}</span>
              <span className="employee-type-label-caps -mt-0.5 block text-[var(--employee-text-secondary)]">Kỳ bảng công</span>
            </span>
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-[min(328px,calc(100vw-32px))] rounded-xl border-[var(--employee-border)] p-3 shadow-none" align="center">
          <div className="flex items-center justify-between gap-3 border-b border-[#EAECF0] pb-3">
            <div>
              <p className="employee-type-strong text-[#101828]">Chọn tháng</p>
              <p className="employee-type-body-sm mt-0.5 text-[#667085]">Xem lịch sử lương và yêu cầu</p>
            </div>
            <label className="employee-type-label text-[#475467]">
              <span className="sr-only">Chọn năm</span>
              <select
                value={pickerYear}
                onChange={(event) => setPickerYear(Number(event.target.value))}
                className="h-11 rounded-[10px] border border-[#D0D5DD] bg-white px-3 text-[#101828] outline-none focus:ring-2 focus:ring-[var(--employee-accent)]"
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
                    "employee-type-action h-11 rounded-[10px] border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]",
                    selected
                      ? "border-[var(--employee-accent)] bg-[var(--employee-accent)] text-white"
                      : "border-[#E4E7EC] bg-white text-[#344054] hover:bg-[#F9FAFB]",
                    "disabled:cursor-not-allowed disabled:border-[#F2F4F7] disabled:bg-[#F9FAFB] disabled:text-[#98A2B3]"
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
        className="ct-btn ct-btn-ghost ct-btn-square h-11 min-h-11 w-11 rounded-xl text-base-content/70 shadow-none transition-colors active:bg-[var(--employee-accent-soft)] disabled:bg-transparent disabled:opacity-40"
      >
        <ChevronRight className="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  );
}
