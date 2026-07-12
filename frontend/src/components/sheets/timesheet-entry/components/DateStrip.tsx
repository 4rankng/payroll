import { memo } from "react";
import { cn } from "@/lib/utils";
import { WEEKDAY, isWeekendDate } from "../utils/timesheetHelpers";
import type { TimesheetEntry } from "../types/multi-timesheet.types";

interface DateStripProps {
  dates: string[];
  activeDate: string;
  entries: TimesheetEntry[];
  entryValidations: Record<string, { valid: boolean; errors: string[] }>;
  getErrorsForRow: (employeeId: number, date: string) => string[] | null;
  employeeId: number;
  onDateSelect: (date: string) => void;
}

export const DateStrip = memo(
  ({
    dates,
    activeDate,
    entries,
    entryValidations,
    getErrorsForRow,
    employeeId,
    onDateSelect,
  }: DateStripProps) => (
    <div className="flex gap-1.5 px-4 py-2.5 overflow-x-auto scrollbar-none">
      {dates.map((date) => {
        const entry = entries.find((e) => e.date === date);
        const filled = entry && Object.values(entry.hours).some((h) => h > 0);
        const markedDel = !!(
          entry?.originalValues &&
          (!entry.hours || Object.keys(entry.hours).length === 0)
        );
        const weekend = isWeekendDate(date);
        const d = new Date(date);
        const isActive = activeDate === date;
        const hasEntryError = !!(
          entry &&
          ((entryValidations[entry.id] && !entryValidations[entry.id].valid) ||
            getErrorsForRow(employeeId, date))
        );

        return (
          <button
            key={date}
            onClick={() => onDateSelect(date)}
            className={cn(
              "flex flex-col items-center gap-0.5 px-2.5 py-1.5 rounded-xl shrink-0 min-w-[40px] transition-all active:scale-95",
              isActive
                ? "bg-primary text-primary-foreground shadow-sm"
                : markedDel
                  ? "bg-red-50 text-red-400 border border-red-200"
                  : hasEntryError
                    ? "bg-red-50 text-red-600 border border-red-200"
                    : filled
                      ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                      : weekend
                        ? "bg-orange-50/80 text-orange-500"
                        : "bg-muted/50 text-muted-foreground",
            )}
          >
            <span className="text-[11px] font-semibold leading-none uppercase">
              {WEEKDAY[d.getDay()]}
            </span>
            <span
              className={cn(
                "text-sm font-bold leading-none tabular-nums",
                markedDel && !isActive && "line-through",
              )}
            >
              {String(d.getDate()).padStart(2, "0")}
            </span>
            <div
              className={cn(
                "w-1 h-1 rounded-full mt-0.5",
                markedDel && !isActive
                  ? "bg-red-300"
                  : filled && !isActive
                    ? "bg-emerald-500"
                    : hasEntryError && !isActive
                      ? "bg-red-400"
                      : "bg-transparent",
              )}
            />
          </button>
        );
      })}
    </div>
  ),
);

DateStrip.displayName = "DateStrip";
