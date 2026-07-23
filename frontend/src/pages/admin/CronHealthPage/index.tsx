import React, { useMemo } from "react";
import { useAppState } from "@/contexts";
import { Clock, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/shared/PageHeader";
import { useCronJobs, useToggleCronJob } from "@/hooks/api/useCronHealth";
import { CronJobTable, computeCronSummary } from "@/components/cron-health";

export default function CronHealthPage() {
  const { setPageTitle } = useAppState();
  React.useEffect(() => { setPageTitle("Tác vụ định kỳ"); }, [setPageTitle]);

  const { data: jobs, isLoading } = useCronJobs();
  const toggleMutation = useToggleCronJob();

  const summary = useMemo(() => {
    if (!jobs) return { total: 0, enabled: 0, failed: 0 };
    return computeCronSummary(jobs);
  }, [jobs]);

  if (isLoading) {
    return (
      <div className="min-h-full flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
        {/* Header */}
        <PageHeader icon={Clock} title="Tác vụ định kỳ">
          <Badge variant="secondary" className="text-xs font-normal">
            {summary.total} tác vụ
          </Badge>
          <Badge variant="secondary" className="text-xs font-normal">
            {summary.enabled} bật
          </Badge>
          {summary.failed > 0 && (
            <Badge variant="destructive" className="text-xs">
              {summary.failed} lỗi
            </Badge>
          )}
        </PageHeader>

        {/* Table */}
        <CronJobTable
          jobs={jobs ?? []}
          onToggle={(name, enabled) => toggleMutation.mutate({ name, enabled })}
          isPending={toggleMutation.isPending}
        />
      </div>
    </div>
  );
}
