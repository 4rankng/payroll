import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS, FILE_LIMITS } from '@/config/api.config';
import { createFileFormData, UploadProgressTracker } from '@/utils/file-upload';
import type {
  Asset,
  CreateAssetData,
  AssetFilters,
  AssetsResponse,
  AssetResponse,
} from '@/types/api/financial.types';

class AssetService {
  /**
   * Upload a new asset file
   */
  async uploadAsset(
    data: CreateAssetData,
    onProgress?: (progress: number) => void
  ): Promise<ApiResponse<Asset>> {
    const formData = createFileFormData(data.file, {
      upload_type: data.upload_type,
    });

    const progressTracker = new UploadProgressTracker();
    if (onProgress) {
      progressTracker.onProgress(onProgress);
    }

    const response = await apiClient.post<AssetResponse>(
      API_ENDPOINTS.assets.upload,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent) => {
          if (progressEvent.total) {
            progressTracker.updateProgress(progressEvent.loaded, progressEvent.total);
          }
        },
      }
    );

    progressTracker.complete();

    // Handle different API response structures
    const responseData = response.data;

    // The API returns: { status: "success", data: { id, filename, ... }, message: "..." }
    // We need to extract the asset data properly
    if (responseData && responseData.data && responseData.data.id) {
      // Wrapped format: { status: "success", data: { asset }, message: "..." }
      return { ...response, data: responseData.data } as ApiResponse<Asset>;
    } else if (responseData && (responseData as unknown as Asset).id) {
      // Direct format: { id: 2, filename: "...", ... } (without status wrapper)
      return { ...response, data: responseData as unknown as Asset } as ApiResponse<Asset>;
    } else {
      throw new Error('Invalid upload response structure');
    }
  }

  /**
   * Get asset details by ID
   */
  async getAsset(id: number): Promise<Asset> {
    const response = await apiClient.get<AssetResponse>(
      API_ENDPOINTS.assets.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * List assets with filtering and pagination
   */
  async getAssets(filters?: AssetFilters): Promise<AssetsResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<AssetsResponse>(
      `${API_ENDPOINTS.assets.base}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }


  /**
   * Download asset file
   */
  async downloadAsset(id: number): Promise<Blob> {
    const blob = await apiClient.downloadBlob(API_ENDPOINTS.assets.download(id));

    if (!blob || !(blob instanceof Blob)) {
      throw new Error('Invalid response format from server');
    }

    if (blob.size === 0) {
      throw new Error('File is empty or does not exist');
    }

    return blob;
  }


  /**
   * Delete asset
   */
  async deleteAsset(id: number): Promise<ApiResponse<void>> {
    const response = await apiClient.delete(API_ENDPOINTS.assets.byId(id));
    return response as ApiResponse<void>;
  }

  /**
   * Validate file before upload
   */
  validateFileForEvidence(file: File): { isValid: boolean; error?: string } {
    // File size validation
    const maxSize = FILE_LIMITS.evidence.maxSize;
    if (file.size > maxSize) {
      const sizeMB = (maxSize / (1024 * 1024)).toFixed(0);
      return {
        isValid: false,
        error: `Kích thước file vượt quá ${sizeMB}MB`,
      };
    }

    // File type validation for evidence files
    const allowedTypes = FILE_LIMITS.evidence.allowedTypes;
    const allowedExtensions = FILE_LIMITS.evidence.allowedFormats;

    const fileExtension = `.${file.name.split('.').pop()?.toLowerCase()}`;
    
    if (!allowedTypes.includes(file.type) && !allowedExtensions.includes(fileExtension)) {
      return {
        isValid: false,
        error: 'Định dạng file không hợp lệ. Chỉ chấp nhận: ảnh (JPG, PNG, GIF), PDF, Word, Excel, TXT',
      };
    }

    return { isValid: true };
  }

  /**
   * Get file icon based on content type or extension
   */
  getFileIcon(asset: Asset): string {
    const contentType = asset.content_type?.toLowerCase() || '';
    const filename = (asset.original_filename || asset.filename).toLowerCase();

    // Images
    if (contentType.startsWith('image/')) {
      return '🖼️';
    }

    // PDFs
    if (contentType.includes('pdf') || filename.endsWith('.pdf')) {
      return '📄';
    }

    // Word documents
    if (contentType.includes('word') || filename.endsWith('.doc') || filename.endsWith('.docx')) {
      return '📝';
    }

    // Excel files
    if (contentType.includes('excel') || contentType.includes('spreadsheet') || 
        filename.endsWith('.xls') || filename.endsWith('.xlsx')) {
      return '📊';
    }

    // Text files
    if (contentType.includes('text') || filename.endsWith('.txt')) {
      return '📋';
    }

    // Default
    return '📎';
  }

  /**
   * Format file size for display
   */
  formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  /**
   * Check if file type is an image for preview
   */
  isImageFile(asset: Asset): boolean {
    return asset.content_type?.startsWith('image/') || false;
  }

  /**
   * Get display name for asset
   */
  getDisplayName(asset: Asset): string {
    return asset.original_filename || asset.filename;
  }

  /**
   * Generate download URL for asset
   */
  getDownloadUrl(id: number): string {
    return `${API_ENDPOINTS.assets.download(id)}`;
  }

  /**
   * Validate evidence URL
   */
  validateEvidenceUrl(url: string): { isValid: boolean; error?: string } {
    if (!url.trim()) {
      return { isValid: true }; // URL is optional
    }

    try {
      new URL(url);
      return { isValid: true };
    } catch {
      return {
        isValid: false,
        error: 'URL không hợp lệ. Vui lòng nhập URL đầy đủ (bao gồm http:// hoặc https://)',
      };
    }
  }
}

export const assetService = new AssetService();