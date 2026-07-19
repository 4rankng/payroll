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
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { useProject, usePauseProject, useResumeProject, useStartProject, useCompleteProject, useCancelProject, useDeleteProject, useProjectApprovedTimesheets } from "@/hooks/api/useProjects";
import { useCanEditProject, useCanManageProjectEmployees } from "@/hooks/useCanEditProject";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";
import { authManager } from "@/lib/auth";
import { GeofenceSection } from "@/components/projects/details/GeofenceSection";
import { ShiftNamesSection } from "@/components/projects/details/ShiftNamesSection";
import { Edit3, Trash2, Plus, LayoutGrid, Users, DollarSign, ShieldCheck } from "lucide-react";
import type { ModalConfig } from "@/types/modal-config.types";

// Internal tab ids. External callers (URL ?tab=, useProjectModals) use a legacy
// vocabulary ('info' | 'employees' | 'payrates' | 'timesheet' | 'settings') which
// we normalize via `mapInitialTab` so existing deep-links keep working.
type ProjectTab = 'overview' | 'employees' | 'payrates' | 'access';

const mapInitialTab = (raw?: string): ProjectTab => {
  switch (raw) {
    case 'employees': return 'employees';
    case 'payrates': return 'payrates';
    case 'settings': return 'access';
    case 'info':
    case 'timesheet':
    default: return 'overview';
  }
};

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
  onProjectDeleted,
  initialTab
}: ProjectDetailsSheetProps) {
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const [activeTab, setActiveTab] = useState<ProjectTab>(() => mapInitialTab(initialTab));
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
          <div className="grid grid-cols-1 gap-4 min-[420px]:grid-cols-2">
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

  // Clamp tab: if a deep-link points to 'access' but the viewer lacks permission,
  // fall back to overview so the sheet never opens on an invisible tab.
  const effectiveTab: ProjectTab = activeTab === 'access' && !canManageProjectAccess ? 'overview' : activeTab;

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={handleClose}
        avatar={avatar}
        size="large"
      >
        <Tabs
          value={effectiveTab}
          onValueChange={(v) => setActiveTab(v as ProjectTab)}
          className="flex flex-col gap-4"
        >
          {/* Sticky pill tab bar — breaks out of template's content padding, re-pads.
              'info' value kept for the overview tab so URL deep-links remain stable. */}
          <div className="-mx-4 sm:-mx-6 sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-b px-4 sm:px-6 pb-2 pt-0.5">
            <TabsList className="bg-transparent p-0 h-auto w-full inline-flex gap-1 overflow-x-auto">
              <TabsTrigger
                value="overview"
                className="min-h-9 flex-1 gap-1.5 rounded-lg px-3 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                <LayoutGrid className="h-3.5 w-3.5" />
                <span className="text-xs font-semibold">Tổng quan</span>
              </TabsTrigger>
              <TabsTrigger
                value="employees"
                className="min-h-9 flex-1 gap-1.5 rounded-lg px-3 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                <Users className="h-3.5 w-3.5" />
                <span className="text-xs font-semibold">Nhân viên</span>
                <span className="ml-0.5 inline-flex min-w-4 items-center justify-center rounded-full bg-muted px-1.5 py-px text-[10px] font-bold tabular-nums leading-none">
                  {activeEmployeesCount}
                </span>
              </TabsTrigger>
              <TabsTrigger
                value="payrates"
                className="min-h-9 flex-1 gap-1.5 rounded-lg px-3 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                <DollarSign className="h-3.5 w-3.5" />
                <span className="text-xs font-semibold">Lương</span>
              </TabsTrigger>
              {canManageProjectAccess && (
                <TabsTrigger
                  value="access"
                  className="min-h-9 flex-1 gap-1.5 rounded-lg px-3 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  <ShieldCheck className="h-3.5 w-3.5" />
                  <span className="text-xs font-semibold">Phân quyền</span>
                </TabsTrigger>
              )}
            </TabsList>
          </div>

          {/* ── Overview tab ── */}
          <TabsContent value="overview" className="mt-0 focus-visible:outline-none">
            <div className="flex items-center justify-between gap-3 mb-2">
              <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
                Thông tin dự án
              </p>
              {canEditProject && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="min-h-9 px-3 text-xs text-muted-foreground hover:text-foreground gap-1.5"
                  onClick={() => setIsEditProjectOpen(true)}
                >
                  <Edit3 className="h-3 w-3" />
                  Chỉnh sửa
                </Button>
              )}
            </div>
            <ProjectInfoTab project={project} />

            {project.is_flexible && (
              <div className="mt-4 space-y-4">
                <GeofenceSection project={project} />
                <ShiftNamesSection project={project} />
              </div>
            )}

            {canDeleteProject && (
              <div className="mt-5">
                <div className="py-2 mb-2">
                  <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
                    Vùng nguy hiểm
                  </p>
                </div>
                <div className="flex flex-col gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-3 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-destructive">Xóa dự án</p>
                    <p className="text-xs text-muted-foreground mt-0.5">Hành động này không thể hoàn tác</p>
                  </div>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => setIsDeleteModalOpen(true)}
                    className="min-h-11 shrink-0"
                  >
                    <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                    Xóa
                  </Button>
                </div>
              </div>
            )}
          </TabsContent>

          {/* ── Employees tab ── */}
          <TabsContent value="employees" className="mt-0 focus-visible:outline-none">
            <div className="flex items-center justify-between gap-3 mb-3">
              <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
                Nhân viên ({activeEmployeesCount})
              </p>
              {canManageEmployees && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="min-h-9 px-3 text-xs text-muted-foreground hover:text-foreground gap-1.5"
                  onClick={() => setIsAddEmployeeOpen(true)}
                >
                  <Plus className="h-3 w-3" />
                  Thêm
                </Button>
              )}
            </div>
            <div className="mb-3">
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
          </TabsContent>

          {/* ── Payrates tab ── */}
          <TabsContent value="payrates" className="mt-0 focus-visible:outline-none">
            <PayrateConfigTab project={project} />
          </TabsContent>

          {/* ── Access tab ── */}
          {canManageProjectAccess && (
            <TabsContent value="access" className="mt-0 focus-visible:outline-none">
              <div className="flex items-center justify-between gap-3 mb-3">
                <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
                  Phân quyền
                </p>
              </div>
              <ProjectUserAccessTab project={project} />
            </TabsContent>
          )}
        </Tabs>
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
          <ConfirmDialog
            open={isCompleteModalOpen}
            onOpenChange={handleCompleteCancel}
            onConfirm={handleCompleteProject}
            title="Xác nhận hoàn thành dự án"
            description={`Bạn có chắc chắn muốn đánh dấu dự án "${project?.name}" là hoàn thành? Sau khi hoàn thành, bạn sẽ không thể thêm/xóa nhân viên hoặc tạo timesheet mới.`}
            confirmText="Hoàn thành"
            cancelText="Hủy bỏ"
            confirmVariant="default"
            loading={completeProjectMutation.isPending}
          />

          {/* Cancel Confirmation Modal */}
          <ConfirmDialog
            open={isCancelModalOpen}
            onOpenChange={handleCancelCancel}
            onConfirm={handleCancelProject}
            title="Xác nhận hủy dự án"
            description={`Bạn có chắc chắn muốn hủy dự án "${project?.name}"? Dự án sẽ được đánh dấu là đã hủy và không thể khôi phục.`}
            confirmText="Hủy dự án"
            cancelText="Hủy bỏ"
            confirmVariant="destructive"
            loading={cancelProjectMutation.isPending}
          />

        </>
      )}

      {/* Delete Confirmation Modal - Available for all users with edit permissions */}
      <ConfirmDialog
        open={isDeleteModalOpen}
        onOpenChange={handleDeleteCancel}
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
        confirmVariant="destructive"
        loading={deleteProjectMutation.isPending || isLoadingApprovedTimesheets}
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
    example: '?modal=project_details_sheet&id=123&tab=overview',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  requiresAuth: true,
  encryptData: false,
};

export default ProjectDetailsSheet;
