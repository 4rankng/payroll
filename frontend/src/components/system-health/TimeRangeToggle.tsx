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
    <div role="tablist" aria-label="Khoảng thời gian" className="inline-flex h-10 shrink-0 items-end rounded-xl border border-border bg-muted/50 p-0.5">
      {OPTIONS.map((opt) => (
        <button
          type="button"
          key={opt.value}
          onClick={() => onChange(opt.value)}
          role="tab"
          aria-selected={value === opt.value}
          className={cn(
            "inline-flex h-9 min-h-9 items-center justify-center whitespace-nowrap rounded-lg px-3 text-xs font-semibold transition-colors select-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-brand",
            value === opt.value
              ? "bg-card text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
