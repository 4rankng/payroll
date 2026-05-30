import { cronToVietnamese } from "@/utils/cron-human";
import { Clock, AlertCircle, CalendarClock, Timer } from "lucide-react";
import { cn } from "@/lib/utils";
import { STATUS_META } from "./constants";
import { CronStatusBadge } from "./CronStatusBadge";
import { formatDuration, formatExactTime } from "./utils";
import type { CronJob } from "@/types/api/cron-health.types";

export interface CronJobCardProps {
  job: CronJob;
  onToggle?: (name: string, enabled: boolean) => void;
  isPending?: boolean;
}

export function CronJobCard({ job, onToggle, isPending }: CronJobCardProps) {
  const meta = job.last_status ? STATUS_META[job.last_status] : null;
  const isDisabled = !job.is_enabled;

  return (
    <div
      className={cn(
        "bg-white rounded-2xl border shadow-sm overflow-hidden",
        isDisabled ? "border-gray-100 opacity-55" : "border-gray-200",
        job.last_status === "failed" && !isDisabled && "border-red-200"
      )}
    >
      {/* Body */}
      <div className="px-4 pt-3.5 pb-3">
        <div className="flex items-center gap-3">
          {/* Live dot — tappable toggle */}
          <button
            onClick={() => onToggle?.(job.name, !job.is_enabled)}
            disabled={isPending}
            aria-label={job.is_enabled ? "Tắt" : "Bật"}
            className="relative shrink-0 w-3 h-3 rounded-full focus:outline-none disabled:cursor-not-allowed"
          >
            {/* Ping ring — only when enabled */}
            {job.is_enabled && (
              <span
                className={cn(
                  "absolute inset-0 rounded-full animate-ping opacity-50",
                  meta?.dot ?? "bg-emerald-400"
                )}
              />
            )}
            <span
              className={cn(
                "relative block w-3 h-3 rounded-full",
                job.is_enabled ? (meta?.dot ?? "bg-emerald-500") : "bg-gray-300"
              )}
            />
          </button>

          {/* Name */}
          <p className="flex-1 min-w-0 text-sm font-semibold text-foreground leading-snug capitalize">
            {job.name.replace(/_/g, " ")}
          </p>
        </div>

        {/* Schedule */}
        <div className="flex items-center gap-1.5 mt-1.5 pl-6">
          <CalendarClock className="h-3 w-3 text-muted-foreground shrink-0" />
          <span className="text-xs text-muted-foreground">{cronToVietnamese(job.cron)}</span>
        </div>
      </div>

      {/* Footer strip */}
      <div className="flex items-center gap-2 px-4 py-2 bg-gray-50 border-t border-gray-100">
        <CronStatusBadge status={job.last_status} variant="pill" />
        <div className="flex items-center gap-3 ml-auto text-xs text-muted-foreground">
          {job.last_run && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3 shrink-0" />
              {formatExactTime(job.last_run)}
            </span>
          )}
          {job.last_duration_ms != null && (
            <span className="flex items-center gap-1 shrink-0">
              <Timer className="h-3 w-3 shrink-0" />
              {formatDuration(job.last_duration_ms)}
            </span>
          )}
        </div>
      </div>

      {/* Error */}
      {job.last_error && (
        <div className="mx-4 mb-3 px-3 py-2 bg-red-50 rounded-lg border border-red-100 flex items-start gap-2">
          <AlertCircle className="h-3.5 w-3.5 text-red-500 shrink-0 mt-0.5" />
          <p className="text-xs text-red-500 leading-relaxed break-words">{job.last_error}</p>
        </div>
      )}
    </div>
  );
}
