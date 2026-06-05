import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  MODAL_IDS,
  getModalMetadata,
  type ModalId
} from '@/constants/modalRegistry';
import { authManager } from '@/lib/auth';
import { toast } from '@/components/ui/sonner';
import {
  encodeReturnPath,
  getParentModal,
  buildModalSearchParams,
  extractCurrentModalState
} from '@/lib/modal-navigation';

/**
 * Type-Safe Modal Navigation Hook with returnTo chain support
 * Provides clean, domain-specific navigation methods for multi-level modals
 */

interface NavigationOptions {
  replace?: boolean;
  preserveSearch?: boolean;
  skipReturnTo?: boolean; // Skip creating returnTo chain (for root modals)
}

// Main navigation hook
export function useModalNavigation() {
  const navigate = useNavigate();

  // Core navigation function
  const openModal = useCallback((
    modalId: ModalId,
    params?: Record<string, string | number | boolean>,
    options: NavigationOptions = {}
  ) => {
    // Get modal metadata
    const metadata = getModalMetadata(modalId);
    if (!metadata) {
      console.error(`Modal metadata not found: ${modalId}`);
      toast({
        title: "Lỗi hệ thống",
        description: "Không tìm thấy thông tin modal.",
        variant: "destructive"
      });
      return;
    }

    // Check authentication
    if (metadata.requiresAuth) {
      const userRole = authManager.getUserRole();

      if (!userRole) {
        toast({
          title: "Cần đăng nhập",
          description: "Vui lòng đăng nhập để truy cập chức năng này.",
          variant: "destructive"
        });
        return;
      }

      // Check role permissions
      if (!metadata.roles.includes(userRole)) {
        toast({
          title: "Không có quyền truy cập",
          description: "Bạn không có quyền truy cập chức năng này.",
          variant: "destructive"
        });
        return;
      }
    }

    // Convert params to string format for URL
    const stringParams: Record<string, string> = {};
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          stringParams[key] = String(value);
        }
      });
    }

    // Handle returnTo chain unless explicitly skipped
    let returnTo: string | undefined;
    if (!options.skipReturnTo) {
      // Get current modal state
      const currentSearch = new URLSearchParams(window.location.search);
      const currentState = extractCurrentModalState(currentSearch);

      // Only create returnTo chain if the current modal is different from the target
      // This prevents circular chains (e.g. timesheet_entry -> timesheet_entry)
      if (currentState.modalId && currentState.modalId !== modalId) {
        // Create returnTo chain with current modal
        returnTo = encodeReturnPath(
          currentState.modalId,
          currentState.params,
          currentState.returnTo || undefined
        );
      }
    }

    // Preserve existing search params if requested
    if (options.preserveSearch) {
      const currentParams = new URLSearchParams(window.location.search);
      currentParams.forEach((value, key) => {
        if (key !== 'modal' && key !== 'returnTo' && !stringParams[key]) {
          stringParams[key] = value;
        }
      });
    }

    // Build search parameters
    const searchParams = buildModalSearchParams(modalId, stringParams, returnTo);

    // Navigate to open the modal
    const url = `${window.location.pathname}?${searchParams.toString()}`;
    navigate(url, { replace: options.replace ?? false });
  }, [navigate]);

  // Close current modal and navigate back using returnTo chain
  const closeModal = useCallback((options: NavigationOptions = {}) => {
    const currentSearch = new URLSearchParams(window.location.search);
    const returnTo = currentSearch.get('returnTo');

    if (returnTo) {
      // Navigate back to parent modal
      const parentModal = getParentModal(returnTo);

      if (parentModal) {
        // Open parent modal with remaining return chain
        const remainingChain = returnTo.split('>').slice(1).join('>');
        const searchParams = buildModalSearchParams(
          parentModal.modalId,
          parentModal.params,
          remainingChain || undefined
        );

        const url = `${window.location.pathname}?${searchParams.toString()}`;
        navigate(url, { replace: options.replace ?? false });
        return;
      }
    }

    // No returnTo chain, close all modals
    const currentParams = new URLSearchParams(window.location.search);

    // Only remove core modal navigation parameters, preserve others for page use
    const coreModalParams = ['modal', 'returnTo', 'tab', 'id'];
    coreModalParams.forEach(param => currentParams.delete(param));

    const newSearch = currentParams.toString();
    const url = `${window.location.pathname}${newSearch ? `?${newSearch}` : ''}`;
    navigate(url, { replace: options.replace ?? false });
  }, [navigate]);

  // Get navigation info for current modal
  const getNavigationInfo = useCallback(() => {
    const currentSearch = new URLSearchParams(window.location.search);
    const currentState = extractCurrentModalState(currentSearch);
    const hasParent = !!currentState.returnTo;
    const parentModal = currentState.returnTo ? getParentModal(currentState.returnTo) : null;

    return {
      currentModal: currentState.modalId,
      hasParent,
      parentModal: parentModal?.modalId || null,
      returnChain: currentState.returnTo || null
    };
  }, []);

  return { openModal, closeModal, getNavigationInfo };
}

