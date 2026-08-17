import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { SlideSheetTemplate } from "@/components/sheets/templates/SlideSheetTemplate";
import { RemoveEmployeeSheet } from "@/components/modals/RemoveEmployeeModal";
import { DeleteEmployeeModal } from "./components/DeleteEmployeeModal";
import { ResetPasswordModalContainer } from "@/components/modals/ResetPasswordModalContainer";
import { useEmployeeForm } from "@/hooks/employees/useEmployeeForm";
import { useEmployeeDetails } from "@/hooks/employees/useEmployeeDetails";
import { useUpdateEmployee, useDeleteEmployee, useRequestEmployeeAccess } from "@/hooks/api/useEmployees";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  AlertCircle,
  Calendar,
  Mail,
  Phone,
  MapPin,
  Hash,
  Clock,
  User,
  AtSign,
  FileDown,
  Pencil,
  HandHeart,
} from "lucide-react";
import { authManager } from "@/lib/auth";
import { format } from "date-fns";
import { employeeService } from "@/services/api/employee.service";
import { showErrorNotification, showSuccessNotification } from "@/utils/error-handler";
import type { Employee } from "@/types/api/employee.types";
import type { EmployeeTimesheetResponse } from "@/types/api/timesheet.types";
import type { Bank } from "@/types/api/bank.types";

// Components
import { EmployeeHeader } from "./components/EmployeeHeader";
import { EmployeeEditForm } from "./components/EmployeeEditForm";
import { EmployeeProjectSection } from "./components/EmployeeProjectSection";
import { EmployeeBankInfo } from "./components/EmployeeBankInfo";
import { EmployeeStatistics } from "./components/EmployeeStatistics";
import { EmployeeActions } from "./components/EmployeeActions";
import { EmployeeUserAccessTab } from "./EmployeeUserAccessTab";

interface EmployeeDetailsSheetProps {
  employee?: Employee | null;
  // True when the detail fetch failed (e.g. partner 403 on GET /employees/:id).
  // Lets us show a real error state instead of hanging on the loading skeleton.
  fetchError?: boolean;
  isOpen: boolean;
  onClose: () => void;
  onUpdate?: (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => Promise<void> | void;
  onDelete?: (employee: Employee) => void;
  onRefetch?: () => void;
}

export function EmployeeDetailsSheet({
  employee: propEmployee,
  fetchError = false,
  isOpen,
  onClose,
  onUpdate,
  onDelete,
  onRefetch
}: EmployeeDetailsSheetProps) {
  // ALL HOOKS MUST BE CALLED AT THE TOP - Rules of Hooks
  const queryClient = useQueryClient();
  const updateEmployeeMutation = useUpdateEmployee();
  const deleteEmployeeMutation = useDeleteEmployee();
  const [projectToRemove, setProjectToRemove] = useState<number | null>(null);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isResetPasswordModalOpen, setIsResetPasswordModalOpen] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const isNarrowViewport = useIsMobile();

  // Use the employee prop from container (container handles data fetching)
  const employee = propEmployee;

  // Form management - MUST be called before any useCallback that uses its return values
  const {
    isEditing,
    setIsEditing,
    formData,
    selectedBank,
    handleInputChange,
    handleBankChange,
    handleSave,
    handleCancel,
    resetFormData,
    projectChanges,
    handleProjectChange,
    applyProjectChanges,
    isDirty,
  } = useEmployeeForm({
    employee,
    onUpdate: onUpdate || (() => {})
  });

  // Data management - MUST be called before any useCallback that uses its return values
  const {
    timesheetData,
    timesheetLoading,
    summaryData,
    summaryLoading,
    projects,
    showRemoveModal,
    setShowRemoveModal,
    removeResponseMessage,
    setRemoveResponseMessage,
    handleRemoveFromProject,
    timesheetFilters,
    timesheetSorting,
    setTimesheetSorting,
    handlePageChange,
    handleDateRangeChange,
    removeEmployee,
  } = useEmployeeDetails({ employee, onRefetch: onRefetch || (() => {}) });

