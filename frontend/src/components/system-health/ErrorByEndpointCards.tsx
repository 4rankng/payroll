import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useAPISummary } from "@/hooks/api/useSystemHealth";
import { EndpointLabel } from "./EndpointLabel";
import { AllClear } from "./AllClear";

const PAGE_SIZE = 8;

interface ErrorByEndpointCardsProps {
  days?: number;
}

export function ErrorByEndpointCards({ days = 1 }: ErrorByEndpointCardsProps) {
  const { data: summaryData, isLoading } = useAPISummary(days, "error_total");
  const [visible, setVisible] = useState(PAGE_SIZE);

  const rows = useMemo(() => {
    if (!summaryData) return [];
    return summaryData
      .filter((s) => (s.error?.total ?? 0) > 0)
      .map((s) => {
        const errorTotal = s.error!.total;
        const errorRate = s.count > 0 ? (errorTotal / s.count) * 100 : 0;
        const codes = Object.entries(s.error ?? {})
          .filter(([k]) => k !== "total")
          .map(([k, v]) => ({ code: Number(k), count: v as number }))
          .sort((a, b) => b.count - a.count);
        return { ...s, errorTotal, errorRate, codes };
      })
      .sort((a, b) => b.errorTotal - a.errorTotal);
  }, [summaryData]);

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-14 w-full rounded-xl" />
        ))}
      </div>
    );
  }

  if (rows.length === 0) return <AllClear text="Không có lỗi" />;

  return (
    <div className="space-y-2">
      {rows.slice(0, visible).map((row) => (
        <div
          key={row.endpoint}
          className={cn(
            "flex flex-col gap-2 px-4 py-3 rounded-xl border text-sm",
            row.errorRate > 20
              ? "bg-utility-error-50/50 border-utility-error-100"
              : row.errorRate > 5
                ? "bg-utility-warning-50/40 border-utility-warning-100"
                : "bg-card"
          )}
        >
          {/* Top row: endpoint + total error count */}
          <div className="flex items-center justify-between gap-4">
            <div className="flex-1 min-w-0">
              <EndpointLabel endpoint={row.endpoint} />
            </div>
            <div className="flex items-center gap-1 shrink-0">
              <span className="text-xs text-muted-foreground">Tổng lỗi:</span>
              <span className="font-bold text-fg-error-primary tabular-nums">
                {row.errorTotal.toLocaleString()}
              </span>
            </div>
          </div>

          {/* Bottom row: per-status breakdown */}
          <div className="flex flex-wrap gap-1">
            {row.codes.slice(0, 5).map(({ code, count }) => (
              <span
                key={code}
                className={cn(
                  "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium border",
                  code >= 500
                    ? "bg-utility-error-100 border-utility-error-200 text-fg-error-primary"
                    : code >= 400
                      ? "bg-utility-warning-50 border-utility-warning-200 text-fg-warning-primary"
                      : "bg-muted border-border text-muted-foreground"
                )}
              >
                <span className="font-mono font-bold">{code}</span>
                <span className="opacity-50">×</span>
                <span>{count}</span>
              </span>
            ))}
          </div>
        </div>
      ))}
      {visible < rows.length && (
        <Button
          variant="ghost"
          size="sm"
          className="w-full text-muted-foreground"
          onClick={() => setVisible((v) => v + PAGE_SIZE)}
        >
          Xem thêm {Math.min(PAGE_SIZE, rows.length - visible)} / {rows.length - visible} còn lại
        </Button>
      )}
    </div>
  );
}
