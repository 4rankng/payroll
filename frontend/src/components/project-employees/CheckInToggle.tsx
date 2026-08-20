import { format } from "date-fns";
import { Switch } from "@/components/ui/switch";
import { ProjectEmployeeAssignment } from "@/types/api/project-employee.types";
import {
  useCancelPendingCheckInEnable,
  useToggleCheckInEnabled,
} from "@/hooks/api/useProjectEmployees";

interface CheckInToggleProps {
  assignment: ProjectEmployeeAssignment;
  disabled?: boolean;
}

function formatEffectiveDate(value?: string | null): string {
  if (!value) return "";
  try {
    return format(new Date(value), "dd/MM");
  } catch {
    return "";
  }
}

export function CheckInToggle({ assignment, disabled }: CheckInToggleProps) {
  const toggleMutation = useToggleCheckInEnabled();
  const cancelMutation = useCancelPendingCheckInEnable();
  const isPending = !assignment.check_in_enabled && !!assignment.pending_check_in_enabled;

  const handleToggle = (checked: boolean) => {
    toggleMutation.mutate({
      projectId: assignment.project_id,
      employeeId: assignment.employee_id,
      enabled: checked,
    });
  };

  const handleCancelPending = () => {
    cancelMutation.mutate({
      projectId: assignment.project_id,
      employeeId: assignment.employee_id,
    });
  };

  if (isPending) {
    const effectiveLabel = formatEffectiveDate(assignment.check_in_effective_from);
    return (
      <div className="flex items-center gap-2">
        <span className="whitespace-nowrap rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 border border-amber-200">
          Kích hoạt {effectiveLabel || "01 tháng sau"}
        </span>
        <button
          type="button"
          className="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground disabled:opacity-50"
          onClick={handleCancelPending}
          disabled={disabled || cancelMutation.isPending}
        >
          Hủy
        </button>
      </div>
    );
  }

  return (
    <div className="flex items-center space-x-2">
      <Switch
        checked={!!assignment.check_in_enabled}
        onCheckedChange={handleToggle}
        disabled={disabled || toggleMutation.isPending}
      />
      <span className="text-xs text-muted-foreground whitespace-nowrap">
        {assignment.check_in_enabled ? "Bật" : "Tắt"}
      </span>
    </div>
  );
}
