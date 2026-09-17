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

/**
 * Payment schedule is already stated by the label, so all three variants share
 * one quiet neutral treatment. Three unrelated hues (sky/teal/emerald) made the
 * list look arbitrary and competed with the bank-status and action colours.
 */
const getPaymentScheduleBadgeConfig = (
  schedule: "weekly" | "monthly" | "flexible",
) => {
  switch (schedule) {
    case "weekly":
      return { className: "border-border bg-muted/60 text-muted-foreground", label: "Tuần" };
    case "monthly":
      return { className: "border-border bg-muted/60 text-muted-foreground", label: "Tháng" };
    case "flexible":
      return { className: "border-border bg-muted/60 text-muted-foreground", label: "Linh động" };
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

  const scheduleBadge = paymentSchedule
    ? getPaymentScheduleBadgeConfig(paymentSchedule)
    : null;

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card transition-colors">
      {/* The whole card body is the tap target. Padding lives on the button so
          there is no min-height dead space below the two content rows. */}
      <button
        type="button"
        onClick={() => onClick(employee)}
        className="flex w-full items-center gap-3 p-3 text-left touch-manipulation active:bg-muted/50"
        aria-label={`Xem chi tiết nhân viên ${employee.fullname}`}
      >
        <UserAvatar
          size="md"
          name={employee.fullname}
          username={employee.username}
          cccd={employee.cccd}
          className="shrink-0"
        />

        <div className="min-w-0 flex-1">
          {/* Row 1: name, with the bank-status glyph inline so it reads as part
              of the identity rather than floating in the far corner. */}
          <div className="flex min-w-0 items-center gap-1.5">
            <span className="min-w-0 truncate text-sm font-semibold leading-tight">
              {employee.fullname}
            </span>
            <span
              className={cn(
                "flex h-4 w-4 shrink-0 items-center justify-center rounded-full",
                hasBankDetails ? "bg-emerald-50" : "bg-amber-50",
              )}
              title={
                hasBankDetails
                  ? "Đã có thông tin ngân hàng"
                  : "Chưa có thông tin ngân hàng"
              }
            >
              <CreditCard
                aria-hidden="true"
                className={cn(
                  "h-2.5 w-2.5",
                  hasBankDetails ? "text-emerald-600" : "text-amber-600",
                )}
              />
              <span className="sr-only">
                {hasBankDetails
                  ? "Đã có thông tin ngân hàng"
                  : "Chưa có thông tin ngân hàng"}
              </span>
            </span>
          </div>

          {/* Row 2: project · CCCD — one truncating line. */}
          <div className="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
            <Building2 aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
            {primaryProject ? (
              <span className="truncate">
                {primaryProject.name}
                {projects.length > 1 && (
                  <span className="font-medium text-primary">
                    {" "}
                    +{projects.length - 1}
                  </span>
                )}
              </span>
            ) : (
              <span className="truncate italic">Chưa phân công</span>
            )}
            {employee.cccd && (
              <>
                <span aria-hidden="true" className="shrink-0 text-border">
                  ·
                </span>
                <IdCard aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
                <span className="shrink-0 font-mono tabular-nums">
                  {employee.cccd}
                </span>
              </>
            )}
          </div>
        </div>

        {/* Single trailing element keeps a clean right edge. */}
        {scheduleBadge && (
          <Badge
            variant="outline"
            className={cn(
              "h-5 shrink-0 px-1.5 text-xs font-medium",
              scheduleBadge.className,
            )}
          >
            {scheduleBadge.label}
          </Badge>
        )}
      </button>

      {onPendingTimesheets && pendingTimesheets > 0 && (
        <button
          type="button"
          className="flex min-h-11 w-full items-center justify-center gap-2 border-t border-warning/30 bg-warning/10 px-3 text-xs font-semibold text-warning"
          onClick={() => onPendingTimesheets(employee)}
        >
          <Clock aria-hidden="true" className="h-4 w-4" />
          {pendingTimesheets} bảng công chờ duyệt
        </button>
      )}
    </div>
  );
});
