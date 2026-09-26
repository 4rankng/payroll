import React from 'react';
import { useSearchParams, useLocation } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { MODAL_IDS, getModalMetadata, isValidModalId, type ModalId } from '@/constants/modalRegistry';
import { authManager } from '@/lib/auth';
import { toast } from '@/components/ui/sonner';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useUpdateEmployee, useEmployee } from '@/hooks/api/useEmployees';
import { useProject, useAssignEmployee, useCreateProject } from '@/hooks/api/useProjects';
import type { AssignEmployeeData, Project, CreateProjectData } from '@/types/api/project.types';
import { useCreateUser, useUsers } from '@/hooks/api/useUsers';
import { payRateService } from '@/services/api/payrate.service';
import type { PayrateStructure } from '@/types/api/payrate.types';
import { useTransaction } from '@/hooks/transactions/useTransactions';

// Import working sheet components (default exports)
import AddUserSheet from '@/components/sheets/AddUserSheet';
import AddEmployeeSheet from '@/components/sheets/AddEmployeeSheet';
import { AddEmployeeToProjectSheetContainer } from '@/components/project-employees/AddEmployeeToProjectSheetContainer';
import AddProjectSheet from '@/components/sheets/AddProjectSheet';
import UserDetailsSheetContainer from '@/components/sheets/UserDetailsSheetContainer';
import UserProfileSheet from '@/components/sheets/UserProfileSheet';
import ProjectDetailsSheet from '@/components/sheets/ProjectDetailsSheet';
import ProjectEditSheet from '@/components/sheets/ProjectEditSheet';
import EmployeeDetailsSheet from '@/components/sheets/EmployeeDetailsSheet';
import ProjectAssignmentSheet from '@/components/sheets/ProjectAssignmentSheet';

// Import working modal components (named exports)
import { ChangePasswordModal } from '@/components/modals/ChangePasswordModal';
import ResetPasswordModalContainer from '@/components/modals/ResetPasswordModalContainer';
import { RemoveEmployeeSheet } from '@/components/modals/RemoveEmployeeModal';
import { TimesheetDetailsModal } from '@/components/modals/TimesheetDetailsModal';
import { TimesheetEntrySheetWrapper } from '@/components/sheets/TimesheetEntrySheetWrapper';
import { FileUploadModal } from '@/components/modals/FileUploadModal';
import { ApprovalDetailsModal } from '@/components/modals/ApprovalDetailsModal';
import { NotificationSheet } from '@/components/notifications/NotificationSheet';
import LedgerEntryDetailsSheet from '@/components/ledger/LedgerEntryDetailsSheet';
import { AddTransactionSheet } from '@/components/sheets/AddTransactionSheet';
import TransactionDetailsSheet from '@/components/sheets/TransactionDetailsSheet';
import { SettleTransactionModal } from '@/components/modals/SettleTransactionModal';
import { ReverseTransactionModal } from '@/components/modals/ReverseTransactionModal';

/**
 * Modal Router - Clean URL-Driven Modal System
 * Currently shows fallback for all modals until components are properly exported
 */

interface ModalRouterProps {
  fallback?: React.ComponentType<{ modalId: string; error?: string }>;
}

