import React, { useMemo, useState } from "react";
import { Activity, ChevronDown } from "lucide-react";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import {
  SectionLabel,
  LatencyGrid,
  MobileErrorByEndpointCards,
  UserInvestigateTable,
  MobileSummaryPills,
  FailedLoginBadges,
  BrowserPlatformStats,
  TimeRangeToggle,
  type TimeRange,
} from "@/components/system-health";
import { useAPISummary, useRecentErrors } from "@/hooks/api/useSystemHealth";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";

type HealthLevel = "healthy" | "degraded" | "critical";

const HEALTH_META: Record<HealthLevel, { label: string; dot: string; pill: string }> = {
  healthy: {
    label: "Bình Thường",
    dot: "bg-emerald-500",
    pill: "bg-emerald-50 text-emerald-700 border-emerald-200",
  },
  degraded: {
    label: "Cảnh Báo",
    dot: "bg-amber-500",
    pill: "bg-amber-50 text-amber-700 border-amber-200",
  },
  critical: {
    label: "Nghiêm Trọng",
    dot: "bg-red-500",
    pill: "bg-red-50 text-red-700 border-red-200",
  },
};

export default function SystemHealthPage() {
  const [errorDays, setErrorDays] = useState<TimeRange>(1);
  const { data: summaryData } = useAPISummary(errorDays);
  const { data: recentErrors } = useRecentErrors(errorDays);

  const healthMeta = useMemo(() => {
    if (!summaryData) return null;
    const total = summaryData.reduce((s, d) => s + d.count, 0);
    const errors = summaryData.reduce((s, d) => s + (d.error?.total ?? 0), 0);
    const errorRate = total > 0 ? (errors / total) * 100 : 0;
    const avgLat = total > 0
      ? summaryData.reduce((s, d) => s + d.latency.avg * d.count, 0) / total
      : 0;
    let level: HealthLevel = "healthy";
    if (errorRate > 5 || avgLat > 500) level = "critical";
    else if (errorRate > 1 || avgLat > 200 || (recentErrors?.length ?? 0) > 10) level = "degraded";
    return HEALTH_META[level];
  }, [summaryData, recentErrors]);

  return (
    <div className="space-y-3 pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        title="Tình trạng API"
        icon={Activity}
        actions={
          <div className="flex items-center gap-1.5">
            <TimeRangeToggle value={errorDays} onChange={setErrorDays} />
            {healthMeta && (
              <div className={cn(
                "inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-semibold border shrink-0",
                healthMeta.pill,
              )}>
                <span className={cn("h-1.5 w-1.5 rounded-full animate-pulse shrink-0", healthMeta.dot)} />
                {healthMeta.label}
              </div>
            )}
          </div>
        }
      />

      {/* Summary pills */}
      <div className="px-4">
        <MobileSummaryPills days={errorDays} />
      </div>

      {/* Latency — collapsible, default open */}
      <div className="px-4">
        <Collapsible defaultOpen={true}>
          <CollapsibleTrigger asChild>
            <button className="flex min-h-11 items-center gap-2 w-full py-2">
              <SectionLabel className="mb-0">Độ Trễ (7 ngày)</SectionLabel>
              <ChevronDown className="h-3.5 w-3.5 ml-auto text-muted-foreground transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
            </button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <LatencyGrid days={7} limit={12} />
          </CollapsibleContent>
        </Collapsible>
      </div>

      {/* Errors */}
      <div className="px-4 space-y-3">
        <SectionLabel>Lỗi</SectionLabel>
        <MobileErrorByEndpointCards days={errorDays} />
      </div>

      {/* Investigate */}
      <div className="px-4 space-y-3">
        <SectionLabel>Cần Điều Tra</SectionLabel>
        <UserInvestigateTable days={errorDays === 1 ? 1 : 7} limit={10} />
      </div>

      {/* Failed logins */}
      <div className="px-4 space-y-3">
        <FailedLoginBadges days={errorDays === 1 ? 1 : 7} />
      </div>

      {/* Browser & Platform */}
      <div className="px-4 space-y-3">
        <BrowserPlatformStats days={30} />
      </div>

    </div>
  );
}
