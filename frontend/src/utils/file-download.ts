import { assetService } from '@/services/api/asset.service';
import { showErrorNotification } from '@/utils/error-handler';

/**
 * Trigger browser file download via an invisible anchor element.
 * Uses try/finally to guarantee the ObjectURL is always revoked.
 */
export const triggerBlobDownload = (blobData: BlobPart, filename: string): void => {
  const blob = blobData instanceof Blob ? blobData : new Blob([blobData]);
  const url = window.URL.createObjectURL(blob);
  try {
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
  } finally {
    window.URL.revokeObjectURL(url);
  }
};

/**
 * Download asset file from server and trigger browser download
 */
export const downloadAssetFile = async (
  assetId: number,
  filename: string
): Promise<void> => {
  try {
    const blob = await assetService.downloadAsset(assetId);
    triggerBlobDownload(blob, filename);
  } catch (error) {
    console.error('Error downloading file:', error);
    showErrorNotification('Lỗi tải file', 'Không thể tải file xuống. Vui lòng thử lại.');
  }
};

/**
 * Extract filename from Content-Disposition header
 * Supports both RFC 2183 and RFC 5987 formats:
 * - attachment; filename="example.xlsx"
 * - attachment; filename*=UTF-8''example.xlsx
 */
export const extractFilenameFromHeaders = (headers: unknown): string | null => {
  let contentDisposition: string | null = null;

  // Handle Axios response headers, Headers object, and plain object
  if (headers && typeof headers === 'object') {
    const maybeHeaders = headers as { [key: string]: unknown; get?: (name: string) => string | null | undefined };
    if (typeof maybeHeaders.get === 'function') {
      // Headers object
      contentDisposition = maybeHeaders.get('Content-Disposition') || maybeHeaders.get('content-disposition') || null;
    } else {
      // Plain object or Axios headers
      const lower = maybeHeaders['content-disposition'];
      const upper = maybeHeaders['Content-Disposition'];
      contentDisposition = typeof lower === 'string' ? lower : (typeof upper === 'string' ? upper : null);
    }
  }

  if (!contentDisposition || typeof contentDisposition !== 'string') {
    return null;
  }

  // Try RFC 5987 format first (filename*=UTF-8''...)
  const rfc5987Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (rfc5987Match) {
    try {
      return decodeURIComponent(rfc5987Match[1]);
    } catch {
      // Fall through to RFC 2183 format
    }
  }

  // Try RFC 2183 format (filename="..." or filename=...)
  const rfc2183Match = contentDisposition.match(/filename="?([^";\n]+)"?/i);
  if (rfc2183Match) {
    return rfc2183Match[1].trim();
  }

  return null;
};

/**
 * Get file extension from filename
 */
export const getFileExtension = (filename: string): string => {
  return filename.split('.').pop()?.toLowerCase() || '';
};

/**
 * Check if file is an image based on extension
 */
export const isImageFile = (filename: string): boolean => {
  const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'];
  const extension = getFileExtension(filename);
  return imageExtensions.includes(extension);
};

/**
 * Get file type icon based on extension
 */
export const getFileTypeIcon = (filename: string): string => {
  const extension = getFileExtension(filename);

  switch (extension) {
    case 'pdf':
      return '📄';
    case 'doc':
    case 'docx':
      return '📝';
    case 'xls':
    case 'xlsx':
      return '📊';
    case 'txt':
      return '📋';
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
    case 'svg':
      return '🖼️';
    default:
      return '📎';
  }
};
