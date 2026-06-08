import React, { useMemo } from "react";
import { Loader2, Clock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useCronJobs, useToggleCronJob } from "@/hooks/api/useCronHealth";
import { CronJobCard, computeCronSummary } from "@/components/cron-health";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";

export default function CronHealthPageMobile() {
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
      <MobilePageHeader
        title="Tác vụ định kỳ"
        icon={Clock}
        subtitle={
          <span className="inline-flex items-center gap-1.5 flex-wrap">
            <Badge variant="outline" className="text-xs font-normal text-muted-foreground">
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
          </span>
        }
      />

      <div className="p-4 flex flex-col gap-3">
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
