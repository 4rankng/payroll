import { useCallback, useMemo, useState } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import DatePicker, { registerLocale } from "react-datepicker";
import "react-datepicker/dist/react-datepicker.css";
import "@/styles/react-datepicker.css";
import { vi } from "date-fns/locale";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

// Register Vietnamese locale once at module load so the popper's
// weekday header renders "CN T2 T3 T4 T5 T6 T7" instead of the
// react-datepicker default "Su Mo Tu We Th Fr Sa".
registerLocale("vi", vi);

const dateRangePickerVariants = cva("flex items-center shrink-0", {
  variants: {
    variant: {
      compact: "gap-1 h-7 px-2 rounded-md border border-transparent hover:border-input hover:bg-accent text-xs",
      default: "gap-1.5 text-xs",
      mobile: "gap-2 h-11 px-3 rounded-lg bg-muted/40 text-sm",
    },
  },
  defaultVariants: { variant: "default" },
});

export interface DateRangePickerProps extends VariantProps<typeof dateRangePickerVariants> {
  startDate?: string;
  endDate?: string;
  onStartDateChange: (date: string) => void;
  onEndDateChange: (date: string) => void;
  minDate?: Date;
  maxDate?: Date;
  className?: string;
  disabled?: boolean;
  // usePortal forces the popover calendar to render in a fullscreen
  // portal instead of the default popper positioned next to the input.
  // Required when the picker is rendered inside a Radix <Dialog> — the
  // dialog's stacking context clips the popper otherwise. Defaults to
  // true for the "mobile" variant; opt in explicitly elsewhere.
  usePortal?: boolean;
}

const MONTHS_VI_FULL = [
  "Tháng 1","Tháng 2","Tháng 3","Tháng 4","Tháng 5","Tháng 6",
  "Tháng 7","Tháng 8","Tháng 9","Tháng 10","Tháng 11","Tháng 12",
];

const formatDateToString = (date: Date): string => {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
};

const parseStringToDate = (value: string): Date | null => {
  if (!value) return null;
  return new Date(value + "T00:00:00");
};

type PickerView = "day" | "month" | "year";

// ─── Shared header used in all three views ────────────────────────────────────
interface HeaderProps {
  date: Date;
  decreaseMonth: () => void;
  increaseMonth: () => void;
  prevMonthButtonDisabled: boolean;
  nextMonthButtonDisabled: boolean;
  view: PickerView;
  onViewChange: (v: PickerView) => void;
}

