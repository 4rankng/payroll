import React from "react";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Badge } from "@/components/ui/badge";
import {
  Employee,
  hasCompleteBankDetails,
  getEmployeeProjects,
} from "@/types/api/employee.types";
import { Building2, Clock, CreditCard, IdCard } from "lucide-react";
import { cn } from "@/lib/utils";

const getPaymentScheduleBadgeConfig = (
  schedule: "weekly" | "monthly" | "flexible",
) => {
  switch (schedule) {
    case "weekly":
      return { className: "bg-sky-50 text-sky-700 border-border", label: "Tuần" };
    case "monthly":
      return { className: "bg-teal-50 text-teal-700 border-teal-200", label: "Tháng" };
    case "flexible":
      return { className: "bg-emerald-50 text-emerald-700 border-emerald-200", label: "Linh động" };
  }
};

interface EmployeeMobileCardProps {
  employee: Employee;
  onClick: (employee: Employee) => void;
  onPendingTimesheets?: (employee: Employee) => void;
}

export const EmployeeMobileCard = React.memo(function EmployeeMobileCard({
  employee,
  onClick,
  onPendingTimesheets,
}: EmployeeMobileCardProps) {
  const hasBankDetails = hasCompleteBankDetails(employee);
  const projects = getEmployeeProjects(employee);
  const primaryProject = projects[0];
  const paymentSchedule = primaryProject?.payment_schedule;
  const pendingTimesheets = employee.timesheet_summary?.pending_timesheets ?? 0;

  return (
    <div
      className="rounded-xl border border-border bg-card p-3.5 transition-colors"
    >
      <button
        type="button"
        onClick={() => onClick(employee)}
        className="min-h-11 w-full text-left touch-manipulation"
        aria-label={`Xem chi tiết nhân viên ${employee.fullname}`}
      >
        <div className="flex items-center gap-3">
          <UserAvatar
            size="sm"
            name={employee.fullname}
            username={employee.username}
            cccd={employee.cccd}
            className="shrink-0"
          />

          <div className="flex-1 min-w-0">
            {/* Row 1: name + bank indicator */}
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 break-words text-sm font-semibold leading-tight line-clamp-2">
                {employee.fullname}
              </span>
              <div
                className={cn(
                  "h-5 w-5 rounded-full flex items-center justify-center shrink-0",
                  hasBankDetails ? "bg-emerald-50" : "bg-muted",
                )}
                title={hasBankDetails ? "Đã có thông tin ngân hàng" : "Chưa có thông tin ngân hàng"}
              >
                <CreditCard
                  className={cn("h-3 w-3", hasBankDetails ? "text-emerald-600" : "text-gray-400")}
                />
              </div>
            </div>

            {/* Row 2: project · CCCD · badge */}
            <div className="flex min-w-0 flex-wrap items-center gap-1.5 mt-0.5">
              <Building2 className="h-3 w-3 text-muted-foreground shrink-0" />
              {primaryProject ? (
                <span className="text-xs text-muted-foreground truncate">
                  {primaryProject.name}
                  {projects.length > 1 && (
                    <span className="text-primary font-medium"> +{projects.length - 1}</span>
                  )}
                </span>
              ) : (
                <span className="text-xs text-muted-foreground italic">Chưa phân công</span>
              )}
              {employee.cccd && (
                <>
                  <span className="text-gray-300 shrink-0">·</span>
                  <IdCard className="h-3 w-3 text-muted-foreground shrink-0" />
                  <span className="text-xs font-mono text-gray-600 shrink-0">
                    {employee.cccd}
                  </span>
                </>
              )}
              {paymentSchedule && (
                <Badge
                  variant="outline"
                  className={cn(
                    "h-5 px-1.5 text-xs shrink-0 border ml-auto",
                    getPaymentScheduleBadgeConfig(paymentSchedule).className,
                  )}
                >
                  {getPaymentScheduleBadgeConfig(paymentSchedule).label}
                </Badge>
              )}
            </div>
          </div>
        </div>
      </button>
      {onPendingTimesheets && pendingTimesheets > 0 && (
        <button
          type="button"
          className="mt-3 inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border border-warning/30 bg-warning/10 px-3 text-xs font-semibold text-warning"
          onClick={() => onPendingTimesheets(employee)}
        >
          <Clock className="h-4 w-4" />
          {pendingTimesheets} bảng công chờ duyệt
        </button>
      )}
    </div>
  );
});
