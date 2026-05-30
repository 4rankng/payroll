import { format } from "date-fns";
import { Badge } from '@/components/ui/badge';
import { Calendar, Users, Building2 } from 'lucide-react';
import { getVietnameseProjectStatus, getProjectStatusVariant } from '@/utils/vietnamese';
import { getEmployeeCountColor, formatEmployeeCount } from '@/utils/projectHelpers';
import type { MobileField } from '@/components/ui/mobile-table';
import type { Project } from '@/types/api/project.types';

interface CreatePartnerProjectMobileConfigProps {
  onRowClick?: (project: Project) => void;
}

export function createPartnerProjectMobileConfig({
  onRowClick,
}: CreatePartnerProjectMobileConfigProps = {}) {
  const mobileFields: MobileField<Project>[] = [
    {
      key: "name",
      label: "Tên dự án",
      priority: 1,
      render: (project) => (
        <div className="typography-body-medium font-medium text-foreground line-clamp-1">
          {project.name}
        </div>
      ),
    },
    {
      key: "code",
      label: "Mã dự án",
      priority: 2,
      render: (project) => (
        <div className="typography-body-small text-muted-foreground">
          #{project.code}
        </div>
      ),
    },
    {
      key: "client",
      label: "Khách hàng",
      priority: 2,
      render: (project) => (
        <div className="flex items-center gap-2">
          <Building2 className="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span className="typography-body-small text-muted-foreground line-clamp-1">
            {project.client_name || "Chưa có thông tin"}
          </span>
        </div>
      ),
    },
    {
      key: "dates",
      label: "Thời gian",
      priority: 3,
      render: (project) => (
        <div className="space-y-1">
          <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
            <Calendar className="h-3 w-3 flex-shrink-0" />
            <span>{format(new Date(project.start_date), 'dd/MM/yyyy')}</span>
          </div>
          {project.end_date && (
            <div className="typography-body-small text-muted-foreground">
              đến {format(new Date(project.end_date), 'dd/MM/yyyy')}
            </div>
          )}
        </div>
      ),
    },
    {
      key: "team_size",
      label: "Nhân sự",
      priority: 3,
      render: (project) => {
        const count = project.employee_count || 0;
        return (
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <span className={`typography-data-small tabular-nums font-semibold ${getEmployeeCountColor(count)}`}>
              {formatEmployeeCount(count)} {count !== 0 ? 'người' : ''}
            </span>
          </div>
        );
      },
    },
    {
      key: "status",
      label: "Trạng thái",
      priority: 1,
      render: (project) => {
        const statusText = getVietnameseProjectStatus(project.status as 'draft' | 'active' | 'paused' | 'completed' | 'cancelled') || project.status || 'Không xác định';
        const statusVariant = getProjectStatusVariant(project.status);

        return (
          <Badge
            variant={statusVariant}
            className="typography-label-small whitespace-nowrap"
          >
            {statusText}
          </Badge>
        );
      },
    },
    {
      key: "priority",
      label: "Ưu tiên",
      priority: 4,
      render: (project) => {
        const getPriorityVariant = (priority: string) => {
          switch (priority?.toLowerCase()) {
            case 'high':
            case 'cao':
              return 'destructive';
            case 'medium':
            case 'trung bình':
              return 'default';
            case 'low':
            case 'thấp':
              return 'secondary';
            default:
              return 'outline';
          }
        };

        const getPriorityText = (priority: string) => {
          switch (priority?.toLowerCase()) {
            case 'high':
              return 'Cao';
            case 'medium':
              return 'Trung bình';
            case 'low':
              return 'Thấp';
            default:
              return priority || 'Chưa xác định';
          }
        };

        if (!project.priority) return null;

        return (
          <Badge 
            variant={getPriorityVariant(project.priority)}
            className="typography-label-small whitespace-nowrap"
          >
            {getPriorityText(project.priority)}
          </Badge>
        );
      },
    },
  ];

  const rowTitle = (project: Project) => (
    <div className="typography-body-medium font-medium text-foreground line-clamp-1">
      {project.name}
    </div>
  );

  const rowSubtitle = (project: Project) => {
    const count = project.employee_count || 0;
    return (
      <div className="flex items-center gap-2 typography-body-small text-muted-foreground">
        <span>#{project.code}</span>
        <span>•</span>
        <span>{project.client_name || "Chưa có khách hàng"}</span>
        <span>•</span>
        <div className="flex items-center gap-1">
          <Users className="h-3 w-3" />
          <span className={`tabular-nums font-semibold ${getEmployeeCountColor(count)}`}>
            {formatEmployeeCount(count)}
          </span>
        </div>
      </div>
    );
  };

  return {
    mobileFields,
    rowTitle,
    rowSubtitle,
  };
}