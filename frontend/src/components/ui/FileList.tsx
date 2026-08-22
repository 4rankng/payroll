import { format } from "date-fns";
import { useState } from 'react';
import { Download, Eye, Trash2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { cn } from '@/lib/utils';
import { assetService } from '@/services/api/asset.service';
import { useDownloadAsset, useDeleteAsset } from '@/hooks/api/useAssets';
import { EmptyState } from '@/components/shared/EmptyState';
import type { Asset } from '@/types/api/financial.types';

interface FileListProps {
  files: Asset[];
  onDelete?: (assetId: number) => void;
  showActions?: boolean;
  compact?: boolean;
  className?: string;
}

interface ImagePreviewProps {
  asset: Asset;
  isOpen: boolean;
  onClose: () => void;
}

function ImagePreview({ asset, isOpen, onClose }: ImagePreviewProps) {
  const downloadUrl = assetService.getDownloadUrl(asset.id);

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{asset.original_filename}</DialogTitle>
        </DialogHeader>
        <div className="flex justify-center">
          <img
            src={downloadUrl}
            alt={asset.original_filename}
            className="max-w-full max-h-[70vh] object-contain"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function FileList({
  files,
  onDelete,
  showActions = true,
  compact = false,
  className,
}: FileListProps) {
  const [previewAsset, setPreviewAsset] = useState<Asset | null>(null);
  const downloadAsset = useDownloadAsset();
  const deleteAsset = useDeleteAsset();

  const handleDownload = (asset: Asset) => {
    downloadAsset.mutate({
      id: asset.id,
      filename: asset.original_filename,
    });
  };

  const handleDelete = (asset: Asset) => {
    deleteAsset.mutate(asset.id, {
      onSuccess: () => {
        onDelete?.(asset.id);
      },
    });
  };

  const handlePreview = (asset: Asset) => {
    if (assetService.isImageFile(asset)) {
      setPreviewAsset(asset);
    } else {
      // For non-image files, trigger download
      handleDownload(asset);
    }
  };

  if (files.length === 0) {
    return (
      <div className={cn("rounded-lg border border-dashed border-border/60 px-3", className)}>
        <EmptyState title="Chưa có tệp chứng từ" size="sm" className="py-3" />
      </div>
    );
  }

  return (
    <>
      <div className={cn("space-y-2", className)}>
        {files.map((asset) => (
          <div
            key={asset.id}
            className={cn(
              "flex items-center gap-3 p-3 border rounded-lg bg-white hover:bg-gray-50 transition-colors",
              compact && "p-2"
            )}
          >
            {/* File Icon/Thumbnail */}
            <div className="flex-shrink-0">
              {assetService.isImageFile(asset) ? (
                <div
                  className="w-10 h-10 rounded bg-gray-100 flex items-center justify-center cursor-pointer hover:bg-gray-200"
                  onClick={() => handlePreview(asset)}
                >
                  <img
                    src={assetService.getDownloadUrl(asset.id)}
                    alt={asset.original_filename}
                    className="w-8 h-8 rounded object-cover"
                    onError={(e) => {
                      // If image fails to load, show icon instead
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                      target.nextElementSibling?.classList.remove('hidden');
                    }}
                  />
                  <span className="hidden typography-title-large">
                    {assetService.getFileIcon(asset)}
                  </span>
                </div>
              ) : (
                <div className="w-10 h-10 rounded bg-gray-100 flex items-center justify-center typography-title-large">
                  {assetService.getFileIcon(asset)}
                </div>
              )}
            </div>

            {/* File Info */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p
                  className={cn(
                    "font-medium text-gray-900 truncate cursor-pointer hover:text-blue-600",
                    compact && "typography-body-medium"
                  )}
                  onClick={() => handlePreview(asset)}
                  title={asset.original_filename}
                >
                  {asset.original_filename}
                </p>
                
                {/* Status badge */}
                {(asset as unknown as { status?: string }).status && (
                  <Badge
                    variant={(asset as unknown as { status: string }).status === 'active' ? 'default' : 'secondary'}
                    className="typography-body-small"
                  >
                    {(asset as unknown as { status: string }).status === 'active' ? 'Đang dùng' : (asset as unknown as { status: string }).status}
                  </Badge>
                )}
              </div>

              <div className="flex items-center gap-4 mt-1">
                <p className={cn("text-gray-500", compact ? "typography-body-small" : "typography-body-medium")}>
                  {assetService.formatFileSize(asset.file_size)}
                </p>
                
                {!compact && (
                  <>
                    <p className="typography-body-small text-gray-400">
                      {format(new Date(asset.created_at), 'dd/MM/yyyy')}
                    </p>
                    
                    {(asset as unknown as { is_public?: boolean }).is_public && (
                      <div className="flex items-center gap-1">
                        <ExternalLink className="w-3 h-3 text-gray-400" />
                        <span className="typography-body-small text-gray-400">Công khai</span>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>

            {/* Actions */}
            {showActions && (
              <div className="flex items-center gap-1">
                {/* Preview/Open Button */}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handlePreview(asset)}
                  disabled={downloadAsset.isPending}
                  className="h-8 w-8 p-0 hover:bg-blue-50 hover:text-blue-600"
                  title={assetService.isImageFile(asset) ? "Xem trước" : "Mở file"}
                >
                  {assetService.isImageFile(asset) ? (
                    <Eye className="w-4 h-4" />
                  ) : (
                    <Download className="w-4 h-4" />
                  )}
                </Button>

                {/* Download Button */}
                {!assetService.isImageFile(asset) && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDownload(asset)}
                    disabled={downloadAsset.isPending}
                    className="h-8 w-8 p-0 hover:bg-green-50 hover:text-green-600"
                    title="Tải xuống"
                  >
                    <Download className="w-4 h-4" />
                  </Button>
                )}

                {/* Delete Button */}
                {onDelete && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(asset)}
                    disabled={deleteAsset.isPending}
                    className="h-8 w-8 p-0 hover:bg-red-50 hover:text-red-600"
                    title="Xóa file"
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Image Preview Modal */}
      {previewAsset && (
        <ImagePreview
          asset={previewAsset}
          isOpen={!!previewAsset}
          onClose={() => setPreviewAsset(null)}
        />
      )}
    </>
  );
}
