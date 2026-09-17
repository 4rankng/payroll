import { CheckCircle2, XCircle, Loader2 } from "lucide-react";

export const STATUS_META = {
  success: {
    icon: CheckCircle2,
    color: "text-emerald-700",
    bg: "bg-emerald-50",
    border: "border-emerald-200",
    label: "Thành công",
    dot: "bg-emerald-500",
  },
  failed: {
    icon: XCircle,
    color: "text-red-700",
    bg: "bg-red-50",
    border: "border-red-200",
    label: "Thất bại",
    dot: "bg-red-500",
  },
  running: {
    icon: Loader2,
    color: "text-amber-800",
    bg: "bg-amber-50",
    border: "border-amber-200",
    label: "Đang chạy",
    dot: "bg-amber-400",
  },
} as const;

export type CronStatus = keyof typeof STATUS_META;

export function computeCronSummary(
  jobs: readonly { is_enabled: boolean; last_status: string | null }[]
) {
  return {
    total: jobs.length,
    enabled: jobs.filter((j) => j.is_enabled).length,
    failed: jobs.filter((j) => j.last_status === "failed").length,
  };
}
