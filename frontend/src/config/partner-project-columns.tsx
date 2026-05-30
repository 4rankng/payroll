import { format } from "date-fns";
import { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Calendar, Users, Building2, Trash2 } from 'lucide-react';
import { formatDateForDisplay } from '@/utils/dateHelpers';
import { getVietnameseProjectStatus, getProjectStatusVariant } from '@/utils/vietnamese';
import { getEmployeeCountColor, formatEmployeeCount } from '@/utils/projectHelpers';
import type { Project } from '@/types/api/project.types';

interface CreatePartnerProjectColumnsProps {
  onRowClick?: (project: Project) => void;
  onDelete?: (project: Project) => void;
  pagination?: {
    page: number;
    pageSize: number;
  };
}

export function createPartnerProjectColumns({
  onRowClick,
  onDelete,
  pagination,
}: CreatePartnerProjectColumnsProps = {}): ColumnDef<Project>[] {
  return [
    {
      id: "stt",
      header: "STT",
      cell: ({ row }) => {
        const baseIndex = pagination
          ? (pagination.page - 1) * pagination.pageSize
          : 0;
        return (
          <div className="typography-body-medium text-center font-medium tabular-nums">
            {baseIndex + row.index + 1}
          </div>
        );
      },
      size: 60,
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Tên dự án",
      cell: ({ row }) => {
        const project = row.original;
        return (
          <div 
            className="space-y-1 cursor-pointer group"
            onClick={() => onRowClick?.(project)}
          >
            <div className="typography-body-medium font-medium text-foreground group-hover:text-primary transition-colors line-clamp-1">
              {project.name}
            </div>
            <div className="typography-body-small text-muted-foreground line-clamp-1">
              #{project.code}
            </div>
          </div>
        );
      },
    },
    {
      id: "client",
      accessorKey: "client_name",
      header: "Khách hàng",
      cell: ({ row }) => {
        const project = row.original;
        const clientName = project.client_name;
        return (
          <div className="flex items-center gap-2">
            <Building2 className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <span className="typography-body-medium text-foreground line-clamp-1">
              {clientName || "Chưa có thông tin"}
            </span>
          </div>
        );
      },
    },
    {
      id: "dates",
      accessorKey: "start_date",
      header: "Thời gian",
      cell: ({ row }) => {
        const project = row.original;
        return (
          <div className="space-y-1">
            <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
              <Calendar className="h-3 w-3 flex-shrink-0" />
              <span>Bắt đầu: {format(new Date(project.start_date), 'dd/MM/yyyy')}</span>
            </div>
            {project.end_date && (
              <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
                <Calendar className="h-3 w-3 flex-shrink-0" />
                <span>Kết thúc: {format(new Date(project.end_date), 'dd/MM/yyyy')}</span>
              </div>
            )}
          </div>
        );
      },
    },
    {
      id: "team_size",
      accessorKey: "employee_count",
      header: "Nhân sự",
      cell: ({ row }) => {
        const project = row.original;
        const count = project.employee_count || 0;
        return (
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <span className={`typography-data-medium tabular-nums font-semibold ${getEmployeeCountColor(count)}`}>
              {formatEmployeeCount(count)} {count !== 0 ? 'người' : ''}
            </span>
          </div>
        );
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: "Trạng thái",
      cell: ({ row }) => {
        const status = row.getValue("status") as string;
        
        const statusText = getVietnameseProjectStatus(status as 'draft' | 'active' | 'paused' | 'completed' | 'cancelled') || status || 'Không xác định';
        const statusVariant = getProjectStatusVariant(status);

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
    ...(onDelete ? [{
      id: "actions",
      header: "Thao tác",
      cell: ({ row }) => {
        const project = row.original;
        return (
          <div className="flex items-center justify-center">
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(project);
              }}
              className="h-8 w-8 p-0 text-red-600 hover:text-red-700 hover:bg-red-50"
              title="Xóa dự án"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        );
      },
      size: 80,
    }] : []),
  ];
}