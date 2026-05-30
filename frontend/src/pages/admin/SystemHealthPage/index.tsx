import React, { useMemo, useState } from "react";
import { useAppState } from "@/contexts";
import { Activity, AlertTriangle, Gauge, Eye } from "lucide-react";
import { cn } from "@/lib/utils";
import { PageHeader } from "@/components/shared/PageHeader";
import {
  SummaryPills,
  SectionLabel,
  LatencyGrid,
  ErrorByEndpointCards,
  UserInvestigateTable,
  FailedLoginBadges,
  BrowserPlatformStats,
  TimeRangeToggle,
  type TimeRange,
} from "@/components/system-health";
import { useAPISummary, useRecentErrors } from "@/hooks/api/useSystemHealth";
import { computeHealthMeta } from "./utils";

export default function SystemHealthPage() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle("Tình trạng API"); }, [setPageTitle]);

  const [errorDays, setErrorDays] = useState<TimeRange>(1);

  const { data: summaryData } = useAPISummary(errorDays);
  const { data: recentErrors } = useRecentErrors(errorDays);

  const healthResult = useMemo(
    () => computeHealthMeta(summaryData, recentErrors),
    [summaryData, recentErrors],
  );
  const healthMeta = healthResult?.meta ?? null;

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">

        <PageHeader icon={Activity} title="Tình trạng API" description="Giám sát hiệu suất và lỗi của hệ thống">
          <div className="flex items-center gap-3">
            <TimeRangeToggle value={errorDays} onChange={setErrorDays} />
            {healthMeta && (
              <div className={cn(
                "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold border",
                healthMeta.pill,
              )}>
                <span className={cn("h-2 w-2 rounded-full animate-pulse", healthMeta.dot)} />
                {healthMeta.label}
              </div>
            )}
          </div>
        </PageHeader>

        <SummaryPills days={errorDays} />

        <section>
          <SectionLabel icon={Gauge}>Độ Trễ — Endpoint (7 ngày)</SectionLabel>
          <LatencyGrid days={7} limit={12} />
        </section>

        <section>
          <SectionLabel icon={AlertTriangle}>Lỗi Theo Endpoint</SectionLabel>
          <ErrorByEndpointCards days={errorDays} />
        </section>

        <section>
          <SectionLabel icon={Eye}>Cần Điều Tra</SectionLabel>
          <UserInvestigateTable days={errorDays === 1 ? 1 : 7} limit={10} />
        </section>

        <section>
          <FailedLoginBadges days={errorDays === 1 ? 1 : 7} />
        </section>

        <section>
          <BrowserPlatformStats days={30} />
        </section>

      </div>
    </div>
  );
}
