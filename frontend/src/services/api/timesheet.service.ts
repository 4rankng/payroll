import { apiClient, buildQueryString, ApiResponse } from './client';
import type { PartnerImportFile, PartnerImportListParams } from '@/types/api/timesheet.types';
import { API_ENDPOINTS, API_CONFIG } from '@/config/api.config';
import { authManager } from '@/lib/auth';
import {
  canApproveTimesheet
} from '@/lib/permissions';
import { extractFilenameFromHeaders } from '@/utils/file-download';
import type {
  Timesheet,
  TimesheetSummary,
  EmployeeTimesheetSummary,
  UpdateTimesheetData,
  TimesheetFilters,
  BulkApproveData,
  BulkApproveResult,
  BulkResetData,
  BulkResetResult,
  RejectTimesheetData,
  ImportTimesheetResult,
  ExportTimesheetResult,
  NewTimesheetEntry,
  BulkCreateTimesheetResult,
  TimesheetListResponse,
  TimesheetEditRequestFilters,
  TimesheetEditRequestListResponse,
  TimesheetEditRequestResponse,
  UploadEntriesExcelResult,
  SettlementUploadResultResponse,
  EmployeeEntryTable,
  ListGroupedTimesheetsResponse,
} from '@/types/api/timesheet.types';

class TimesheetService {
  /**
   * Get timesheet summary statistics
   */
  async getSummary(params?: {
    project_id?: number;
    employee_id?: number;
    fromDate?: string;
    toDate?: string;
  }): Promise<TimesheetSummary> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<TimesheetSummary>(
      `${API_ENDPOINTS.timesheets.summary}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get paginated list of timesheets
   */
  async getTimesheets(filters?: TimesheetFilters): Promise<TimesheetListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    // The backend returns the response structure directly, so we use unknown type
    // and then cast it to the expected TimesheetListResponse
    const response = await apiClient.get<unknown>(
      `${API_ENDPOINTS.timesheets.base}${queryString}`
    );

    // The API client wraps the response in ApiResponse format
    // Return the entire response which includes status, data, and pagination
    return response as TimesheetListResponse;
  }

  /**
   * Get paginated list of timesheets grouped by employee (for partner view)
   * Uses server-side grouping to ensure consistent pagination by employee count
   */
  async getGroupedTimesheets(filters?: TimesheetFilters): Promise<ListGroupedTimesheetsResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<unknown>(
      `${API_ENDPOINTS.timesheets.grouped}${queryString}`
    );
    return response as ListGroupedTimesheetsResponse;
  }

  /**
   * Get single timesheet by ID
   */
  async getTimesheetById(id: number): Promise<Timesheet> {
    const response = await apiClient.get<Timesheet>(
      API_ENDPOINTS.timesheets.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get timesheets by project and date
   */
  async getTimesheetsByProjectAndDate(projectId: number, date: string) {
    const queryString = buildQueryString({ date });
    const response = await apiClient.get<Timesheet[]>(
      `${API_ENDPOINTS.timesheets.byProjectAndDate(projectId)}${queryString}`
    );
    return response;
  }

  /**
   * Create new timesheet entry
   */
  async createTimesheet(data: {
    projectId: number;
    employeeId: number;
    date: string;
    hoursWorked: number;
    paytype: string;
    description?: string;
  }): Promise<Timesheet> {
    const response = await apiClient.post<Timesheet>(
      API_ENDPOINTS.timesheets.base,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create timesheet entries using new simplified format (UPSERT behavior)
   * Accepts array of individual timesheet entries
   * Note: For same day and same hourType, values will be updated instead of creating new entries
   */
  async createTimesheets(entries: NewTimesheetEntry[]): Promise<BulkCreateTimesheetResult> {
    const response = await apiClient.post<{
      created: Timesheet[];
      failed: Array<{ employeeId: number; error: string; details?: string; }> | null;
      total_created: number;
      total_failed: number;
      total_success?: number;
    }>(
      API_ENDPOINTS.timesheets.base,
      entries
    );

    // Extract count from message if structured data is not available
    let totalCreated = 0;
    let totalFailed = 0;

    if (response.data && typeof response.data === 'object') {
      // Use structured data if available
      // Backend returns 'total_success' not 'total_created'
      totalCreated = response.data.total_success || response.data.total_created || 0;
      totalFailed = response.data.total_failed || 0;
    }

    // Always try to parse from message as backup or primary method
    const message = response.message || '';

    // Handle Vietnamese message format: "Tạo hàng loạt bảng chấm công thành công: 3"
    // Handle English formats: "Successfully created 3 records", "Created 3 timesheets"
    // Handle generic: "thành công: 3", "created: 3", "successful: 3"

    // Try multiple patterns to catch the count
    let messageMatch = message.match(/thành công\s*:\s*(\d+)/i);
    if (!messageMatch) {
      messageMatch = message.match(/(?:created|successful)\s*:\s*(\d+)/i);
    }
    if (!messageMatch) {
      // Fallback: Look for any number after "thành công" or "created"
      messageMatch = message.match(/thành công[^0-9]*(\d+)/i);
    }
    if (!messageMatch) {
      // Final fallback: Look for any number at the end of the message
      messageMatch = message.match(/(\d+)(?!.*\d)/);
    }

    if (messageMatch) {
      const parsedCount = parseInt(messageMatch[1], 10);
      if (!isNaN(parsedCount)) {
        // Use message count if it's greater than structured data or if structured data is 0
        totalCreated = Math.max(totalCreated, parsedCount);
      }
    }

    // Create proper data structure for the notification
    const data = {
      total_created: totalCreated,
      total_failed: totalFailed,
      created: response.data?.created || [],
      failed: response.data?.failed || null
    };

    // Transform the API response to match our BulkCreateTimesheetResult format
    return {
      status: response.status,
      message: response.message || '',
      data
    };
  }

  /**
   * Update existing timesheet
   *
   * Edit eligibility (paid status, role/approval rules) is enforced by the backend
   * (paid check + CanBeEditedByUser), which returns localized messages surfaced via
   * the global React Query error handler. We intentionally do NOT pre-fetch via
   * getTimesheetById: that endpoint returns 403 for partners on some timesheets,
   * which would abort the PUT and leave the UI showing stale cached data.
   */
  async updateTimesheet(id: number, data: UpdateTimesheetData): Promise<Timesheet> {
    const response = await apiClient.put<Timesheet>(
      API_ENDPOINTS.timesheets.byId(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete timesheet (admin/partner)
   *
   * Deletion eligibility (paid status; partner may only delete pending/rejected) is
   * enforced by the backend DELETE handler, which returns localized messages surfaced
   * via the global React Query error handler. We intentionally do NOT pre-fetch via
   * getTimesheetById: that endpoint returns 403 for partners on some timesheets, which
   * aborts the DELETE and skips cache invalidation — leaving the deleted row visible.
   */
  async deleteTimesheet(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.timesheets.byId(id));
  }

  /**
   * Approve timesheet entry
   */
  async approveTimesheet(id: number): Promise<Timesheet> {
    // Validate approval permissions
    if (!canApproveTimesheet()) {
      throw new Error('Chỉ Admin mới có thể phê duyệt timesheet');
    }

    const response = await apiClient.put<Timesheet>(
      API_ENDPOINTS.timesheets.approve(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Reject timesheet entry
   */
  async rejectTimesheet(id: number, data: RejectTimesheetData): Promise<Timesheet> {
    // Validate rejection permissions
    if (!canApproveTimesheet()) {
      throw new Error('Chỉ Admin mới có thể loại timesheet');
    }

    const response = await apiClient.put<Timesheet>(
      API_ENDPOINTS.timesheets.reject(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Manually add timesheet to next payroll run (Admin only)
   */
  async addToPayroll(id: number): Promise<{ status: string; message?: string }> {
    if (!canApproveTimesheet()) {
      throw new Error('Chỉ Admin mới có thể thêm timesheet vào kỳ lương');
    }

    const response = await apiClient.post(
      API_ENDPOINTS.timesheets.addToPayroll(id)
    );
    return { status: response.status as string, message: response.message };
  }

  /**
   * Bulk approve timesheets
   */
  async bulkApprove(data: BulkApproveData): Promise<BulkApproveResult> {
    const response = await apiClient.post<BulkApproveResult>(
      API_ENDPOINTS.timesheets.bulkApprove,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Approve all pending timesheets
   */
  async approveAll(): Promise<BulkApproveResult> {
    // Validate approval permissions
    if (!canApproveTimesheet()) {
      throw new Error('Chỉ Admin mới có thể phê duyệt timesheet');
    }

    const response = await apiClient.post<BulkApproveResult>(
      API_ENDPOINTS.timesheets.approveAll
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Bulk reset timesheets (Admin only)
   * Resets approved/rejected timesheets back to pending_approval status
   */
  async bulkReset(data: BulkResetData): Promise<BulkResetResult> {
    if (!canApproveTimesheet()) {
      throw new Error('Chỉ Admin mới có thể đặt lại trạng thái timesheet');
    }

    const response = await apiClient.post<BulkResetResult>(
      API_ENDPOINTS.timesheets.bulkReset,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Import timesheets from Excel
   */
  async importTimesheets(
    file: File,
    projectId?: number,
    onProgress?: (progress: number) => void
  ): Promise<ImportTimesheetResult> {
    const formData = new FormData();
    formData.append('file', file);
    if (projectId) {
      formData.append('project_id', projectId.toString());
    }

    const response = await apiClient.upload<ImportTimesheetResult>(
      API_ENDPOINTS.timesheets.import,
      formData,
      onProgress
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Import settlement result from payroll report
   */
  async uploadSettlementResult(file: File): Promise<SettlementUploadResultResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.post<SettlementUploadResultResponse>(
      API_ENDPOINTS.timesheets.uploadSettlementResult,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );

    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Export timesheets to Excel
   */
  async exportTimesheets(filters?: TimesheetFilters): Promise<ExportTimesheetResult> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<ExportTimesheetResult>(
      `${API_ENDPOINTS.timesheets.export}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Export timesheets to Excel with date range (direct download)
   */
  async exportTimesheetsExcel(params: {
    fromDate: string;
    toDate: string;
    project_ids?: string;
    employeeId?: number; // Using camelCase to match API
    status?: string;
  }): Promise<void> {
    const queryString = buildQueryString(params);
    const downloadUrl = `${API_ENDPOINTS.timesheets.export}${queryString}`;
    await apiClient.download(downloadUrl, 'timesheets_export.xlsx');
  }

  /**
   * Export payroll report for a specific report date (direct download)
   */
  async exportPayrollReport(params: {
    atDate: string;
  }): Promise<void> {
    const queryString = buildQueryString({ atDate: params.atDate });
    const downloadUrl = `${API_ENDPOINTS.timesheets.payrollReport}${queryString}`;
    await apiClient.download(downloadUrl, 'payroll_report.xlsx');
  }

  /**
   * Send payroll report via email
   */
  async sendPayrollReportEmail(params: {
    reportAtDate: string;
    recipients: string[];
    cc?: string[];
    bcc?: string[];
  }): Promise<{ status: string; message: string }> {
    const response = await apiClient.post<{ status: string; message: string }>(
      API_ENDPOINTS.timesheets.emailPayrollReport,
      params
    );

    return response.data || { status: response.status, message: response.message || '' };
  }

  /**
   * Download exported timesheet file
   */
  async downloadExport(downloadUrl: string, filename: string): Promise<void> {
    await apiClient.download(downloadUrl, filename);
  }

  /**
   * Validate timesheet entry with 3-level payrate structure
   */
  async validateEntry(data: {
    employee_id: number;
    project_id: number;
    date: string;
    position: string;
    hour_type: string;
    hours_worked: number;
    exclude_id?: number;
  }): Promise<{
    valid: boolean;
    errors: string[];
    warnings?: string[];
    calculated_rate?: number;
    calculated_amount?: number;
    calculated_day_type?: string;
  }> {
    try {
      const response = await apiClient.post(
        `${API_ENDPOINTS.timesheets.base}/validate`,
        data
      );
      const defaultResponse = {
        valid: false,
        errors: ['Validation failed']
      };

      if (response.data && typeof response.data === 'object') {
        return { ...defaultResponse, ...response.data };
      }

      return defaultResponse;
    } catch (error: unknown) {
      const apiError = error as { response?: { data?: { message?: string } } };
      return {
        valid: false,
        errors: [apiError?.response?.data?.message || 'Validation failed']
      };
    }
  }

  /**
   * Get employee timesheets using the employee-specific endpoint
   */
  async getEmployeeTimesheets(
    employeeId: number,
    filters?: {
      page?: number;
      pageSize?: number;
      fromDate?: string;
      toDate?: string;
      sortBy?: string;
      sortOrder?: 'asc' | 'desc';
    }
  ) {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.timesheet(employeeId)}${queryString}`
    );
    return response;
  }

  /**
   * Get employee timesheet summary
   */
  async getEmployeeTimesheetSummary(
    employeeId: number,
    params?: {
      project_id?: number;
      fromDate?: string;
      toDate?: string;
    }
  ): Promise<EmployeeTimesheetSummary> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<EmployeeTimesheetSummary>(
      `${API_ENDPOINTS.employees.timesheetSummary(employeeId)}${queryString}`
    );
    return response.data!;
  }

  /**
   * Get approval queue for timesheets
   */
  async getApprovalQueue(params?: {
    page?: number;
    pageSize?: number;
    project_id?: number;
    priority?: 'high' | 'normal' | 'low';
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
  }) {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.timesheets.base}/approval-queue${queryString}`
    );
    return response;
  }

  /**
   * Bulk reject multiple timesheets
   */
  async bulkReject(data: {
    timesheet_ids: number[];
    rejection_reason: string;
    notify_partners?: boolean;
  }): Promise<{
    rejected_count: number;
    failed: number;
    results: Array<{
      id: number;
      status: string;
      rejection_reason?: string;
      error?: string;
    }>;
    notifications_sent?: number;
  }> {
    const response = await apiClient.post<{
      rejected_count: number;
      failed: number;
      results: Array<{
        id: number;
        status: string;
        rejection_reason?: string;
        error?: string;
      }>;
      notifications_sent?: number;
    }>(
      API_ENDPOINTS.timesheets.bulkReject,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get detailed calculation breakdown for a timesheet
   */
  async getCalculationDetails(id: number) {
    const response = await apiClient.get(
      `${API_ENDPOINTS.timesheets.byId(id)}/calculation-details`
    );
    return response.data;
  }

  /**
   * Bulk approve timesheets by project
   */
  async bulkApproveByProject(projectId: number, data: {
    approvalNote?: string;
  }): Promise<BulkApproveResult> {
    const response = await apiClient.post<BulkApproveResult>(
      API_ENDPOINTS.timesheets.approveByProject(projectId),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Bulk approve timesheets by employee
   */
  async bulkApproveByEmployee(employeeId: number, data: {
    timesheet_ids: number[];
    approvalNote?: string;
  }): Promise<BulkApproveResult> {
    const response = await apiClient.post<BulkApproveResult>(
      API_ENDPOINTS.timesheets.approveByEmployee(employeeId),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create edit request for approved timesheet (Partner)
   */
  async createEditRequest(timesheetId: number): Promise<TimesheetEditRequestResponse> {
    const response = await apiClient.post<TimesheetEditRequestResponse>(
      API_ENDPOINTS.timesheets.requestEdit(timesheetId),
      {}
    );
    return response.data!;
  }

  /**
   * Get list of edit requests with filters
   */
  async getEditRequests(filters?: TimesheetEditRequestFilters): Promise<TimesheetEditRequestListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<TimesheetEditRequestListResponse>(
      `${API_ENDPOINTS.timesheets.editRequests}${queryString}`
    );
    return response.data!;
  }

  /**
   * Get single edit request by ID
   */
  async getEditRequestById(id: number): Promise<TimesheetEditRequestResponse> {
    const response = await apiClient.get<TimesheetEditRequestResponse>(
      API_ENDPOINTS.timesheets.editRequestById(id)
    );
    return response.data!;
  }

  /**
   * Approve edit request (Admin)
   */
  async approveEditRequest(id: number): Promise<TimesheetEditRequestResponse> {
    const response = await apiClient.put<TimesheetEditRequestResponse>(
      API_ENDPOINTS.timesheets.approveEditRequest(id),
      {}
    );
    return response.data!;
  }

  /**
   * Reject edit request (Admin)
   */
  async rejectEditRequest(id: number): Promise<TimesheetEditRequestResponse> {
    const response = await apiClient.put<TimesheetEditRequestResponse>(
      API_ENDPOINTS.timesheets.rejectEditRequest(id),
      {}
    );
    return response.data!;
  }

  /**
   * Cancel edit request (Partner)
   */
  async cancelEditRequest(timesheetId: number): Promise<ApiResponse<null>> {
    const response = await apiClient.post<ApiResponse<null>>(
      API_ENDPOINTS.timesheets.cancelEditRequest(timesheetId),
      {}
    );
    return response as ApiResponse<null>;
  }

  /**
   * Export timesheet entries template (direct download)
   * Generates an Excel template for entering timesheet entries for a specific project and date range
   */
  async exportEntriesTemplate(params: {
    fromDate: string;
    toDate: string;
    projectId: number;
  }): Promise<void> {
    await apiClient.downloadPost(
      `${API_ENDPOINTS.timesheets.base}/export-entries-template`,
      params,
      'timesheet_entries_template.xlsx'
    );
  }

  /**
   * Preview timesheet entries before creation
   * Performs dry-run validation without creating actual entries
   */
  async previewTimesheets(entries: NewTimesheetEntry[]): Promise<{
    errors?: Array<{ employeeId: number; date?: string; message: string }> | null;
  }> {
    const response = await apiClient.post<{ errors?: Array<{ employeeId: number; date?: string; message: string }> | null }>(
      API_ENDPOINTS.timesheets.preview,
      entries
    );
    return { errors: response.data?.errors ?? null };
  }

  /**
   * Upload timesheet entries from Excel file
   */
  async uploadEntriesExcel(
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<UploadEntriesExcelResult> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.upload<UploadEntriesExcelResult>(
      API_ENDPOINTS.timesheets.uploadEntriesExcel,
      formData,
      onProgress
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get timesheet entry table data
   */
  async getTimesheetEntryTable(
    projectId: number,
    params: {
      fromDate: string;
      toDate: string;
      employeeIDs?: number[];
    }
  ): Promise<EmployeeEntryTable[]> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<EmployeeEntryTable[]>(
      `${API_ENDPOINTS.projects.base}/${projectId}/timesheet-entry-table${queryString}`
    );
    return response.data || [];
  }

  // ─── BCC Partner Import ─────────────────────────────────────────────────

  async uploadBCCFile(file: File, projectId: number, forMonth: string): Promise<PartnerImportFile> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('project_id', String(projectId));
    formData.append('for_month', forMonth);
    const response = await apiClient.post<PartnerImportFile>(
      API_ENDPOINTS.timesheets.partnerImport,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    if (!response.data) throw new Error(response.message || 'Upload thất bại');
    return response.data;
  }

  async listPartnerImports(params?: PartnerImportListParams): Promise<ApiResponse<PartnerImportFile[]>> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<PartnerImportFile[]>(
      `${API_ENDPOINTS.timesheets.partnerImport}${queryString}`
    );
    return response;
  }

  async getPartnerImport(id: number): Promise<PartnerImportFile> {
    const response = await apiClient.get<PartnerImportFile>(
      API_ENDPOINTS.timesheets.partnerImportById(id)
    );
    if (!response.data) throw new Error('Không tìm thấy import');
    return response.data;
  }

  async downloadPartnerImport(id: number, filename?: string): Promise<void> {
    await apiClient.download(
      API_ENDPOINTS.timesheets.partnerImportDownload(id),
      filename || 'bcc_import.xlsx'
    );
  }
}

export const timesheetService = new TimesheetService();
