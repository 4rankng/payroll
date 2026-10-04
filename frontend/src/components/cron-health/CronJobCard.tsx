import { cronToVietnamese } from "@/utils/cron-human";
// W13c: lucide → verified @untitledui/icons; CalendarClock/Timer →
// CalendarDate/ClockStopwatch (verified names).
import {
  AlertCircle,
  CalendarDate,
  Clock,
  ClockStopwatch,
} from "@untitledui/icons";
import { cn } from "@/lib/utils";
import { CronStatusBadge } from "./CronStatusBadge";
import { formatDuration, formatExactTime } from "./utils";
import type { CronJob } from "@/types/api/cron-health.types";

export interface CronJobCardProps {
  job: CronJob;
  onToggle?: (name: string, enabled: boolean) => void;
  isPending?: boolean;
}

export function CronJobCard({ job, onToggle, isPending }: CronJobCardProps) {
  const isDisabled = !job.is_enabled;

  return (
    <div
      className={cn(
        "bg-white rounded-2xl border shadow-sm overflow-hidden",
        isDisabled ? "border-utility-gray-300 bg-utility-gray-50" : "border-utility-gray-300",
        job.last_status === "failed" && !isDisabled && "border-utility-error-200"
      )}
    >
      {/* Body */}
      <div className="px-4 pt-3.5 pb-3">
        <div className="flex items-center gap-3">
          {/* Explicit switch separates scheduling state from the run result below. */}
          <button
            onClick={() => onToggle?.(job.name, !job.is_enabled)}
            disabled={isPending}
            aria-label={`${job.is_enabled ? "Tắt" : "Bật"} tác vụ ${job.name.replace(/_/g, " ")}`}
            role="switch"
            aria-checked={job.is_enabled}
            className="relative flex h-11 w-11 shrink-0 flex-col items-center justify-center gap-0.5 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:cursor-not-allowed"
          >
            <span className={cn("flex h-6 w-10 items-center rounded-full p-0.5 transition-colors", job.is_enabled ? "bg-primary" : "bg-slate-300")} aria-hidden="true">
              <span className={cn("h-5 w-5 rounded-full bg-white shadow-sm transition-transform", job.is_enabled ? "translate-x-4" : "translate-x-0")} />
            </span>
            <span className="text-xs font-semibold leading-none text-muted-foreground" aria-hidden="true">{job.is_enabled ? "Bật" : "Tắt"}</span>
          </button>

          {/* Name */}
          <p className="flex-1 min-w-0 text-sm font-semibold text-foreground leading-snug capitalize">
            {job.name.replace(/_/g, " ")}
          </p>
        </div>

        {/* Schedule */}
        <div className="flex items-center gap-1.5 mt-1.5 pl-14">
          <CalendarDate className="h-3 w-3 text-muted-foreground shrink-0" />
          <span className="text-xs text-muted-foreground">{cronToVietnamese(job.cron)}</span>
        </div>
      </div>

      {/* Footer strip */}
      <div className="flex flex-wrap items-center gap-2 px-4 py-2 bg-utility-gray-50 border-t border-utility-gray-200">
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
              <ClockStopwatch className="h-3 w-3 shrink-0" />
              {formatDuration(job.last_duration_ms)}
            </span>
          )}
        </div>
      </div>

      {/* Error */}
      {job.last_error && (
        <div className="mx-4 mb-3 px-3 py-2 bg-utility-error-50 rounded-lg border border-utility-error-100 flex items-start gap-2">
          <AlertCircle className="h-3.5 w-3.5 text-fg-error-primary shrink-0 mt-0.5" />
          <p className="text-xs text-fg-error-primary leading-relaxed break-words">{job.last_error}</p>
        </div>
      )}
    </div>
  );
}