// Container component for ProjectAssignmentSheet to handle hooks properly
function ProjectAssignmentSheetContainer({
  isOpen,
  onClose,
  onAssign,
  employeeId: employeeIdParam,
  projectId: projectIdParam,
  returnTo
}: {
  isOpen: boolean;
  onClose: () => void;
  onAssign: () => void;
  employeeId?: string | null;
  projectId?: string | null;
  returnTo?: string | null;
}) {
  const { closeModal } = useModalNavigation();

  // Handle deeplink parameters for project assignment
  const employeeId = employeeIdParam ? parseInt(String(employeeIdParam), 10) : undefined;
  const projectId = projectIdParam ? parseInt(String(projectIdParam), 10) : undefined;


  const { data: employee, isLoading: isEmployeeLoading } = useEmployee(employeeId || 0, !!employeeId);
  const { data: project, isLoading: isProjectLoading } = useProject(projectId || 0, !!projectId);
  const assignEmployeeMutation = useAssignEmployee();

  // Prevent race conditions with loading states
  const isDataLoading = (!!employeeId && isEmployeeLoading) || (!!projectId && isProjectLoading);

  const handleAssign = async (projectId: number, data: AssignEmployeeData, selectedProject?: Project) => {
    // Prevent duplicate submissions if already in progress
    if (assignEmployeeMutation.isPending) {
      return;
    }

    try {
      // Additional validation to prevent conflicts
      if (!data.employee_id || !projectId) {
        toast({
          title: "Lỗi dữ liệu",
          description: "Thiếu thông tin nhân viên hoặc dự án.",
          variant: "destructive"
        });
        return;
      }

      await assignEmployeeMutation.mutateAsync({
        projectId,
        data,
        selectedProject
      });

      toast({
        title: "Thành công",
        description: "Đã phân công nhân viên vào dự án thành công.",
      });

      closeModal();
    } catch (error: unknown) {
      // Handle specific conflict errors from API
      const apiError = error as { response?: { status?: number; data?: { message?: string } } };
      if (apiError?.response?.status === 409) {
        toast({
          title: "Xung đột",
          description: "Nhân viên đã được phân công vào dự án này hoặc có xung đột thời gian.",
          variant: "destructive"
        });
      } else if (apiError?.response?.status === 422) {
        toast({
          title: "Lỗi dữ liệu",
          description: apiError?.response?.data?.message || "Dữ liệu không hợp lệ.",
          variant: "destructive"
        });
      } else if (apiError?.response?.status === 400) {
        toast({
          title: "Dữ liệu không hợp lệ",
          description: apiError?.response?.data?.message || "Vui lòng kiểm tra lại thông tin.",
          variant: "destructive"
        });
      } else {
        toast({
          title: "Lỗi",
          description: "Không thể phân công nhân viên. Vui lòng thử lại.",
          variant: "destructive"
        });
      }
    }
  };

  return (
    <ProjectAssignmentSheet
      isOpen={isOpen}
      onClose={onClose}
      onAssign={handleAssign}
      employee={employee || null}
      selectedProject={project || undefined}
      loading={assignEmployeeMutation.isPending || isDataLoading}
    />
  );
}

