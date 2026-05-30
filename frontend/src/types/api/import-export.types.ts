// Import/Export Management API Types

// Base types
export interface ImportSummary {
  totalImports: number;
  successfulImports: number;
  failedImports: number;
  processingImports: number;
  totalRecordsProcessed: number;
  totalRecordsSuccessful: number;
  totalRecordsFailed: number;
  averageProcessingTime: number;
  lastImportDate: string;
}

export interface ImportBatch {
  id: number;
  batchId: string;
  importType: ImportType;
  fileName: string;
  fileSize: number;
  totalRecords?: number;
  successfulRecords?: number;
  failedRecords?: number;
  status: BatchStatus;
  errorSummary?: string;
  importedBy: number;
  importedByName: string;
  projectId?: number;
  projectName?: string;
  payrollId?: number;
  processingProgress?: number;
  requiresApproval?: boolean;
  errors?: ImportError[];
  createdAt: string;
  completedAt?: string;
  estimatedCompletionTime?: string;
}

export interface ImportError {
  rowNumber: number;
  errorType: string;
  errorMessage: string;
  rawData: Record<string, unknown>;
}

export interface ExportJob {
  exportId: string;
  exportType: ExportType;
  status: BatchStatus;
  downloadUrl?: string;
  filename?: string;
  fileSize?: number;
  totalRecords?: number;
  createdBy: number;
  createdByName: string;
  createdAt: string;
  completedAt?: string;
  expiresAt?: string;
  estimatedCompletionTime?: string;
}

export interface ImportTemplate {
  downloadUrl: string;
  filename: string;
  fileSize: number;
  version: string;
  lastUpdated: string;
}

// Enums
export type ImportType = 'employees' | 'timesheets';
export type ExportType = 'employees' | 'timesheets' | 'projects' | 'financial';
export type BatchStatus = 'processing' | 'completed' | 'failed' | 'cancelled' | 'expired';
export type ExportFormat = 'excel' | 'csv' | 'pdf';
export type TemplateType = 'employees' | 'timesheets';

// Request types
export interface ImportSummaryParams {
  import_type?: ImportType;
  fromDate?: string;
  toDate?: string;
}

