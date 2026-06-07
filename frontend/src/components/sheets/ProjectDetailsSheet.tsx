import { useState, useMemo, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { Project } from "@/types/api/project.types";
import { ProjectEmployeesList } from "@/components/project-employees/ProjectEmployeesList";
import { AddEmployeesToProject } from "@/components/project-employees/AddEmployeesToProject";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { PayrateConfigTab } from "@/components/payrates/PayrateConfigTab";
import { ProjectHeader } from "@/components/projects/details/ProjectHeader";
import { ProjectInfoTab } from "@/components/projects/details/ProjectInfoTab";
import { ProjectUserAccessTab } from "@/components/projects/details/ProjectUserAccessTab";
import { GroupedStatCard } from "@/components/shared/GroupedStatCard";
import ProjectEditSheet from "./ProjectEditSheet";
import { Button } from "@/components/ui/button";
import ConfirmationModal from "@/components/modals/ConfirmationModal";
import { useProject, usePauseProject, useResumeProject, useStartProject, useCompleteProject, useCancelProject, useDeleteProject, useProjectApprovedTimesheets } from "@/hooks/api/useProjects";
import { useCanEditProject, useCanManageProjectEmployees } from "@/hooks/useCanEditProject";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";
import { authManager } from "@/lib/auth";
import { GeofenceSection } from "@/components/projects/details/GeofenceSection";
import { Edit3, Trash2, Plus } from "lucide-react";
import type { ModalConfig } from "@/types/modal-config.types";

interface ProjectDetailsSheetProps {
  project?: Project | null;
  isOpen: boolean;
  onClose: () => void;
  id?: string;
  onProjectDeleted?: (deletedProjectId: number) => void;
  initialTab?: string;
  tab?: string;
}

function ProjectDetailsSheet({
  project: propProject,
  isOpen,
  onClose,
  id,
  onProjectDeleted
}: ProjectDetailsSheetProps) {
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const [isAddEmployeeOpen, setIsAddEmployeeOpen] = useState(false);
  const [isEditProjectOpen, setIsEditProjectOpen] = useState(false);
  const [isCompleteModalOpen, setIsCompleteModalOpen] = useState(false);
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);



  // Status transition hooks
  const startProjectMutation = useStartProject();
  const pauseProjectMutation = usePauseProject();
  const resumeProjectMutation = useResumeProject();
  const completeProjectMutation = useCompleteProject();
  const cancelProjectMutation = useCancelProject();
  const deleteProjectMutation = useDeleteProject();

  // Fetch project data if ID is provided but no project prop
  const { data: fetchedProject, isLoading: projectLoading } = useProject(
    id ? parseInt(id) : 0,
    isOpen && !!id && !propProject
  );

  // Use prop project or fetched project
  const project = propProject || fetchedProject;

  // Get current user role
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';

  // Check permissions using hooks
  const canEditProject = useCanEditProject(project);
  const canManageEmployees = useCanManageProjectEmployees(project);

  // Check for approved timesheets (to prevent deletion)
  const { data: approvedTimesheetsData, isLoading: isLoadingApprovedTimesheets } = useProjectApprovedTimesheets(
    project?.id || 0,
    !!project?.id && isDeleteModalOpen // Only fetch when delete modal is open
  );

  // Determine if project has approved timesheets
  const hasApprovedTimesheets = useMemo(() => {
    return approvedTimesheetsData?.data && approvedTimesheetsData.data.length > 0;
  }, [approvedTimesheetsData]);

  // Determine if project can be deleted (not completed status, no approved timesheets, and user has permission)
  const canDeleteProject = useMemo(() => {
    if (!project) return false;

    // Check user permissions - use the same permission system as canEditProject but allow deletion of cancelled projects
    const userRole = authManager.getUserRole();
    const hasPermission = userRole === 'admin' || userRole === 'partner';
    if (!hasPermission) return false;

    // Check project constraints - can delete any status except completed, and no approved timesheets
    return project.status !== 'completed' && !hasApprovedTimesheets;
  }, [project, hasApprovedTimesheets]);

  // Track employee count from the list API's pagination response (accurate total)
  const [listEmployeeCount, setListEmployeeCount] = useState(0);

  // Fallback to backend data for positions until list loads
  const activePositions = useMemo(
    () => project?.employee_assignments?.positions || [],
    [project?.employee_assignments?.positions]
  );

  // Use list count when available (from pagination totalRecords), fallback to project data
  const activeEmployeesCount = listEmployeeCount > 0
    ? listEmployeeCount
    : (project?.employee_assignments?.total_employees || 0);

  // Memoize stats configuration to prevent recreating on every render
  const statsConfig = useMemo(() => [
    {
      title: "Tổng nhân viên",
      value: activeEmployeesCount,
    },
    ...activePositions.map(pos => ({
      title: pos.position,
      value: pos.count,
    }))
  ], [activeEmployeesCount, activePositions]);

  // Update local tab state when prop changes
  // Handle tab change
  // Handle project update
  const handleProjectUpdated = (updatedProject: Project) => {
    // The project data will be refreshed automatically by react-query
    // since we're using optimistic updates in the hook
  };


  // Handle project pause
  const handlePauseProject = async () => {
    if (!project) return;

    try {
      await pauseProjectMutation.mutateAsync(project.id);
    } catch (error) {
      console.error('Failed to pause project:', error);
    }
  };

  // Handle project resume
  const handleResumeProject = async () => {
    if (!project) return;

    try {
      await resumeProjectMutation.mutateAsync(project.id);
    } catch (error) {
      console.error('Failed to resume project:', error);
    }
  };

  // Handle project start
  const handleStartProject = async () => {
    if (!project) return;

    try {
      await startProjectMutation.mutateAsync(project.id);
    } catch (error) {
      console.error('Failed to start project:', error);
    }
  };

  // Handle project complete
  const handleCompleteProject = async () => {
    if (!project) return;

    try {
      await completeProjectMutation.mutateAsync(project.id);
      setIsCompleteModalOpen(false);
    } catch (error) {
      console.error('Failed to complete project:', error);
    }
  };

  // Handle project cancel
  const handleCancelProject = async () => {
    if (!project) return;

    try {
      await cancelProjectMutation.mutateAsync(project.id);
      setIsCancelModalOpen(false);
    } catch (error) {
      console.error('Failed to cancel project:', error);
    }
  };

  // Handle project delete
  const handleDeleteProject = async () => {
    if (!project) return;

    try {
      await deleteProjectMutation.mutateAsync(project.id);
      setIsDeleteModalOpen(false);

      // Call the onProjectDeleted callback if provided
      if (onProjectDeleted) {
        onProjectDeleted(project.id);
      }

      // Close the sheet after successful deletion
      handleClose();
    } catch (error) {
      console.error('Failed to delete project:', error);
      // Error notification is handled by the global error handler
    }
  };

  // Wrapped close handler with cache invalidation
  const handleClose = useCallback(() => {
    // Invalidate cache to ensure fresh data on next open
    queryClient.invalidateQueries({ queryKey: ['projects'] });
    queryClient.invalidateQueries({ queryKey: ['employees'] });

    // Use modal navigation system if opened via URL, otherwise use prop callback
    if (searchParams.has('modal')) {
      closeModal();
    } else {
      onClose();
    }
  }, [searchParams, closeModal, onClose, queryClient]);

  const handleCompleteCancel = () => {
    setIsCompleteModalOpen(false);
  };

  const handleCancelCancel = () => {
    setIsCancelModalOpen(false);
  };

  const handleDeleteCancel = () => {
    setIsDeleteModalOpen(false);
  };

  // Show loading state when fetching project
  if (projectLoading && !propProject) {
    return (
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={handleClose}
        avatar={{
          custom: (
            <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
              <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                <div className="h-5 w-5 rounded-full bg-primary/20 animate-pulse" />
              </div>
              <div className="space-y-1 flex-1 min-w-0">
                <h2 className="typography-headline-medium font-semibold">
                  Đang tải...
                </h2>
              </div>
            </div>
          )
        }}
        size="large"
      >
        <div className="p-4 space-y-4">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <div className="grid grid-cols-2 gap-4">
            <Skeleton className="h-20" />
            <Skeleton className="h-20" />
          </div>
          <Skeleton className="h-48" />
        </div>
      </SlideSheetTemplate>
    );
  }

  if (!project) return null;

  const avatar = {
    custom: (
      <div className="w-full">
        <ProjectHeader project={project} showName={true} />
      </div>
    )
  };

  // Check if current user can manage project access (admin or project creator)
  const isProjectCreator = project.created_by === authManager.getUserId();
  const canManageProjectAccess = isAdmin || isProjectCreator;

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={handleClose}
        avatar={avatar}
        size="large"
      >
        <div className="space-y-0 py-2">

          {/* ── Info section ── */}
          <div className="pb-2">
            <div className="flex items-center justify-between px-4 sm:px-6 py-2">
              <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">
                Thông tin dự án
              </p>
              {canEditProject && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
                  onClick={() => setIsEditProjectOpen(true)}
                >
                  <Edit3 className="h-3 w-3" />
                  Chỉnh sửa
                </Button>
              )}
            </div>
            <ProjectInfoTab project={project} />
          </div>

          {/* ── Employees section ── */}
          <div className="border-t pt-2 pb-4">
            <div className="flex items-center justify-between px-4 sm:px-6 py-2">
              <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">
                Nhân viên ({activeEmployeesCount})
              </p>
              {canManageEmployees && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
                  onClick={() => setIsAddEmployeeOpen(true)}
                >
                  <Plus className="h-3 w-3" />
                  Thêm
                </Button>
              )}
            </div>
            <div className="px-4 sm:px-6 mb-3">
              <GroupedStatCard
                title="Nhân viên"
                stats={statsConfig.map(s => ({ label: s.title, value: s.value }))}
              />
            </div>
            <ProjectEmployeesList
              project={project}
              onEmployeeRemoved={() => {}}
              onAddEmployees={() => setIsAddEmployeeOpen(true)}
              onTotalCountChange={setListEmployeeCount}
            />
          </div>

          {/* ── Payrates section ── */}
          <div className="border-t pt-2 pb-4">
            <div className="px-4 sm:px-6">
              <PayrateConfigTab
                project={project}
              />
            </div>
          </div>

          {/* ── Geofence section (flexible projects only) ── */}
          {project.is_flexible && (
            <GeofenceSection project={project} />
          )}

          {/* ── Permissions section ── */}
          {canManageProjectAccess && (
            <div className="border-t pt-2 pb-4">
              <div className="flex items-center justify-between px-4 sm:px-6 py-2">
                <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">
                  Phân quyền
                </p>
              </div>
              <div className="px-4 sm:px-6">
                <ProjectUserAccessTab project={project} />
              </div>
            </div>
          )}

          {/* ── Danger zone ── */}
          {canDeleteProject && (
            <div className="border-t pt-2 pb-4 px-4 sm:px-6">
              <div className="py-2 mb-2">
                <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">
                  Vùng nguy hiểm
                </p>
              </div>
              <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-3 flex items-center justify-between gap-3">
                <div>
                  <p className="text-sm font-medium text-destructive">Xóa dự án</p>
                  <p className="text-xs text-muted-foreground mt-0.5">Hành động này không thể hoàn tác</p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setIsDeleteModalOpen(true)}
                  className="shrink-0"
                >
                  <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                  Xóa
                </Button>
              </div>
            </div>
          )}

        </div>
      </SlideSheetTemplate>

      {/* Add Employee Sheet */}
      <AddEmployeesToProject
        project={project}
        isOpen={isAddEmployeeOpen}
        onClose={() => setIsAddEmployeeOpen(false)}
        onEmployeesAdded={() => {
          // Note: No need to invalidate here as the mutation already handles optimistic updates
          // This callback is for UI state management only
        }}
      />

      {/* Edit Project Sheet */}
      <ProjectEditSheet
        project={project}
        isOpen={isEditProjectOpen}
        onClose={() => setIsEditProjectOpen(false)}
        onProjectUpdated={handleProjectUpdated}
      />

      {/* Status Transition Modals - Admin Only */}
      {isAdmin && (
        <>
          {/* Complete Confirmation Modal */}
          <ConfirmationModal
            isOpen={isCompleteModalOpen}
            onClose={handleCompleteCancel}
            onConfirm={handleCompleteProject}
            title="Xác nhận hoàn thành dự án"
            description={`Bạn có chắc chắn muốn đánh dấu dự án "${project?.name}" là hoàn thành? Sau khi hoàn thành, bạn sẽ không thể thêm/xóa nhân viên hoặc tạo timesheet mới.`}
            confirmText="Hoàn thành"
            cancelText="Hủy bỏ"
            variant="default"
            isLoading={completeProjectMutation.isPending}
          />

          {/* Cancel Confirmation Modal */}
          <ConfirmationModal
            isOpen={isCancelModalOpen}
            onClose={handleCancelCancel}
            onConfirm={handleCancelProject}
            title="Xác nhận hủy dự án"
            description={`Bạn có chắc chắn muốn hủy dự án "${project?.name}"? Dự án sẽ được đánh dấu là đã hủy và không thể khôi phục.`}
            confirmText="Hủy dự án"
            cancelText="Hủy bỏ"
            variant="destructive"
            isLoading={cancelProjectMutation.isPending}
          />

        </>
      )}

      {/* Delete Confirmation Modal - Available for all users with edit permissions */}
      <ConfirmationModal
        isOpen={isDeleteModalOpen}
        onClose={handleDeleteCancel}
        onConfirm={canDeleteProject ? handleDeleteProject : undefined}
        title="Xác nhận xóa dự án"
        description={
          <div className="space-y-4">
            <p>Bạn có chắc chắn muốn xóa dự án <strong>"{project?.name}"</strong>?</p>

            {project?.status === 'completed' ? (
              <div className="bg-red-50 border border-red-200 rounded-xl p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-red-600 font-medium">🚫 Không thể xóa:</span>
                </div>
                <ul className="text-sm text-red-800 space-y-1">
                  <li>• Dự án đã hoàn thành không thể xóa</li>
                  <li>• Vui lòng liên hệ quản trị viên nếu cần hỗ trợ</li>
                </ul>
              </div>
            ) : hasApprovedTimesheets ? (
              <div className="bg-red-50 border border-red-200 rounded-xl p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-red-600 font-medium">🚫 Không thể xóa:</span>
                </div>
                <ul className="text-sm text-red-800 space-y-1">
                  <li>• Dự án có bảng công đã duyệt</li>
                  <li>• Vui lòng liên hệ quản trị viên nếu cần hỗ trợ</li>
                </ul>
              </div>
            ) : (
              <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-yellow-600 font-medium">⚠️ Cảnh báo:</span>
                </div>
                <ul className="text-sm text-yellow-800 space-y-1">
                  <li>• Nhân viên được giao: <strong>{activeEmployeesCount} người</strong></li>
                  <li>• Tất cả dữ liệu chấm công của dự án sẽ bị xóa</li>
                  <li>• Hành động này <strong>không thể hoàn thành</strong></li>
                </ul>
              </div>
            )}

            {canDeleteProject && (
              <p className="text-sm text-muted-foreground">
                Việc xóa sẽ bị từ chối nếu có bảng công đã duyệt.
              </p>
            )}
          </div>
        }
        confirmText={canDeleteProject ? "Xóa dự án" : undefined}
        cancelText={canDeleteProject ? "Hủy bỏ" : "Đóng"}
        variant="destructive"
        isLoading={deleteProjectMutation.isPending || isLoadingApprovedTimesheets}
      />
    </>
  );
}

export const modalConfig: ModalConfig = {
  id: 'project_details_sheet',
  name: 'Chi tiết dự án',
  description: 'Xem chi tiết dự án, nhân viên và cấu hình lương.',
  category: 'project',
  permissions: {
    action: 'read',
    subject: 'Project',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id', 'tab'],
    example: '?modal=project_details_sheet&id=123&tab=employees',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  requiresAuth: true,
  encryptData: false,
};

export default ProjectDetailsSheet;