function PickerHeader({
  date,
  decreaseMonth,
  increaseMonth,
  prevMonthButtonDisabled,
  nextMonthButtonDisabled,
  view,
  onViewChange,
}: HeaderProps) {
  return (
    <div className="flex items-center justify-between px-1 pb-1.5">
      <button
        type="button"
        onClick={decreaseMonth}
        disabled={prevMonthButtonDisabled || view !== "day"}
        className="h-6 w-6 flex items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        aria-label="Tháng trước"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
      </button>

      <div className="flex items-center gap-0.5">
        <button
          type="button"
          onClick={() => onViewChange(view === "month" ? "day" : "month")}
          className={cn(
            "px-2 py-0.5 rounded-md text-xs font-semibold transition-colors",
            view === "month"
              ? "bg-primary text-primary-foreground"
              : "text-foreground hover:bg-accent"
          )}
        >
          {MONTHS_VI_FULL[date.getMonth()]}
        </button>
        <button
          type="button"
          onClick={() => onViewChange(view === "year" ? "day" : "year")}
          className={cn(
            "px-2 py-0.5 rounded-md text-xs font-semibold transition-colors",
            view === "year"
              ? "bg-primary text-primary-foreground"
              : "text-foreground hover:bg-accent"
          )}
        >
          {date.getFullYear()}
        </button>
      </div>

      <button
        type="button"
        onClick={increaseMonth}
        disabled={nextMonthButtonDisabled || view !== "day"}
        className="h-6 w-6 flex items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        aria-label="Tháng sau"
      >
        <ChevronRight className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

// ─── Month grid (3×4) ─────────────────────────────────────────────────────────
const MONTHS_SHORT = ["T1","T2","T3","T4","T5","T6","T7","T8","T9","T10","T11","T12"];

function MonthGrid({ currentMonth, onSelect }: { currentMonth: number; onSelect: (m: number) => void }) {
  return (
    <div className="grid grid-cols-3 gap-1 p-2">
      {MONTHS_SHORT.map((label, i) => (
        <button
          key={i}
          type="button"
          onClick={() => onSelect(i)}
          className={cn(
            "h-8 rounded-md text-xs font-medium transition-colors",
            i === currentMonth
              ? "bg-primary text-primary-foreground"
              : "text-foreground hover:bg-accent"
          )}
        >
          {label}
        </button>
      ))}
    </div>
  );
}

// ─── Year grid (3×4, decade-based) ───────────────────────────────────────────
function YearGrid({ currentYear, onSelect }: { currentYear: number; onSelect: (y: number) => void }) {
  const years = useMemo(() => {
    const base = Math.floor(currentYear / 10) * 10 - 1;
    return Array.from({ length: 12 }, (_, i) => base + i);
  }, [currentYear]);

  return (
    <div className="grid grid-cols-3 gap-1 p-2">
      {years.map((y) => (
        <button
          key={y}
          type="button"
          onClick={() => onSelect(y)}
          className={cn(
            "h-8 rounded-md text-xs font-medium transition-colors",
            y === currentYear
              ? "bg-primary text-primary-foreground"
              : "text-foreground hover:bg-accent"
          )}
        >
          {y}
        </button>
      ))}
    </div>
  );
}

// ─── Single picker with view-switching ───────────────────────────────────────
interface SinglePickerProps {
  selected: Date | null;
  onChange: (date: Date | null) => void;
  minDate?: Date;
  maxDate?: Date;
  placeholderText: string;
  inputClassName: string;
  ariaLabel: string;
  disabled: boolean;
  // portalMode: "fullscreen" overlays a full-screen dialog (mobile UX);
  // "popper" renders the popper into a body-level portal so it escapes
  // ancestor overflow/stacking contexts (Radix <Dialog> needs this);
  // "inline" keeps the popper inside the input's DOM ancestor.
  portalMode: "fullscreen" | "popper" | "inline";
  popperClassName?: string;
}

function SinglePicker({
  selected,
  onChange,
  minDate,
  maxDate,
  placeholderText,
  inputClassName,
  ariaLabel,
  disabled,
  portalMode,
  popperClassName,
}: SinglePickerProps) {
  const [view, setView] = useState<PickerView>("day");

  const handleViewChange = useCallback((v: PickerView) => setView(v), []);

  const renderHeader = useCallback(
    (props: {
      date: Date;
      changeMonth: (m: number) => void;
      changeYear: (y: number) => void;
      decreaseMonth: () => void;
      increaseMonth: () => void;
      prevMonthButtonDisabled: boolean;
      nextMonthButtonDisabled: boolean;
    }) => {
      if (view === "month") {
        return (
          <>
            <PickerHeader
              date={props.date}
              decreaseMonth={props.decreaseMonth}
              increaseMonth={props.increaseMonth}
              prevMonthButtonDisabled={props.prevMonthButtonDisabled}
              nextMonthButtonDisabled={props.nextMonthButtonDisabled}
              view={view}
              onViewChange={handleViewChange}
            />
            <MonthGrid
              currentMonth={props.date.getMonth()}
              onSelect={(m) => { props.changeMonth(m); setView("day"); }}
            />
          </>
        );
      }

      if (view === "year") {
        return (
          <>
            <PickerHeader
              date={props.date}
              decreaseMonth={props.decreaseMonth}
              increaseMonth={props.increaseMonth}
              prevMonthButtonDisabled={props.prevMonthButtonDisabled}
              nextMonthButtonDisabled={props.nextMonthButtonDisabled}
              view={view}
              onViewChange={handleViewChange}
            />
            <YearGrid
              currentYear={props.date.getFullYear()}
              onSelect={(y) => { props.changeYear(y); setView("day"); }}
            />
          </>
        );
      }

      return (
        <PickerHeader
          date={props.date}
          decreaseMonth={props.decreaseMonth}
          increaseMonth={props.increaseMonth}
          prevMonthButtonDisabled={props.prevMonthButtonDisabled}
          nextMonthButtonDisabled={props.nextMonthButtonDisabled}
          view={view}
          onViewChange={handleViewChange}
        />
      );
    },
    [view, handleViewChange]
  );

  return (
    <DatePicker
      selected={selected}
      onChange={onChange}
      minDate={minDate}
      maxDate={maxDate}
      dateFormat="dd/MM/yyyy"
      locale="vi"
      placeholderText={placeholderText}
      className={inputClassName}
      customInput={<input aria-label={ariaLabel} />}
      disabled={disabled}
      renderCustomHeader={renderHeader}
      calendarClassName={view !== "day" ? "dp-grid-view" : undefined}
      popperClassName={popperClassName}
      // Mobile keeps the full-screen overlay (better touch UX). Desktop
      // pickers render the popper into a body-level portal so a parent
      // <Dialog>'s overflow-hidden / stacking context can't clip it,
      // but stay positioned next to the input (not full-screen).
      {...(portalMode === "fullscreen"
        ? { withPortal: true, portalId: "datepicker-portal" }
        : portalMode === "popper"
          ? { portalId: "datepicker-portal" }
          : {})}
    />
  );
}

// ─── Public component ─────────────────────────────────────────────────────────
export function DateRangePicker({
  variant = "default",
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  minDate,
  maxDate,
  className,
  disabled = false,
  usePortal: usePortalProp,
}: DateRangePickerProps) {
  const inputClasses = useMemo(() => {
    const base = "bg-transparent border-0 p-0 font-medium text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-0 cursor-pointer";
    switch (variant) {
      case "compact": return cn(base, "w-[76px] text-xs");
      case "mobile":  return cn(base, "h-11 flex-1 text-sm w-[95px]");
      default:        return cn(base, "w-[72px] text-xs");
    }
  }, [variant]);

  const iconSize = variant === "compact" ? "h-3 w-3" : "h-4 w-4";
  // portalMode: mobile → fullscreen overlay; desktop+usePortal → body
  // portal popper (escapes ancestor clipping); else inline.
  const portalMode: "fullscreen" | "popper" | "inline" =
    variant === "mobile"
      ? "fullscreen"
      : usePortalProp
        ? "popper"
        : "inline";
  const popperClassName = variant === "compact" ? "datepicker-left-shift" : undefined;

  const handleStartChange = useCallback((date: Date | null) => {
    if (!date) return;
    onStartDateChange(formatDateToString(date));
  }, [onStartDateChange]);

  const handleEndChange = useCallback((date: Date | null) => {
    if (!date) return;
    onEndDateChange(formatDateToString(date));
  }, [onEndDateChange]);

  return (
    <div className={cn(dateRangePickerVariants({ variant }), className)}>
      <CalendarDays className={cn(iconSize, "text-muted-foreground shrink-0")} />
      <SinglePicker
        selected={startDate ? parseStringToDate(startDate) : null}
        onChange={handleStartChange}
        minDate={minDate}
        maxDate={maxDate}
        placeholderText="Từ ngày"
        inputClassName={inputClasses}
        ariaLabel="Ngày bắt đầu"
        disabled={disabled}
        portalMode={portalMode}
        popperClassName={popperClassName}
      />
      <span className={cn(
        "select-none text-muted-foreground/40",
        variant === "compact" ? "text-[11px]" : "text-xs"
      )}>–</span>
      <SinglePicker
        selected={endDate ? parseStringToDate(endDate) : null}
        onChange={handleEndChange}
        minDate={minDate}
        maxDate={maxDate}
        placeholderText="Đến ngày"
        inputClassName={inputClasses}
        ariaLabel="Ngày kết thúc"
        disabled={disabled}
        portalMode={portalMode}
        popperClassName={popperClassName}
      />
    </div>
  );
}
