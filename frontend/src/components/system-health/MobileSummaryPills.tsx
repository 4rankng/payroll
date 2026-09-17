import { useMemo } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { useAPISummary, useErrorCount } from "@/hooks/api/useSystemHealth";
import { StatusPill } from "./StatusPill";
import { AlertTriangle, AlertCircle } from "lucide-react";

interface MobileSummaryPillsProps {
  days?: number;
}

/** 2×1 grid variant for mobile */
export function MobileSummaryPills({ days = 1 }: MobileSummaryPillsProps) {
  const { data: summaryData, isLoading } = useAPISummary(days);
  const { data: errorCountData } = useErrorCount(days);

  const stats = useMemo(() => {
    if (!summaryData) return null;
    const total = summaryData.reduce((s, d) => s + d.count, 0);
    const errors = summaryData.reduce((s, d) => s + (d.error?.total ?? 0), 0);
    const errorRate = total > 0 ? (errors / total) * 100 : 0;
    return { total, errorRate };
  }, [summaryData]);

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-[72px] rounded-xl" />
        ))}
      </div>
    );
  }
  if (!stats) return null;

  const rc = errorCountData?.total ?? 0;

  return (
    <div className="grid grid-cols-2 gap-2">
      <StatusPill
        icon={AlertTriangle}
        label="Tỷ Lệ Lỗi"
        value={`${stats.errorRate.toFixed(2)}%`}
        sub={`${stats.total.toLocaleString()} req`}
        status={stats.errorRate > 5 ? "danger" : stats.errorRate > 1 ? "warn" : "ok"}
      />
      <StatusPill
        icon={AlertCircle}
        label={days === 1 ? "Lỗi 24h" : "Lỗi 7 ngày"}
        value={String(rc)}
        sub="lượt lỗi"
        status={rc > 20 ? "danger" : rc > 0 ? "warn" : "ok"}
      />
    </div>
  );
}
