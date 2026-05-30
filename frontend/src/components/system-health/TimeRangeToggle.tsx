import { cn } from "@/lib/utils";

export type TimeRange = 1 | 7;

interface TimeRangeToggleProps {
  value: TimeRange;
  onChange: (v: TimeRange) => void;
}

const OPTIONS: { value: TimeRange; label: string }[] = [
  { value: 1, label: "24h" },
  { value: 7, label: "7 ngày" },
];

export function TimeRangeToggle({ value, onChange }: TimeRangeToggleProps) {
  return (
    <div className="inline-flex rounded-lg border border-border bg-muted/50 p-0.5">
      {OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={cn(
            "px-3 py-1 text-xs font-medium rounded-md transition-colors",
            value === opt.value
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
