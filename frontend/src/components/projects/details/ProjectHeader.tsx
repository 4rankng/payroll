import { Badge } from "@/components/ui/badge";
import { getStatusColor } from "@/utils/projectHelpers";
import { getVietnameseProjectStatus } from "@/utils/vietnamese";
import type { Project } from "@/types/api/project.types";
import { format } from "date-fns";

interface ProjectHeaderProps {
  project: Project;
  showName?: boolean;
}

export function ProjectHeader({ project, showName = true }: ProjectHeaderProps) {
  const statusLabel = getVietnameseProjectStatus(project.status as 'draft' | 'active' | 'paused' | 'completed' | 'cancelled');
  const statusColor = getStatusColor(project.status);

  const formatDateRange = () => {
    const startDate = format(new Date(project.start_date), 'dd/MM/yyyy');
    return `từ ${startDate}`;
  };

  return (
    <div className="flex items-start justify-between gap-3 w-full">
      <div className="flex-1 min-w-0">
        {showName && (
          <div className="flex items-center gap-2 mb-0.5 flex-wrap">
            <h1 className="text-sm font-semibold leading-tight">
              {project.name}
            </h1>
            <Badge className={statusColor} variant="secondary">
              {statusLabel}
            </Badge>
            <span className="text-xs font-mono bg-muted/60 px-1.5 py-0.5 rounded text-muted-foreground">
              {project.code}
            </span>
          </div>
        )}
        <div className="text-xs text-muted-foreground">
          <span>{project.client_name || 'Chưa có khách hàng'}</span>
          <span className="mx-1.5">·</span>
          <span>{formatDateRange()}</span>
        </div>
      </div>
    </div>
  );
}
