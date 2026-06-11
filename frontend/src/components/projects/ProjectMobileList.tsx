import React from 'react';
import { Calendar, Users } from 'lucide-react';
import type { Project } from '@/types/api/project.types';
import { Badge } from '@/components/ui/badge';
import { ProjectStatusBadge } from '@/components/projects/ProjectStatusBadge';
import { Briefcase } from 'lucide-react';
import { formatDate } from '@/utils/formatters';

interface ProjectMobileListProps {
  projects: Project[];
  onRowClick?: (project: Project) => void;
  emptyState?: React.ReactNode;
}

function formatDateShort(dateStr: string | null | undefined): string {
  if (!dateStr) return '—';
  return formatDate(dateStr);
}

const ProjectCard = React.memo(function ProjectCard({
  project,
  onRowClick,
}: {
  project: Project;
  onRowClick?: (project: Project) => void;
}) {
  return (
    <div
      onClick={() => onRowClick?.(project)}
      role="button"
      tabIndex={0}
      aria-label={`Xem chi tiết dự án ${project.name}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onRowClick?.(project);
        }
      }}
      className="border border-border rounded-xl bg-card px-3.5 py-3 shadow-sm card-lift active:bg-muted/50 transition-all cursor-pointer touch-manipulation"
    >
      {/* Line 1: name + status badge */}
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-semibold truncate leading-tight">
          {project.name}
        </span>
        <ProjectStatusBadge status={project.status} className="shrink-0 text-[10px] h-5 px-1.5" />
      </div>

      {/* Line 2: code · dates · employee badges */}
      <div className="flex items-center gap-1.5 mt-0.5 min-w-0">
        {project.code && (
          <>
            <span className="text-xs font-mono text-muted-foreground shrink-0">{project.code}</span>
            <span className="text-gray-300 shrink-0">·</span>
          </>
        )}
        <Calendar className="h-3 w-3 text-muted-foreground shrink-0" />
        <span className="text-xs text-muted-foreground shrink-0">
          {formatDateShort(project.start_date)}
          {project.end_date ? ` – ${formatDateShort(project.end_date)}` : ''}
        </span>
        <span className="text-gray-300 shrink-0">·</span>
        <Users className="h-3 w-3 text-muted-foreground shrink-0" />
        <Badge className="font-semibold bg-emerald-50 text-emerald-700 border-emerald-200 border text-[10px] h-4 px-1 shrink-0">
          {project.employee_count || 0}
        </Badge>
        {(project.weekly_salary_employee_count ?? 0) > 0 && (
          <Badge className="font-semibold bg-sky-50 text-sky-700 border-border border text-[10px] h-4 px-1 shrink-0">
            {project.weekly_salary_employee_count}T
          </Badge>
        )}
        {(project.monthly_salary_employee_count ?? 0) > 0 && (
          <Badge className="font-semibold bg-violet-50 text-violet-700 border-violet-200 border text-[10px] h-4 px-1 shrink-0">
            {project.monthly_salary_employee_count}M
          </Badge>
        )}
      </div>
    </div>
  );
});

export function ProjectMobileList({
  projects,
  onRowClick,
  emptyState,
}: ProjectMobileListProps) {
  if (projects.length === 0) {
    return (
      <div className="text-center py-12">
        {emptyState || (
          <>
            <Briefcase className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 typography-title-large">Không tìm thấy dự án nào</h3>
            <p className="mt-2 typography-body-medium text-muted-foreground">
              Không có dữ liệu để hiển thị.
            </p>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="w-full flex flex-col gap-3">
      {projects.map((project) => (
        <ProjectCard key={project.id} project={project} onRowClick={onRowClick} />
      ))}
    </div>
  );
}
