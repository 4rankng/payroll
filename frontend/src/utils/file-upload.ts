import { API_ENDPOINTS, FILE_LIMITS } from '@/config/api.config';
import type { apiClient } from '@/services/api/client';

export interface FileValidationResult {
  isValid: boolean;
  error?: string;
}

/**
 * Validate file before upload
 */
export const validateFile = (
  file: File,
  options?: {
    maxSize?: number;
    allowedFormats?: string[];
  }
): FileValidationResult => {
  const maxSize = options?.maxSize || FILE_LIMITS.excel.maxSize;
  const allowedFormats = options?.allowedFormats || FILE_LIMITS.excel.allowedFormats;

  // Check file size
  if (file.size > maxSize) {
    const sizeMB = (maxSize / (1024 * 1024)).toFixed(0);
    return {
      isValid: false,
      error: `Kích thước file vượt quá ${sizeMB}MB`,
    };
  }

  // Check file format
  const fileExtension = `.${file.name.split('.').pop()?.toLowerCase()}`;
  if (!allowedFormats.includes(fileExtension)) {
    return {
      isValid: false,
      error: `Định dạng file không hợp lệ. Chỉ chấp nhận: ${allowedFormats.join(', ')}`,
    };
  }

  return { isValid: true };
};

/**
 * Format file size for display
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

/**
 * Create FormData for file upload
 */
export const createFileFormData = (
  file: File,
  additionalData?: Record<string, unknown>
): FormData => {
  const formData = new FormData();
  formData.append('file', file);

  if (additionalData) {
    Object.entries(additionalData).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        formData.append(key, String(value));
      }
    });
  }

  return formData;
};

/**
 * Download file from URL
 */
export const downloadFile = (url: string, filename: string): void => {
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  // Safely remove the link
  if (link.parentNode) {
    link.parentNode.removeChild(link);
  }
};

/**
 * Generate XLS filename with timestamp
 */
export const generateXLSFilename = (prefix: string): string => {
  const timestamp = new Date().toISOString().split('T')[0].replace(/-/g, '');
  return `${prefix}_${timestamp}.xls`;
};

/**
 * @deprecated Use generateXLSFilename instead
 */
export const generateExcelFilename = (prefix: string): string => {
  return generateXLSFilename(prefix);
};

/**
 * File upload progress tracker
 */
export class UploadProgressTracker {
  private progress = 0;
  private onProgressCallbacks: ((progress: number) => void)[] = [];

  public onProgress(callback: (progress: number) => void): void {
    this.onProgressCallbacks.push(callback);
  }

  public updateProgress(loaded: number, total: number): void {
    this.progress = Math.round((loaded / total) * 100);
    this.onProgressCallbacks.forEach(callback => callback(this.progress));
  }

  public reset(): void {
    this.progress = 0;
    this.onProgressCallbacks.forEach(callback => callback(0));
  }

  public complete(): void {
    this.progress = 100;
    this.onProgressCallbacks.forEach(callback => callback(100));
  }

  public getProgress(): number {
    return this.progress;
  }
}

/**
 * Template download helper
 */
export const downloadTemplate = async (
  templateType: 'employees' | 'timesheets' | 'payroll_results',
  apiClient: typeof import('@/services/api/client').apiClient
): Promise<void> => {
  try {
    const response = await apiClient.get<{ downloadUrl: string; filename: string }>(
      `/imports/templates/${templateType}`
    );

    if (response.data?.downloadUrl) {
      downloadFile(response.data.downloadUrl, response.data.filename);
    }
  } catch (error) {
    throw new Error('Không thể tải xuống mẫu file');
  }
};

/**
 * Batch file processor for large imports
 */
export class BatchFileProcessor {
  private batchSize = 100;

  constructor(batchSize?: number) {
    if (batchSize) {
      this.batchSize = batchSize;
    }
  }

  async processBatches<T>(
    data: T[],
    processor: (batch: T[]) => Promise<void>,
    onProgress?: (processed: number, total: number) => void
  ): Promise<void> {
    const totalBatches = Math.ceil(data.length / this.batchSize);
    let processed = 0;

    for (let i = 0; i < totalBatches; i++) {
      const start = i * this.batchSize;
      const end = Math.min(start + this.batchSize, data.length);
      const batch = data.slice(start, end);

      await processor(batch);

      processed += batch.length;
      if (onProgress) {
        onProgress(processed, data.length);
      }
    }
  }
}

/**
 * Check if browser supports file upload
 */
export const isFileUploadSupported = (): boolean => {
  return typeof File !== 'undefined' &&
         typeof FileReader !== 'undefined' &&
         typeof FormData !== 'undefined';
};

/**
 * Get file icon based on extension
 */
export const getFileIcon = (filename: string): string => {
  const extension = filename.split('.').pop()?.toLowerCase();

  switch (extension) {
    case 'xlsx':
    case 'xls':
      return '📊';
    case 'csv':
      return '📋';
    case 'pdf':
      return '📄';
    default:
      return '📎';
  }
};

/**
 * Validate email attachments (multiple files)
 */
export const validateEmailAttachments = (files: File[]): FileValidationResult => {
  const { maxFiles, maxSize, allowedFormats } = FILE_LIMITS.emailAttachment;

  // Check number of files
  if (files.length > maxFiles) {
    return {
      isValid: false,
      error: `Chỉ được đính kèm tối đa ${maxFiles} file`,
    };
  }

  // Validate each file
  for (const file of files) {
    const validation = validateFile(file, {
      maxSize,
      allowedFormats,
    });

    if (!validation.isValid) {
      return {
        isValid: false,
        error: `${file.name}: ${validation.error}`,
      };
    }
  }

  return { isValid: true };
};
