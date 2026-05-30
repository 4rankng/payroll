import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import { Employee, getEmployeeProjects, getEmployeeProjectCount } from "@/types/api/employee.types";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Badge } from "@/components/ui/badge";
import { Building, Phone, MapPin } from "lucide-react";

export interface PartnerEmployeeColumnsDeps {
  onRowClick: (employee: Employee) => void;
}

export const createPartnerEmployeeColumns = (
  deps: PartnerEmployeeColumnsDeps,
): ColumnDef<Employee>[] => {
  const { onRowClick } = deps;

  return [
    {
      id: "stt",
      header: "STT",
      size: 60,
      maxSize: 80,
      cell: ({ row }) => (
        <div className="text-center typography-label-medium">
          {row.index + 1}
        </div>
      ),
      enableHiding: false,
    },
    {
      id: "employee_info",
      header: "Nhân viên",
      size: 250,
      cell: ({ row }) => {
        const employee = row.original;
        return (
          <div
            className="flex items-center gap-3 cursor-pointer hover:bg-accent/50 -mx-2 px-2 py-1 rounded-md transition-colors min-w-[200px]"
            onClick={(e) => {
              e.stopPropagation();
              onRowClick(employee);
            }}
          >
            <UserAvatar
              size="md"
              name={employee.fullname}
              username={employee.username}
              cccd={employee.cccd}
            />
            <div className="min-w-0 flex-1">
              <p className="typography-label-medium truncate">
                {employee.fullname}
              </p>
              <p className="typography-body-medium text-muted-foreground truncate">
                {employee.email || "-"}
              </p>
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "mobile",
      header: "Điện thoại",
      size: 140,
      cell: ({ row }) => (
        <div className="flex items-center gap-2 min-w-[120px]">
          <Phone className="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span className="truncate">{row.getValue("mobile") || "-"}</span>
        </div>
      ),
    },
    {
      id: "current_projects",
      header: "Dự án hiện tại",
      size: 200,
      cell: ({ row }) => {
        const employee = row.original;
        const projects = getEmployeeProjects(employee);
        const projectCount = getEmployeeProjectCount(employee);

        return (
          <div className="flex items-center gap-2 min-w-[180px]">
            <Building className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <div className="min-w-0 flex-1">
              {projectCount === 0 ? (
                <span className="text-muted-foreground">Chưa có dự án</span>
              ) : projectCount === 1 ? (
                <div>
                  <p className="typography-label-medium truncate">
                    {projects[0].name}
                  </p>
                  <p className="typography-body-medium text-muted-foreground truncate">
                    {projects[0].code}
                  </p>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <div className="min-w-0 flex-1">
                    <p className="typography-label-medium truncate">
                      {projects[0].name}
                    </p>
                    <p className="typography-body-medium text-muted-foreground truncate">
                      {projects[0].code}
                    </p>
                  </div>
                  <Badge variant="secondary" className="text-xs">
                    +{projectCount - 1}
                  </Badge>
                </div>
              )}
            </div>
          </div>
        );
      },
    },
    {
      id: "position",
      header: "Vị trí",
      size: 140,
      cell: ({ row }) => {
        const employee = row.original;
        const projects = getEmployeeProjects(employee);

        // Show position from the first project, or show multiple positions if different
        const positions = projects.map(p => p.position).filter(Boolean);
        const uniquePositions = [...new Set(positions)];

        return (
          <div className="min-w-[120px]">
            <span className="truncate">
              {uniquePositions.length === 0
                ? "-"
                : uniquePositions.length === 1
                  ? uniquePositions[0]
                  : `${uniquePositions[0]} +${uniquePositions.length - 1}`
              }
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      size: 120,
      cell: ({ row }) => {
        const status = row.getValue("status") as string;
        const statusLabels = {
          active: "Đang dùng",
          inactive: "Ngừng hoạt động",
          pending: "Chờ xử lý"
        };

        const statusColors = {
          active: "bg-green-100 text-green-800",
          inactive: "bg-red-100 text-red-800",
          pending: "bg-yellow-100 text-yellow-800"
        };

        return (
          <div className="min-w-[100px]">
            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${statusColors[status as keyof typeof statusColors] || 'bg-gray-100 text-gray-800'}`}>
              {statusLabels[status as keyof typeof statusLabels] || status}
            </span>
          </div>
        );
      },
    },
  ];
};

export default createPartnerEmployeeColumns;
