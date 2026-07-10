/**
 * Advance Payment Service
 * Handles API calls for the Advance Payment (Ứng lương tháng) feature
 */

import { apiClient, buildQueryString, type ApiResponse } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  AdvancePaymentInfo,
  AdvancePaymentHistoryItem,
  AdvancePaymentListItem,
  AdvancePaymentFilters,
  AdvancePaymentSummary,
  AdvancePaymentFileHistoryItem,
  CreateAdvancePaymentRequest,
  CalculateFeeRequest,
  CalculateFeeResponse,
  UploadBankResultResponse,
  ImportJobResponse,
  ImportJobStatusResponse,
  ImportPollingOptions,
  FlexPayEmployeeListItem,
  FlexPayEmployeeFilters,
  FlexPayEmployeeListMeta,
  AvailableMonth,
} from "@/types/api/advance-payment.types";
import type { EmployeeImportResponse, EmployeeImportStatus } from "@/types/api/employee.types";
import { employeeService } from "./employee.service";

class AdvancePaymentService {
  // ============================================================================
  // Employee Endpoints
  // ============================================================================

  /**
   * Get current month's advance payment info for employee
   * Returns: max advance, completed, pending, remaining amounts
   */
  async getAdvancePaymentInfo(): Promise<ApiResponse<AdvancePaymentInfo>> {
    return apiClient.get<AdvancePaymentInfo>(
      API_ENDPOINTS.employee.advancePayment,
    );
  }

  /**
   * Get self-check-in advance info (dedicated /me/check-in-advance path for
   * check-in-enabled employees). Same AdvancePaymentInfo shape, plus salary
   * (100% earned), disclaimer, and windowOpenDay.
   */
  async getCheckInAdvanceInfo(): Promise<ApiResponse<AdvancePaymentInfo>> {
    return apiClient.get<AdvancePaymentInfo>(
      API_ENDPOINTS.employee.checkInAdvance,
    );
  }

  /**
   * Submit advance payment request for employee
   */
  async requestAdvancePayment(
    data: CreateAdvancePaymentRequest,
  ): Promise<ApiResponse<AdvancePaymentHistoryItem>> {
    return apiClient.post<AdvancePaymentHistoryItem>(
      API_ENDPOINTS.employee.advancePaymentRequest,
      data,
    );
  }

  /**
   * Submit an advance request under the self-check-in flow
   * (dedicated /me/check-in-advance/request path).
   */
  async requestCheckInAdvance(
    data: CreateAdvancePaymentRequest,
  ): Promise<ApiResponse<AdvancePaymentHistoryItem>> {
    return apiClient.post<AdvancePaymentHistoryItem>(
      API_ENDPOINTS.employee.checkInAdvanceRequest,
      data,
    );
  }

  /**
   * Get advance payment history for employee (paginated, optional month filter).
   * `forMonth` (`yyyy-MM`) groups by salary period (preferred — an advance is
   * charged to a period, which can differ from the calendar month it was sent in).
   * `fromDate`/`toDate` are a legacy created_at fallback.
   * API returns: { data: AdvancePaymentHistoryItem[], pagination: {...}, status, message }
   */
  async getAdvancePaymentHistory(filters?: {
    page?: number;
    pageSize?: number;
    /** `yyyy-MM` — salary period to group by (preferred over the date range). */
    forMonth?: string;
    /** `yyyy-MM-dd` — inclusive lower bound on created_at (legacy fallback). */
    fromDate?: string;
    /** `yyyy-MM-dd` — inclusive upper bound on created_at (legacy fallback). */
    toDate?: string;
  }): Promise<ApiResponse<AdvancePaymentHistoryItem[]>> {
    const queryString = filters ? buildQueryString(filters) : "";
    return apiClient.get<AdvancePaymentHistoryItem[]>(
      `${API_ENDPOINTS.employee.advancePaymentHistory}${queryString}`,
    );
  }

