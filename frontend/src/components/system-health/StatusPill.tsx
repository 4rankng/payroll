import { cn } from "@/lib/utils";
// W13c: icon slot widened (W13a pattern); accents on the UU ladders.
import type { ComponentType, SVGProps } from "react";
import type { PillStatus } from "./utils";

const iconBg: Record<PillStatus, string> = {
  ok:     "bg-utility-success-50 text-fg-success-primary",
  warn:   "bg-utility-warning-50 text-fg-warning-primary",
  danger: "bg-utility-error-50 text-fg-error-primary",
};

const valueCls: Record<PillStatus, string> = {
  ok:     "text-fg-success-primary",
  warn:   "text-fg-warning-primary",
  danger: "text-fg-error-primary",
};

const borderCls: Record<PillStatus, string> = {
  ok:     "border-utility-success-100",
  warn:   "border-utility-warning-100",
  danger: "border-utility-error-100",
};

interface Props {
  label: string;
  value: string;
  sub?: string;
  status: PillStatus;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  className?: string;
}

export function StatusPill({ label, value, sub, status, icon: Icon, className }: Props) {
  return (
    <div className={cn(
      "flex h-full min-w-0 items-start gap-2 rounded-xl border bg-card px-3 py-3 shadow-sm sm:gap-3 sm:px-4 sm:py-3.5",
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
