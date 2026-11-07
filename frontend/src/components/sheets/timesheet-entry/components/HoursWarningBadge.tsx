import { memo } from "react";
import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

interface HoursWarningBadgeProps {
  totalHours: number;
  status: "exceeded" | "excessive";
}

export const HoursWarningBadge = memo(
  ({ totalHours, status }: HoursWarningBadgeProps) => (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 rounded-lg",
        status === "excessive"
          ? "bg-destructive/15 border border-destructive/30"
          : "bg-warning/15 border border-warning/30",
      )}
    >
      <AlertTriangle
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          status === "excessive" ? "text-destructive" : "text-warning",
        )}
      />
      <p
        className={cn(
          "text-xs font-semibold",
          status === "excessive" ? "text-destructive" : "text-warning",
        )}
      >
        {totalHours}h —{" "}
        {status === "excessive"
          ? "Vượt quá 16h, kiểm tra lại"
          : "Vượt quá 12h, chú ý tăng ca"}
      </p>
    </div>
  ),
);

HoursWarningBadge.displayName = "HoursWarningBadge";
