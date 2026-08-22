import { useEffect, useMemo, useState } from "react";
import {
  addMonths,
  format,
  isAfter,
  isBefore,
  startOfMonth,
  subMonths,
} from "date-fns";
import { vi } from "date-fns/locale";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

const MONTHS_BACK = 24;

interface CheckInMonthSelectorProps {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
}

function monthValueToDate(value: string): Date {
  const [year, month] = value.split("-").map(Number);
  return startOfMonth(new Date(year, month - 1, 1));
}

export function CheckInMonthSelector({
  value,
  onValueChange,
  disabled = false,
  className,
}: CheckInMonthSelectorProps) {
  const selectedMonth = useMemo(() => monthValueToDate(value), [value]);
  const currentMonth = useMemo(() => startOfMonth(new Date()), []);
  const oldestMonth = useMemo(() => subMonths(currentMonth, MONTHS_BACK), [currentMonth]);
  const [open, setOpen] = useState(false);
  const [pickerYear, setPickerYear] = useState(selectedMonth.getFullYear());

  useEffect(() => {
    setPickerYear(selectedMonth.getFullYear());
  }, [selectedMonth]);

  const years = useMemo(() => {
    const result: number[] = [];
    for (let year = currentMonth.getFullYear(); year >= oldestMonth.getFullYear(); year -= 1) {
      result.push(year);
    }
    return result;
  }, [currentMonth, oldestMonth]);

  const handleSelect = (nextMonth: Date) => {
    onValueChange(format(nextMonth, "yyyy-MM"));
    setOpen(false);
  };

  const canGoPrevious = isBefore(oldestMonth, selectedMonth);
  const canGoNext = isBefore(selectedMonth, currentMonth);
  const label = format(selectedMonth, "MM/yyyy");

  return (
    <div
      role="group"
      aria-label={`Tháng điểm danh ${label}`}
      className={cn(
        "inline-flex h-11 items-center rounded-lg border border-border bg-background p-0.5 sm:h-9",
        className,
      )}
    >
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-10 w-10 rounded-md sm:h-8 sm:w-8"
        aria-label="Xem tháng điểm danh trước"
        title="Tháng trước"
        disabled={disabled || !canGoPrevious}
        onClick={() => onValueChange(format(addMonths(selectedMonth, -1), "yyyy-MM"))}
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            className="h-10 min-w-[8.75rem] gap-2 rounded-md px-2.5 tabular-nums sm:h-8"
            aria-label={`Chọn tháng điểm danh. Đang chọn tháng ${label}`}
            disabled={disabled}
          >
            <CalendarDays className="h-4 w-4 text-muted-foreground" aria-hidden />
            <span className="font-semibold">Tháng {label}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="end"
          className="w-[min(20rem,calc(100vw-1.5rem))] rounded-xl p-3"
        >
          <div className="flex items-center justify-between gap-3 border-b border-border pb-3">
            <div>
              <p className="text-sm font-semibold text-foreground">Chọn tháng điểm danh</p>
              <p className="mt-0.5 text-xs text-muted-foreground">Tối đa 24 tháng gần nhất</p>
            </div>
            <label className="text-xs font-medium text-muted-foreground">
              <span className="sr-only">Chọn năm điểm danh</span>
              <select
                value={pickerYear}
                onChange={(event) => setPickerYear(Number(event.target.value))}
                className="h-11 rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring sm:h-9"
                aria-label="Chọn năm điểm danh"
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
              const isDisabled = isBefore(candidate, oldestMonth) || isAfter(candidate, currentMonth);
              const isSelected = candidate.getTime() === selectedMonth.getTime();
              return (
                <button
                  key={monthIndex}
                  type="button"
                  disabled={isDisabled}
                  aria-pressed={isSelected}
                  aria-label={`Tháng ${monthIndex + 1} năm ${pickerYear}`}
                  onClick={() => handleSelect(candidate)}
                  className={cn(
                    "h-11 rounded-md border text-sm font-medium capitalize transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:h-9",
                    isSelected
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-border bg-background text-foreground hover:bg-muted",
                    "disabled:cursor-not-allowed disabled:bg-muted/40 disabled:text-muted-foreground disabled:opacity-50",
                  )}
                >
                  {format(candidate, "MMM", { locale: vi })}
                </button>
              );
            })}
          </div>
        </PopoverContent>
      </Popover>

      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-10 w-10 rounded-md sm:h-8 sm:w-8"
        aria-label="Xem tháng điểm danh sau"
        title="Tháng sau"
        disabled={disabled || !canGoNext}
        onClick={() => onValueChange(format(addMonths(selectedMonth, 1), "yyyy-MM"))}
      >
        <ChevronRight className="h-4 w-4" aria-hidden />
      </Button>
    </div>
  );
}
