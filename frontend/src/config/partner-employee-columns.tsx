import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Mail, Phone, MapPin, Calendar, Building2 } from "lucide-react";
import { formatCurrency } from "@/utils/employeeHelpers";
import type { Employee } from "@/types/api/employee.types";

const getPaymentScheduleBadgeConfig = (
  schedule: "weekly" | "monthly" | "flexible",
) => {
  switch (schedule) {
    case "weekly":
      return {
        className: "border-blue-300 text-blue-700 bg-blue-50",
        label: "lương tuần",
      };
    case "monthly":
      return {
        className: "border-amber-300 text-amber-700 bg-amber-50",
        label: "lương tháng",
      };
    case "flexible":
      return {
        className: "border-emerald-300 text-emerald-700 bg-emerald-50",
        label: "linh động",
      };
  }
};

interface CreatePartnerEmployeeColumnsProps {
  onRowClick?: (employee: Employee) => void;
}

export function createPartnerEmployeeColumns({
  onRowClick,
}: CreatePartnerEmployeeColumnsProps = {}): ColumnDef<Employee>[] {
  return [
    {
      id: "employee",
      accessorKey: "name",
      header: "Nhân viên",
      cell: ({ row }) => {
        const employee = row.original as Employee & { name?: string; avatar?: string; employee_code?: string };
        return (
          <div
            className="flex items-center gap-3 cursor-pointer group max-w-xs"
            onClick={() => onRowClick?.(employee)}
          >
            <UserAvatar
              name={employee.name ?? employee.fullname}
              email={employee.email}
              src={employee.avatar}
              size="lg"
            />
            <div className="min-w-0 flex-1">
              <div className="typography-body-medium font-medium text-foreground group-hover:text-primary transition-colors line-clamp-1">
                {employee.name ?? employee.fullname}
              </div>
              <div className="typography-body-small text-muted-foreground line-clamp-1">
                #{employee.employee_code ?? employee.id}
              </div>
            </div>
          </div>
        );
      },
    },
    {
      id: "position",
      accessorKey: "position",
      header: "Vị trí",
      cell: ({ row }) => {
        const employee = row.original as Employee & { department?: string };
        return (
          <div className="space-y-1">
            <div className="typography-body-medium text-foreground line-clamp-1">
              {employee.position || "Chưa xác định"}
            </div>
            {employee.department && (
              <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
                <Building2 className="h-3 w-3 flex-shrink-0" />
                <span className="line-clamp-1">{employee.department}</span>
              </div>
            )}
          </div>
        );
      },
    },
    {
      id: "projects",
      accessorKey: "current_projects",
      header: "Dự án",
      cell: ({ row }) => {
        const employee = row.original;
        const projects = employee.current_projects || [];
        const projectCount = projects.length;

        if (projectCount === 0) {
          return (
            <span className="typography-body-medium text-muted-foreground">
              Chưa có dự án
            </span>
          );
        }

        if (projectCount === 1) {
          const project = projects[0];
          return (
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <p className="typography-body-medium font-medium line-clamp-1">
                  {project.name}
                </p>
                {project.payment_schedule && (
                  <Badge
                    variant="outline"
                    className={`text-[10px] px-1.5 py-0 h-4 shrink-0 rounded-[4px] ${getPaymentScheduleBadgeConfig(project.payment_schedule).className}`}
                  >
                    {
                      getPaymentScheduleBadgeConfig(project.payment_schedule)
                        .label
                    }
                  </Badge>
                )}
              </div>
              <p className="typography-body-small text-muted-foreground">
                {project.code}
                {project.position && ` • ${project.position}`}
              </p>
            </div>
          );
        }

        // Multiple projects - show first and badge with count
        const firstProject = projects[0];
        const remainingCount = projectCount - 1;

        return (
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <div className="flex-1 min-w-0">
                <p className="typography-body-medium font-medium line-clamp-1">
                  {firstProject.name}
                </p>
                <p className="typography-body-small text-muted-foreground">
                  {firstProject.code}
                  {firstProject.position && ` • ${firstProject.position}`}
                </p>
              </div>
              {firstProject.payment_schedule && (
                <Badge
                  variant="outline"
                  className={`text-[10px] px-1.5 py-0 h-4 shrink-0 rounded-[4px] ${getPaymentScheduleBadgeConfig(firstProject.payment_schedule).className}`}
                >
                  {
                    getPaymentScheduleBadgeConfig(firstProject.payment_schedule)
                      .label
                  }
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="text-xs">
                +{remainingCount} dự án
              </Badge>
            </div>
          </div>
        );
      },
    },
    {
      id: "contact",
      accessorKey: "email",
      header: "Liên hệ",
      cell: ({ row }) => {
        const employee = row.original;
        return (
          <div className="space-y-1 max-w-xs">
            {employee.email && (
              <div className="flex items-center gap-2">
                <Mail className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                <span className="typography-body-small text-muted-foreground line-clamp-1">
                  {employee.email}
                </span>
              </div>
            )}
          </div>
        );
      },
    },
    {
      id: "location",
      accessorKey: "address",
      header: "Địa chỉ",
      cell: ({ row }) => {
        const address = row.getValue("location") as string;
        if (!address)
          return (
            <span className="typography-body-small text-muted-foreground">
              Chưa cập nhật
            </span>
          );

        return (
          <div className="flex items-start gap-2 max-w-xs">
            <MapPin className="h-3 w-3 text-muted-foreground flex-shrink-0 mt-0.5" />
            <span className="typography-body-small text-muted-foreground line-clamp-2">
              {address}
            </span>
          </div>
        );
      },
    },
    {
      id: "salary",
      accessorKey: "base_salary",
      header: "Lương cơ bản",
      cell: ({ row }) => {
        const salary = row.getValue("salary") as number | undefined;
        if (salary == null)
          return (
            <span className="typography-body-small text-muted-foreground">
              Chưa thiết lập
            </span>
          );

        return (
          <div className="typography-data-medium tabular-nums font-medium text-foreground">
            {formatCurrency(salary)}
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

        const getStatusVariant = (status: string) => {
          switch (status?.toLowerCase()) {
            case "active":
            case "hoạt động":
              return "default";
            case "inactive":
            case "ngừng hoạt động":
              return "secondary";
            case "pending":
            case "chờ xử lý":
              return "outline";
            case "terminated":
            case "đã nghỉ việc":
              return "destructive";
            default:
              return "secondary";
          }
        };

        const getStatusText = (status: string) => {
          switch (status?.toLowerCase()) {
            case "active":
              return "Đang dùng";
            case "inactive":
              return "Ngừng hoạt động";
            case "pending":
              return "Chờ xử lý";
            case "terminated":
              return "Đã nghỉ việc";
            default:
              return status || "Không xác định";
          }
        };

        return (
          <Badge
            variant={getStatusVariant(status)}
            className="typography-label-small whitespace-nowrap"
          >
            {getStatusText(status)}
          </Badge>
        );
      },
    },
  ];
}