export interface ImportListParams {
  page?: number;
  pageSize?: number;
  import_type?: ImportType;
  status?: BatchStatus;
  project_id?: number;
  imported_by?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface ImportEmployeesRequest extends FormData {
  file: File;
}

export interface ImportTimesheetsRequest extends FormData {
  file: File;
  project_id: number;
}


export interface ExportDataRequest {
  filters?: {
    project_id?: number;
    fromDate?: string;
    toDate?: string;
    status?: string;
    employee_id?: number;
    payroll_id?: number;
    [key: string]: unknown;
  };
  format: ExportFormat;
  includeHeaders?: boolean;
  columns?: string[];
}

export interface ExportListParams {
  page?: number;
  pageSize?: number;
  export_type?: ExportType;
  status?: BatchStatus;
  created_by?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface FileValidationRequest {
  file: File;
  import_type: ImportType;
  project_id?: number;
}

export interface BulkRetryRequest {
  batch_ids: string[];
  retry_failed_only?: boolean;
}

// Response wrapper types
export interface ImportSummaryResponse {
  data: ImportSummary;
}

export interface ImportListResponse {
  data: {
    imports: ImportBatch[];
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
  };
}

export interface ImportBatchResponse {
  data: ImportBatch;
}

export interface ImportCreateResponse {
  data: {
    batchId: string;
    importType: ImportType;
    fileName: string;
    fileSize: number;
    projectId?: number;
    payrollId?: number;
    status: BatchStatus;
    requiresApproval?: boolean;
    estimatedCompletionTime?: string;
    createdAt: string;
  };
}

export interface ImportCancelResponse {
  data: {
    batchId: string;
    status: 'cancelled';
    cancelledAt: string;
  };
}

export interface ImportRetryResponse {
  data: {
    newBatchId: string;
    originalBatchId: string;
    recordsToRetry: number;
    status: 'processing';
  };
}

export interface ImportTemplateResponse {
  data: ImportTemplate;
}

export interface ExportCreateResponse {
  data: {
    exportId: string;
    exportType: ExportType;
    status: BatchStatus;
    estimatedCompletionTime?: string;
    createdAt: string;
  };
}

export interface ExportJobResponse {
  data: ExportJob;
}

export interface ExportListResponse {
  data: {
    exports: ExportJob[];
    pagination: {
      page: number;
      pageSize: number;
      total: number;
      totalPages: number;
    };
  };
}

export interface FileValidationResponse {
  data: {
    valid: boolean;
    errors: ImportError[];
    warnings: string[];
    previewData?: Record<string, unknown>[];
    totalRows: number;
    columnsFound: string[];
    requiredColumns: string[];
    missingColumns: string[];
  };
}

export interface BatchStatusResponse {
  data: {
    batchId: string;
    status: BatchStatus;
    progress: number;
    currentStep: string;
    estimatedTimeRemaining?: number;
    lastUpdated: string;
  };
}

export interface BulkRetryResponse {
  data: {
    totalBatches: number;
    retriedBatches: number;
    skippedBatches: number;
    newBatchIds: string[];
    errors: {
      batchId: string;
      error: string;
    }[];
  };
}

// Advanced features
export interface ImportMetadata {
  batchId: string;
  originalFileName: string;
  uploadedBy: {
    userId: number;
    userName: string;
    userRole: string;
  };
  fileInfo: {
    size: number;
    checksum: string;
    mimeType: string;
  };
  validationRules: {
    requiredColumns: string[];
    optionalColumns: string[];
    dataTypes: Record<string, string>;
    businessRules: string[];
  };
  processingStats: {
    startTime: string;
    endTime?: string;
    duration?: number;
    memoryUsed?: number;
    cpuTime?: number;
  };
}

export interface ExportMetadata {
  exportId: string;
  requestedBy: {
    userId: number;
    userName: string;
    userRole: string;
  };
  exportConfig: {
    filters: Record<string, unknown>;
    columns: string[];
    format: ExportFormat;
    includeHeaders: boolean;
  };
  generationStats: {
    startTime: string;
    endTime?: string;
    duration?: number;
    recordsProcessed: number;
    fileSize?: number;
  };
}

// Error types specific to imports/exports
export interface ImportExportError {
  code: 'INVALID_FILE_FORMAT' | 'MISSING_REQUIRED_COLUMNS' | 'EXPORT_FILE_EXPIRED' |
        'FILE_TOO_LARGE' | 'PROCESSING_TIMEOUT' | 'VALIDATION_FAILED' |
        'DUPLICATE_BATCH' | 'PROJECT_NOT_FOUND' | 'INSUFFICIENT_PERMISSIONS';
  message: string;
  details: {
    filename?: string;
    allowedFormats?: string[];
    missingColumns?: string[];
    providedColumns?: string[];
    exportId?: string;
    expiredAt?: string;
    maxSize?: number;
    actualSize?: number;
    [key: string]: unknown;
  };
}

// Vietnamese translations
export const VIETNAMESE_IMPORT_EXPORT_LABELS = {
  importTypes: {
    employees: 'Nhân viên',
    timesheets: 'Bảng chấm công',
  },
  exportTypes: {
    employees: 'Nhân viên',
    timesheets: 'Bảng chấm công',
    projects: 'Dự án',
    financial: 'Tài chính'
  },
  statuses: {
    processing: 'Đang xử lý',
    completed: 'Hoàn thành',
    failed: 'Thất bại',
    cancelled: 'Đã hủy',
    expired: 'Hết hạn'
  },
  formats: {
    excel: 'Excel',
    csv: 'CSV',
    pdf: 'PDF'
  }
} as const;
