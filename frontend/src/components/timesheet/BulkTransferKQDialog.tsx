import { useState, memo, useCallback, useMemo } from 'react';
import { Dialog, DialogContent, DialogNavyHeader, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { FileUpload } from '@/components/ui/FileUpload';
import { Upload, Loader2, FileSpreadsheet, X, FileDown, Info } from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useGenerateBulkTransferKQ } from '@/hooks/api/usePayrolls';

interface BulkTransferKQDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * "Tạo KQ CK" — turns an outbound chuyển-lô workbook (the file handed to the
 * bank) into the bank result workbook the system accepts back through the
 * "Nhập KQ chuyển lô" dialog.
 *
 * Every line is written as "Thành công" with an empty FT column, so the admin
 * is expected to edit the downloaded workbook (mark real failures, paste the
 * bank's FT numbers) before uploading it.
 */
export const BulkTransferKQDialog = memo(function BulkTransferKQDialog({
  open,
  onOpenChange,
}: BulkTransferKQDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const generateMutation = useGenerateBulkTransferKQ();

  const validateExcelFile = useCallback((file: File) => {
    const allowedExtensions = ['.xlsx', '.xls'];
    const hasValidExtension = allowedExtensions.some((ext) =>
      file.name.toLowerCase().endsWith(ext)
    );

    if (!hasValidExtension) {
      return { isValid: false, error: 'Chỉ chấp nhận file Excel (.xls, .xlsx)' };
    }

    if (file.size > FILE_LIMITS.excel.maxSize) {
      return {
        isValid: false,
        error: `File quá lớn. Kích thước tối đa: ${FILE_LIMITS.excel.maxSize / 1024 / 1024}MB`,
      };
    }

    return { isValid: true };
  }, []);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      if (files.length > 0) setSelectedFile(files[0]);
    },
    []
  );

  const handleGenerate = useCallback(async () => {
    if (!selectedFile) return;
    try {
      await generateMutation.mutateAsync(selectedFile);
      setSelectedFile(null);
    } catch {
      // Error is already handled in the mutation hook
    }
  }, [selectedFile, generateMutation]);

  const handleClose = useCallback(() => {
    if (generateMutation.isPending) return;
    setSelectedFile(null);
    onOpenChange(false);
  }, [generateMutation.isPending, onOpenChange]);

  const canGenerate = useMemo(
    () => Boolean(selectedFile) && !generateMutation.isPending,
    [selectedFile, generateMutation.isPending]
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="flex max-h-[92dvh] max-w-2xl flex-col gap-0 overflow-hidden"
        contentPadding="none"
        hideCloseButton
      >
        <DialogNavyHeader
          title="Tạo KQ CK"
          description="Tạo file kết quả chuyển khoản từ file chuyển lô đã gửi ngân hàng."
        />

        <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-4 py-5 sm:px-6">
          <div className="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50/60 p-3">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-blue-100 text-blue-700">
              <Info className="h-4 w-4" />
            </div>
            <div className="min-w-0 space-y-0.5">
              <p className="typography-body-medium text-blue-900">
                Cách dùng
              </p>
              <p className="typography-body-small text-blue-800">
                Tải lên file chuyển lô → file KQ CK tải xuống → sửa lại trạng
                thái/FT nếu cần → nhập lại qua “Nhập KQ chuyển lô”.
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <Label className="typography-body-medium">
              FILE CHUYỂN LÔ <span className="text-red-700">*</span>
            </Label>

            {!selectedFile ? (
              <FileUpload
                onFilesSelected={handleFilesSelected}
                accept=".xls,.xlsx"
                multiple={false}
                maxFiles={1}
                validateFile={validateExcelFile}
                disabled={generateMutation.isPending}
              >
                <div className="flex flex-col items-center justify-center text-center space-y-2 py-6">
                  <div className="w-12 h-12 rounded-full flex items-center justify-center bg-muted">
                    <Upload className="w-6 h-6 text-muted-foreground" />
                  </div>

                  <div className="space-y-1">
                    <p className="typography-body-medium text-foreground">
                      Tải lên file chuyển lô
                    </p>
                    <p className="typography-body-small text-muted-foreground">
                      Kéo thả hoặc{' '}
                      <span className="text-blue-700 hover:text-blue-700 font-medium">
                        chọn file
                      </span>
                    </p>
                    <p className="typography-body-small text-gray-600">
                      File xuất từ “Chuyển lô” hoặc “Chuyển OnePay”
                    </p>
                  </div>
                </div>
              </FileUpload>
            ) : (
              <div className="border rounded-xl p-4 bg-muted/50">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className="w-10 h-10 rounded bg-green-100 flex items-center justify-center">
                      <FileSpreadsheet className="w-5 h-5 text-green-700" />
                    </div>
                    <div className="min-w-0">
                      <p className="typography-body-medium break-words font-medium text-foreground">
                        {selectedFile.name}
                      </p>
                      <p className="typography-body-small text-muted-foreground">
                        {(selectedFile.size / 1024).toFixed(1)} KB
                      </p>
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedFile(null)}
                    disabled={generateMutation.isPending}
                    className="text-red-700 hover:text-red-700 hover:bg-red-50"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="grid grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-2 sm:gap-4 sm:px-6">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={generateMutation.isPending}
            className="h-11 w-full"
          >
            Đóng
          </Button>
          <Button
            onClick={handleGenerate}
            disabled={!canGenerate}
            className="h-11 w-full"
          >
            {generateMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang tạo...
              </>
            ) : (
              <>
                <FileDown className="w-4 h-4 mr-2" />
                Tạo &amp; tải file
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
