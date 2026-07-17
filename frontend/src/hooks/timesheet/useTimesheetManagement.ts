import { useState, useMemo } from 'react';
import { SortingState } from '@tanstack/react-table';
import { Timesheet, TimesheetFilters } from '@/types/api/timesheet.types';
import {
  useGroupedTimesheets,
  useBulkApproveTimesheets,
  useBulkRejectTimesheets,
  useApproveTimesheet,
  useRejectTimesheet,
  useDeleteTimesheet,
  useExportTimesheets,
  useExportTimesheetsExcel
} from '@/hooks/api/useTimesheets';
import { useProjects, useProjectEmployeesSimple } from '@/hooks/api/useProjects';
import { useEmployees } from '@/hooks/api/useEmployees';
import { useTimesheetModals } from '@/hooks/useModalNavigation';
import { buildTimesheetStatusFilters, TimesheetStatusFilter } from '@/utils/timesheetFilterHelpers';

interface TimesheetManagementConfig {
  userRole?: 'admin' | 'partner';
  useYearToDate?: boolean;
  extraFilters?: Partial<TimesheetFilters>;
}

export function useTimesheetManagement(config: TimesheetManagementConfig = {}) {
  const { userRole = 'admin', useYearToDate = false, extraFilters } = config;
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedMonth, setSelectedMonth] = useState(() => {
    const today = new Date();
    return `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`;
  });
  const [selectedProject, setSelectedProject] = useState('all');
  const [selectedEmployee, setSelectedEmployee] = useState('all');
  const [statusFilter, setStatusFilter] = useState<TimesheetStatusFilter>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sorting, setSorting] = useState<SortingState>([]);

  // Column ID to API field mapping
  const SORT_FIELD_MAP: Record<string, string> = useMemo(() => ({
    employeeName: 'employee_name',
    projectName: 'project_name',
    date: 'date',
    paytype: 'paytype',
    hours_payrate: 'hours_worked',
    payment_summary: 'amount',
    status: 'status',
    created_at: 'created_at',
    created_by: 'created_by',
  }), []);

  const { openTimesheetDetails } = useTimesheetModals();

  // Calculate year-to-date range for partner view
  const yearToDateRange = useMemo(() => {
    const currentDate = new Date();
    const currentYear = currentDate.getFullYear();
    return {
      fromDate: `${currentYear}-01-01`,
      toDate: currentDate.toISOString().split('T')[0]
    };
  }, []);

  // Build filters for API
  const filters: TimesheetFilters = useMemo(() => {
    const apiFilters: TimesheetFilters = {
      // Pagination and sorting
      page: currentPage,
      pageSize: pageSize,
      ...(sorting.length > 0
        ? {
            sortBy: SORT_FIELD_MAP[sorting[0].id] ?? sorting[0].id,
            sortOrder: (sorting[0].desc ? 'desc' : 'asc') as 'desc' | 'asc',
          }
        : {
            sortBy: userRole === 'partner' ? 'date' : 'created_at',
            sortOrder: 'desc' as const,
          }),
    };

    // Apply search filter if provided
    if (searchTerm.trim()) {
      apiFilters.search = searchTerm.trim();
    }

    // Apply month filter if selected (not for "all") - takes priority over year-to-date
    if (selectedMonth !== 'all') {
      apiFilters.fromDate = `${selectedMonth}-01`;
      const lastDay = new Date(parseInt(selectedMonth.split('-')[0]), parseInt(selectedMonth.split('-')[1]), 0).getDate();
      apiFilters.toDate = `${selectedMonth}-${lastDay.toString().padStart(2, '0')}`;
    }
    // For partner role, use year-to-date if useYearToDate is true and no specific month is selected
    else if (useYearToDate && userRole === 'partner') {
      apiFilters.fromDate = yearToDateRange.fromDate;
      apiFilters.toDate = yearToDateRange.toDate;
    }

    // Apply project filter if selected
    if (selectedProject !== 'all') {
      const projectId = parseInt(selectedProject);
      if (!isNaN(projectId)) {
        apiFilters.project_ids = projectId.toString();
      }
    }

    // Apply employee filter if selected
    if (selectedEmployee !== 'all') {
      const employeeId = parseInt(selectedEmployee);
      if (!isNaN(employeeId)) {
        apiFilters.employee_id = employeeId;
      }
    }

    Object.assign(apiFilters, buildTimesheetStatusFilters(statusFilter));

    return extraFilters
      ? { ...apiFilters, ...extraFilters }
      : apiFilters;
  }, [searchTerm, selectedMonth, selectedProject, selectedEmployee, statusFilter, currentPage, pageSize, sorting, userRole, useYearToDate, yearToDateRange, extraFilters, SORT_FIELD_MAP]);


  // Fetch data — both admin and partner use the grouped endpoint for employee-based pagination
  const groupedQuery = useGroupedTimesheets(filters);

  const timesheetResponse = groupedQuery.data;
  const isLoading = groupedQuery.isLoading;
  const error = groupedQuery.error;

  const { data: projectsData } = useProjects();

  // Fetch all employees for calendar view
  const { data: employeesData } = useEmployees({
    page: 1,
    pageSize: 1000,
    status: 'working',
    sortBy: 'fullname',
    sortOrder: 'asc'
  });

  // Fetch project-specific employees when a project is selected (needed for project filtering logic)
  const selectedProjectId = selectedProject !== 'all' ? parseInt(selectedProject) : null;
  const { data: projectEmployeesData } = useProjectEmployeesSimple(
    selectedProjectId || 0,
    !!selectedProjectId
  );

  // API mutations
  const bulkApproveMutation = useBulkApproveTimesheets();
  const bulkRejectMutation = useBulkRejectTimesheets();
  const approveMutation = useApproveTimesheet();
  const rejectMutation = useRejectTimesheet();
  const deleteMutation = useDeleteTimesheet();
  const exportMutation = useExportTimesheets();
  const exportExcelMutation = useExportTimesheetsExcel();

  // Extract timesheets array and pagination from the API response
  const { rawTimesheets, paginationInfo: apiPaginationInfo } = useMemo(() => {
    if (!timesheetResponse) {
      return { rawTimesheets: [], paginationInfo: null };
    }

    // Handle grouped response structure: { status, data: { groups: [] }, pagination }
    if (timesheetResponse.status === 'success' && timesheetResponse.data?.groups) {
      const groups = timesheetResponse.data.groups;
      const flattened: Timesheet[] = groups.flatMap(group => group.entries);
      const pagination = timesheetResponse.pagination || null;

      return {
        rawTimesheets: flattened,
        paginationInfo: pagination
      };
    }

    // Handle regular response structure: { status, data: [], pagination }
    if (timesheetResponse.status === 'success' && timesheetResponse.data) {
      const data = Array.isArray(timesheetResponse.data) ? timesheetResponse.data : [];
      const pagination = timesheetResponse.pagination || null;

      return {
        rawTimesheets: data,
        paginationInfo: pagination
      };
    }

    // Handle case where response is directly an array (no pagination from backend)
    if (Array.isArray(timesheetResponse)) {
      return { rawTimesheets: timesheetResponse, paginationInfo: null };
    }

    return { rawTimesheets: [], paginationInfo: null };
  }, [timesheetResponse]);

  const allProjects = useMemo(() => projectsData?.data || [], [projectsData?.data]);

  // Get all employees for calendar view
  const employees = useMemo(() => {
    if (!employeesData?.data) return [];
    return employeesData.data.map(emp => ({
      id: emp.id,
      name: emp.fullname,
      subtitle: emp.code
    }));
  }, [employeesData?.data]);

  // Timesheets — backend now handles approved+not-paid filtering via payment_status param
  const timesheets = rawTimesheets;

  // Calculate pagination info based on API response
  const paginationInfo = useMemo(() => {
    if (apiPaginationInfo) {
      return {
        page: apiPaginationInfo.page || currentPage,
        pageSize: apiPaginationInfo.pageSize || pageSize,
        totalPages: apiPaginationInfo.totalPages,
        totalRecords: apiPaginationInfo.totalRecords
      };
    }
    return {
      page: currentPage,
      pageSize: pageSize,
      totalPages: 1,
      totalRecords: rawTimesheets.length
    };
  }, [apiPaginationInfo, rawTimesheets, currentPage, pageSize]);

  // Get pending timesheets from the main response - filter for pending_approval status
  const allPendingTimesheets = useMemo(() => {
    // Filter rawTimesheets for pending approval status
    return rawTimesheets.filter((timesheet: Timesheet) => timesheet.status === 'pending_approval');
  }, [rawTimesheets]);

  // Use pagination info for summary stats
  const summary = useMemo(() => {
    return {
      totalEntries: paginationInfo?.totalRecords || timesheets.length,
      pendingApproval: allPendingTimesheets.length, // Filter from main response
      approvedEntries: 0, // Would need separate API call for accurate counts
      rejectedEntries: 0  // Would need separate API call for accurate counts
    };
  }, [paginationInfo, timesheets.length, allPendingTimesheets.length]);

  // Pending timesheets — backend handles search filtering
  const pendingTimesheets = useMemo(() => {
    return allPendingTimesheets;
  }, [allPendingTimesheets]);

  // Get project-specific pending timesheets when a project is selected
  const projectPendingTimesheets = useMemo(() => {
    if (selectedProject === 'all') return [];
    const projectId = parseInt(selectedProject);
    if (isNaN(projectId)) return [];

    return pendingTimesheets.filter((t: Timesheet) => t.project_id === projectId);
  }, [pendingTimesheets, selectedProject]);

  // Get selected project details
  const selectedProjectDetails = useMemo(() => {
    if (selectedProject === 'all') return null;
    const projectId = parseInt(selectedProject);
    if (isNaN(projectId)) return null;

    return allProjects.find(p => p.id === projectId);
  }, [allProjects, selectedProject]);

  // Get all projects for filter dropdown
  const projects = useMemo(() => {
    return allProjects.map(p => ({ id: p.id, code: p.code, name: p.name }));
  }, [allProjects]);

  // Get project-specific employees for filter dropdown (needed for TimesheetFilters)
  const projectEmployees = useMemo(() => {
    // If a project is selected, use project-specific employees
    if (selectedProject !== 'all' && projectEmployeesData) {
      return projectEmployeesData;
    }

    // Otherwise return empty array - EmployeeDropdownAdapter will handle fetching all employees
    return [];
  }, [selectedProject, projectEmployeesData]);

  // Get current month for comparison
  const currentMonth = useMemo(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  }, []);

  const hasFilters = searchTerm !== '' || selectedMonth !== currentMonth || selectedProject !== 'all' || selectedEmployee !== 'all' || statusFilter !== 'all';

  const clearFilters = () => {
    setSearchTerm('');
    setSelectedMonth(currentMonth);
    setSelectedProject('all');
    setSelectedEmployee('all');
    setStatusFilter('all');
    setCurrentPage(1); // Reset to first page when clearing filters
  };

  // Enhanced filter setters that reset page to 1
  const setSearchTermWithReset = (term: string) => {
    setSearchTerm(term);
    setCurrentPage(1);
  };

  const setSelectedMonthWithReset = (month: string) => {
    setSelectedMonth(month);
    setCurrentPage(1);
  };

  const setSelectedProjectWithReset = (project: string) => {
    setSelectedProject(project);

    // Reset employee selection when project changes
    // This ensures that the employee filter is compatible with the new project
    if (selectedEmployee !== 'all') {
      setSelectedEmployee('all');
    }

    setCurrentPage(1);
  };

  const setSelectedEmployeeWithReset = (employee: string) => {
    setSelectedEmployee(employee);
    setCurrentPage(1);
  };

  const setStatusFilterWithReset = (status: TimesheetStatusFilter) => {
    setStatusFilter(status);
    setCurrentPage(1);
  };


  // Pagination handlers
  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    setPageSize(newPageSize);
    setCurrentPage(1); // Reset to first page when changing page size
  };

  // Real API actions
  const handleBulkApprove = async (ids: number[]) => {
    await bulkApproveMutation.mutateAsync({ timesheet_ids: ids });
    // Cache invalidation is handled by the mutation hook
  };

  const handleBulkReject = async (ids: number[], rejectionReason: string) => {
    await bulkRejectMutation.mutateAsync({
      timesheet_ids: ids,
      rejection_reason: rejectionReason,
      notify_partners: true
    });
    // Cache invalidation is handled by the mutation hook
  };

  const handleProjectBulkApprove = async () => {
    if (projectPendingTimesheets.length === 0) return;

    const ids = projectPendingTimesheets.map(t => t.id);
    await handleBulkApprove(ids);
  };

  const handleProjectBulkReject = async (rejectionReason: string) => {
    if (projectPendingTimesheets.length === 0) return;

    const ids = projectPendingTimesheets.map(t => t.id);
    await handleBulkReject(ids, rejectionReason);
  };

  const handleView = (timesheet: Timesheet) => {
    openTimesheetDetails(timesheet.id.toString());
  };

  const handleEdit = (timesheet: Timesheet) => {
    // This is handled by parent component
  };

  const handleApprove = async (timesheet: Timesheet) => {
    await approveMutation.mutateAsync(timesheet.id);
    // Cache invalidation is handled by the mutation hook
  };

  const handleReject = async (timesheet: Timesheet, rejectionReason: string) => {
    await rejectMutation.mutateAsync({
      id: timesheet.id,
      data: { rejection_reason: rejectionReason }
    });
    // Cache invalidation is handled by the mutation hook
  };

  const handleDelete = async (timesheet: Timesheet) => {
    // Check if this timesheet can be deleted based on business rules
    if (!canDeleteTimesheet(timesheet)) {
      return;
    }
    await deleteMutation.mutateAsync(timesheet.id);
    // Cache invalidation is handled by the mutation hook
  };

  const handleExport = async () => {
    await exportMutation.mutateAsync(filters);
  };

  const handleExportExcel = async () => {
    const defaultDateRange = useYearToDate && userRole === 'partner' ? yearToDateRange : {
      fromDate: (() => {
        const now = new Date();
        return `${now.getFullYear()}-01-01`;
      })(),
      toDate: (() => {
        const now = new Date();
        return now.toISOString().split('T')[0];
      })()
    };

    const exportParams: {
      fromDate: string;
      toDate: string;
      project_ids?: string;
      employee_id?: number;
    } = {
      fromDate: filters.fromDate || defaultDateRange.fromDate,
      toDate: filters.toDate || defaultDateRange.toDate,
    };

    if (filters.project_ids) {
      exportParams.project_ids = filters.project_ids;
    }

    if (filters.employee_id && typeof filters.employee_id === 'number') {
      exportParams.employee_id = filters.employee_id;
    }

    await exportExcelMutation.mutateAsync(exportParams);
  };

  const handleImport = () => {
    // This should open import modal
  };

  const canDeleteTimesheet = (timesheet: Timesheet) => {
    // Admin can delete if payment_status is pending
    if (config.userRole === 'admin' && timesheet.payment_status === 'pending') {
      return true;
    }

    // For all users, can delete if status is pending_approval
    return timesheet.status === 'pending_approval';
  };

  return {
    // Data
    timesheets,
    summary,
    projects,
    projectEmployees,
    employees,
    pendingTimesheets,
    projectPendingTimesheets,
    selectedProjectDetails,
    error,
    isLoading: isLoading || bulkApproveMutation.isPending || bulkRejectMutation.isPending || approveMutation.isPending || rejectMutation.isPending || deleteMutation.isPending || exportExcelMutation.isPending,

    // Sorting
    sorting,
    setSorting,

    // Pagination
    paginationInfo,
    currentPage,
    pageSize,

    // Filters
    searchTerm,
    selectedMonth,
    selectedProject,
    selectedEmployee,
    statusFilter,
    hasFilters,

    // Filter actions
    setSearchTerm: setSearchTermWithReset,
    setSelectedMonth: setSelectedMonthWithReset,
    setSelectedProject: setSelectedProjectWithReset,
    setSelectedEmployee: setSelectedEmployeeWithReset,
    setStatusFilter: setStatusFilterWithReset,
    clearFilters,

    // Pagination actions
    handlePageChange,
    handlePageSizeChange,

    // Actions
    handleBulkApprove,
    handleBulkReject,
    handleProjectBulkApprove,
    handleProjectBulkReject,
    handleView,
    handleEdit,
    handleApprove,
    handleReject,
    handleDelete,
    handleExport,
    handleExportExcel,
    handleImport,

    // Utility functions
    canDeleteTimesheet
  };
}