  // Default update handler
  const defaultOnUpdate = useCallback(async (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => {
    return new Promise<void>((resolve, reject) => {
      updateEmployeeMutation.mutate({
        id: employeeId,
        data: employeeData,
        bank: bankObject
      }, {
        onSuccess: () => {
          // Cache is already updated with backend response in useUpdateEmployee hook
          resolve();
        },
        onError: (error) => {
          reject(error);
        }
      });
    });
  }, [updateEmployeeMutation]);

  const defaultOnDelete = useCallback((employee: Employee) => {
    setIsDeleteModalOpen(true);
  }, []);

  // Handle employee deletion
  const handleDeleteEmployee = async () => {
    if (!employee) return;

    try {
      await deleteEmployeeMutation.mutateAsync(employee.id);
      setIsDeleteModalOpen(false);
      handleClose(); // Close the sheet after successful deletion
    } catch (error) {
      console.error('Failed to delete employee:', error);
    }
  };

  // Wrapped close handler with cache invalidation
  const handleClose = useCallback(() => {
    // Invalidate cache to ensure fresh data on next open
    queryClient.invalidateQueries({ queryKey: ['employees'] });
    onClose();
  }, [queryClient, onClose]);

  // Check if current user can manage employee access (admin or employee creator)
  const isEmployeeCreator = employee?.created_by === authManager.getUserId();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';
  const canManageEmployeeAccess = isAdmin || isEmployeeCreator;

  // Preview mode: partner viewing an employee they don't manage (global pool).
  // Backend already masks bank/address and omits summaries; hide all mutating
  // actions and offer the one action that makes sense: request management.
  const isPreviewOnly = !isAdmin && employee?.is_accessible === false;
  const requestAccess = useRequestEmployeeAccess();
  const handleClaimAccess = useCallback(() => {
    if (!employee) return;
    requestAccess.mutate(employee.id, {
      onSuccess: () => handleClose(),
    });
  }, [employee, requestAccess, handleClose]);

  const avatar = useMemo(() => ({
    custom: employee ? <EmployeeHeader employee={employee} showName={true} /> : null
  }), [employee]);

  // Custom handlers with direct form state management
  const handleEditClick = useCallback(() => {
    setIsEditing(true);
  }, [setIsEditing]);

  const handleSaveClick = useCallback(async () => {
    await handleSave();
    // Cache is already updated with backend response in useUpdateEmployee hook
    setIsEditing(false);
  }, [handleSave, setIsEditing]);

  const handleCancelClick = useCallback(() => {
    handleCancel(); // This is from useEmployeeForm hook
  }, [handleCancel]);

  // Handle project removal
  const handleProjectRemove = useCallback((projectId: number) => {
    setProjectToRemove(projectId);
    setShowRemoveModal(true);
  }, [setShowRemoveModal]);

  // Handle confirm removal
  const handleConfirmRemoval = useCallback(async (employeeId: number, projectId: number, lastDate?: string) => {
    if (employee && projectToRemove) {
      await handleRemoveFromProject(employeeId, projectId, lastDate);
      setProjectToRemove(null);
    }
  }, [employee, projectToRemove, handleRemoveFromProject]);

  const handleExport = useCallback(async () => {
    if (!employee) return;
    setIsExporting(true);
    try {
      await employeeService.exportEmployeeDetail(employee.id, employee.fullname);
      showSuccessNotification("Xuất Excel thành công");
    } catch {
      showErrorNotification("Lỗi xuất Excel", "Không thể xuất hồ sơ nhân viên. Vui lòng thử lại.");
    } finally {
      setIsExporting(false);
    }
  }, [employee]);

  const headerActions = useMemo(() => {
    if (!employee || isEditing || isPreviewOnly) return undefined;
    return (
      <Button
        variant="outline"
        size="sm"
        onClick={handleExport}
        disabled={isExporting}
        className={isNarrowViewport ? "min-h-11 px-3 gap-1 text-xs" : "min-h-11 px-3 gap-1.5"}
      >
        <FileDown className="h-3.5 w-3.5" />
        {!isNarrowViewport && "Xuất Excel"}
      </Button>
    );
  }, [employee, isEditing, isPreviewOnly, handleExport, isExporting, isNarrowViewport]);

  const footer = useMemo(() => {
    if (!employee || isEditing) return null;

    const currentProjects = employee.current_projects ?? [];
    const canDelete = employee.can_delete !== false && !currentProjects.some(
      (project) => project.payment_schedule === "flexible",
    );

    return (
      <EmployeeActions
        onDelete={isPreviewOnly ? undefined : (canDelete ? () => (onDelete || defaultOnDelete)(employee) : undefined)}
        onResetPassword={isPreviewOnly ? undefined : () => setIsResetPasswordModalOpen(true)}
        onClose={handleClose}
      />
    );
  }, [employee, isEditing, onDelete, defaultOnDelete, handleClose, isPreviewOnly]);

  const editingFooter = useMemo(() => {
    if (!employee || !isEditing) return null;

    return (
      <EmployeeActions
        onSave={handleSaveClick}
        onCancel={handleCancelClick}
        isDirty={isDirty}
        onDelete={() => (onDelete || defaultOnDelete)(employee)}
        onResetPassword={() => setIsResetPasswordModalOpen(true)}
        onClose={undefined}
      />
    );
  }, [employee, isEditing, handleSaveClick, handleCancelClick, isDirty, onDelete, defaultOnDelete]);

  // Early returns AFTER all hooks are called
  if (!isOpen) {
    return null;
  }

  if (!employee) {
    if (fetchError) {
      return (
        <SlideSheetTemplate
          isOpen={isOpen}
          onClose={onClose}
          title="Không tải được"
          size="default"
        >
          <div className="p-6 flex flex-col items-center justify-center text-center gap-3">
            <AlertCircle className="h-10 w-10 text-destructive" />
            <div className="space-y-1">
              <p className="font-medium">Không thể tải thông tin nhân viên</p>
              <p className="text-sm text-muted-foreground">
                Bạn không có quyền xem nhân viên này hoặc dữ liệu không tồn tại.
              </p>
            </div>
            <Button variant="outline" size="sm" onClick={onClose} className="min-h-11">
              Đóng
            </Button>
          </div>
        </SlideSheetTemplate>
      );
    }
    return (
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={onClose}
        title="Đang tải..."
        size="default"
      >
        <div className="p-4 space-y-4">
          <div className="text-center text-muted-foreground">
            <p>Đang tải thông tin nhân viên...</p>
            <p className="text-sm">ID: Loading...</p>
          </div>
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        </div>
      </SlideSheetTemplate>
    );
  }

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={handleClose}
        title=""
        avatar={avatar}
        footer={isEditing ? editingFooter : footer}
        headerActions={headerActions}
        className={isNarrowViewport ? "w-full p-0" : "w-full md:w-[60%] min-w-[800px] p-0"}
      >
        <div className="space-y-3 py-1">
          {/* ── Personal info / Edit form ── */}
          {isEditing ? (
            <EmployeeEditForm
              employee={employee}
              formData={formData}
              selectedBank={selectedBank}
              onInputChange={handleInputChange}
              onBankChange={handleBankChange}
              canCreateBank={isAdmin}
            />
          ) : (
            <>
              <div className="flex items-center justify-between gap-3">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Thông tin cá nhân</p>
                {!isPreviewOnly && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleEditClick}
                    className={isNarrowViewport ? "min-h-11 px-3 gap-1 text-xs" : "min-h-11 px-3 gap-1 text-xs"}
                  >
                    <Pencil className="h-3 w-3" />
                    Chỉnh sửa
                  </Button>
                )}
              </div>
              {isPreviewOnly && (
                <div className="flex items-center justify-between gap-3 rounded-xl border border-info/30 bg-info/5 px-3 py-2.5">
                  <p className="text-xs text-foreground/80">
                    Bạn đang xem bản xem trước. Thông tin ngân hàng, địa chỉ và lịch sử được ẩn.
                  </p>
                  <Button
                    size="sm"
                    onClick={handleClaimAccess}
                    disabled={requestAccess.isPending}
                    className="shrink-0 gap-1.5"
                  >
                    <HandHeart className="h-3.5 w-3.5" />
                    Yêu cầu quản lý
                  </Button>
                </div>
              )}
              <EmployeePersonalInfo employee={employee} />

              {/* ── Bank info (view only) ── */}
              <div className="border-t pt-3 space-y-2">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Thông tin ngân hàng</p>
                <EmployeeBankInfo employee={employee} />
              </div>

              {/* ── Projects ── */}
              <div className="border-t pt-3">
                <EmployeeProjectSection
                  employee={employee}
                  onRemove={isPreviewOnly ? undefined : handleProjectRemove}
                  isRemoving={removeEmployee.isPending}
                  isEditing={false}
                  projectChanges={projectChanges}
                  onProjectChange={handleProjectChange}
                  onProjectApply={applyProjectChanges}
                  allowProjectLinking={!isPreviewOnly}
                />
              </div>

              {/* ── Statistics (management data — hidden in preview) ── */}
              {!isPreviewOnly && (
                <div className="border-t pt-3">
                  <EmployeeStatistics
                    employee={employee}
                    summaryData={summaryData}
                    summaryLoading={summaryLoading}
                    timesheetData={timesheetData}
                    timesheetLoading={timesheetLoading}
                    timesheetFilters={timesheetFilters}
                    timesheetSorting={timesheetSorting}
                    onTimesheetSortingChange={setTimesheetSorting}
                    onPageChange={handlePageChange}
                    onDateRangeChange={handleDateRangeChange}
                  />
                </div>
              )}

              {/* ── Permissions ── */}
              {canManageEmployeeAccess && (
                <div className="border-t pt-3">
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Phân quyền</p>
                  <EmployeeUserAccessTab employee={employee} />
                </div>
              )}
            </>
          )}
        </div>
      </SlideSheetTemplate>

      <RemoveEmployeeSheet
        employee={employee}
        projectId={projectToRemove}
        isOpen={showRemoveModal}
        onClose={() => {
          setShowRemoveModal(false);
          setProjectToRemove(null);
          setRemoveResponseMessage(undefined);
        }}
        onConfirm={handleConfirmRemoval}
        isLoading={removeEmployee.isPending}
        responseMessage={removeResponseMessage}
      />

      {/* Delete Employee Confirmation Modal */}
      <DeleteEmployeeModal
        employee={employee}
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        onConfirm={handleDeleteEmployee}
        isLoading={deleteEmployeeMutation.isPending}
      />

      {/* Reset Employee Password Modal */}
      <ResetPasswordModalContainer
        isOpen={isResetPasswordModalOpen}
        onClose={() => setIsResetPasswordModalOpen(false)}
        targetType="employee"
        employeeId={employee.id.toString()}
      />
    </>
  );
}

