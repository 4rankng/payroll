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
          : "border-emerald-100/60 bg-white shadow-[0_1px_3px_rgba(0,0,0,0.04),0_4px_12px_rgba(0,0,0,0.03)]",
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
              ? "text-white/70 hover:bg-white/15 hover:text-white disabled:hover:text-white/70"
              : "text-slate-500 hover:bg-emerald-50 hover:text-emerald-700 disabled:hover:text-slate-500"
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
                isCanopy ? "h-12 hover:bg-white/15" : "h-14 hover:bg-emerald-50/60"
              )}
              aria-label={`Kỳ lương tháng ${month.shortLabel}. Nhấn để chọn tháng và năm khác`}
            >
              <span
                className={cn(
                  "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                  isCanopy
                    ? "border border-white/20 bg-white/15 text-white"
                    : "bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-md shadow-emerald-500/20"
                )}
              >
                <CalendarDays className="h-[1.125rem] w-[1.125rem]" aria-hidden="true" />
              </span>
              <span className="min-w-0 text-left">
                <span
                  className={cn(
                    "employee-type-card-title block tabular-nums",
                    isCanopy ? "text-white" : "text-slate-900"
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
          <PopoverContent data-employee-ui="" data-theme="employee" className="w-[min(340px,calc(100vw-32px))] rounded-2xl border border-slate-300 p-4 shadow-xl shadow-slate-900/10" align="center">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 pb-3">
              <div>
                <p className="employee-type-card-title text-slate-900">Chọn tháng</p>
                <p className="employee-type-body-sm mt-0.5 text-slate-500">Xem lịch sử lương và yêu cầu</p>
              </div>
              <label className="text-[0.8125rem] font-medium text-slate-600">
                <span className="sr-only">Chọn năm</span>
                <select
                  value={pickerYear}
                  onChange={(event) => setPickerYear(Number(event.target.value))}
                  className="employee-type-action h-11 rounded-xl border border-slate-300 bg-white px-3 text-slate-800 outline-none transition-colors focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20"
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
                      "employee-type-action min-h-11 rounded-xl px-1 py-2 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2",
                      selected
                        ? "bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-md shadow-emerald-500/25"
                        : "border border-slate-300 bg-white text-slate-700 hover:border-emerald-300 hover:bg-emerald-50/50 hover:text-emerald-700 active:scale-95",
                      "disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-50 disabled:text-slate-300 disabled:hover:border-slate-200 disabled:hover:bg-slate-50 disabled:hover:text-slate-300"
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
              ? "text-white/70 hover:bg-white/15 hover:text-white disabled:hover:text-white/70"
              : "text-slate-500 hover:bg-emerald-50 hover:text-emerald-700 disabled:hover:text-slate-500"
          )}
        >
          <ChevronRight className="h-5 w-5" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
