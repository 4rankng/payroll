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
    <div role="tablist" aria-label="Khoảng thời gian" className="ct-tabs ct-tabs-box inline-flex h-10 shrink-0 rounded-xl border border-border bg-muted/50 p-0.5">
      {OPTIONS.map((opt) => (
        <button
          type="button"
          key={opt.value}
          onClick={() => onChange(opt.value)}
          role="tab"
          aria-selected={value === opt.value}
          className={cn(
            "ct-tab h-9 min-h-9 whitespace-nowrap rounded-lg px-3 text-xs font-semibold transition-colors",
            value === opt.value
              ? "ct-tab-active bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
