import { useState, useCallback, useMemo, memo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { FileUpload } from '@/components/ui/FileUpload';
import { Upload, Loader2, X } from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useUploadSettlementResult } from '@/hooks/api/usePayrolls';
import { useQueryClient } from '@tanstack/react-query';

interface SettlementResultUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const allowedMimeTypes = [
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
];

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 Bytes';
  const units = ['Bytes', 'KB', 'MB', 'GB'];
  const index = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const value = bytes / Math.pow(1024, index);
  return `${value.toFixed(2)} ${units[index]}`;
};

export const SettlementResultUploadDialog = memo(function SettlementResultUploadDialog({
  open,
  onOpenChange,
}: SettlementResultUploadDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const queryClient = useQueryClient();
  const uploadMutation = useUploadSettlementResult();

  const validateExcelFile = useCallback((file: File) => {
    const extension = file.name.slice(file.name.lastIndexOf('.')).toLowerCase();
    const hasValidExtension = FILE_LIMITS.excel.allowedFormats.includes(extension);
    const hasValidMimeType = allowedMimeTypes.includes(file.type);

    if (!hasValidMimeType && !hasValidExtension) {
      return {
        isValid: false,
        error: 'Chỉ chấp nhận file Excel (.xls, .xlsx)',
      };
    }

    if (file.size > FILE_LIMITS.excel.maxSize) {
      const maxSizeMb = FILE_LIMITS.excel.maxSize / 1024 / 1024;
      return {
        isValid: false,
        error: `File quá lớn. Kích thước tối đa: ${maxSizeMb}MB`,
      };
    }

    return { isValid: true };
  }, []);

  const handleFilesSelected = useCallback((files: File[]) => {
    if (files.length === 0) {
      setSelectedFile(null);
      return;
    }

    setSelectedFile(files[files.length - 1]);
  }, []);

  const handleUpload = useCallback(async () => {
    if (!selectedFile) return;

    try {
      await uploadMutation.mutateAsync(selectedFile);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['timesheets'] }),
        queryClient.invalidateQueries({ queryKey: ['timesheet-summary'] }),
        queryClient.invalidateQueries({ queryKey: ['transactions'] }),
        queryClient.invalidateQueries({ queryKey: ['ledger-summary'] }),
      ]);
      setSelectedFile(null);
      onOpenChange(false);
    } catch {
      // Returned error is handled by useUploadSettlementResult
    }
  }, [selectedFile, uploadMutation, queryClient, onOpenChange]);

  const handleClose = useCallback(() => {
    if (uploadMutation.isPending) return;
    setSelectedFile(null);
    onOpenChange(false);
  }, [onOpenChange, uploadMutation.isPending]);

  const handleRemoveFile = useCallback(() => {
    if (uploadMutation.isPending) return;
    setSelectedFile(null);
  }, [uploadMutation.isPending]);

  const onDialogOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        handleClose();
      }
    },
    [handleClose]
  );

  const canUpload = useMemo(() => {
    return !!selectedFile && !uploadMutation.isPending;
  }, [selectedFile, uploadMutation.isPending]);

  return (
    <Dialog open={open} onOpenChange={onDialogOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Nhập kết quả sao kê</DialogTitle>
        </DialogHeader>



          {!selectedFile && (
            <FileUpload
              onFilesSelected={handleFilesSelected}
              accept=".xls,.xlsx"
              multiple={false}
              maxFiles={1}
              validateFile={validateExcelFile}
              className="w-full min-h-[320px]"
              showSelectedFilesList={false}
            >
              <div className="flex flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
                <span className="font-semibold text-foreground">Kéo thả hoặc nhấn để chọn file</span>
                <span>Chấp nhận định dạng .xls, .xlsx (tối đa {FILE_LIMITS.excel.maxSize / 1024 / 1024}MB)</span>
              </div>
            </FileUpload>
          )}

          {selectedFile && (
            <div className="flex flex-col gap-2 rounded-xl bg-muted/50 p-3 text-sm text-foreground">
              <div className="flex items-center justify-between gap-4">
                <div className="font-medium truncate">{selectedFile.name}</div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={handleRemoveFile}
                  disabled={uploadMutation.isPending}
                  className="h-8 w-8 p-0 text-muted-foreground hover:bg-slate-100:bg-slate-900"
                >
                  <X className="h-4 w-4" />
                  <span className="sr-only">Xóa file</span>
                </Button>
              </div>
              <div className="text-xs text-muted-foreground">{formatBytes(selectedFile.size)}</div>
            </div>
          )}


        <DialogFooter className="space-x-2">
          <Button type="button" variant="outline" onClick={handleClose} disabled={uploadMutation.isPending}>
            Hủy
          </Button>
          <Button
            type="button"
            onClick={handleUpload}
            disabled={!canUpload}
            variant="default"
            className="gap-2"
            aria-label="Tải lên kết quả sao kê"
          >
            {uploadMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
            {uploadMutation.isPending ? 'Đang xử lý...' : 'Tải lên sao kê'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