// Personal Information component for view mode
const EmployeePersonalInfo = React.memo(function EmployeePersonalInfo({ employee }: { employee: Employee }) {
  return (
    <div className="rounded-xl bg-muted/30 px-3 py-1">
        {/* Row 1: username + CCCD */}
        <div className="grid grid-cols-1 gap-x-4 border-b border-border/50 min-[420px]:grid-cols-2">
          <div className="flex flex-col gap-1 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
            <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><AtSign className="w-3 h-3" />Đăng nhập</span>
            <span className="text-xs font-mono font-medium break-all min-[360px]:text-right">{employee.username || '-'}</span>
          </div>
          <div className="flex flex-col gap-1 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
            <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><Hash className="w-3 h-3" />CCCD</span>
            <span className="text-xs font-mono font-medium break-all min-[360px]:text-right">{employee.cccd || '-'}</span>
          </div>
        </div>
        {/* Row 2: phone + DOB */}
        <div className="grid grid-cols-1 gap-x-4 border-b border-border/50 min-[420px]:grid-cols-2">
          <div className="flex flex-col gap-1 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
            <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><Phone className="w-3 h-3" />Điện thoại</span>
            <span className="text-xs font-medium break-all min-[360px]:text-right">{employee.mobile || '-'}</span>
          </div>
          <div className="flex flex-col gap-1 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
            <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><Calendar className="w-3 h-3" />Ngày sinh</span>
            <span className="text-xs font-medium min-[360px]:text-right">
              {employee.date_of_birth ? format(new Date(employee.date_of_birth), 'dd/MM/yyyy') : '-'}
            </span>
          </div>
        </div>
        {/* Row 3: email + address */}
        <div className="flex flex-col gap-1 border-b border-border/50 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><Mail className="w-3 h-3" />Email</span>
          <span className="text-xs font-medium break-all min-[360px]:text-right">{employee.email || '-'}</span>
        </div>
        <div className="flex flex-col gap-1 border-b border-border/50 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><MapPin className="w-3 h-3" />Địa chỉ</span>
          <span className="text-xs font-medium leading-relaxed break-words min-[360px]:text-right">{employee.address || '-'}</span>
        </div>
        {/* Row 4: join date */}
        <div className="flex flex-col gap-1 py-2 min-[360px]:flex-row min-[360px]:items-baseline min-[360px]:justify-between">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1"><Clock className="w-3 h-3" />Tham gia</span>
          <span className="text-xs font-medium min-[360px]:text-right">
            {employee.created_at ? format(new Date(employee.created_at), 'dd/MM/yyyy') : '-'}
          </span>
        </div>
      </div>
  );
});
