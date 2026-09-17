import { Minus } from "lucide-react";
import { cn } from "@/lib/utils";
import { STATUS_META } from "./constants";
import type { CronStatus } from "./constants";

export interface CronStatusBadgeProps {
  status: CronStatus | null;
  variant?: "pill" | "inline";
  className?: string;
}

const NULL_STYLE = {
  color: "text-gray-600",
  label: "Chưa chạy",
} as const;

export function CronStatusBadge({
  status,
  variant = "inline",
  className,
}: CronStatusBadgeProps) {
  const meta = status ? STATUS_META[status] : null;
  const Icon = meta?.icon ?? Minus;
  const color = meta?.color ?? NULL_STYLE.color;
  const label = meta?.label ?? NULL_STYLE.label;
  const animateClass =
    status === "running"
      ? variant === "pill"
        ? "animate-spin"
        : "animate-spin"
      : "";

  if (variant === "pill") {
    if (!meta) {
      return (
        <div
          className={cn(
            "shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap bg-gray-50 text-gray-600",
            className
          )}
        >
          <Icon className="h-3 w-3" />
          <span>{label}</span>
        </div>
      );
    }
    return (
      <div
        className={cn(
          "shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap",
          meta.bg,
          meta.color,
          className
        )}
      >
        <Icon className={cn("h-3 w-3", animateClass)} />
        <span>{label}</span>
      </div>
    );
  }

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <Icon className={cn("h-3.5 w-3.5", color, animateClass)} />
      <span className={cn("text-xs font-medium", color)}>{label}</span>
    </div>
  );
}
