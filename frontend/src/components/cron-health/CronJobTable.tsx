import { cronToVietnamese } from "@/utils/cron-human";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { CronStatusBadge } from "./CronStatusBadge";
import { formatDuration, formatExactTime } from "./utils";
import type { CronJob } from "@/types/api/cron-health.types";

export interface CronJobTableProps {
  jobs: CronJob[];
  onToggle: (name: string, enabled: boolean) => void;
  isPending: boolean;
}

export function CronJobTable({ jobs, onToggle, isPending }: CronJobTableProps) {
  return (
    <Card>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[50px]">Bật/Tắt</TableHead>
              <TableHead>Tên Job</TableHead>
              <TableHead>Lịch trình</TableHead>
              <TableHead>Trạng thái</TableHead>
              <TableHead>Lần chạy cuối</TableHead>
              <TableHead>Thời lượng</TableHead>
              <TableHead className="max-w-[200px]">Lỗi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {jobs.map((job) => (
              <TableRow key={job.name} className={cn(!job.is_enabled && "bg-muted/30")}>
                <TableCell>
                  <Switch
                    checked={job.is_enabled}
                    aria-label={`${job.is_enabled ? "Tắt" : "Bật"} tác vụ ${job.name.replace(/_/g, " ")}`}
                    onCheckedChange={(checked) => onToggle(job.name, checked)}
                    disabled={isPending}
                    className={cn(
                      "data-[state=checked]:bg-emerald-500 data-[state=unchecked]:bg-red-400"
                    )}
                  />
                </TableCell>
                <TableCell className="font-medium text-sm">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="cursor-default">{job.name.replace(/_/g, " ")}</span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="font-mono text-xs">{job.name}</p>
                    </TooltipContent>
                  </Tooltip>
                </TableCell>
                <TableCell>
                  <span className="text-xs text-muted-foreground">{cronToVietnamese(job.cron)}</span>
                </TableCell>
                <TableCell>
                  <CronStatusBadge status={job.last_status} variant="inline" />
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {job.last_run ? formatExactTime(job.last_run) : "—"}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {job.last_duration_ms != null ? formatDuration(job.last_duration_ms) : "—"}
                </TableCell>
                <TableCell className="max-w-[200px] text-xs text-red-700 truncate">
                  {job.last_error ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-default truncate block">{job.last_error}</span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-sm">
                        <p className="text-xs">{job.last_error}</p>
                      </TooltipContent>
                    </Tooltip>
                  ) : "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
