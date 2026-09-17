import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { METHOD_COLORS, latencyStatus, parseEndpoint } from "./utils";
import type { SlowestEndpoint } from "@/types/api/system-health.types";

interface Props {
  endpoint: SlowestEndpoint;
  days?: number;
}

export function EndpointLatencyCard({ endpoint }: Props) {
  const p95Status = latencyStatus(endpoint.p95_ms);
  const parsed = parseEndpoint(endpoint.endpoint);

  return (
    <Card className="px-3 py-2.5">
      {/* Method chip and path on one line; the path wraps to at most two lines
          so a long route stays readable without the row growing unbounded. */}
      <div className="flex items-start gap-1.5">
        {parsed && (
          <span
            className={cn(
              "mt-px shrink-0 rounded px-1.5 py-0.5 text-xs font-bold leading-none",
              METHOD_COLORS[parsed.method] ?? "bg-muted text-muted-foreground",
            )}
          >
            {parsed.method}
          </span>
        )}
        <span
          className="min-w-0 font-mono text-xs leading-snug text-foreground line-clamp-2 break-all"
          title={endpoint.endpoint}
        >
          {parsed ? parsed.path : endpoint.endpoint}
        </span>
      </div>

      {/* One left-aligned metric sentence — no stray element pushed to the far
          right, which previously left a ragged gap on every row. */}
      <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
        <span
          className={cn(
            "font-bold tabular-nums",
            p95Status === "danger"
              ? "text-red-700"
              : p95Status === "warn"
                ? "text-amber-700"
                : "text-foreground",
          )}
        >
          P95 {endpoint.p95_ms.toLocaleString()}ms
        </span>
        <span aria-hidden="true" className="h-3 w-px shrink-0 bg-border" />
        <span className="font-medium tabular-nums text-foreground">
          TB {endpoint.avg_ms.toLocaleString()}ms
        </span>
        <span aria-hidden="true" className="h-3 w-px shrink-0 bg-border" />
        <span className="tabular-nums text-muted-foreground">
          {endpoint.count.toLocaleString()} lượt gọi
        </span>
      </div>
    </Card>
  );
}
