import React from 'react';
import { Calendar, Users } from 'lucide-react';
import type { Project } from '@/types/api/project.types';
import { Badge } from '@/components/ui/badge';
import { ProjectStatusBadge } from '@/components/projects/ProjectStatusBadge';
import { formatDate } from '@/utils/formatters';
import { EmptyState } from '@/components/shared/EmptyState';

interface ProjectMobileListProps {
  projects: Project[];
  onRowClick?: (project: Project) => void;
  onTimesheet?: (project: Project) => void;
  emptyState?: React.ReactNode;
}

function formatDateShort(dateStr: string | null | undefined): string {
  if (!dateStr) return '—';
  return formatDate(dateStr);
}

const ProjectCard = React.memo(function ProjectCard({
  project,
  onRowClick,
  onTimesheet,
}: {
  project: Project;
  onRowClick?: (project: Project) => void;
  onTimesheet?: (project: Project) => void;
}) {
  return (
    <div
      className="rounded-xl border border-border bg-card p-3.5 transition-colors"
    >
      <button
        type="button"
        onClick={() => onRowClick?.(project)}
        aria-label={`Xem chi tiết dự án ${project.name}`}
        className="min-h-11 w-full text-left touch-manipulation"
      >
        {/* Line 1: name + status badge */}
        <div className="flex items-start justify-between gap-2">
          <span className="min-w-0 break-words text-sm font-semibold leading-snug line-clamp-2">
            {project.name}
          </span>
          <ProjectStatusBadge status={project.status} className="shrink-0 text-[11px] h-5 px-1.5" />
        </div>

        {/* Line 2: code · dates · employee badges */}
        <div className="flex min-w-0 flex-wrap items-center gap-1.5 mt-0.5">
          {project.code && (
            <>
              <span className="text-xs font-mono text-muted-foreground shrink-0">{project.code}</span>
              <span className="text-gray-300 shrink-0">·</span>
            </>
          )}
          <Calendar className="h-3 w-3 text-muted-foreground shrink-0" />
          <span className="min-w-0 break-words text-xs text-muted-foreground">
            {formatDateShort(project.start_date)}
            {project.end_date ? ` – ${formatDateShort(project.end_date)}` : ''}
          </span>
          <span className="text-gray-300 shrink-0">·</span>
          <Users className="h-3 w-3 text-muted-foreground shrink-0" />
          <Badge className="font-semibold bg-emerald-50 text-emerald-700 border-emerald-200 border text-[11px] h-4 px-1 shrink-0">
            {project.employee_count || 0}
          </Badge>
          {(project.weekly_salary_employee_count ?? 0) > 0 && (
            <Badge className="font-semibold bg-sky-50 text-sky-700 border-border border text-[11px] h-4 px-1 shrink-0">
              {project.weekly_salary_employee_count}T
            </Badge>
          )}
          {(project.monthly_salary_employee_count ?? 0) > 0 && (
            <Badge className="font-semibold bg-emerald-50 text-emerald-700 border-emerald-200 border text-[11px] h-4 px-1 shrink-0">
              {project.monthly_salary_employee_count}M
            </Badge>
          )}
        </div>
      </button>
      {onTimesheet && (
        <button
          type="button"
          className="mt-3 inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border border-border bg-background px-3 text-xs font-semibold text-foreground"
          onClick={() => onTimesheet(project)}
        >
          <Calendar className="h-4 w-4" />
          Bảng công
        </button>
      )}
    </div>
  );
});

export function ProjectMobileList({
  projects,
  onRowClick,
  onTimesheet,
  emptyState,
}: ProjectMobileListProps) {
  if (projects.length === 0) {
    return (
      <div>
        {emptyState || (
          <EmptyState title="Không tìm thấy dự án nào" description="Không có dữ liệu để hiển thị." />
        )}
      </div>
    );
  }

  return (
    <div className="w-full flex flex-col gap-3">
      {projects.map((project) => (
        <ProjectCard
          key={project.id}
          project={project}
          onRowClick={onRowClick}
          onTimesheet={onTimesheet}
        />
      ))}
    </div>
  );
}