// Container component for AddProjectSheet to handle cache invalidation before closing
function AddProjectSheetContainer({
  isOpen,
  onClose
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const createProjectMutation = useCreateProject();

  const handleProjectCreate = async (projectData: CreateProjectData, payrates?: PayrateStructure) => {
    try {


      // Create the project and wait for the mutation to complete
      const response = await createProjectMutation.mutateAsync(projectData);
      const createdProject = response.data;

      // If payrates are provided, create payrate configuration for the project
      if (payrates && createdProject?.id && Object.keys(payrates).length > 0) {


        try {
          await payRateService.createProjectPayRate(createdProject.id, {
            rates: payrates,
            effective_from: projectData.start_date,
            // No end date - payrate is indefinite unless manually updated
          });

        } catch (payrateError) {
          console.error('❌ Failed to create payrate configuration:', payrateError);
          // Don't throw here - we want the project creation to succeed even if payrate creation fails
          // The user will be notified through the mutation's error handling
        }
      }

      // Wait a bit for the onSuccess cache invalidation to complete
      await new Promise<void>((resolve) => {
        setTimeout(async () => {
          // Ensure all project list queries are refreshed
          await queryClient.refetchQueries({
            predicate: (query) => {
              const key = query.queryKey;
              return Array.isArray(key) &&
                     key[0] === 'projects' &&
                     key[1] === 'list' &&
                     query.getObserversCount() > 0;
            }
          });

          // Also refresh summary data
          await queryClient.refetchQueries({
            predicate: (query) => {
              const key = query.queryKey;
              return Array.isArray(key) &&
                     key[0] === 'projects' &&
                     key[1] === 'summary';
            }
          });


          resolve();
        }, 150); // Small delay to ensure hook's onSuccess cache invalidation completes first
      });

    } catch (error) {
      console.error('❌ Project creation failed:', error);
      throw error; // Re-throw to let the error handling in the hook work
    }
  };

  return (
    <AddProjectSheet
      isOpen={isOpen}
      onClose={onClose}
      onProjectCreate={handleProjectCreate}
    />
  );
}

// Container component for SettleTransactionModal to handle transaction data
function SettleTransactionModalContainer({
  isOpen,
  onClose,
  transactionId: transactionIdParam
}: {
  isOpen: boolean;
  onClose: () => void;
  transactionId?: string | null;
}) {
  const transactionId = transactionIdParam ? parseInt(String(transactionIdParam), 10) : 0;
  const { data: transaction, isLoading } = useTransaction(transactionId);

  if (isLoading || !transaction) {
    return null;
  }

  return (
    <SettleTransactionModal
      isOpen={isOpen}
      onClose={onClose}
      transaction={transaction}
    />
  );
}

// Container component for ReverseTransactionModal to handle transaction data
function ReverseTransactionModalContainer({
  isOpen,
  onClose,
  transactionId: transactionIdParam
}: {
  isOpen: boolean;
  onClose: () => void;
  transactionId?: string | null;
}) {
  const transactionId = transactionIdParam ? parseInt(String(transactionIdParam), 10) : 0;
  const { data: transaction, isLoading } = useTransaction(transactionId);

  if (isLoading || !transaction) {
    return null;
  }

  return (
    <ReverseTransactionModal
      isOpen={isOpen}
      onClose={onClose}
      transaction={transaction}
    />
  );
}

// Container component for TimesheetEntrySheet to handle success callback
function TimesheetEntrySheetContainer({
  isOpen,
  onClose
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();

  const handleSuccess = async () => {
    // Additional comprehensive cache invalidation as a safety net
    // This acts as a backup to the hook's invalidation
    // IMPORTANT: This function must complete before the sheet closes

    try {


      // Invalidate all timesheet-related queries and wait for completion
      await queryClient.invalidateQueries({
        predicate: (query) => {
          const key = query.queryKey;
          return Array.isArray(key) && key[0] === 'timesheets';
        },
        refetchType: 'active'
      });

      // Invalidate employee timesheet summaries and wait for completion
      await queryClient.invalidateQueries({
        predicate: (query) => {
          const key = query.queryKey;
          return Array.isArray(key) &&
                 key[0] === 'employees' &&
                 key.some(part => typeof part === 'string' && part.includes('timesheet'));
        },
        refetchType: 'active'
      });

      // Manual refetch for currently active timesheet list queries with Promise
      await new Promise<void>((resolve) => {
        setTimeout(async () => {
          await queryClient.refetchQueries({
            predicate: (query) => {
              const key = query.queryKey;
              return Array.isArray(key) &&
                     key[0] === 'timesheets' &&
                     key[1] === 'list' &&
                     query.getObserversCount() > 0;
            }
          });
          resolve();
        }, 100); // Slightly longer delay to ensure invalidation completes first
      });


    } catch (error) {
      console.error('❌ Cache invalidation failed:', error);
      // Don't throw - we still want the sheet to close even if cache invalidation fails
    }
  };

  return (
    <TimesheetEntrySheetWrapper
      isOpen={isOpen}
      onClose={onClose}
      onSuccess={handleSuccess}
    />
  );
}

// Default fallback for unknown or unauthorized modals
function DefaultModalFallback({ modalId, error, onClose }: { modalId: string; error?: string; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-card/80 backdrop-blur-sm">
      <div className="bg-card rounded-xl p-6 shadow-sm max-w-md">
        <h3 className="typography-title-large text-destructive mb-2">
          {error || 'Modal không tồn tại'}
        </h3>
        <p className="text-muted-foreground">
          {error ? error : `Không tìm thấy modal với ID: ${modalId}`}
        </p>
        <button
          className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-xl typography-body-medium"
          onClick={onClose}
        >
          Đóng
        </button>
      </div>
    </div>
  );
}

export function ModalRouter({ fallback: CustomFallback }: ModalRouterProps) {
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const updateEmployeeMutation = useUpdateEmployee();
  const createUserMutation = useCreateUser();

  // This ensures a remount only when modal or tab changes, not other params
  const modalParam = searchParams.get('modal');
  const tabParam = searchParams.get('tab');
  const instanceKey = `${modalParam || ''}:${tabParam || ''}`;
  const employeeId = searchParams.get('employeeId');
  const projectId = searchParams.get('projectId');
  const returnTo = searchParams.get('returnTo');
  const userRole = authManager.getUserRole();

  // Only fetch users data for admin users and when actually needed
  const shouldFetchUsers = userRole === 'admin' && modalParam === MODAL_IDS.ADD_USER;
  const { data: usersResponse } = useUsers(undefined, { enabled: shouldFetchUsers });


  // No modal requested
  if (!modalParam) {
    return null;
  }

  // Validate modal ID
  if (!isValidModalId(modalParam)) {
    console.warn(`Invalid modal ID: ${modalParam}`);
    const FallbackComponent = CustomFallback || DefaultModalFallback;
    return <FallbackComponent modalId={modalParam} error={`Modal ID không hợp lệ: ${modalParam}`} onClose={closeModal} />;
  }

  const modalId = modalParam as ModalId;
  const metadata = getModalMetadata(modalId);

  if (!metadata) {
    console.warn(`No metadata found for modal: ${modalId}`);
    const FallbackComponent = CustomFallback || DefaultModalFallback;
    return <FallbackComponent modalId={modalId} error={`Không tìm thấy thông tin modal: ${modalId}`} onClose={closeModal} />;
  }

  // Check authentication
  if (metadata.requiresAuth) {
    const userRole = authManager.getUserRole();

    if (!userRole) {
      const FallbackComponent = CustomFallback || DefaultModalFallback;
      return <FallbackComponent modalId={modalId} error="Cần đăng nhập để truy cập" onClose={closeModal} />;
    }

    // Check role permissions
    if (!metadata.roles.includes(userRole)) {
      const FallbackComponent = CustomFallback || DefaultModalFallback;
      return <FallbackComponent modalId={modalId} error="Không có quyền truy cập" onClose={closeModal} />;
    }
  }

  // Parse parameters
  const params: Record<string, string> = {};
  if (metadata.hasParams) {
    searchParams.forEach((value, key) => {
      if (key !== 'modal') {
        params[key] = value;
      }
    });
  }

  // Render modal component based on ID
  switch (modalId) {
    // User modals
    case MODAL_IDS.ADD_USER:
      return (
        <AddUserSheet
          key={instanceKey}
          isOpen={true}
          onClose={closeModal}
          onAddUser={async (userData) => {
            await createUserMutation.mutateAsync(userData);
          }}
          existingUsers={usersResponse?.data || []}
          loading={createUserMutation.isPending}
        />
      );
    case MODAL_IDS.USER_DETAILS:
      return <UserDetailsSheetContainer key={instanceKey} isOpen={true} onClose={closeModal} {...params} />;
    case MODAL_IDS.RESET_PASSWORD:
      return <ResetPasswordModalContainer key={instanceKey} isOpen={true} onClose={closeModal} {...params} />;
    case MODAL_IDS.USER_PROFILE:
      return <UserProfileSheet key={instanceKey} isOpen={true} onClose={closeModal} />;
    case MODAL_IDS.CHANGE_PASSWORD:
      return <ChangePasswordModal key={instanceKey} isOpen={true} onClose={closeModal} />;

    // Project modals
    case MODAL_IDS.ADD_PROJECT:
      return (
        <AddProjectSheetContainer
          isOpen={true}
          onClose={closeModal}
        />
      );
    case MODAL_IDS.PROJECT_DETAILS:
      return <ProjectDetailsSheet isOpen={true} onClose={closeModal} {...params} initialTab={params.tab} />;
    case MODAL_IDS.PROJECT_CREATE:
      return (
        <AddProjectSheetContainer
          isOpen={true}
          onClose={closeModal}
        />
      );
    case MODAL_IDS.PROJECT_EDIT:
      return <ProjectEditSheet isOpen={true} onClose={closeModal} project={null} {...params} />;
    case MODAL_IDS.PROJECT_ASSIGNMENT:
      return (
        <ProjectAssignmentSheetContainer
          key={instanceKey}
          isOpen={true}
          onClose={closeModal}
          onAssign={() => {}} // This is just for the interface, actual assignment is handled in container
          employeeId={employeeId}
          projectId={projectId}
          returnTo={returnTo}
        />
      );
    case MODAL_IDS.REMOVE_EMPLOYEE:
      return <RemoveEmployeeSheet isOpen={true} onClose={closeModal} onConfirm={() => {}} employee={null} {...params} />;
    case MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT:
      return (
        <AddEmployeeToProjectSheetContainer
          key={instanceKey}
          isOpen={true}
          onClose={closeModal}
          projectId={projectId}
        />
      );

    // Employee modals
    case MODAL_IDS.ADD_EMPLOYEE:
      return <AddEmployeeSheet isOpen={true} onClose={closeModal} />;
    case MODAL_IDS.EMPLOYEE_DETAILS:
      return <EmployeeDetailsSheet
        key={instanceKey}
        isOpen={true}
        onClose={closeModal}
        onUpdate={(employeeId, employeeData, bankObject) => {
          return updateEmployeeMutation.mutateAsync({
            id: employeeId,
            data: employeeData,
            bank: bankObject
          });
        }}
        {...params}
      />;

    // Timesheet modals
    case MODAL_IDS.TIMESHEET_DETAILS:
      return <TimesheetDetailsModal {...params} />;
    case MODAL_IDS.TIMESHEET_ENTRY:
      return <TimesheetEntrySheetContainer isOpen={true} onClose={closeModal} />;

    // General modals
    case MODAL_IDS.FILE_UPLOAD:
      return <FileUploadModal isOpen={true} onClose={closeModal} type="upload" title="Upload File" />;
    case MODAL_IDS.NOTIFICATION_SHEET:
      return <NotificationSheet variant="corporate" isOpen={true} onClose={closeModal} />;
    case MODAL_IDS.LEDGER_ENTRY_DETAILS:
      return <LedgerEntryDetailsSheet isOpen={true} onClose={closeModal} {...params} />;
    case MODAL_IDS.ADD_LEDGER_ENTRY:
      return <AddTransactionSheet isOpen={true} onClose={closeModal} />;
    case MODAL_IDS.APPROVAL_DETAILS:
      return <ApprovalDetailsModal {...params} />;

    // Transaction modals
    case MODAL_IDS.TRANSACTION_DETAILS:
      return <TransactionDetailsSheet isOpen={true} onClose={closeModal} />;
    case MODAL_IDS.SETTLE_TRANSACTION:
      return (
        <SettleTransactionModalContainer
          isOpen={true}
          onClose={closeModal}
          transactionId={params.id}
        />
      );
    case MODAL_IDS.REVERSE_TRANSACTION:
      return (
        <ReverseTransactionModalContainer
          isOpen={true}
          onClose={closeModal}
          transactionId={params.id}
        />
      );

    default:
      console.warn(`Modal component not implemented: ${modalId}`);
      const FallbackComponent = CustomFallback || DefaultModalFallback;
      return <FallbackComponent modalId={modalId} error="Modal chưa được triển khai" onClose={closeModal} />;
  }
}

// Export default and named
export default ModalRouter;

// Utility hook to get current modal state
export function useCurrentModal() {
  const [searchParams] = useSearchParams();
  const modalParam = searchParams.get('modal');

  return {
    isOpen: !!modalParam,
    modalId: modalParam as ModalId | null,
    isValid: modalParam ? isValidModalId(modalParam) : false,
    metadata: modalParam && isValidModalId(modalParam) ? getModalMetadata(modalParam) : null
  };
}
