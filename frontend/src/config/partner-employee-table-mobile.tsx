import { format } from "date-fns";
import React from "react";
import {
  Employee,
  getEmployeeProjects,
  getEmployeeProjectCount,
} from "@/types/api/employee.types";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";

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

export interface PartnerEmployeeMobileFieldConfig {
  key: string;
  label: string;
  render?: (value: unknown, row: Employee, index?: number) => React.ReactNode;
}

export interface PartnerEmployeeMobileConfigDeps {
  onRowClick: (employee: Employee) => void;
}

export const createPartnerEmployeeMobileConfig = (
  deps: PartnerEmployeeMobileConfigDeps,
) => {
  const { onRowClick } = deps;

  const mobileFields: PartnerEmployeeMobileFieldConfig[] = [
    {
      key: "stt",
      label: "STT",
      render: (value, row, index) => (
        <span className="typography-body-medium font-medium">
          {(index as number) + 1}
        </span>
      ),
    },
    {
      key: "fullname",
      label: "Họ và tên",
      render: (value) => (
        <span className="typography-body-medium font-medium">{value}</span>
      ),
    },
    {
      key: "email",
      label: "Email",
      render: (value) => (
        <span className="typography-body-medium text-muted-foreground truncate">
          {value || "-"}
        </span>
      ),
    },
    {
      key: "mobile",
      label: "Điện thoại",
      render: (value) => (
        <span className="typography-body-medium">{value || "-"}</span>
      ),
    },
    {
      key: "current_projects",
      label: "Dự án",
      render: (value, row) => {
        const projects = getEmployeeProjects(row);
        const projectCount = getEmployeeProjectCount(row);

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
              <p className="typography-body-medium">{project.name}</p>
              <p className="typography-body-small text-muted-foreground">
                {project.code}
                {project.position &&
                  project.position !== "-" &&
                  ` • ${project.position}`}
              </p>
              <div className="flex items-center gap-2 flex-wrap">
                <p className="typography-body-small text-muted-foreground">
                  {project.start_date &&
                    `BĐ: ${format(new Date(project.start_date), 'dd/MM/yyyy')}`}
                  {project.last_date &&
                    project.start_date &&
                    ` • KT: ${format(new Date(project.last_date), 'dd/MM/yyyy')}`}
                </p>
                {project.payment_schedule && !project.last_date && (
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
            </div>
          );
        }

        const firstProject = projects[0];
        return (
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <div className="flex-1">
                <p className="typography-body-medium">{firstProject.name}</p>
                <p className="typography-body-small text-muted-foreground">
                  {firstProject.code}
                  {firstProject.position &&
                    firstProject.position !== "-" &&
                    ` • ${firstProject.position}`}
                </p>
              </div>
              <Badge variant="secondary" className="text-xs">
                +{projectCount - 1} dự án
              </Badge>
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              <p className="typography-body-small text-muted-foreground">
                {firstProject.start_date &&
                  `BĐ: ${format(new Date(firstProject.start_date), 'dd/MM/yyyy')}`}
                {firstProject.last_date &&
                  firstProject.start_date &&
                  ` • KT: ${format(new Date(firstProject.last_date), 'dd/MM/yyyy')}`}
              </p>
              {firstProject.payment_schedule && !firstProject.last_date && (
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
          </div>
        );
      },
    },
    {
      key: "position",
      label: "Vị trí",
      render: (value, row) => {
        const projects = getEmployeeProjects(row);
        const positions = projects.map((p) => p.position).filter(Boolean);
        const uniquePositions = [...new Set(positions)];

        return (
          <span className="typography-body-medium">
            {uniquePositions.length === 0
              ? "-"
              : uniquePositions.length === 1
                ? uniquePositions[0]
                : `${uniquePositions[0]} +${uniquePositions.length - 1}`}
          </span>
        );
      },
    },
    {
      key: "status",
      label: "Trạng thái",
      render: (value) => {
        const statusLabels = {
          active: "Đang dùng",
          inactive: "Ngừng hoạt động",
          pending: "Chờ xử lý",
        };

        const statusColors = {
          active:
            "bg-green-100 text-green-800",
          inactive: "bg-red-100 text-red-800",
          pending:
            "bg-yellow-100 text-yellow-800",
        };

        return (
          <span
            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${statusColors[value as keyof typeof statusColors] || "bg-gray-100 text-gray-800"}`}
          >
            {statusLabels[value as keyof typeof statusLabels] || value}
          </span>
        );
      },
    },
  ];

  const rowTitle = (row: Employee) => (
    <div className="flex items-center gap-3">
      <UserAvatar name={row.fullname} username={row.username} cccd={row.cccd} size="md" />
      <div>
        <p className="typography-body-medium font-medium">{row.fullname}</p>
        <p className="typography-caption text-muted-foreground">
          CCCD: {row.cccd || "-"}
        </p>
      </div>
    </div>
  );

  const rowSubtitle = (row: Employee) => {
    const projects = getEmployeeProjects(row);
    const projectCount = getEmployeeProjectCount(row);

    let projectInfo: string;
    if (projectCount === 0) {
      projectInfo = "Chưa có dự án";
    } else if (projectCount === 1) {
      const project = projects[0];
      const scheduleInfo =
        project.payment_schedule && !project.last_date
          ? ` • ${getPaymentScheduleBadgeConfig(project.payment_schedule).label}`
          : "";
      projectInfo = `${project.code} • ${project.name}${project.start_date ? ` • BĐ: ${format(new Date(project.start_date), 'dd/MM/yyyy')}` : ""}${scheduleInfo}`;
    } else {
      const project = projects[0];
      const scheduleInfo =
        project.payment_schedule && !project.last_date
          ? ` • ${getPaymentScheduleBadgeConfig(project.payment_schedule).label}`
          : "";
      projectInfo = `${project.code} • ${project.name} (+${projectCount - 1})${project.start_date ? ` • BĐ: ${format(new Date(project.start_date), 'dd/MM/yyyy')}` : ""}${scheduleInfo}`;
    }

    return `${row.email || "Không có email"} • ${projectInfo}`;
  };

  return {
    mobileFields,
    rowTitle,
    rowSubtitle,
  };
};

export default createPartnerEmployeeMobileConfig;
