import { memo } from "react";
import { cn } from "@/lib/utils";
import { handleNumericInputKeyDown } from "@/utils/numericInput";
import { formatCurrency as formatVND } from "@/utils/formatters";

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

  if (thisFieldDeleted) return "border-destructive/40 bg-destructive/10 text-destructive";
  if (thisFieldEdited)
    return "border-warning/40 bg-warning/10 text-warning font-semibold";
  if (thisFieldNew)
    return "border-success/40 bg-success/10 text-success font-bold";

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
            "border bg-card",
            inputColor,
            "focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20",
            disabled && "cursor-not-allowed",
          )}
        />
        {entryAmount > 0 && (
          <span className="text-xs text-success font-medium tabular-nums">
            {formatVND(entryAmount)}
          </span>
        )}
      </div>
    );
  },
);

HourInputField.displayName = "HourInputField";