  /**
   * Calculate fee for advance payment amount
   * Returns: fee and netAmount
   */
  async calculateFee(data: CalculateFeeRequest): Promise<CalculateFeeResponse> {
    const response = await apiClient.post<CalculateFeeResponse>(
      API_ENDPOINTS.employee.advancePaymentCalculateFee,
      data,
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Cancel a pending advance payment request
   * Only PENDING requests can be cancelled
   */
  async cancelAdvancePaymentRequest(
    id: number,
  ): Promise<ApiResponse<AdvancePaymentHistoryItem>> {
    return apiClient.post<AdvancePaymentHistoryItem>(
      API_ENDPOINTS.employee.advancePaymentCancelRequest(id),
    );
  }

  // ============================================================================
  // Admin Endpoints
  // ============================================================================

  /**
   * Get paginated list of advance payment requests
   */
  async getAdvancePayments(
    filters?: AdvancePaymentFilters,
  ): Promise<ApiResponse<AdvancePaymentListItem[]>> {
    const queryString = filters
      ? buildQueryString(filters as Record<string, unknown>)
      : "";
    return apiClient.get<AdvancePaymentListItem[]>(
      `${API_ENDPOINTS.advancePayments.base}${queryString}`,
    );
  }

  /**
   * Get advance payment summary statistics
   */
  async getAdvancePaymentSummary(params?: {
    fromDate?: string;
    toDate?: string;
    forMonth?: string;
  }): Promise<ApiResponse<AdvancePaymentSummary>> {
    const queryString = params ? buildQueryString(params) : "";
    return apiClient.get<AdvancePaymentSummary>(
      `${API_ENDPOINTS.advancePayments.summary}${queryString}`,
    );
  }

  /**
   * Export pending requests to Excel
   * Triggers browser download
   */
  async exportAdvancePayments(params?: {
    fromDate?: string;
    toDate?: string;
  }): Promise<void> {
    const queryString = params ? buildQueryString(params) : "";
    const endpoint = `${API_ENDPOINTS.advancePayments.export}${queryString}`;
    await apiClient.download(
      endpoint,
      `ung_luong_thang_${new Date().toISOString().split("T")[0]}.xlsx`,
    );
  }

  /**
   * Upload bank transfer result file
   * Updates request statuses based on bank transfer results
   */
  async uploadBankResult(
    formData: FormData,
  ): Promise<ApiResponse<UploadBankResultResponse>> {
    return apiClient.upload<UploadBankResultResponse>(
      API_ENDPOINTS.advancePayments.uploadResult,
      formData,
    );
  }

  /**
   * Import Flexible Payroll Template (Async)
   * Starts an async import job and returns the job ID
   */
  async importFlexTemplate(
    formData: FormData,
  ): Promise<ApiResponse<ImportJobResponse>> {
    return apiClient.upload<ImportJobResponse>(
      API_ENDPOINTS.advancePayments.import,
      formData,
    );
  }

  /**
   * Get import job status
   * Returns the current status and result of an import job
   */
  async getImportJobStatus(
    jobId: number,
  ): Promise<ApiResponse<ImportJobStatusResponse>> {
    return apiClient.get<ImportJobStatusResponse>(
      API_ENDPOINTS.advancePayments.importStatus(jobId),
    );
  }

  /**
   * Import Flexible Payroll Template with polling
   * Starts an async import job and waits for completion
   */
  async importFlexTemplateWithPolling(
    formData: FormData,
    options: ImportPollingOptions = {},
  ): Promise<ImportJobStatusResponse> {
    const {
      maxWaitTime = 300000, // 5 minutes
      pollInterval = 1000, // 1 second
      onProgress,
      onStatusChange,
    } = options;

    // Start the import job
    const startResponse = await this.importFlexTemplate(formData);
    if (!startResponse.data) {
      throw new Error("Failed to start import job");
    }

    const jobId = startResponse.data.id;
    const startTime = Date.now();

    // Poll for status
    while (Date.now() - startTime < maxWaitTime) {
      const statusResponse = await this.getImportJobStatus(jobId);
      if (!statusResponse.data) {
        throw new Error("Failed to get import job status");
      }

      const { status, processedRows, totalRows, percentage, error } =
        statusResponse.data;

      // Notify status change
      if (onStatusChange) {
        onStatusChange(status);
      }

      // Notify progress with percentage from backend
      if (onProgress) {
        onProgress(percentage, processedRows, totalRows);
      }

      // Check completion
      if (status === "completed") {
        return statusResponse.data;
      }

      if (status === "failed") {
        throw new Error(error || "Import failed");
      }

      // Wait before next poll
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    throw new Error("Import timed out");
  }

  /**
   * Get uploaded file history
   * Returns list of previously uploaded files (templates and results)
   */
  async getFileHistory(): Promise<
    ApiResponse<AdvancePaymentFileHistoryItem[]>
  > {
    return apiClient.get<AdvancePaymentFileHistoryItem[]>(
      API_ENDPOINTS.advancePayments.files,
    );
  }

  /**
   * Download an uploaded file by asset ID
   */
  async downloadUploadedFile(id: number, filename: string): Promise<void> {
    await apiClient.download(
      API_ENDPOINTS.advancePayments.fileDownload(id),
      filename,
    );
  }

  /**
   * Cancel an advance payment request (admin)
   * Only PENDING requests can be cancelled
   */
  async cancelAdvancePayment(
    id: number,
  ): Promise<ApiResponse<AdvancePaymentListItem>> {
    return apiClient.post<AdvancePaymentListItem>(
      API_ENDPOINTS.advancePayments.cancel(id),
    );
  }

  /**
   * Retry disbursement for an APPROVED or FAILED advance payment request (admin)
   * Re-enqueues a disbursement task to the payment provider
   */
  async retryDisbursement(
    id: number,
  ): Promise<ApiResponse<{ requestId: number; disbursementTaskId: string; message: string }>> {
    return apiClient.post<{
      requestId: number; disbursementTaskId: string; message: string
    }>(
      API_ENDPOINTS.advancePayments.retryDisbursement(id),
    );
  }

  /**
   * Get disbursement status for an advance payment request (admin)
   * Used for polling progress after retry
   */
  async getDisbursementStatus(
    id: number,
  ): Promise<ApiResponse<{ requestId: number; requestStatus: string; isTerminal: boolean; paymentRef?: string | null; paidAt?: string | null }>> {
    return apiClient.get<{
      requestId: number; requestStatus: string; isTerminal: boolean; paymentRef?: string | null; paidAt?: string | null
    }>(
      API_ENDPOINTS.advancePayments.disbursementStatus(id),
    );
  }

  /**
   * Get flex pay employee list
   * Returns employees from the latest imported flex pay month
   */
  async getFlexPayEmployees(
    filters?: FlexPayEmployeeFilters,
  ): Promise<ApiResponse<FlexPayEmployeeListItem[]>> {
    const queryString = filters
      ? buildQueryString(filters as Record<string, unknown>)
      : "";
    return apiClient.get<FlexPayEmployeeListItem[]>(
      `${API_ENDPOINTS.advancePayments.employees}${queryString}`,
    );
  }

  /**
   * Export flex pay employee list to Excel
   * Downloads Excel file with columns: STT, Ho va ten, CCCD, Du an, Ten dang nhap, Ngay tao
   */
  async exportFlexPayEmployees(params?: {
    forMonth?: string;
  }): Promise<void> {
    const queryString = params ? buildQueryString(params) : "";
    const endpoint = `${API_ENDPOINTS.advancePayments.exportEmployees}${queryString}`;
    await apiClient.download(
      endpoint,
      `danh_sach_nhan_vien_${params?.forMonth ?? new Date().toISOString().split("T")[0]}.xlsx`,
    );
  }

  /**
   * Get available months with flex pay data
   * Returns list of months that have flex pay data imported
   */
  async getAvailableMonths(): Promise<ApiResponse<AvailableMonth[]>> {
    return apiClient.get<AvailableMonth[]>(
      API_ENDPOINTS.advancePayments.availableMonths,
    );
  }

  
  /**
   * Send reconciliation email
   * Sends email with reconciliation report for a specific month
   */
  async sendReconciliationEmail(data: {
    forMonth: string;
    recipients: string[];
    cc?: string[];
  }): Promise<ApiResponse> {
    return apiClient.post(API_ENDPOINTS.advancePayments.reconciliation.sendEmail, data);
  }

  /**
   * Upload and settle reconciliation file
   * Processes uploaded settlement results for advance payments
   */
  async uploadAndSettleReconciliation(formData: FormData): Promise<ApiResponse> {
    return apiClient.upload(
      API_ENDPOINTS.advancePayments.reconciliation.settle,
      formData
    );
  }

  /**
   * Download transfer history file (original uploaded Excel)
   */
  async downloadTransferHistoryFile(id: number): Promise<void> {
    await apiClient.download(
      API_ENDPOINTS.advancePayments.transferHistoryDownload(id),
      `ket_qua_chuyen_tien_${id}.xlsx`,
    );
  }

  /**
   * Import flexible employee list (async)
   * Uploads an Excel file containing employees under flexible payment schedule
   * Returns import_id for tracking progress via status endpoint
   */
  async importFlexibleEmployeeList(
    formData: FormData,
  ): Promise<ApiResponse<EmployeeImportResponse>> {
    // Extract file from FormData
    const file = formData.get('file') as File;
    if (!file) {
      throw new Error('No file provided');
    }

    // Use the new async employee import endpoint
    // Use upload() method to properly set Content-Type: multipart/form-data
    const response = await apiClient.upload<EmployeeImportResponse>(
      API_ENDPOINTS.employees.import,
      formData,
    );

    return response;
  }

  /**
   * Import flexible employee list with polling
   * Starts an async import job and waits for completion
   */
  async importFlexibleEmployeeListWithPolling(
    formData: FormData,
    options: ImportPollingOptions = {},
  ): Promise<EmployeeImportStatus> {
    const {
      maxWaitTime = 300000, // 5 minutes
      pollInterval = 1000, // 1 second
      onProgress,
      onStatusChange,
    } = options;

    // Start the import job
    const startResponse = await this.importFlexibleEmployeeList(formData);
    if (!startResponse.data?.import_id) {
      throw new Error("Failed to start import job");
    }

    const importId = startResponse.data.import_id;
    const startTime = Date.now();

    // Poll for status
    while (Date.now() - startTime < maxWaitTime) {
      const status = await employeeService.getImportStatus(importId);

      // Notify status change
      if (onStatusChange) {
        onStatusChange(status.status);
      }

      // Notify progress with percentage from backend
      if (onProgress) {
        onProgress(status.percentage, status.processed_rows, status.total_rows);
      }

      // Check completion
      if (status.status === "completed") {
        return status;
      }

      if (status.status === "failed") {
        throw new Error("Import failed");
      }

      // Wait before next poll
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    throw new Error("Import timed out");
  }
}

export const advancePaymentService = new AdvancePaymentService();
