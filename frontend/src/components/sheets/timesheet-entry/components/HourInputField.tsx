import { memo } from "react";
import { cn } from "@/lib/utils";
import { handleNumericInputKeyDown } from "@/utils/numericInput";
import { formatVND } from "../utils/timesheetHelpers";

interface HourInputFieldProps {
  hourType: string;
  value: number;
  displayValue: string;
  disabled?: boolean;
  originalValue?: number;
  hasOriginalValues?: boolean;
  rate?: number;
  onFocus?: () => void;
  onChange: (raw: string) => void;
  onBlur?: () => void;
}

function getInputColor(opts: {
  disabled: boolean;
  hasOriginalValues: boolean;
  originalValue: number;
  value: number;
}): string {
  const { disabled, hasOriginalValues, originalValue, value } = opts;
  if (disabled) return "";

  // Unsaved diff states — only shown before saving
  const thisFieldDeleted = hasOriginalValues && originalValue > 0 && value === 0;
  const thisFieldEdited =
    hasOriginalValues && value !== originalValue && !thisFieldDeleted;
  const thisFieldNew = !hasOriginalValues && value > 0;

  if (thisFieldDeleted) return "border-red-300 bg-red-50/70 text-red-400";
  if (thisFieldEdited)
    return "border-amber-300 bg-amber-50/70 text-amber-700 font-semibold";
  if (thisFieldNew)
    return "border-emerald-300 bg-emerald-50/70 text-emerald-700 font-bold";

  // Saved / unchanged data — no color coding, just plain styling
  return "border-input text-foreground";
}

export const HourInputField = memo(
  ({
    hourType,
    value,
    displayValue,
    disabled = false,
    originalValue = 0,
    hasOriginalValues = false,
    rate = 0,
    onFocus,
    onChange,
    onBlur,
  }: HourInputFieldProps) => {
    const entryAmount = value > 0 && rate > 0 ? value * rate : 0;
    const inputColor = getInputColor({
      disabled,
      hasOriginalValues,
      originalValue,
      value,
    });

    return (
      <div
        className={cn("flex items-center gap-1.5", disabled && "opacity-40")}
      >
        <span className="text-xs text-muted-foreground whitespace-nowrap font-medium">
          {hourType}
        </span>
        <input
          type="text"
          inputMode="decimal"
          value={displayValue}
          placeholder="0"
          disabled={disabled}
          onFocus={onFocus}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          onKeyDown={handleNumericInputKeyDown}
          className={cn(
            "w-14 h-9 text-center text-base tabular-nums rounded-lg transition-all duration-150",
            "border bg-background",
            inputColor,
            "focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20",
            disabled && "cursor-not-allowed",
          )}
        />
        {entryAmount > 0 && (
          <span className="text-[10px] text-emerald-600 font-medium tabular-nums">
            {formatVND(entryAmount)}
          </span>
        )}
      </div>
    );
  },
);

HourInputField.displayName = "HourInputField";
