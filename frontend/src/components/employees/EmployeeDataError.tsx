import { AlertCircle, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";

interface EmployeeDataErrorProps {
  title: string;
  onRetry: () => void;
  isRetrying?: boolean;
  className?: string;
}

export function EmployeeDataError({ title, onRetry, isRetrying = false, className }: EmployeeDataErrorProps) {
  return (
    <div className={cn("rounded-xl border border-[var(--employee-border)] bg-[var(--employee-surface)] px-4 py-5 text-center", className)} role="alert">
      <AlertCircle className="mx-auto h-6 w-6 text-[var(--employee-text-secondary)]" aria-hidden="true" />
      <p className="employee-type-strong mt-2 text-[var(--employee-text)]">{title}</p>
      <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Kiểm tra kết nối rồi thử lại.</p>
      <button
        type="button"
        onClick={onRetry}
        disabled={isRetrying}
        className="employee-type-action mt-3 inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-[var(--employee-border-strong)] px-4 text-[var(--employee-accent)] transition-colors hover:bg-[var(--employee-accent-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2 disabled:opacity-60"
      >
        <RefreshCw className={cn("h-4 w-4", isRetrying && "animate-spin")} aria-hidden="true" />
        {isRetrying ? "Đang tải lại…" : "Tải lại"}
      </button>
    </div>
  );
}
