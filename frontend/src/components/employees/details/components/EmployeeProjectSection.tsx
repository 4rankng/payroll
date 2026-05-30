import { format } from "date-fns";
import { useMemo, useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Building, Link, Unlink, Plus, Calendar, Clock, AlertCircle, X, Pencil, Check } from "lucide-react";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { getEmployeeProjects, getEmployeeProjectCount } from "@/types/api/employee.types";
import { useCurrentPayRate } from "@/hooks/api/usePayRates";
import { extractPositionsFromPayrates } from "@/types/api/payrate.types";
import { useCancelScheduleChange } from "@/hooks/api/useProjectEmployees";
import { VIETNAMESE_ASSIGNMENT_LABELS } from "@/types/api/project-employee.types";
import { CheckInToggle } from "@/components/project-employees/CheckInToggle";
import { cn } from "@/lib/utils";
import { dateToString } from "@/utils/dateHelpers";
import type { EmployeeDetailsProps } from "../types";
import type { CurrentProject } from "@/types/api/employee.types";

type ProjectChange = {
  position?: string;
  start_date?: string;
  payment_schedule?: 'weekly' | 'monthly';
  assignmentId?: number;
};

interface EmployeeProjectSectionProps extends EmployeeDetailsProps {
  onRemove?: (projectId: number) => void;
  isRemoving?: boolean;
  isEditing?: boolean;
  projectChanges?: Record<number, ProjectChange>;
  onProjectChange?: (projectId: number, changes: ProjectChange | null) => void;
  onProjectApply?: (projectId: number, changes: ProjectChange) => Promise<void>;
  allowProjectLinking?: boolean; // New prop to control project link/unlink in read-only mode
}

interface ProjectCardProps {
  project: CurrentProject;
  employeeId?: number;
  onRemove?: (projectId: number) => void;
  isRemoving?: boolean;
  onChange?: (projectId: number, changes: ProjectChange | null) => void;
  isProjectEditing?: boolean;
  changes?: ProjectChange;
  allowProjectLinking?: boolean; // Allow unlink in read-only mode
  assignmentId?: number;
  onEditStateChange?: (projectId: number, editing: boolean) => void;
  onApplyChanges?: (projectId: number, changes: ProjectChange) => Promise<void> | void;
}

interface ProjectPaymentScheduleControlProps {
  assignmentId?: number;
  currentSchedule?: 'weekly' | 'monthly';
  pendingSchedule?: 'weekly' | 'monthly' | null;
  effectiveFrom?: string | null;
  draftSchedule?: 'weekly' | 'monthly';
  isEditing?: boolean;
  onScheduleChange?: (schedule: 'weekly' | 'monthly') => void;
}

