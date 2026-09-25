import { Project } from "@/types/api/project.types";
import { formatEmployeeCount } from "@/utils/projectHelpers";
import { MobileField } from "@/components/ui/responsive-table";
import { ProjectStatusBadge } from "@/components/projects/ProjectStatusBadge";

interface CreateProjectMobileConfigProps {}

export const createProjectMobileConfig = (
  _props: CreateProjectMobileConfigProps = {}
) => {
  const mobileFields: MobileField<Project>[] = [
    {
      key: 'status',
      label: 'Trạng thái',
      priority: 1,
      render: (row) => <ProjectStatusBadge status={row.status} />,
    },
    {
      key: 'client_name',
      label: 'Khách hàng',
      priority: 1,
      render: (row) => (
        <span className="typography-body-medium text-foreground/80">
          {row.client_name || "—"}
        </span>
      ),
    },
    {
      key: 'employee_summary',
      label: 'Nhân viên',
      priority: 2,
      render: (row) => {
        const weekly = row.weekly_salary_employee_count || 0;
        const monthly = row.monthly_salary_employee_count || 0;
        const total = weekly + monthly;

        if (total === 0) return <span className="text-muted-foreground">—</span>;

        return (
          <div className="flex flex-col gap-0.5">
            {weekly > 0 && (
              <div className="flex items-center gap-1.5">
                <span className="typography-body-medium font-semibold text-foreground tabular-nums">{formatEmployeeCount(weekly)}</span>
                <span className="typography-label-medium text-muted-foreground">Lương tuần</span>
              </div>
            )}
            {monthly > 0 && (
              <div className="flex items-center gap-1.5">
                <span className="typography-body-medium font-semibold text-foreground tabular-nums">{formatEmployeeCount(monthly)}</span>
                <span className="typography-label-medium text-muted-foreground">Lương tháng</span>
              </div>
            )}
          </div>
        );
      },
    },
  ];

  const rowTitle = (row: Project) => (
    <div className="min-w-0 flex-1">
      <p className="typography-body-medium font-semibold text-foreground truncate">
        {row.name}
      </p>
      {row.code && (
        <p className="typography-label-medium text-muted-foreground font-mono truncate">
          {row.code}
        </p>
      )}
    </div>
  );

  const rowSubtitle = () => null;

  return {
    mobileFields,
    rowTitle,
    rowSubtitle,
  };
};
