import React from "react";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Badge } from "@/components/ui/badge";
import {
  Employee,
  hasCompleteBankDetails,
  getEmployeeProjects,
} from "@/types/api/employee.types";
import { Building2, CreditCard, IdCard } from "lucide-react";
import { cn } from "@/lib/utils";

const getPaymentScheduleBadgeConfig = (
  schedule: "weekly" | "monthly" | "flexible",
) => {
  switch (schedule) {
    case "weekly":
      return { className: "bg-sky-50 text-sky-700 border-border", label: "Tuần" };
    case "monthly":
      return { className: "bg-violet-50 text-violet-700 border-violet-200", label: "Tháng" };
    case "flexible":
      return { className: "bg-emerald-50 text-emerald-700 border-emerald-200", label: "Linh động" };
  }
};

interface EmployeeMobileCardProps {
  employee: Employee;
  onClick: (employee: Employee) => void;
}

export const EmployeeMobileCard = React.memo(function EmployeeMobileCard({
  employee,
  onClick,
}: EmployeeMobileCardProps) {
  const hasBankDetails = hasCompleteBankDetails(employee);
  const projects = getEmployeeProjects(employee);
  const primaryProject = projects[0];
  const paymentSchedule = primaryProject?.payment_schedule;

  return (
    <div
      onClick={() => onClick(employee)}
      className="bg-card/80 border border-border rounded-xl px-3.5 py-3 shadow-sm card-lift transition-all cursor-pointer touch-manipulation"
      style={{ backdropFilter: 'blur(8px)', boxShadow: '0 1px 4px rgba(2,132,199,0.08)' }}
      role="button"
      aria-label={`Xem chi tiết nhân viên ${employee.fullname}`}
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick(employee);
        }
      }}
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
            <span className="text-sm font-semibold truncate leading-tight">
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
          <div className="flex items-center gap-1.5 mt-0.5 min-w-0">
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
                <span className="text-xs font-mono text-gray-400 shrink-0">
                  {employee.cccd}
                </span>
              </>
            )}
            {paymentSchedule && (
              <Badge
                variant="outline"
                className={cn(
                  "text-[11px] h-4 px-1.5 shrink-0 border ml-auto",
                  getPaymentScheduleBadgeConfig(paymentSchedule).className,
                )}
              >
                {getPaymentScheduleBadgeConfig(paymentSchedule).label}
              </Badge>
            )}
          </div>
        </div>
      </div>
    </div>
  );
});
