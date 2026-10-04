// W13c: lucide → verified @untitledui/icons; Loading01 replaces lucide's Loader2.
import { CheckCircle, Loading01, XCircle } from "@untitledui/icons";

export const STATUS_META = {
  success: {
    icon: CheckCircle,
    color: "text-fg-success-primary",
    bg: "bg-utility-success-50",
    border: "border-utility-success-200",
    label: "Thành công",
    dot: "bg-utility-success-500",
  },
  failed: {
    icon: XCircle,
    color: "text-fg-error-primary",
    bg: "bg-utility-error-50",
    border: "border-utility-error-200",
    label: "Thất bại",
    dot: "bg-utility-error-500",
  },
  running: {
    icon: Loading01,
    color: "text-fg-warning-primary",
    bg: "bg-utility-warning-50",
    border: "border-utility-warning-200",
    label: "Đang chạy",
    dot: "bg-utility-warning-400",
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
