import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  ImportSummaryResponse,
  ImportSummaryParams,
  ImportListResponse,
  ImportListParams,
  ImportBatchResponse,
  ImportCreateResponse,
  ImportCancelResponse,
  ImportRetryResponse,
  ImportTemplateResponse,
  ExportCreateResponse,
  ExportDataRequest,
  ExportJobResponse,
  ExportListResponse,
  ExportListParams,
  FileValidationResponse,
  FileValidationRequest,
  BatchStatusResponse,
  BulkRetryResponse,
  BulkRetryRequest,
  ImportType,
  ExportType,
  TemplateType
} from '@/types/api/import-export.types';

/**
 * Import/Export Management API Service
 * Handles bulk data operations, batch tracking, file validation, and export generation
 */
export class ImportExportService {

  // ========== IMPORT OPERATIONS ==========

  /**
   * Get import summary statistics
   */
  async getImportSummary(params: ImportSummaryParams = {}): Promise<ImportSummaryResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<ImportSummaryResponse>(`${API_ENDPOINTS.imports.summary}${queryString}`);
    return response.data!;
  }

  /**
   * Get paginated list of import batches
   */
  async getImportBatches(params: ImportListParams = {}): Promise<ImportListResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<ImportListResponse>(`${API_ENDPOINTS.imports.base}${queryString}`);
    return response.data!;
  }

  /**
   * Get detailed information about an import batch
   */
  async getImportBatch(batchId: string): Promise<ImportBatchResponse> {
    const response = await apiClient.get<ImportBatchResponse>(API_ENDPOINTS.imports.byBatchId(batchId));
    return response.data!;
  }

  /**
   * Import employee data from Excel file
   * @param file Excel file containing employee data
   */
  async importEmployees(file: File): Promise<ImportCreateResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.upload<ImportCreateResponse>(API_ENDPOINTS.imports.employees, formData);
    return response.data!;
  }

  /**
   * Import timesheet data from Excel file
   * @param file Excel file containing timesheet data
   * @param projectId Project ID for the timesheets
   */
  async importTimesheets(file: File, projectId: number): Promise<ImportCreateResponse> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('project_id', projectId.toString());

    const response = await apiClient.upload<ImportCreateResponse>(API_ENDPOINTS.imports.timesheets, formData);
    return response.data!;
  }


  /**
   * Cancel a processing import batch
   */
  async cancelImport(batchId: string): Promise<ImportCancelResponse> {
    const response = await apiClient.post<ImportCancelResponse>(API_ENDPOINTS.imports.cancel(batchId));
    return response.data!;
  }

  /**
   * Retry processing failed records from an import batch
   */
  async retryImport(batchId: string): Promise<ImportRetryResponse> {
    const response = await apiClient.post<ImportRetryResponse>(API_ENDPOINTS.imports.retry(batchId));
    return response.data!;
  }

  /**
   * Bulk retry multiple failed import batches
   */
  async bulkRetryImports(data: BulkRetryRequest): Promise<BulkRetryResponse> {
    const response = await apiClient.post<BulkRetryResponse>(`${API_ENDPOINTS.imports.base}/bulk-retry`, data);
    return response.data!;
  }

  /**
   * Download import template for specific type
   */
  async getImportTemplate(templateType: TemplateType): Promise<ImportTemplateResponse> {
    const response = await apiClient.get<ImportTemplateResponse>(API_ENDPOINTS.imports.templates(templateType));
    return response.data!;
  }

  /**
   * Download template file
   */
  async downloadTemplate(templateType: TemplateType, filename?: string): Promise<void> {
    const response = await this.getImportTemplate(templateType);
    return apiClient.download(response.data.downloadUrl, filename || response.data.filename);
  }

  /**
   * Validate import file before processing
   */
  async validateImportFile(
    file: File,
    importType: ImportType,
    projectId?: number
  ): Promise<FileValidationResponse> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('import_type', importType);
    if (projectId) {
      formData.append('project_id', projectId.toString());
    }

    const response = await apiClient.upload<FileValidationResponse>(`${API_ENDPOINTS.imports.base}/validate-file`, formData);
    return response.data!;
  }

  /**
   * Get real-time batch processing status
   */
  async getBatchStatus(batchId: string): Promise<BatchStatusResponse> {
    const response = await apiClient.get<BatchStatusResponse>(`${API_ENDPOINTS.imports.base}/batch-status-tracking`, {
      params: { batch_id: batchId }
    });
    return response.data!;
  }

  /**
   * Get error report for failed import batch
   */
  async getImportErrorReport(batchId: string): Promise<{
    data: {
      batchId: string;
      errorReportUrl: string;
      filename: string;
      totalErrors: number;
      generatedAt: string;
      expiresAt: string;
    };
  }> {
    const response = await apiClient.get(`${API_ENDPOINTS.imports.byBatchId(batchId)}/error-report`);
    return response.data! as {
      data: {
        batchId: string;
        errorReportUrl: string;
        filename: string;
        totalErrors: number;
        generatedAt: string;
        expiresAt: string;
      };
    };
  }

  // ========== EXPORT OPERATIONS ==========

  /**
   * Create export job for specific data type
   */
  async createExport(
    exportType: ExportType,
    data: ExportDataRequest
  ): Promise<ExportCreateResponse> {
    const response = await apiClient.post<ExportCreateResponse>(API_ENDPOINTS.exports.byType(exportType), data);
    return response.data!;
  }

  /**
   * Get export job status and details
   */
  async getExportJob(exportId: string): Promise<ExportJobResponse> {
    const response = await apiClient.get<ExportJobResponse>(API_ENDPOINTS.exports.byId(exportId));
    return response.data!;
  }

  /**
   * Get paginated list of export jobs
   */
  async getExportHistory(params: ExportListParams = {}): Promise<ExportListResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<ExportListResponse>(`${API_ENDPOINTS.exports.base}${queryString}`);
    return response.data!;
  }

  /**
   * Download completed export file
   */
  async downloadExport(exportId: string, filename?: string): Promise<void> {
    return apiClient.download(API_ENDPOINTS.exports.download(exportId), filename);
  }

  /**
   * Cancel processing export job
   */
  async cancelExport(exportId: string): Promise<{
    data: {
      exportId: string;
      status: 'cancelled';
      cancelledAt: string;
    };
  }> {
    const response = await apiClient.post(`${API_ENDPOINTS.exports.byId(exportId)}/cancel`);
    return response.data! as {
      data: {
        exportId: string;
        status: 'cancelled';
        cancelledAt: string;
      };
    };
  }

  // ========== CONVENIENCE METHODS ==========

  /**
   * Export employees with common filters
   */
  async exportEmployees(filters: {
    project_id?: number;
    status?: string;
    fromDate?: string;
    toDate?: string;
  } = {}, format: 'excel' | 'csv' = 'excel'): Promise<ExportCreateResponse> {
    return this.createExport('employees', {
      filters,
      format,
      includeHeaders: true
    });
  }

  /**
   * Export timesheets with common filters
   */
  async exportTimesheets(filters: {
    project_id?: number;
    employee_id?: number;
    fromDate?: string;
    toDate?: string;
    status?: string;
  } = {}, format: 'excel' | 'csv' = 'excel'): Promise<ExportCreateResponse> {
    return this.createExport('timesheets', {
      filters,
      format,
      includeHeaders: true,
      columns: ['employee_name', 'date', 'hours_worked', 'paytype', 'amount', 'status']
    });
  }


  /**
   * Export financial data with common filters
   */
  async exportFinancial(filters: {
    account_type?: string;
    fromDate?: string;
    toDate?: string;
    project_id?: number;
  } = {}, format: 'excel' | 'csv' = 'excel'): Promise<ExportCreateResponse> {
    return this.createExport('financial', {
      filters,
      format,
      includeHeaders: true
    });
  }

  /**
   * Export bulk bank transfer file with UTC datetime
   */
  async exportBulkBankTransfer(filters: {
    project_id?: number[];
    employee_id?: number[];
    fromDate: string;
    toDate: string;
  }): Promise<ExportCreateResponse> {
    // Ensure dates are in UTC format
    const utcFilters = {
      ...filters,
      fromDate: new Date(filters.fromDate).toISOString(),
      toDate: new Date(filters.toDate).toISOString(),
    };

    const response = await apiClient.post<ExportCreateResponse>(API_ENDPOINTS.payrolls.exportBulkTransfer, {
      filters: utcFilters,
      format: 'excel',
      includeHeaders: true,
      timezone: 'UTC'
    });
    return response.data!;
  }

  /**
   * Wait for export completion with polling
   */
  async waitForExportCompletion(
    exportId: string,
    maxWaitTime: number = 300000, // 5 minutes
    pollInterval: number = 2000 // 2 seconds
  ): Promise<ExportJobResponse> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitTime) {
      const response = await this.getExportJob(exportId);

      if (response.data.status === 'completed') {
        return response;
      }

      if (response.data.status === 'failed') {
        throw new Error(`Export failed: ${exportId}`);
      }

      // Wait before polling again
      await new Promise(resolve => setTimeout(resolve, pollInterval));
    }

    throw new Error(`Export timeout: ${exportId}`);
  }

  /**
   * Wait for import completion with polling
   */
  async waitForImportCompletion(
    batchId: string,
    maxWaitTime: number = 300000, // 5 minutes
    pollInterval: number = 2000 // 2 seconds
  ): Promise<ImportBatchResponse> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitTime) {
      const response = await this.getImportBatch(batchId);

      if (response.data.status === 'completed') {
        return response;
      }

      if (response.data.status === 'failed') {
        throw new Error(`Import failed: ${batchId}`);
      }

      // Wait before polling again
      await new Promise(resolve => setTimeout(resolve, pollInterval));
    }

    throw new Error(`Import timeout: ${batchId}`);
  }
}

// Export singleton instance
export const importExportService = new ImportExportService();
