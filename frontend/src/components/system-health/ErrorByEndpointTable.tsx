import { useMemo, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronDown, ChevronUp, ChevronsUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAPISummary } from "@/hooks/api/useSystemHealth";
import { EndpointLabel } from "./EndpointLabel";
import { AllClear } from "./AllClear";

type SortKey = "endpoint" | "errorTotal" | "errorRate";
type SortDir = "asc" | "desc";

function SortIcon({ col, sortKey, sortDir }: { col: SortKey; sortKey: SortKey; sortDir: SortDir }) {
  if (col !== sortKey) return <ChevronsUpDown className="inline h-3 w-3 ml-0.5 opacity-40" />;
  return sortDir === "asc"
    ? <ChevronUp className="inline h-3 w-3 ml-0.5" />
    : <ChevronDown className="inline h-3 w-3 ml-0.5" />;
}

export function ErrorByEndpointTable() {
  const { data: summaryData, isLoading } = useAPISummary(1, "error_total");
  const [sortKey, setSortKey] = useState<SortKey>("errorTotal");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  function handleSort(key: SortKey) {
    if (key === sortKey) setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    else { setSortKey(key); setSortDir("desc"); }
  }

  const rows = useMemo(() => {
    if (!summaryData) return [];
    const mapped = summaryData
      .filter((s) => (s.error?.total ?? 0) > 0)
      .map((s) => {
        const errorTotal = s.error!.total;
        const errorRate = s.count > 0 ? (errorTotal / s.count) * 100 : 0;
        const codes = Object.entries(s.error ?? {})
          .filter(([k]) => k !== "total")
          .map(([k, v]) => ({ code: Number(k), count: v as number }))
          .sort((a, b) => b.count - a.count);
        return { ...s, errorTotal, errorRate, codes };
      });

    return mapped.sort((a, b) => {
      let cmp = 0;
      if (sortKey === "endpoint") cmp = a.endpoint.localeCompare(b.endpoint);
      else if (sortKey === "errorTotal") cmp = a.errorTotal - b.errorTotal;
      else if (sortKey === "errorRate") cmp = a.errorRate - b.errorRate;
      return sortDir === "asc" ? cmp : -cmp;
    });
  }, [summaryData, sortKey, sortDir]);

  return (
    <div>
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <AllClear text="Không có lỗi" />
      ) : (
        <div className="border rounded-xl overflow-hidden">
          <div className="grid grid-cols-[1fr_64px_72px_160px] gap-x-4 px-4 py-2 border-b bg-muted/30
                          text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <button onClick={() => handleSort("endpoint")} className="text-left hover:text-foreground transition-colors">
              Endpoint <SortIcon col="endpoint" sortKey={sortKey} sortDir={sortDir} />
            </button>
            <button onClick={() => handleSort("errorTotal")} className="text-right w-full hover:text-foreground transition-colors">
              Lỗi <SortIcon col="errorTotal" sortKey={sortKey} sortDir={sortDir} />
            </button>
            <button onClick={() => handleSort("errorRate")} className="text-right w-full hover:text-foreground transition-colors">
              Tỷ Lệ <SortIcon col="errorRate" sortKey={sortKey} sortDir={sortDir} />
            </button>
            <span className="text-right">Mã Lỗi</span>
          </div>
          <div className="divide-y max-h-96 overflow-y-auto">
            {rows.map((row) => (
              <div
                key={row.endpoint}
                className={cn(
                  "grid grid-cols-[1fr_64px_72px_160px] gap-x-4 px-4 py-3 items-center",
                  row.errorRate > 20
                    ? "bg-red-50/50"
                    : row.errorRate > 5
                      ? "bg-amber-50/40"
                      : ""
                )}
              >
                <div className="overflow-x-auto">
                  <EndpointLabel endpoint={row.endpoint} />
                </div>
                <span className="text-right text-sm font-bold text-red-600 tabular-nums">
                  {row.errorTotal.toLocaleString()}
                </span>
                <span className={cn(
                  "text-right text-sm font-semibold tabular-nums",
                  row.errorRate > 20 ? "text-red-600"
                    : row.errorRate > 5 ? "text-amber-600"
                    : "text-muted-foreground"
                )}>
                  {row.errorRate.toFixed(1)}%
                </span>
                <div className="flex flex-wrap gap-1 justify-end">
                  {row.codes.slice(0, 4).map(({ code, count }) => (
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
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
