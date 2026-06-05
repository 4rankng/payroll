import { useCallback } from 'react';
import { useModalStore, type ModalParams, type OpenModalOptions, type CloseModalOptions } from '@/lib/modal-state-manager';
import { getAllModalIds } from '@/lib/modal-registry-auto';

/**
 * Enhanced Modal Manager Hook
 * Uses centralized state management with full deep linking support
 */

export function useModalManager() {
  const {
    modalId,
    params,
    returnTo,
    isLoading,
    error,
    openModal: storeOpenModal,
    closeModal: storeCloseModal,
    updateParams,
    setLoading,
    setError
  } = useModalStore();

  const openModal = useCallback((
    modalId: string,
    params?: ModalParams,
    options?: OpenModalOptions
  ) => {
    return storeOpenModal(modalId, params, options);
  }, [storeOpenModal]);

  const closeModal = useCallback((options?: CloseModalOptions) => {
    storeCloseModal(options);
  }, [storeCloseModal]);

  const isModalOpen = useCallback((targetModalId?: string) => {
    if (targetModalId) {
      return modalId === targetModalId;
    }
    return modalId !== null;
  }, [modalId]);

  const getModalParam = useCallback(<T = unknown>(paramName: string, defaultValue?: T): T => {
    return (params[paramName] as T) ?? defaultValue;
  }, [params]);

  const setModalParam = useCallback((paramName: string, value: unknown) => {
    updateParams({ [paramName]: value as string });
  }, [updateParams]);

  return {
    // State
    currentModal: modalId,
    modalParams: params,
    returnChain: returnTo,
    isLoading,
    error,

    // Actions
    openModal,
    closeModal,
    updateParams,
    setLoading,
    setError,

    // Utilities
    isModalOpen,
    getModalParam,
    setModalParam,
    hasParent: !!returnTo
  };
}

/**
 * Domain-specific modal hooks for better developer experience
 */

// User Management Modals
export function useUserModals() {
  const { openModal } = useModalManager();

  return {
    openUserDetails: useCallback((userId: string | number, tab?: string) => {
      return openModal('user_details_sheet', { id: userId, ...(tab && { tab }) });
    }, [openModal]),

    openAddUser: useCallback(() => {
      return openModal('add_user_sheet');
    }, [openModal]),

    openEditUser: useCallback((userId: string | number) => {
      return openModal('edit_user_sheet', { id: userId });
    }, [openModal]),

    openResetPassword: useCallback((userId: string | number) => {
      return openModal('reset_password_modal', { userId });
    }, [openModal]),

    openUserProfile: useCallback(() => {
      return openModal('user_profile_sheet');
    }, [openModal]),

    openChangePassword: useCallback(() => {
      return openModal('change_password');
    }, [openModal])
  };
}

// Project Management Modals
export function useProjectModals() {
  const { openModal } = useModalManager();

  return {
    openProjectDetails: useCallback((projectId: string | number, tab?: string) => {
      return openModal('project_details_sheet', { id: projectId, ...(tab && { tab }) });
    }, [openModal]),

    openPartnerProjectDetails: useCallback((projectId: string | number, tab?: string) => {
      return openModal('project_details_sheet', { id: projectId, ...(tab && { tab }) });
    }, [openModal]),

    openProjectEdit: useCallback((projectId: string | number) => {
      return openModal('project-edit-sheet', { id: projectId });
    }, [openModal]),


    openAddProject: useCallback(() => {
      return openModal('add-project-sheet');
    }, [openModal]),


    openProjectAssignment: useCallback((projectId: string | number, employeeId?: string | number) => {
      return openModal('project_assignment_sheet', {
        projectId,
        ...(employeeId && { employeeId })
      });
    }, [openModal]),

    openAddEmployeeToProject: useCallback((projectId: string | number) => {
      return openModal('add_employee_to_project_sheet', { projectId });
    }, [openModal])
  };
}

// Employee Management Modals
export function useEmployeeModals() {
  const { openModal } = useModalManager();

  return {
    openEmployeeDetails: useCallback((employeeId: string | number, tab?: string) => {
      return openModal('employee_details', { id: employeeId, ...(tab && { tab }) });
    }, [openModal]),

    openAddEmployee: useCallback((projectId?: string | number) => {
      return openModal('add_employee_sheet', projectId ? { projectId } : {});
    }, [openModal])
  };
}

// Timesheet Management Modals
export function useTimesheetModals() {
  const { openModal } = useModalManager();

  return {
    openTimesheetDetails: useCallback((timesheetId: string | number, tab?: string) => {
      return openModal('timesheet_details_sheet', { id: timesheetId, ...(tab && { tab }) });
    }, [openModal]),

    openTimesheetEntry: useCallback((params?: {
      projectId?: string | number;
      employeeId?: string | number;
      date?: string;
      entryId?: string | number;
    }) => {
      return openModal('timesheet_entry_sheet', params || {});
    }, [openModal]),

    openSaturdayDayType: useCallback((date: string, projectId?: string | number) => {
      return openModal('saturday_day_type_modal', { date, ...(projectId && { projectId }) });
    }, [openModal])
  };
}

// Ledger & Reports Modals
export function useLedgerModals() {
  const { openModal } = useModalManager();

  return {
    openLedgerDetails: useCallback((entryId: string | number, tab?: string) => {
      return openModal('ledger_entry_details', { id: entryId, ...(tab && { tab }) });
    }, [openModal]),

    openAddLedgerEntry: useCallback(() => {
      return openModal('add_ledger_entry');
    }, [openModal]),

    openApprovalDetails: useCallback((approvalId: string | number, type?: string) => {
      return openModal('approval_details', { id: approvalId, ...(type && { type }) });
    }, [openModal])
  };
}

// General Purpose Modals
export function useGeneralModals() {
  const { openModal } = useModalManager();

  return {
    openFileUpload: useCallback(() => {
      return openModal('file_upload');
    }, [openModal]),

    openNotifications: useCallback(() => {
      return openModal('notification_sheet');
    }, [openModal])
  };
}

// Development helper
export function useModalRegistry() {
  const getAllIds = useCallback(async () => {
    return await getAllModalIds();
  }, []);

  return {
    getAllModalIds: getAllIds
  };
}