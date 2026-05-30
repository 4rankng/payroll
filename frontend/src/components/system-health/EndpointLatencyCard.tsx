import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { METHOD_COLORS, latencyStatus, parseEndpoint } from "./utils";
import type { SlowestEndpoint } from "@/types/api/system-health.types";

interface Props {
  endpoint: SlowestEndpoint;
}

export function EndpointLatencyCard({ endpoint }: Props) {
  const p95Status = latencyStatus(endpoint.p95_ms);
  const parsed = parseEndpoint(endpoint.endpoint);

  return (
    <Card className="px-3 py-2.5">
      {/* Badge inline, path wraps to next line naturally */}
      <div className="mb-1.5 flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5">
        {parsed ? (
          <>
            <span className={cn(
              "shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded leading-none",
              METHOD_COLORS[parsed.method] ?? "bg-muted text-muted-foreground",
            )}>
              {parsed.method}
            </span>
            <span className="font-mono text-xs text-foreground break-all leading-snug">
              {parsed.path}
            </span>
          </>
        ) : (
          <span className="font-mono text-xs text-foreground break-all leading-snug">
            {endpoint.endpoint}
          </span>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs">
        <span className="flex items-center gap-1">
          <span className="inline-block w-3 h-0.5 rounded shrink-0 bg-destructive" />
          <span>P95: </span>
          <span className={cn(
            "font-bold",
            p95Status === "danger" ? "text-red-600"
              : p95Status === "warn" ? "text-amber-500"
              : "text-foreground"
          )}>
            {endpoint.p95_ms.toLocaleString()}ms
          </span>
        </span>
        <span className="flex items-center gap-1 text-muted-foreground">
          <span className="inline-block w-3 h-0.5 rounded shrink-0 bg-primary" />
          <span>Avg: </span>
          <span className="font-medium text-foreground">{endpoint.avg_ms.toLocaleString()}ms</span>
        </span>
        <span className="text-muted-foreground">{endpoint.count.toLocaleString()} calls</span>
      </div>
    </Card>
  );
}
