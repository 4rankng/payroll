import { CheckCircle2, Clock3, Loader2, Power, PowerOff } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { CheckInConfigurationEmployee } from "@/types/api/project-employee.types";
import { formatLastCheckIn, getCheckInEmployeeState } from "@/utils/checkInSettingsHelpers";

const employeeStatePresentation = {
  pending: { label: "Chờ kích hoạt", icon: Clock3, badge: "warning" as const },
  disabled: { label: "Đang tắt", icon: PowerOff, badge: "secondary" as const },
  used: { label: "Đã điểm danh", icon: CheckCircle2, badge: "success" as const },
  unused: { label: "Chưa điểm danh", icon: Power, badge: "outline" as const },
};

interface CheckInEmployeeCardProps {
  employee: CheckInConfigurationEmployee;
  disabled: boolean;
  pending: boolean;
  onToggle: (employee: CheckInConfigurationEmployee) => void;
}

export function CheckInEmployeeCard({
  employee,
  disabled,
  pending,
  onToggle,
}: CheckInEmployeeCardProps) {
  const state = getCheckInEmployeeState(employee);
  const presentation = employeeStatePresentation[state];
  const StatusIcon = presentation.icon;
  const isEnabledOrPending = employee.check_in_enabled || employee.pending_check_in_enable;
  const actionLabel = employee.pending_check_in_enable
    ? "Hủy chờ"
    : employee.check_in_enabled
      ? "Tắt"
      : "Bật";

  return (
    <Card
      role="listitem"
      className="flex min-h-[8.5rem] flex-col overflow-hidden shadow-none transition-colors hover:border-primary/35"
    >
      <div className="min-w-0 space-y-2 border-b border-border/70 p-3 pb-2.5">
        <div className="min-w-0">
          <h2 className="break-words text-sm font-semibold leading-5 text-foreground">
            {employee.employee_name}
          </h2>
          <p className="mt-0.5 break-words text-xs tabular-nums text-muted-foreground">
            {employee.employee_cccd}
            {employee.employee_code ? ` · ${employee.employee_code}` : ""}
          </p>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Badge variant={presentation.badge} className="shrink-0 gap-1 whitespace-nowrap">
            <StatusIcon className="h-3.5 w-3.5" aria-hidden />
            {presentation.label}
          </Badge>
          <Button
            type="button"
            variant={isEnabledOrPending ? "outline" : "default"}
            size="sm"
            className="h-11 min-h-11 min-w-11 shrink-0 px-3 sm:h-8 sm:min-h-0 sm:px-2.5"
            disabled={disabled}
            onClick={() => onToggle(employee)}
            aria-label={`${actionLabel} điểm danh cho ${employee.employee_name}`}
          >
            {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
            {actionLabel}
          </Button>
        </div>
      </div>

      <div className="grid flex-1 grid-cols-[minmax(0,0.72fr)_minmax(0,1.28fr)] gap-3 p-3 py-2.5">
        <div>
          <p className="text-[11px] font-medium text-muted-foreground">Lượt điểm danh</p>
          <p className="mt-1 text-base font-semibold tabular-nums text-foreground">
            {employee.attendance_count}
          </p>
        </div>
        <div className="min-w-0 border-l border-border/70 pl-3">
          <p className="text-[11px] font-medium text-muted-foreground">Lần gần nhất</p>
          <p className="mt-1 truncate text-xs tabular-nums text-foreground" title={formatLastCheckIn(employee.last_check_in_at)}>
            {formatLastCheckIn(employee.last_check_in_at)}
          </p>
        </div>
      </div>

    </Card>
  );
}
