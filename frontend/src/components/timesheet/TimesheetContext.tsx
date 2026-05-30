import { createContext, useContext, useMemo, type ReactNode } from 'react';
import type { Timesheet } from '@/types/api/timesheet.types';
import type { SortingState, OnChangeFn } from '@tanstack/react-table';
import type { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';

interface TimesheetState {
  timesheets: Timesheet[];
  isLoading: boolean;
  error: unknown;
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  sorting: SortingState;
  userRole: 'admin' | 'partner';
}

interface TimesheetActions {
  view: (t: Timesheet) => void;
  edit: (t: Timesheet) => void;
  approve: (t: Timesheet) => void;
  reject: (t: Timesheet, reason: string) => void;
  delete: (t: Timesheet) => void;
  canDelete: (t: Timesheet) => boolean;
  exportExcel: () => void;
  requestEdit: ((t: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void) | undefined;
  pageChange: (page: number) => void;
  pageSizeChange: (size: number) => void;
  sortingChange: OnChangeFn<SortingState>;
}

interface TimesheetMeta {
  requestingTimesheetId: number | null;
  bulkTransferPercentage: number;
  isExportLoading: boolean;
}

interface TimesheetFilterState {
  selectedMonth: string;
  onMonthChange: (v: string) => void;
  selectedProject: string;
  onProjectChange: (v: string) => void;
  selectedEmployee: string;
  onEmployeeChange: (v: string) => void;
  statusFilter: Timesheet['status'] | Timesheet['payment_status'] | 'all' | 'pending_payment';
  onStatusChange: (v: Timesheet['status'] | Timesheet['payment_status'] | 'all' | 'pending_payment') => void;
  projects: Array<{ id: number; code: string; name: string }>;
  projectEmployees?: Array<{ id: number; name: string; subtitle?: string; cccd?: string; fullname?: string; current_projects?: Array<{ project_id: number }> }>;
  searchTerm?: string;
  onSearchChange?: (v: string) => void;
  userRole: 'admin' | 'partner';
  onAddTimesheet?: () => void;
  onExportExcel?: () => void;
  onPaymentHistory?: () => void;
  isExportLoading?: boolean;
}

export interface TimesheetContextValue {
  state: TimesheetState;
  actions: TimesheetActions;
  meta: TimesheetMeta;
  filters: TimesheetFilterState;
}

const TimesheetContext = createContext<TimesheetContextValue | null>(null);

interface TimesheetProviderProps {
  management: ReturnType<typeof useTimesheetManagement>;
  onEdit: (t: Timesheet) => void;
  onApprove?: (t: Timesheet) => void;
  onDelete?: (t: Timesheet) => void;
  onAddTimesheet?: () => void;
  onRequestEdit?: (t: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void;
  requestingTimesheetId?: number | null;
  onBulkApprove?: () => void;
  bulkTransferPercentage?: number;
  showEditRequestTable?: boolean;
  userRole?: 'admin' | 'partner';
  onEditRequestRowClick?: (t: Timesheet) => void;
  onExportExcel?: () => void;
  onPaymentHistory?: () => void;
  isExportLoading?: boolean;
  children: ReactNode;
}

export function TimesheetProvider({
  management,
  onEdit,
  onApprove,
  onDelete,
  userRole = 'admin',
  onRequestEdit,
  requestingTimesheetId = null,
  bulkTransferPercentage = 0,
  onExportExcel,
  onPaymentHistory,
  isExportLoading = false,
  children,
}: TimesheetProviderProps) {
  const value = useMemo<TimesheetContextValue>(() => {
    const handleApprove = userRole === 'partner' ? () => {} : (onApprove || management.handleApprove);
    const handleReject = userRole === 'partner' ? () => {} : management.handleReject;
    const handleDelete = onDelete || management.handleDelete;

    return {
    state: {
      timesheets: management.timesheets,
      isLoading: management.isLoading,
      error: management.error,
      pagination: management.paginationInfo,
      sorting: management.sorting,
      userRole,
    },
    actions: {
      view: management.handleView,
      edit: onEdit,
      approve: handleApprove,
      reject: handleReject,
      delete: handleDelete,
      canDelete: management.canDeleteTimesheet,
      exportExcel: management.handleExportExcel,
      requestEdit: onRequestEdit,
      pageChange: management.handlePageChange,
      pageSizeChange: management.handlePageSizeChange,
      sortingChange: management.setSorting,
    },
    meta: {
      requestingTimesheetId,
      bulkTransferPercentage,
      isExportLoading,
    },
    filters: {
      selectedMonth: management.selectedMonth,
      onMonthChange: management.setSelectedMonth,
      selectedProject: management.selectedProject,
      onProjectChange: management.setSelectedProject,
      selectedEmployee: management.selectedEmployee,
      onEmployeeChange: management.setSelectedEmployee,
      statusFilter: management.statusFilter,
      onStatusChange: management.setStatusFilter,
      projects: management.projects,
      projectEmployees: management.projectEmployees,
      searchTerm: management.searchTerm,
      onSearchChange: management.setSearchTerm,
      userRole,
      onAddTimesheet: undefined,
      onExportExcel,
      onPaymentHistory,
      isExportLoading,
    },
  };}, [
    management.timesheets, management.isLoading, management.error,
    management.paginationInfo, management.sorting, management.handleView,
    management.canDeleteTimesheet, management.handleExportExcel,
    management.handlePageChange, management.handlePageSizeChange, management.setSorting,
    management.selectedMonth, management.setSelectedMonth,
    management.selectedProject, management.setSelectedProject,
    management.selectedEmployee, management.setSelectedEmployee,
    management.statusFilter, management.setStatusFilter,
    management.projects, management.projectEmployees, management.searchTerm,
    management.setSearchTerm,
    onEdit, onRequestEdit, userRole,
    requestingTimesheetId, bulkTransferPercentage, isExportLoading,
    onExportExcel, onPaymentHistory,
    onApprove, onDelete, management.handleApprove, management.handleReject, management.handleDelete,
  ]);

  return (
    <TimesheetContext.Provider value={value}>
      {children}
    </TimesheetContext.Provider>
  );
}

export function useTimesheetContext(): TimesheetContextValue {
  const ctx = useContext(TimesheetContext);
  if (!ctx) {
    throw new Error('useTimesheetContext must be used within a TimesheetProvider');
  }
  return ctx;
}
