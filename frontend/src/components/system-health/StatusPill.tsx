import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import type { PillStatus } from "./utils";

const iconBg: Record<PillStatus, string> = {
  ok:     "bg-emerald-50 text-emerald-600",
  warn:   "bg-amber-50 text-amber-600",
  danger: "bg-red-50 text-red-600",
};

const valueCls: Record<PillStatus, string> = {
  ok:     "text-emerald-700",
  warn:   "text-amber-700",
  danger: "text-red-700",
};

const borderCls: Record<PillStatus, string> = {
  ok:     "border-emerald-100",
  warn:   "border-amber-100",
  danger: "border-red-100",
};

interface Props {
  label: string;
  value: string;
  sub?: string;
  status: PillStatus;
  icon: LucideIcon;
  className?: string;
}

export function StatusPill({ label, value, sub, status, icon: Icon, className }: Props) {
  return (
    <div className={cn(
      "flex items-start gap-3 rounded-xl border bg-card px-4 py-3.5 shadow-sm",
      borderCls[status],
      className,
    )}>
      <div className={cn("mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl", iconBg[status])}>
        <Icon className="h-4.5 w-4.5" strokeWidth={2} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground leading-none mb-1.5">
          {label}
        </p>
        <div className="flex items-baseline gap-2">
          <p className={cn("text-2xl font-bold tabular-nums leading-none", valueCls[status])}>
            {value}
          </p>
          {sub && (
            <span className="text-xs text-muted-foreground whitespace-nowrap">{sub}</span>
          )}
        </div>
      </div>
    </div>
  );
}
