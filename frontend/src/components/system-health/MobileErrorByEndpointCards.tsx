import { useMemo, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useAPISummary } from "@/hooks/api/useSystemHealth";
import { EndpointLabel } from "./EndpointLabel";
import { AllClear } from "./AllClear";

const PAGE_SIZE = 5;

interface MobileErrorByEndpointCardsProps {
  days?: number;
}

export function MobileErrorByEndpointCards({ days = 1 }: MobileErrorByEndpointCardsProps) {
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
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-16 w-full rounded-xl" />
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
            "px-3 py-3 rounded-xl border",
            row.errorRate > 20
              ? "bg-red-50/50 border-red-100"
              : row.errorRate > 5
                ? "bg-amber-50/40 border-amber-100"
                : "bg-background"
          )}
        >
          {/* Endpoint */}
          <div className="mb-1.5 min-w-0">
            <EndpointLabel
              endpoint={row.endpoint}
              className="whitespace-normal break-all text-xs leading-snug"
            />
          </div>
          {/* Stats row */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-bold text-red-600 tabular-nums text-sm">
              {row.errorTotal.toLocaleString()} lỗi
            </span>
            <span className={cn(
              "text-xs font-semibold tabular-nums",
              row.errorRate > 20 ? "text-red-600"
                : row.errorRate > 5 ? "text-amber-600"
                : "text-muted-foreground"
            )}>
              {row.errorRate.toFixed(1)}%
            </span>
            <div className="flex gap-1 ml-auto">
              {row.codes.slice(0, 3).map(({ code, count }) => (
                <Badge
                  key={code}
                  variant={code >= 500 ? "destructive" : "outline"}
                  className="text-[11px] px-1.5 py-0 h-5 font-mono"
                >
                  {code} ×{count}
                </Badge>
              ))}
            </div>
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
