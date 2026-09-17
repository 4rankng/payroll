import { format } from "date-fns";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { MobileField } from "@/components/ui/responsive-table";
import {
  Employee,
  getEmployeeProjects,
  getEmployeeProjectCount,
  hasCompleteBankDetails,
  VIETNAMESE_EMPLOYEE_LABELS,
} from "@/types/api/employee.types";
import {
  Building,
  Calendar,
  IdCard,
  Phone,
  MapPin,
  CreditCard,
  Check,
  X,
} from "lucide-react";
import React from "react";

const getPaymentScheduleBadgeClasses = (
  schedule: "weekly" | "monthly" | "flexible",
) => {
  switch (schedule) {
    case "weekly":
      return "border-blue-300 text-blue-700 bg-blue-50";
    case "monthly":
      return "border-amber-300 text-amber-700 bg-amber-50";
    case "flexible":
      return "border-emerald-300 text-emerald-700 bg-emerald-50";
  }
};

export interface EmployeeMobileConfigDeps {
  formatCurrency: (amount: number) => string;
  onRowClick: (employee: Employee) => void;
}

export interface EmployeeMobileConfig {
  mobileFields: MobileField<Employee>[];
  rowTitle: (row: Employee) => React.ReactNode;
  rowSubtitle: (row: Employee) => React.ReactNode;
}

export const createEmployeeMobileConfig = (
  deps: EmployeeMobileConfigDeps,
): EmployeeMobileConfig => {
  const { formatCurrency, onRowClick } = deps;

  return {
    mobileFields: [
      {
        key: "mobile",
        label: "Điện thoại",
        priority: 2,
        render: (row) => (
          <div className="typography-body-medium text-gray-600">
            {row.mobile || "-"}
          </div>
        ),
      },
      {
        key: "date_of_birth",
        label: "Ngày sinh",
        priority: 2,
        render: (row) => (
          <div className="typography-body-medium text-gray-600">
            {row.date_of_birth
              ? format(new Date(row.date_of_birth), 'dd/MM/yyyy')
              : "-"}
          </div>
        ),
      },
      {
        key: "address",
        label: "Địa chỉ",
        priority: 2,
        render: (row) => (
          <div className="typography-body-medium text-gray-600 line-clamp-2">
            {row.address || "-"}
          </div>
        ),
      },
      {
        key: "bank",
        label: "Ngân hàng",
        priority: 3,
        render: (row) => (
          <div className="typography-body-medium text-gray-600">
            {row.bank?.branch_name || "-"}
          </div>
        ),
      },
    ],
    rowTitle: (row) => {
      const hasBankDetails = hasCompleteBankDetails(row);
      return (
        <div className="flex items-center gap-3">
          <UserAvatar size="md" name={row.fullname} username={row.username} cccd={row.cccd} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              <p className="typography-body-large truncate">{row.fullname}</p>
            </div>
            <p className="typography-body-medium text-muted-foreground font-mono truncate">
              {row.cccd}
            </p>
          </div>
        </div>
      );
    },
    rowSubtitle: (row) => {
      const projects = getEmployeeProjects(row);
      const projectCount = getEmployeeProjectCount(row);

      if (projectCount === 0) {
        return (
          <div className="typography-body-medium text-muted-foreground">
            Chưa phân công dự án
          </div>
        );
      }

      if (projectCount === 1) {
        return (
          <div>
            <div className="typography-title-small text-muted-foreground">
              {projects[0].name}
            </div>
            {projects[0].payment_schedule && (
              <Badge
                variant="outline"
                className={`text-xs px-1.5 py-0 h-4 shrink-0 mt-1 rounded-[4px] ${getPaymentScheduleBadgeClasses(projects[0].payment_schedule)}`}
              >
                {
                  VIETNAMESE_EMPLOYEE_LABELS.schedule[
                    projects[0].payment_schedule
                  ]
                }
              </Badge>
            )}
          </div>
        );
      }

      return (
        <div>
          <div className="typography-title-small text-muted-foreground">
            {projects[0].name} (+{projectCount - 1} dự án khác)
          </div>
          {projects[0].payment_schedule && (
            <Badge
              variant="outline"
              className={`text-xs px-1.5 py-0 h-4 shrink-0 mt-1 rounded-[2px] ${getPaymentScheduleBadgeClasses(projects[0].payment_schedule)}`}
            >
              {
                VIETNAMESE_EMPLOYEE_LABELS.schedule[
                  projects[0].payment_schedule
                ]
              }
            </Badge>
          )}
        </div>
      );
    },
  };
};

export default createEmployeeMobileConfig;
