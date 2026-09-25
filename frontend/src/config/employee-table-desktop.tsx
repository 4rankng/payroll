import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
  Employee,
  getEmployeeProjects,
  getEmployeeProjectCount,
  hasCompleteBankDetails,
  VIETNAMESE_EMPLOYEE_LABELS,
} from "@/types/api/employee.types";
import { UserAvatar } from "@/components/ui/user-avatar";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { MapPin, Phone, Landmark, AlertTriangle, CreditCard } from "lucide-react";
import { formatVietnameseDate } from "@/utils/vietnamese";
import { cn } from "@/lib/utils";

const SCHEDULE_STYLES: Record<string, string> = {
  weekly: "bg-sky-50 text-sky-700 border-sky-200/80",
  monthly: "bg-teal-50 text-teal-700 border-teal-200/80",
  flexible: "bg-emerald-50 text-emerald-700 border-emerald-200/80",
};

export interface EmployeeColumnsDeps {
  onRowClick: (employee: Employee) => void;
}

export const createEmployeeColumns = (
  deps: EmployeeColumnsDeps,
): ColumnDef<Employee>[] => {
  const { onRowClick } = deps;
  return [
    {
      id: "employee_info",
      accessorKey: "fullname",
      header: "Nhân viên",
      size: 0,
      minSize: 0,
      cell: ({ row }) => {
        const employee = row.original;
        const bankOk = hasCompleteBankDetails(employee);
        return (
          <div
            className="flex items-center gap-3 cursor-pointer -mx-2 px-2 py-1.5 rounded-lg transition-colors hover:bg-accent/30 min-w-0 group"
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
            <div className="min-w-0 flex-1 space-y-0.5">
              <div className="flex items-center gap-2">
                <p className="typography-body-medium font-semibold text-foreground truncate group-hover:text-primary transition-colors">
                  {employee.fullname}
                </p>
                {!bankOk && (
                  <TooltipProvider delayDuration={200}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <AlertTriangle className="h-3 w-3 text-amber-700 shrink-0" />
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs">
                        Thiếu thông tin ngân hàng
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </div>
              <div className="flex items-center gap-3 text-muted-foreground">
                <div className="flex items-center gap-1">
                  <CreditCard className="h-3 w-3 shrink-0" />
                  <span className="typography-label-medium font-mono">{employee.cccd}</span>
                </div>
                {employee.mobile && (
                  <div className="flex items-center gap-1">
                    <Phone className="h-3 w-3 shrink-0" />
                    <span className="typography-label-medium tabular-nums">{employee.mobile}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      },
    },
    {
      id: "projects",
      accessorKey: "project_name",
      header: "Dự án",
      size: 0,
      minSize: 0,
      cell: ({ row }) => {
        const employee = row.original;
        const projects = getEmployeeProjects(employee);
        const projectCount = getEmployeeProjectCount(employee);

        if (projectCount === 0) {
          return (
            <span className="inline-flex items-center gap-1 text-xs text-muted-foreground px-2 py-1 rounded-md bg-muted/30">
              Chưa phân công
            </span>
          );
        }

        const first = projects[0];
        const more = projectCount - 1;

        return (
          <div className="space-y-1 min-w-0">
            <div className="flex items-center gap-2 min-w-0">
              <span className="typography-body-medium text-foreground truncate">
                {first.name}
              </span>
              {first.payment_schedule && (
                <span className={cn(
                  "inline-flex items-center px-1.5 py-px rounded text-xs font-semibold border shrink-0",
                  SCHEDULE_STYLES[first.payment_schedule],
                )}>
                  {VIETNAMESE_EMPLOYEE_LABELS.schedule[first.payment_schedule]}
                </span>
              )}
            </div>
            <div className="flex items-center gap-1.5">
              <span className="typography-label-medium text-muted-foreground font-mono truncate">
                {first.code}
              </span>
              {more > 0 && (
                <TooltipProvider delayDuration={200}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex items-center justify-center h-4 min-w-[1.25rem] px-1 rounded text-xs font-bold bg-primary/5 text-primary/60 border border-primary/10 shrink-0">
                        +{more}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" className="text-xs max-w-[200px]">
                      {projects.slice(1).map(p => p.name).join(', ')}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </div>
          </div>
        );
      },
    },
    {
      id: "bank",
      accessorKey: "bank_name",
      header: "Ngân hàng",
      size: 0,
      cell: ({ row }) => {
        const employee = row.original;
        const branch = employee.bank?.branch_name;
        if (!branch) {
          return <span className="text-muted-foreground/40">—</span>;
        }
        return (
          <div className="flex items-center gap-2 min-w-0 max-w-[180px]">
            <div className="flex h-6 w-6 items-center justify-center rounded bg-primary/5 border border-primary/10 shrink-0">
              <Landmark className="h-3 w-3 text-primary/50" />
            </div>
            <span className="typography-body-medium text-foreground/80 truncate" title={branch}>
              {branch}
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: "address",
      header: "Địa chỉ",
      cell: ({ row }) => {
        const raw = (row.getValue("address") as string) || "";
        const cleaned = raw.replace(/\\'/g, "'");
        if (!cleaned) return <span className="text-muted-foreground/40">—</span>;
        return (
          <div className="flex items-center gap-1.5 min-w-0" title={cleaned}>
            <MapPin className="h-3 w-3 shrink-0 text-muted-foreground/40" />
            <span className="typography-body-medium text-muted-foreground truncate">
              {cleaned}
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: "created_at",
      header: "Ngày tạo",
      size: 110,
      cell: ({ row }) => {
        const date = row.getValue("created_at") as string;
        return (
          <span className="typography-body-medium text-muted-foreground tabular-nums">
            {date ? formatVietnameseDate(date) : "—"}
          </span>
        );
      },
    },
  ];
};

export default createEmployeeColumns;
