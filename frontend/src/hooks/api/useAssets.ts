import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { assetService } from '@/services/api/asset.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';
import type {
  Asset,
  CreateAssetData,
  AssetFilters,
  AssetsResponse,
} from '@/types/api/financial.types';

/**
 * Get single asset by ID
 */
export const useAsset = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.assets.detail(id),
    queryFn: () => assetService.getAsset(id),
    enabled: enabled && id > 0,
  });
};

/**
 * Get assets with filtering and pagination
 */
export const useAssets = (filters?: AssetFilters) => {
  return useQuery({
    queryKey: QueryKeys.assets.list(filters),
    queryFn: () => assetService.getAssets(filters),
  });
};


/**
 * Upload asset file
 */
export const useUploadAsset = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      data,
      onProgress,
    }: {
      data: CreateAssetData;
      onProgress?: (progress: number) => void;
    }) => assetService.uploadAsset(data, onProgress),
    onSuccess: (response, variables) => {
      // Invalidate assets lists
      queryClient.invalidateQueries({ queryKey: QueryKeys.assets.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};


/**
 * Delete asset
 */
export const useDeleteAsset = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => assetService.deleteAsset(id),
    onSuccess: (response, deletedId) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: QueryKeys.assets.detail(deletedId) });

      // Invalidate list queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.assets.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Download asset file
 */
export const useDownloadAsset = () => {

  return useMutation({
    mutationFn: async ({ id, filename }: { id: number; filename: string }) => {
      try {
        const blob = await assetService.downloadAsset(id);

        // Validate blob
        if (!blob || !(blob instanceof Blob)) {
          console.error('Download failed: Invalid blob response', blob);
          throw new Error('Không thể tải file: Phản hồi không hợp lệ từ server');
        }

        if (blob.size === 0) {
          console.error('Download failed: Empty blob');
          throw new Error('Không thể tải file: File rỗng hoặc không tồn tại');
        }

        // Create download link
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();

        // Cleanup
        if (link.parentNode) {
          link.parentNode.removeChild(link);
        }
        window.URL.revokeObjectURL(url);

        return { success: true, message: `Đã tải xuống: ${filename}` };
      } catch (error) {
        console.error('Asset download error:', error);
        if (error instanceof Error) {
          throw error;
        }
        throw new Error('Không thể tải file. Vui lòng thử lại.');
      }
    },
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Bulk upload assets
 */
export const useBulkUploadAssets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      files,
      uploadType = 'ledger_evidence',
      onProgress,
    }: {
      files: File[];
      uploadType?: 'document' | 'ledger_evidence' | 'bulk_transfer_result' | 'general';
      onProgress?: (fileIndex: number, progress: number) => void;
    }) => {
      const uploadPromises = files.map((file, index) => {
        const createAssetData: CreateAssetData = {
          file,
          upload_type: uploadType,
        };

        return assetService.uploadAsset(createAssetData, (progress) => {
          if (onProgress) {
            onProgress(index, progress);
          }
        });
      });

      const results = await Promise.all(uploadPromises);
      return results;
    },
    onSuccess: (assets) => {
      // Invalidate relevant queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.assets.lists() });

      // Show success notification for bulk upload
      const successCount = assets.filter(asset => asset.data).length;
      showSuccessNotification(`Đã tải lên thành công ${successCount} file chứng từ`);
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Export query keys for use in other hooks
 */
export const ASSET_QUERY_KEYS = QueryKeys.assets;