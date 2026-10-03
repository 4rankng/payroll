import { Badge } from "@/components/ui/badge";
import { getStatusColor } from "@/utils/projectHelpers";
import { getVietnameseProjectStatus } from "@/utils/vietnamese";
import type { Project } from "@/types/api/project.types";
import { Building2 } from "lucide-react";

interface ProjectHeaderProps {
  project: Project;
  showName?: boolean;
}

export function ProjectHeader({ project, showName = true }: ProjectHeaderProps) {
  const statusLabel = getVietnameseProjectStatus(project.status as 'draft' | 'active' | 'paused' | 'completed' | 'cancelled');
  const statusColor = getStatusColor(project.status);

  return (
    <div className="flex items-start justify-between gap-3 w-full">
      <div className="flex-1 min-w-0">
        {showName && (
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <h1 className="text-base font-semibold leading-tight break-words">
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
        {/* Metadata row — inline icon-label pairs inspired by Tailkit a-c-page-headings-05 */}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1 min-w-0">
            <Building2 className="h-3 w-3 shrink-0 opacity-70" />
            <span className="truncate">{project.client_name || 'Chưa có khách hàng'}</span>
          </span>
        </div>
      </div>
    </div>
  );
}
