import React, { useMemo } from "react";
import { useAppState } from "@/contexts";
import { Loader2, Clock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useCronJobs, useToggleCronJob } from "@/hooks/api/useCronHealth";
import { CronJobCard, computeCronSummary } from "@/components/cron-health";

export default function CronHealthPageMobile() {
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
    <div className="min-h-full pb-20">
      {/* Header */}
      <div className="bg-white border-b border-gray-100 px-4 pt-4 pb-4">
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-lg font-bold text-foreground flex items-center gap-2 shrink-0">
            <Clock className="h-5 w-5 text-primary" />
            Tác vụ định kỳ
          </h1>
          <div className="flex items-center gap-1.5 flex-wrap justify-end">
            <Badge variant="outline" className="text-xs font-normal text-gray-500">
              {summary.total} jobs
            </Badge>
            <Badge variant="outline" className="text-xs font-medium text-emerald-600 border-emerald-200 bg-emerald-50">
              {summary.enabled} bật
            </Badge>
            {summary.failed > 0 && (
              <Badge variant="destructive" className="text-xs">
                {summary.failed} lỗi
              </Badge>
            )}
          </div>
        </div>
      </div>

      {/* Job list */}
      <div className="p-4 space-y-3">
        {jobs?.map((job) => (
          <CronJobCard
            key={job.name}
            job={job}
            onToggle={(name, enabled) => toggleMutation.mutate({ name, enabled })}
            isPending={toggleMutation.isPending}
          />
        ))}
      </div>
    </div>
  );
}
