import { useState, useEffect, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { ProjectSelector } from "@/components/ui/project-selector";
import { Calendar, Plus, Clock } from "lucide-react";
import { Employee, getEmployeeProjects } from "@/types/api/employee.types";
import { Project } from "@/types/api/project.types";
import { AssignEmployeeData } from "@/types/api/project.types";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { useCurrentPayRate, isForbiddenError } from "@/hooks/api/usePayRates";
import { extractPositionsFromPayrates } from "@/types/api/payrate.types";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { useQueryClient } from "@tanstack/react-query";
import { VIETNAMESE_ASSIGNMENT_LABELS } from "@/types/api/project-employee.types";
import { cn } from "@/lib/utils";
import type { ModalConfig } from "@/types/modal-config.types";

interface ProjectAssignmentSheetProps {
  employee: Employee | null;
  isOpen: boolean;
  onClose: () => void;
  onAssign: (projectId: number, data: AssignEmployeeData, selectedProject?: Project) => void;
  loading?: boolean;
  selectedProject?: Project;
}

function ProjectAssignmentSheet({
  employee,
  isOpen,
  onClose,
  onAssign,
  loading = false,
  selectedProject: initialSelectedProject
}: ProjectAssignmentSheetProps) {
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState<AssignEmployeeData>({
    employee_id: employee?.id || 0,
    employee_code: "",
    position: "",
    // Empty by default: the backend continues coverage from the employee's
    // last recorded timesheet (else 1st of current month) instead of today.
    start_date: "",
    end_date: "",
    payment_schedule: "weekly",
  });

  const [selectedProject, setSelectedProject] = useState<Project | null>(initialSelectedProject || null);

  // Fetch current payrate for the selected project to get available positions
  const { data: payrateData, error: payrateError } = useCurrentPayRate(
    selectedProject?.id || 0,
    !!selectedProject?.id
  );

  // Modal navigation
  const { openProjectDetails } = useProjectModals();

  // Extract available positions from payrate data
  const availablePositions = useMemo(() => {
    if (payrateData?.rates) {
      return extractPositionsFromPayrates(payrateData);
    }
    return [];
  }, [payrateData]);

  // Check if payrates exist for the selected project
  const hasPayrates = availablePositions.length > 0;

  // 403 from the payrate query means the current user (partner) has no access
  // to the selected project — distinct from "project has no payrate config".
  const payrateForbidden = isForbiddenError(payrateError);

  // Check if project can be assigned employees (not completed or cancelled)
  const canAssignEmployees = useMemo(() => {
    if (!selectedProject) return false;
    return selectedProject.status !== 'completed' && selectedProject.status !== 'cancelled';
  }, [selectedProject]);

  // Get employee's current project IDs to exclude from dropdown
  const excludeProjectIds = useMemo(() => {
    if (!employee) return [];
    const currentProjects = getEmployeeProjects(employee);
    return currentProjects.map(project => project.project_id);
  }, [employee]);

  useEffect(() => {
    if (employee) {
      setFormData(prev => ({
        ...prev,
        employee_id: employee.id,
      }));
    }
  }, [employee]);

  // Update selectedProject when initialSelectedProject changes (deeplink scenario)
  // but check if employee is already assigned to that project
  useEffect(() => {
    if (initialSelectedProject && !selectedProject) {
      // Check if employee is already assigned to this project
      const isAlreadyAssigned = excludeProjectIds.includes(initialSelectedProject.id);
      if (!isAlreadyAssigned) {
        setSelectedProject(initialSelectedProject);
      }
    }
  }, [initialSelectedProject, selectedProject, excludeProjectIds]);

  // Force refetch payrate data when modal opens or project changes
  useEffect(() => {
    if (selectedProject?.id && isOpen) {
      // Invalidate and refetch the current project payrate data to ensure fresh data
      queryClient.invalidateQueries({
        queryKey: ['projects', selectedProject.id, 'payrate'],
        refetchType: 'active'
      });
    }
  }, [selectedProject?.id, isOpen, queryClient]);

  // Reset position when project changes
  useEffect(() => {
    if (selectedProject) {
      setFormData(prev => ({
        ...prev,
        position: hasPayrates ? availablePositions[0] || "" : "",
      }));
    }
  }, [selectedProject, hasPayrates, availablePositions]);

  // Clear selected project if it becomes excluded (employee assigned elsewhere)
  useEffect(() => {
    if (selectedProject && excludeProjectIds.includes(selectedProject.id)) {
      setSelectedProject(null);
    }
  }, [selectedProject, excludeProjectIds]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // Prevent multiple submissions while loading
    if (loading) return;

    // Validate all required fields and conditions
    if (!selectedProject || !employee || !hasPayrates || !formData.position?.trim() || !canAssignEmployees) return;

    // Prepare the assignment data with proper defaults
    const finalAssignmentData: AssignEmployeeData = {
      employee_id: employee.id, // Use employee.id directly to ensure it's always correct
      // If employee_code is empty, let API use employee cccd as default
      employee_code: formData.employee_code?.trim() || undefined,
      // Position must be selected from available positions
      position: formData.position,
      // If start_date is empty, the backend continues coverage from the
      // employee's last recorded timesheet (else 1st of current month)
      start_date: formData.start_date || undefined,
      end_date: formData.end_date || undefined,
      payment_schedule: formData.payment_schedule,
    };

    onAssign(selectedProject.id, finalAssignmentData, selectedProject);
  };

  const handleClose = () => {
    setFormData({
      employee_id: 0,
      employee_code: "",
      position: "",
      start_date: "",
      end_date: "",
      payment_schedule: "weekly",
    });
    setSelectedProject(null);
    onClose();
  };

  if (!employee) return null;

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={handleClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
            <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
              <Calendar className="h-5 w-5 text-primary" />
            </div>
            <div className="space-y-1 flex-1 min-w-0">
              <h2 className="typography-headline-medium text-base sm:text-lg font-semibold">
                Phân công dự án
              </h2>
              <p className="typography-body-medium text-muted-foreground text-sm">
                Phân công {employee.fullname} vào dự án
              </p>
            </div>
          </div>
        )
      }}
      footer={
        <form onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
            <Button type="button" variant="outline" onClick={handleClose} className="min-h-11">
              Đóng
            </Button>
            <Button
              type="submit"
              className="min-h-11"
              variant="default"
              disabled={loading || !selectedProject || !hasPayrates || !formData.position?.trim() || !canAssignEmployees}
            >
              {loading ? "Đang phân công..." : "Phân công"}
            </Button>
          </div>
        </form>
      }
    >
      <div className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="project">Dự án *</Label>
              <ProjectSelector
                value={selectedProject}
                onSelect={setSelectedProject}
                placeholder="Chọn dự án"
                className="w-full"
                excludeProjectIds={excludeProjectIds}
              />
              {excludeProjectIds.length > 0 && (
                <p className="typography-body-small text-muted-foreground">
                  Nhân viên hiện đang được phân công vào {excludeProjectIds.length} dự án. Các dự án này đã được loại bỏ khỏi danh sách.
                </p>
              )}
            </div>

            {selectedProject && !canAssignEmployees && (
              <div className="p-4 bg-amber-50 border border-amber-200 rounded-xl">
                <div className="flex items-center gap-2">
                  <div className="h-5 w-5 rounded-full bg-amber-100 flex items-center justify-center">
                    <span className="text-amber-700 text-xs">!</span>
                  </div>
                  <p className="typography-body-medium text-amber-800 font-medium">
                    Không thể phân công nhân viên
                  </p>
                </div>
                <p className="typography-body-small text-amber-700 mt-1 ml-7">
                  Dự án này đã {selectedProject.status === 'completed' ? 'hoàn thành' : 'bị hủy'}.
                  Chỉ có thể phân công nhân viên vào dự án đang ở trạng thái nháp hoặc đang hoạt động.
                </p>
              </div>
            )}

            {selectedProject && canAssignEmployees && (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="position">Vị trí *</Label>
                  <div className="space-y-3">
                    <div className="flex flex-col gap-2 min-[420px]:flex-row">
                      <Select
                        value={hasPayrates ? formData.position : ""}
                        onValueChange={(value) => setFormData(prev => ({ ...prev, position: value }))}
                        disabled={!hasPayrates}
                      >
                        <SelectTrigger className="h-11 w-full">
                          <SelectValue
                            placeholder={
                              payrateForbidden
                                ? "Bạn không có quyền xem bảng lương của dự án này"
                                : hasPayrates
                                  ? "Chọn vị trí"
                                  : "Dự án chưa có bảng lương"
                            }
                          />
                        </SelectTrigger>
                        <SelectContent>
                          {availablePositions.map((position) => (
                            <SelectItem key={position} value={position}>
                              {position}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {!hasPayrates && (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            if (selectedProject) {
                              openProjectDetails(selectedProject.id.toString(), 'payrates');
                            }
                          }}
                          className="min-h-11 shrink-0"
                        >
                          <Plus className="h-4 w-4 mr-2" />
                          Thêm bảng lương
                        </Button>
                      )}
                    </div>
                    {!hasPayrates && (
                      <p className="typography-body-small text-muted-foreground">
                        Dự án này chưa có bảng lương. Vui lòng tạo bảng lương trước khi phân công.
                      </p>
                    )}
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="employee_code">Mã nhân viên tại công ty</Label>
                  <Input
                    id="employee_code"
                    value={formData.employee_code}
                    onChange={(e) => setFormData(prev => ({ ...prev, employee_code: e.target.value }))}
                    placeholder="Để trống sẽ sử dụng CCCD"
                    className="h-11 w-full"
                  />
                  <p className="typography-body-small text-muted-foreground">Nếu để trống, hệ thống sẽ sử dụng số CCCD của nhân viên</p>
                </div>
              </div>
            )}

            {selectedProject && canAssignEmployees && hasPayrates && (
              <div className="space-y-2">
                <Label>Chu kỳ trả lương</Label>
                <ToggleGroup
                  type="single"
                  value={formData.payment_schedule}
                  onValueChange={(value) => {
                    if (value) {
                      setFormData(prev => ({ ...prev, payment_schedule: value as 'weekly' | 'monthly' }));
                    }
                  }}
                  className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2"
                  aria-label="Chọn chu kỳ trả lương"
                >
                  <ToggleGroupItem
                    value="weekly"
                    aria-label={VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.weekly}
                    className={cn(
                      "min-h-11 justify-start gap-2 typography-body-small",
                      "data-[state=on]:bg-primary/10 data-[state=on]:text-primary data-[state=on]:border-primary data-[state=on]:border-2 data-[state=on]:font-medium",
                      "data-[state=off]:bg-muted/30 data-[state=off]:text-muted-foreground data-[state=off]:border data-[state=off]:border-border"
                    )}
                  >
                    <Clock className="h-4 w-4" />
                    {VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.weekly}
                  </ToggleGroupItem>
                  <ToggleGroupItem
                    value="monthly"
                    aria-label={VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.monthly}
                    className={cn(
                      "min-h-11 justify-start gap-2 typography-body-small",
                      "data-[state=on]:bg-primary/10 data-[state=on]:text-primary data-[state=on]:border-primary data-[state=on]:border-2 data-[state=on]:font-medium",
                      "data-[state=off]:bg-muted/30 data-[state=off]:text-muted-foreground data-[state=off]:border data-[state=off]:border-border"
                    )}
                  >
                    <Calendar className="h-4 w-4" />
                    {VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule.monthly}
                  </ToggleGroupItem>
                </ToggleGroup>
              </div>
            )}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="start_date">Ngày bắt đầu</Label>
                <Input
                  id="start_date"
                  type="date"
                  value={formData.start_date}
                  onChange={(e) => setFormData(prev => ({ ...prev, start_date: e.target.value }))}
                  className="h-11 w-full"
                />
                <p className="typography-body-small text-muted-foreground">Để trống để hệ thống tự chọn (ngày sau lần chấm công gần nhất, hoặc đầu tháng này)</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="end_date">Ngày kết thúc</Label>
                <Input
                  id="end_date"
                  type="date"
                  value={formData.end_date}
                  onChange={(e) => setFormData(prev => ({ ...prev, end_date: e.target.value }))}
                  className="h-11 w-full"
                />
              </div>
            </div>
      </div>
    </SlideSheetTemplate>
  );
}

export const modalConfig: ModalConfig = {
  id: 'project_assignment',
  name: 'Phân công dự án',
  description: 'Phân công nhân viên vào dự án.',
  category: 'project',
  permissions: {
    action: 'update',
    subject: 'Project',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['projectId', 'employeeId', 'returnTo'],
    example: '?modal=project_assignment&projectId=123&employeeId=456&returnTo=employee_details:456',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.projectId || params.employeeId);
    }
  },
  requiresAuth: true,
  encryptData: false,
};

export default ProjectAssignmentSheet;