// Domain-specific hooks for better DX
export function useUserModals() {
  const { openModal } = useModalNavigation();

  return {
    openUserDetails: useCallback((userId: string, tab?: 'details' | 'activity' | 'permissions') => {
      openModal(MODAL_IDS.USER_DETAILS, { id: userId, ...(tab && { tab }) });
    }, [openModal]),

    openAddUser: useCallback(() => {
      openModal(MODAL_IDS.ADD_USER);
    }, [openModal]),

    openResetPassword: useCallback((userId: string) => {
      openModal(MODAL_IDS.RESET_PASSWORD, { userId });
    }, [openModal]),

    openUserProfile: useCallback(() => {
      openModal(MODAL_IDS.USER_PROFILE);
    }, [openModal]),

    openChangePassword: useCallback(() => {
      openModal(MODAL_IDS.CHANGE_PASSWORD);
    }, [openModal])
  };
}

export function useProjectModals() {
  const { openModal } = useModalNavigation();

  return {
    openProjectDetails: useCallback((projectId: string, tab?: 'info' | 'employees' | 'payrates' | 'timesheet' | 'settings') => {
      openModal(MODAL_IDS.PROJECT_DETAILS, { id: projectId, ...(tab && { tab }) });
    }, [openModal]),

    openPartnerProjectDetails: useCallback((projectId: string, tab?: 'details' | 'employees' | 'timesheet' | 'settings') => {
      openModal(MODAL_IDS.PROJECT_DETAILS, { id: projectId, ...(tab && { tab }) });
    }, [openModal]),

    openProjectEdit: useCallback((projectId: string) => {
      openModal(MODAL_IDS.PROJECT_EDIT, { id: projectId });
    }, [openModal]),


    openProjectAssignment: useCallback((projectId?: string, employeeId?: string) => {
      openModal(MODAL_IDS.PROJECT_ASSIGNMENT, {
        ...(projectId && { projectId }),
        ...(employeeId && { employeeId })
      });
    }, [openModal]),

    openCreateProject: useCallback(() => {
      openModal(MODAL_IDS.PROJECT_CREATE);
    }, [openModal]),


    openAddProject: useCallback(() => {
      openModal(MODAL_IDS.ADD_PROJECT);
    }, [openModal]),

    openRemoveEmployee: useCallback((projectId: string, employeeId: string) => {
      openModal(MODAL_IDS.REMOVE_EMPLOYEE, { projectId, employeeId });
    }, [openModal]),

    openAddEmployeeToProject: useCallback((projectId: string) => {
      openModal(MODAL_IDS.ADD_EMPLOYEE_TO_PROJECT, { projectId });
    }, [openModal])
  };
}

export function useEmployeeModals() {
  const { openModal } = useModalNavigation();

  return {
    openEmployeeDetails: useCallback((employeeId: string, tab?: 'details' | 'projects' | 'timesheet' | 'payroll') => {
      openModal(MODAL_IDS.EMPLOYEE_DETAILS, { id: employeeId, ...(tab && { tab }) });
    }, [openModal]),

    openAddEmployee: useCallback((projectId?: string) => {
      openModal(MODAL_IDS.ADD_EMPLOYEE, projectId ? { projectId } : {});
    }, [openModal])
  };
}

export function useTimesheetModals() {
  const { openModal } = useModalNavigation();

  return {
    openTimesheetDetails: useCallback((timesheetId: string, tab?: 'details' | 'entries' | 'history') => {
      openModal(MODAL_IDS.TIMESHEET_DETAILS, { id: timesheetId, ...(tab && { tab }) });
    }, [openModal]),

    openTimesheetEntry: useCallback((params?: {
      projectId?: string;
      employeeId?: string;
      date?: string;
      entryId?: string
    }) => {
      openModal(MODAL_IDS.TIMESHEET_ENTRY, params || {});
    }, [openModal]),

    openSaturdayDayType: useCallback((date: string, projectId?: string) => {
      openModal('saturday_day_type' as unknown as Parameters<typeof openModal>[0], { date, ...(projectId && { projectId }) });
    }, [openModal])
  };
}

export function useGeneralModals() {
  const { openModal } = useModalNavigation();

  return {
    openFileUpload: useCallback(() => {
      openModal(MODAL_IDS.FILE_UPLOAD);
    }, [openModal]),

    openNotifications: useCallback(() => {
      openModal(MODAL_IDS.NOTIFICATION_SHEET);
    }, [openModal]),

    openLedgerDetails: useCallback((entryId: string, tab?: 'details' | 'entries' | 'adjustments') => {
      openModal(MODAL_IDS.LEDGER_ENTRY_DETAILS, { id: entryId, ...(tab && { tab }) });
    }, [openModal]),

    openAddLedgerEntry: useCallback(() => {
      openModal(MODAL_IDS.ADD_LEDGER_ENTRY);
    }, [openModal]),

    openApprovalDetails: useCallback((approvalId: string, type?: 'timesheet' | 'project' | 'employee') => {
      openModal(MODAL_IDS.APPROVAL_DETAILS, { id: approvalId, ...(type && { type }) });
    }, [openModal]),

    openTransactionDetails: useCallback((transactionId: string) => {
      openModal(MODAL_IDS.TRANSACTION_DETAILS, { id: transactionId });
    }, [openModal])
  };
}

// Development helper - get all available modal methods
export function useModalRegistry() {
  return {
    getAllModalIds: () => Object.values(MODAL_IDS),
    getModalMetadata,
    MODAL_IDS
  };
}