function ProjectPaymentScheduleControl({
  assignmentId,
  currentSchedule = 'weekly',
  pendingSchedule,
  effectiveFrom,
  draftSchedule,
  isEditing = false,
  onScheduleChange
}: ProjectPaymentScheduleControlProps) {
  const baselineSchedule = pendingSchedule ?? currentSchedule;
  const selectedSchedule = draftSchedule ?? baselineSchedule;
  const cancelScheduleMutation = useCancelScheduleChange();
  const scheduleOptions = useMemo(
    () => [
      {
        value: 'weekly' as const,
        label: VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.weekly,
        icon: Clock
      },
      {
        value: 'monthly' as const,
        label: VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.monthly,
        icon: Calendar
      }
    ],
    []
  );

  const hasPendingChange = Boolean(pendingSchedule);
  const isSelectionDisabled = !assignmentId || !isEditing;

  const handleScheduleChange = useCallback(
    (value: string) => {
      if (!value || !isEditing) {
        return;
      }
      onScheduleChange?.(value as 'weekly' | 'monthly');
    },
    [isEditing, onScheduleChange]
  );

  const handleCancelPending = useCallback(async () => {
    if (!assignmentId) {
      return;
    }

    try {
      await cancelScheduleMutation.mutateAsync({
        assignmentId
      });
    } catch (error) {
      // Notification already handled in hook
    }
  }, [assignmentId, cancelScheduleMutation]);

  return (
    <div className="space-y-2">
      <ToggleGroup
        type="single"
        value={selectedSchedule}
        onValueChange={handleScheduleChange}
        className="grid grid-cols-2 gap-2"
        aria-label="Chọn chu kỳ trả lương"
      >
        {scheduleOptions.map(({ value, label, icon: Icon }) => (
          <ToggleGroupItem
            key={value}
            value={value}
            aria-label={label}
            className={cn(
              "justify-start gap-2 typography-body-small",
              "data-[state=on]:bg-primary/10 data-[state=on]:text-primary data-[state=on]:border-primary data-[state=on]:border-2 data-[state=on]:font-medium",
              "data-[state=off]:bg-muted/30 data-[state=off]:text-muted-foreground data-[state=off]:border data-[state=off]:border-border",
              isSelectionDisabled && "pointer-events-none text-foreground" // Use dark color instead of gray
            )}
            disabled={isSelectionDisabled}
          >
            <Icon className="h-4 w-4" />
            {label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>

      <div className="flex flex-wrap items-center gap-2 typography-label-small text-foreground/80">
        {hasPendingChange && (
          <Badge variant="outline" className="text-financial-pending border-amber-400 flex items-center gap-1">
            <AlertCircle className="h-3 w-3" />
            Chờ: {pendingSchedule ? VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule[pendingSchedule] : ''}
          </Badge>
        )}

        {hasPendingChange && effectiveFrom && (
          <span className="typography-label-small text-foreground/80">
            từ {format(new Date(effectiveFrom), 'dd/MM/yyyy')}
          </span>
        )}

        {hasPendingChange && assignmentId && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={handleCancelPending}
            disabled={cancelScheduleMutation.isPending}
          >
            <X className="h-3 w-3" />
            <span className="sr-only">Hủy thay đổi chu kỳ</span>
          </Button>
        )}
      </div>

      {!assignmentId && (
        <p className="typography-label-small text-foreground/80">
          Không thể đổi chu kỳ do thiếu mã phân công.
        </p>
      )}
    </div>
  );
}

function ProjectCard({
  project,
  employeeId,
  onRemove,
  isRemoving,
  onChange,
  isProjectEditing = false,
  changes,
  allowProjectLinking = false,
  assignmentId,
  onEditStateChange,
  onApplyChanges
}: ProjectCardProps) {
  const canEditFields = isProjectEditing;
  const canRemoveProject = isProjectEditing || allowProjectLinking;
  const baselineSchedule = project.pending_payment_schedule ?? project.payment_schedule ?? 'weekly';
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Fetch current payrate for this project to get available positions
  const { data: payrateData } = useCurrentPayRate(
    project.project_id,
    true
  );

  // Extract available positions from payrate data
  const availablePositions = useMemo(() => {
    if (payrateData?.rates) {
      return extractPositionsFromPayrates(payrateData);
    }
    return [];
  }, [payrateData]);

  // Memoize today's date to prevent recreating on every render
  const todayDateString = useMemo(() => dateToString(new Date()), []);

  const updateProjectChanges = (delta: ProjectChange) => {
    const nextChange: ProjectChange = { ...delta };
    const targetAssignmentId = assignmentId ?? project.project_employee_id;
    if (targetAssignmentId !== undefined) {
      nextChange.assignmentId = targetAssignmentId;
    }
    onChange?.(project.project_id, nextChange);
  };

  const handlePositionChange = (value: string) => {
    // Always set date to today when position changes
    // Prepare changes object - always update both position and date
    const updateData: ProjectChange = {
      position: value,
      start_date: todayDateString  // Always set to today when position changes
    };

    updateProjectChanges(updateData);
  };

  const handleDateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    updateProjectChanges({ start_date: e.target.value });
  };

  const handleScheduleDraftChange = (schedule: 'weekly' | 'monthly') => {
    if (schedule === baselineSchedule) {
      updateProjectChanges({ payment_schedule: undefined });
      return;
    }
    updateProjectChanges({ payment_schedule: schedule });
  };

  const hasMeaningfulChanges = (changeSet?: ProjectChange | null) => {
    if (!changeSet) {
      return false;
    }
    return Object.keys(changeSet).some(
      (key) => key !== 'assignmentId' && changeSet[key as keyof ProjectChange] !== undefined
    );
  };

  const handleCancelEditing = () => {
    onChange?.(project.project_id, null);
    onEditStateChange?.(project.project_id, false);
  };

  const handleApplyEditing = async () => {
    if (!isProjectEditing || isSubmitting) {
      return;
    }
    if (!hasMeaningfulChanges(changes)) {
      onEditStateChange?.(project.project_id, false);
      return;
    }
    if (!onApplyChanges) {
      onEditStateChange?.(project.project_id, false);
      return;
    }

    try {
      setIsSubmitting(true);
      await onApplyChanges(project.project_id, changes);
      onChange?.(project.project_id, null);
      onEditStateChange?.(project.project_id, false);
    } catch (error) {
      // Errors handled upstream
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleStartEditing = () => {
    if (isSubmitting) {
      return;
    }
    onEditStateChange?.(project.project_id, true);
  };

  return (
    <div className="border rounded-xl p-3 space-y-2 min-h-[120px] flex flex-col">
      {/* Project Header */}
      <div className="flex items-start gap-2">
        <div className="flex-1 min-w-0">
          <div className="font-medium typography-body-small truncate text-foreground" title={project.name}>
            {project.name}
          </div>
          <div className="typography-label-small text-foreground/80 mt-0.5 flex flex-wrap items-center gap-3">
            <span className="truncate text-foreground/80">{project.code}</span>
            {project.client_name && (
              <span className="truncate text-foreground/80">
                KH: {project.client_name}
              </span>
            )}
          </div>
        </div>
        {onEditStateChange && (
          <div className="flex items-center gap-1">
            {isProjectEditing ? (
              <>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={handleCancelEditing}
                  disabled={isSubmitting}
                  title="Hủy chỉnh sửa"
                >
                  <X className="h-3.5 w-3.5" />
                  <span className="sr-only">Hủy chỉnh sửa</span>
                </Button>
                <Button
                  size="icon"
                  variant="default"
                  className="h-7 w-7"
                  onClick={handleApplyEditing}
                  disabled={isSubmitting}
                  title="Áp dụng thay đổi"
                >
                  <Check className="h-3.5 w-3.5" />
                  <span className="sr-only">Áp dụng thay đổi</span>
                </Button>
              </>
            ) : (
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={handleStartEditing}
                title="Chỉnh sửa phân công"
              >
                <Pencil className="h-3.5 w-3.5" />
                <span className="sr-only">Chỉnh sửa phân công</span>
              </Button>
            )}
          </div>
        )}
      </div>

      {/* Project Details */}
      <div className="typography-body-small text-foreground flex-1 space-y-2">
        {/* Payment schedule controller */}
        <div>
          <label className="block typography-label-medium font-medium mb-1 text-foreground">Chu kỳ trả lương:</label>
          <ProjectPaymentScheduleControl
            assignmentId={assignmentId}
            currentSchedule={project.payment_schedule}
            pendingSchedule={project.pending_payment_schedule}
            effectiveFrom={project.schedule_effective_from}
            draftSchedule={changes?.payment_schedule}
            isEditing={canEditFields && !isSubmitting}
            onScheduleChange={handleScheduleDraftChange}
          />
        </div>

        {/* Check-in toggle for flexible projects */}
        {project.is_flexible && (
          <div className="flex items-center justify-between">
            <label className="typography-label-medium font-medium text-foreground">Điểm danh:</label>
            <CheckInToggle
              assignment={{
                project_id: project.project_id,
                employee_id: employeeId ?? 0,
                check_in_enabled: project.check_in_enabled,
              } as any}
            />
          </div>
        )}

        <div className="grid gap-3 sm:grid-cols-2">
          {/* Position Selector */}
          <div>
            <label className="block typography-label-medium font-medium mb-1 text-foreground">Vị trí:</label>
            {canEditFields && availablePositions.length === 0 ? (
              <Input
                value={(changes?.position ?? project.position) || ""}
                onChange={(e) => handlePositionChange(e.target.value)}
                placeholder="Nhập vị trí công việc"
                disabled={!canEditFields || isSubmitting}
                className={cn(
                  "h-7 typography-body-small",
                  !canEditFields && "opacity-100" // Ensure proper contrast in read-only mode
                )}
              />
            ) : (
              <Select
                value={changes?.position ?? project.position ?? undefined}
                onValueChange={handlePositionChange}
                disabled={!canEditFields || isSubmitting}
              >
                <SelectTrigger className={cn(
                  "h-7 typography-body-small",
                  !canEditFields && "opacity-100" // Ensure proper contrast in read-only mode
                )}>
                  <SelectValue
                    placeholder={
                      availablePositions.length === 0
                        ? "Chưa có bảng lương"
                        : "Chọn vị trí"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {availablePositions.map((position: string) => (
                    <SelectItem key={position} value={position} className="typography-body-small">
                      {position}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            {canEditFields && availablePositions.length === 0 && (
              <p className="typography-body-small text-financial-pending mt-1">
                Dự án chưa có bảng lương. Vui lòng nhập vị trí thủ công.
              </p>
            )}
          </div>

          {/* Start Date Picker with Unlink Button */}
          <div>
            <label className="block typography-label-medium font-medium mb-1 text-foreground">Ngày bắt đầu:</label>
            <div className="flex items-center gap-2">
              <Input
                type="date"
                value={(changes?.start_date ?? project.start_date) || ''}
                onChange={handleDateChange}
                disabled={!canEditFields || isSubmitting}
                className={cn(
                  "h-7 typography-body-small flex-1",
                  !canEditFields && "opacity-100" // Ensure proper contrast in read-only mode
                )}
              />
            {canRemoveProject && (
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-7 w-7 p-0 text-red-600 hover:text-red-700 hover:bg-red-50 flex-shrink-0"
                  onClick={() => onRemove?.(project.project_id)}
                  disabled={isRemoving || isSubmitting}
                  title="Gỡ khỏi dự án"
                >
                  <Unlink className="h-3 w-3" />
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* End Date Display */}
        {project.last_date && (
          <div className="flex items-center gap-1 text-foreground/80">
            <span className="text-foreground/80">Đến:</span>
            <span className="text-foreground/80">
              {format(new Date(project.last_date), 'dd/MM/yyyy')}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

export function EmployeeProjectSection({
  employee,
  onRemove,
  isRemoving,
  isEditing: _isEditing = false,
  projectChanges = {},
  onProjectChange,
  onProjectApply,
  allowProjectLinking = false
}: EmployeeProjectSectionProps) {
  const { openProjectAssignment } = useProjectModals();
  const [editableProjects, setEditableProjects] = useState<Record<number, boolean>>({});

  // Use current_projects from backend response
  const currentProjects = employee ? getEmployeeProjects(employee) : [];
  const projectCount = employee ? getEmployeeProjectCount(employee) : 0;

  useEffect(() => {
    setEditableProjects({});
  }, [employee?.id]);

  const handleAssignClick = () => {
    if (!allowProjectLinking) {
      return;
    }
    if (employee?.id) {
      openProjectAssignment(
        undefined,
        employee.id.toString()
      );
    }
  };

  const setProjectEditingState = (projectId: number, editing: boolean) => {
    setEditableProjects((prev) => {
      const next = { ...prev };
      if (editing) {
        next[projectId] = true;
      } else {
        delete next[projectId];
      }
      return next;
    });
  };

  const isProjectEditable = (projectId: number) => Boolean(editableProjects[projectId]);
  const canAssignProjects = allowProjectLinking;

  return (
    <div className="space-y-4">
      {/* Projects Header */}
      <div className="flex items-center gap-3">
        <Building className="w-4 h-4 text-muted-foreground" />
        <dt className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Dự án hiện tại
        </dt>
        {projectCount > 0 && (
          <Badge variant="secondary">
            {projectCount} dự án
          </Badge>
        )}
      </div>

      {/* Projects Grid */}
      {currentProjects.length === 0 ? (
        <div className="text-center py-4 border-2 border-dashed border-border rounded-xl">
          <div className="flex flex-col items-center gap-2">
            <Building className="w-8 h-8 text-muted-foreground" />
            <p className="typography-body-medium text-muted-foreground">
              Nhân viên chưa được gán dự án nào
            </p>
            {canAssignProjects && (
              <Button
                size="sm"
                variant="outline"
                onClick={handleAssignClick}
              >
                <Link className="h-4 w-4 mr-2" />
                Gán dự án
              </Button>
            )}
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {currentProjects.map((project) => (
            <ProjectCard
              key={project.project_id}
              project={project}
              employeeId={employee?.id}
              onRemove={onRemove}
              isRemoving={isRemoving}
              onChange={onProjectChange}
              isProjectEditing={isProjectEditable(project.project_id)}
              changes={projectChanges[project.project_id]}
              allowProjectLinking={allowProjectLinking}
              assignmentId={project.project_employee_id}
              onEditStateChange={setProjectEditingState}
              onApplyChanges={onProjectApply}
            />
          ))}

          {/* Add Project Card - Show in edit mode or when project linking is allowed */}
          {canAssignProjects && (
            <div
              className="border-2 border-dashed border-muted-foreground/30 rounded-xl p-3 min-h-[120px] flex flex-col items-center justify-center cursor-pointer hover:border-blue-400 hover:bg-blue-50/50 transition-colors group"
              onClick={handleAssignClick}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleAssignClick();
                }
              }}
            >
              <Plus className="h-6 w-6 text-muted-foreground group-hover:text-blue-600 mb-2" />
              <span className="typography-body-small text-foreground/80 group-hover:text-blue-600 text-center">
                Gán thêm dự án
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
