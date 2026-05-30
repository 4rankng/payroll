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
        "flex items-center gap-2 px-3 py-1.5 rounded-xl",
        status === "excessive"
          ? "bg-red-100 border border-red-200"
          : "bg-amber-100 border border-amber-200",
      )}
    >
      <AlertTriangle
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          status === "excessive" ? "text-red-500" : "text-amber-500",
        )}
      />
      <p
        className={cn(
          "text-xs font-semibold",
          status === "excessive" ? "text-red-700" : "text-amber-700",
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
