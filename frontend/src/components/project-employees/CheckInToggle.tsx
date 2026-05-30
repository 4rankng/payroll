import { Switch } from "@/components/ui/switch";
import { ProjectEmployeeAssignment } from "@/types/api/project-employee.types";
import { useToggleCheckInEnabled } from "@/hooks/api/useProjectEmployees";

interface CheckInToggleProps {
  assignment: ProjectEmployeeAssignment;
  disabled?: boolean;
}

export function CheckInToggle({ assignment, disabled }: CheckInToggleProps) {
  const toggleMutation = useToggleCheckInEnabled();

  const handleToggle = (checked: boolean) => {
    toggleMutation.mutate({
      projectId: assignment.project_id,
      employeeId: assignment.employee_id,
      enabled: checked,
    });
  };

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
