import { ErrorState } from "@/components/ui/error-state";
import React, { useMemo } from "react";
import { Loader2, Clock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useCronJobs, useToggleCronJob } from "@/hooks/api/useCronHealth";
import { CronJobCard, computeCronSummary } from "@/components/cron-health";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";

export default function CronHealthPageMobile() {
  const { data: jobs, isLoading, isError, refetch } = useCronJobs();
  const toggleMutation = useToggleCronJob();

  const summary = useMemo(() => {
    if (!jobs) return { total: 0, enabled: 0, failed: 0 };
    return computeCronSummary(jobs);
  }, [jobs]);

  if (isLoading) {
    return (
      <div className="min-h-[100dvh] bg-[hsl(var(--surface-page))] flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="min-h-[100dvh] bg-[hsl(var(--surface-page))]">
        <MobilePageHeader
          title="Tác vụ định kỳ"
          icon={Clock}
        />
        <div role="alert" className="px-4 pt-4">
          <ErrorState message="Không thể tải danh sách tác vụ. Vui lòng thử lại." onRetry={() => void refetch()} />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[100dvh] max-w-full overflow-x-clip bg-[hsl(var(--surface-page))]">
      <MobilePageHeader
        title="Tác vụ định kỳ"
        icon={Clock}
        subtitle={
          <span className="inline-flex items-center gap-1.5 flex-wrap">
            <Badge variant="outline" className="text-xs font-normal text-muted-foreground">
              {summary.total} jobs
            </Badge>
            <Badge variant="outline" className="text-xs font-medium text-emerald-700 border-emerald-200 bg-emerald-50">
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

      <div className="px-4 pt-4 pb-[calc(5.75rem+env(safe-area-inset-bottom))] flex flex-col gap-3">
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
