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
      "flex min-w-0 items-start gap-2 rounded-xl border bg-card px-3 py-3 shadow-sm sm:gap-3 sm:px-4 sm:py-3.5",
      borderCls[status],
      className,
    )}>
      <div className={cn("mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-xl sm:h-9 sm:w-9", iconBg[status])}>
        <Icon className="h-[18px] w-[18px]" strokeWidth={2} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="mb-1.5 text-xs font-semibold uppercase leading-none tracking-[0.08em] text-muted-foreground sm:tracking-wider">
          {label}
        </p>
        <div className="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1">
          <p className={cn("text-xl font-bold tabular-nums leading-none sm:text-2xl", valueCls[status])}>
            {value}
          </p>
          {sub && (
            <span className="basis-full text-xs leading-none text-muted-foreground sm:basis-auto sm:whitespace-nowrap">
              {sub}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
