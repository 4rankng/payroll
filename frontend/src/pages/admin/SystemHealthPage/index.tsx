import React, { useMemo, useState } from "react";
import { useAppState } from "@/contexts";
// UU PRO restyle (W13c): lucide → verified @untitledui/icons; Gauge →
// Speedometer01. Layout, copy and data flow unchanged.
import { Activity, AlertTriangle, Eye, Speedometer01 } from "@untitledui/icons";
import { cn } from "@/lib/utils";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
} from "@/components/shared/AdminPageFrame";
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
import { useLocation } from "react-router-dom";
import { useAPISummary, useRecentErrors } from "@/hooks/api/useSystemHealth";
import { getConsoleKicker } from "@/lib/console";
import { computeHealthMeta } from "./utils";

export default function SystemHealthPage() {
  const { setPageTitle } = useAppState();
  const location = useLocation();
  React.useEffect(() => { setPageTitle("Tình trạng API"); }, [setPageTitle]);

  const [errorDays, setErrorDays] = useState<TimeRange>(1);

  const { data: summaryData } = useAPISummary(errorDays);
  const { data: recentErrors } = useRecentErrors(errorDays);

  const headerMeta = useMemo(() => {
    if (!summaryData) return undefined;
    const total = summaryData.reduce((s, d) => s + d.count, 0);
    return `${total.toLocaleString("vi-VN")} yêu cầu · ${errorDays === 1 ? "24 giờ qua" : "7 ngày qua"}`;
  }, [summaryData, errorDays]);

  const healthResult = useMemo(
    () => computeHealthMeta(summaryData, recentErrors),
    [summaryData, recentErrors],
  );
  const healthMeta = healthResult?.meta ?? null;

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          icon={Activity}
          title="Tình trạng API"
          kicker={getConsoleKicker(location.pathname)}
          meta={headerMeta}
        >
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
      </AdminPageHeaderCard>

      <SummaryPills days={errorDays} />

      <section>
        <SectionLabel icon={Speedometer01}>Độ Trễ — Endpoint (7 ngày)</SectionLabel>
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
    </AdminPageCanvas>
  );
}
