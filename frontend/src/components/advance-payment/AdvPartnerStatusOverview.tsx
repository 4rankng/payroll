import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { CheckCircle2 } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";

export interface AdvPartnerStatusOverviewProps {
  totalPaid: number;
  totalPending: number;
  totalFailed: number;
  totalCancelled: number;
  totalRequests: number;
  totalPaidAmount?: number;
  totalPendingAmount?: number;
  totalFailedAmount?: number;
  totalCancelledAmount?: number;
  /** From backend — totalPaid / (totalRequests - totalCancelled) * 100 */
  successRate: number;
  isLoading?: boolean;
  className?: string;
  /** Strip the card chrome (border/bg/padding) so the pipeline can be embedded
   *  in a shared surface, e.g. the admin "pipeline + health" band. */
  bare?: boolean;
}

interface StatusCellDef {
  key: string;
  label: string;
  /** Progress-rail segment fill. */
  barClass: string;
  /** Status-cell dot fill. */
  dotClass: string;
  getValue: (props: AdvPartnerStatusOverviewProps) => number;
  getAmount: (props: AdvPartnerStatusOverviewProps) => number | undefined;
}

const CELLS: StatusCellDef[] = [
  {
    key: "completed",
    label: "Hoàn tất",
    barClass: "bg-emerald-500",
    dotClass: "bg-emerald-500",
    getValue: (p) => p.totalPaid,
    getAmount: (p) => p.totalPaidAmount,
  },
  {
    key: "pending",
    label: "Chờ xử lý",
    barClass: "bg-amber-400",
    dotClass: "bg-amber-400",
    getValue: (p) => p.totalPending,
    getAmount: (p) => p.totalPendingAmount,
  },
  {
    key: "failed",
    label: "Thất bại",
    barClass: "bg-rose-500",
    dotClass: "bg-rose-500",
    getValue: (p) => p.totalFailed,
    getAmount: (p) => p.totalFailedAmount,
  },
  {
    key: "cancelled",
    label: "Đã hủy",
    barClass: "bg-slate-400",
    dotClass: "bg-slate-400",
    getValue: (p) => p.totalCancelled,
    getAmount: (p) => p.totalCancelledAmount,
  },
];

export const AdvPartnerStatusOverview = memo(function AdvPartnerStatusOverview({
  totalPaid,
  totalPending,
  totalFailed,
  totalCancelled,
  totalRequests,
  totalPaidAmount,
  totalPendingAmount,
  totalFailedAmount,
  totalCancelledAmount,
  successRate,
  isLoading = false,
  className,
  bare = false,
}: AdvPartnerStatusOverviewProps) {
  const props = {
    totalPaid,
    totalPending,
    totalFailed,
    totalCancelled,
    totalRequests,
    totalPaidAmount,
    totalPendingAmount,
    totalFailedAmount,
    totalCancelledAmount,
    successRate,
  };

  const chrome = bare
    ? "rounded-none border-0 bg-transparent p-4 shadow-none"
    : "rounded-xl border border-border/70 bg-card p-4 shadow-[0_1px_2px_0_rgb(15_23_42/0.04)]";

  return (
    <section
      aria-label="Tiến độ yêu cầu"
      className={cn("flex h-full flex-col justify-center", chrome, className)}
    >
      {/* Header — title + total + success rate badges */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h2 className="text-[13px] font-bold tracking-tight text-foreground">
            Tiến độ yêu cầu
          </h2>
          <span className="rounded bg-muted px-1.5 py-px font-financial text-[11px] font-medium tabular-nums text-muted-foreground">
            {totalRequests}
          </span>
          {totalRequests > 0 && (
            <span className="inline-flex items-center gap-1 rounded bg-muted px-1.5 py-px font-financial text-[11px] font-medium tabular-nums text-foreground/70">
              <CheckCircle2 className="h-3 w-3" />
              {successRate.toFixed(0)}%
            </span>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="mt-3 space-y-3">
          <Skeleton className="h-1.5 w-full rounded-full" />
          <div className="grid grid-cols-2 gap-x-4 gap-y-3 lg:grid-cols-4">
            <Skeleton className="h-9 rounded-lg" />
            <Skeleton className="h-9 rounded-lg" />
            <Skeleton className="h-9 rounded-lg" />
            <Skeleton className="h-9 rounded-lg" />
          </div>
        </div>
      ) : totalRequests === 0 ? (
        <div className="mt-3 flex min-h-10 items-center rounded-lg border border-dashed border-border/70 bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          Chưa có yêu cầu trong kỳ đã chọn
        </div>
      ) : (
        <>
          {/* Segmented progress rail — thin futuristic bar; numbers live in the
              cells below so no segment ever clips its value. */}
          <div
            className="mt-2.5 flex h-1.5 overflow-hidden rounded-full bg-muted ring-1 ring-inset ring-black/[0.06]"
            role="progressbar"
            aria-valuenow={totalPaid}
            aria-valuemin={0}
            aria-valuemax={totalRequests}
          >
            {CELLS.map((cell) => {
              const value = cell.getValue(props);
              const width = totalRequests > 0 ? (value / totalRequests) * 100 : 0;
              if (width === 0) return null;

              return (
                <div
                  key={cell.key}
                  title={`${cell.label}: ${value}`}
                  aria-label={`${cell.label}: ${value}`}
                  className={cn(
                    "h-full transition-[filter] hover:brightness-110",
                    cell.barClass,
                  )}
                  style={{ width: `${width}%` }}
                />
              );
            })}
          </div>

          {/* Status cells — dot + uppercase micro label over count + amount */}
          <div className="mt-3 grid grid-cols-2 gap-x-4 gap-y-3 lg:grid-cols-4 lg:gap-x-3">
            {CELLS.map((cell) => {
              const value = cell.getValue(props);
              const amount = cell.getAmount(props);

              return (
                <div key={cell.key} className="min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", cell.dotClass)} />
                    <span className="truncate text-[10px] font-semibold uppercase leading-tight tracking-[0.08em] text-muted-foreground">
                      {cell.label}
                    </span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-baseline gap-x-1.5">
                    <span className="font-financial text-[17px] font-semibold leading-none tracking-[-0.01em] tabular-nums text-foreground">
                      {value}
                    </span>
                    {amount !== undefined && amount > 0 && (
                      <span className="font-financial text-[10.5px] font-medium leading-none text-muted-foreground tabular-nums">
                        {formatCurrency(amount)}
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
    </section>
  );
});
