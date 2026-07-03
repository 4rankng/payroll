import { useState, memo, useCallback, useMemo } from 'react';
import { Dialog, DialogContent, DialogNavyHeader, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { FileDropZone } from './FileDropZone';
import { Upload, Loader2, CheckCircle, XCircle, FileSpreadsheet, Users } from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useImportFlexibleEmployeeList } from '@/hooks/api/useAdvancePayments';
import type { EmployeeImportStatus } from '@/types/api/employee.types';

interface FlexibleEmployeeListUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export const FlexibleEmployeeListUploadDialog = memo(function FlexibleEmployeeListUploadDialog({
  open,
  onOpenChange,
  onSuccess,
}: FlexibleEmployeeListUploadDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<EmployeeImportStatus | null>(null);
  const [showResults, setShowResults] = useState(false);
  const [uploadProgress, setUploadProgress] = useState({ percentage: 0, processedRows: 0, totalRows: 0 });

  const importMutation = useImportFlexibleEmployeeList({
    onProgress: (percentage, processedRows, totalRows) => {
      setUploadProgress({ percentage, processedRows, totalRows });
    },
    onStatusChange: () => {},
  });

  const validateExcelFile = useCallback((file: File) => {
    const allowedExtensions = ['.xls', '.xlsx'];
    const hasValidExtension = allowedExtensions.some(ext => file.name.toLowerCase().endsWith(ext));
    if (!hasValidExtension) return { isValid: false, error: 'Chỉ chấp nhận file Excel (.xls, .xlsx)' };
    if (file.size > FILE_LIMITS.excel.maxSize) {
      return { isValid: false, error: `File quá lớn. Tối đa ${FILE_LIMITS.excel.maxSize / 1024 / 1024}MB` };
    }
    return { isValid: true };
  }, []);

  const handleUpload = useCallback(async () => {
    if (!selectedFile) return;
    const validation = validateExcelFile(selectedFile);
    if (!validation.isValid) return;
    const formData = new FormData();
    formData.append('file', selectedFile);
    try {
      const result = await importMutation.mutateAsync(formData);
      if (result) {
        setImportResult(result);
        setShowResults(true);
        onSuccess?.();
      }
    } catch {
      // handled in hook
    }
  }, [selectedFile, importMutation, onSuccess, validateExcelFile]);

  const handleReset = useCallback(() => {
    setSelectedFile(null);
    setImportResult(null);
    setShowResults(false);
    setUploadProgress({ percentage: 0, processedRows: 0, totalRows: 0 });
  }, []);

  const handleClose = useCallback(() => {
    if (importMutation.isPending) return;
    handleReset();
    onOpenChange(false);
  }, [importMutation.isPending, handleReset, onOpenChange]);

  const canUpload = useMemo(() => selectedFile && !importMutation.isPending, [selectedFile, importMutation.isPending]);
  const hasProgress = uploadProgress.totalRows > 0;

  // ── Results view ──
  if (showResults && importResult) {
    const hasErrors = importResult.errors && importResult.errors.length > 0;
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[92dvh] max-w-lg flex-col gap-0 overflow-hidden p-0" hideCloseButton>
          <DialogNavyHeader
            title="Kết quả nhập"
            description="Tổng kết quá trình xử lý file Excel"
          />

          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-5">
            {/* Stats row */}
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              {[
                { label: "Tổng dòng", value: importResult.total_rows, color: "text-foreground" },
                { label: "Nhân viên mới", value: importResult.created_count, color: "text-emerald-600" },
                { label: "Đã cập nhật", value: importResult.updated_count, color: "text-primary" },
              ].map(({ label, value, color }) => (
                <div key={label} className="rounded-xl border bg-muted/30 px-3 py-2.5 text-center">
                  <p className={`text-xl font-semibold tabular-nums ${color}`}>{value}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
                </div>
              ))}
            </div>

            {hasErrors && (
              <div className="space-y-2">
                <div className="flex items-center gap-1.5">
                  <XCircle className="h-3.5 w-3.5 text-destructive" />
                  <p className="text-sm font-medium text-destructive">
                    {importResult.error_count} lỗi
                  </p>
                </div>
                <ScrollArea className="h-44 rounded-xl border">
                  <div className="p-3 space-y-2">
                    {importResult.errors?.map((error, index) => (
                      <div key={index} className="flex items-start gap-2.5 rounded-xl border border-destructive/10 bg-destructive/5 p-2.5">
                        <XCircle className="w-3.5 h-3.5 text-destructive mt-0.5 flex-shrink-0" />
                        <div className="min-w-0">
                          <p className="break-words text-xs font-medium text-destructive">
                            Dòng {error.row_number}{error.cccd ? ` · CCCD: ${error.cccd}` : ''}
                          </p>
                          <p className="mt-0.5 break-words text-xs text-muted-foreground">{error.message}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </div>
            )}
          </div>

          <DialogFooter className="grid shrink-0 grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-2 sm:px-5">
            <Button variant="outline" onClick={handleReset} className="h-11 w-full">
              Nhập file khác
            </Button>
            <Button onClick={handleClose} className="h-11 w-full">
              Đóng
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // ── Upload view ──
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[92dvh] max-w-md flex-col gap-0 overflow-hidden p-0" hideCloseButton>
        <DialogNavyHeader
          title="Nhập danh sách nhân viên"
          description="File Excel danh sách nhân viên lịch trả lương linh hoạt"
        />

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-5">
          <FileDropZone
            file={selectedFile}
            onFileChange={setSelectedFile}
            accept=".xls,.xlsx"
            inputId="flex-employee-file-input"
          />

          {/* Progress */}
          {importMutation.isPending && (
            <div className="space-y-2 px-3 py-2.5 rounded-xl bg-muted/50 border">
              <div className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-1.5">
                  <Loader2 className="w-3.5 h-3.5 animate-spin text-primary" />
                  <span className="text-muted-foreground">
                    {hasProgress ? `${uploadProgress.processedRows} / ${uploadProgress.totalRows} dòng` : 'Đang chuẩn bị...'}
                  </span>
                </div>
                {hasProgress && (
                  <span className="font-medium tabular-nums">{uploadProgress.percentage}%</span>
                )}
              </div>
              <Progress value={hasProgress ? uploadProgress.percentage : 0} className="h-1.5" />
            </div>
          )}

          {/* Format hint */}
          <div className="rounded-xl border border-border/60 bg-muted/20 px-3 py-2.5">
            <p className="text-xs font-medium text-muted-foreground mb-1.5">Định dạng cột:</p>
            <div className="grid grid-cols-1 gap-x-4 gap-y-1 text-xs text-muted-foreground min-[380px]:grid-cols-2">
              {[
                ["B", "CCCD"], ["C", "Dự án / Khách hàng"],
                ["D", "Vị trí"], ["E", "Số điện thoại"],
                ["F", "Họ và tên"], ["G", "Tên tài khoản NH"],
                ["H", "Số tài khoản NH"], ["I", "Tên ngân hàng"],
              ].map(([col, label]) => (
                <div key={col} className="flex min-w-0 items-center gap-1.5">
                  <span className="font-mono font-semibold text-foreground/70 w-4">{col}</span>
                  <span className="min-w-0 break-words">{label}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <DialogFooter className="grid shrink-0 grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-[minmax(72px,auto)_1fr] sm:px-5">
          <Button variant="outline" onClick={handleClose} disabled={importMutation.isPending} className="h-11 w-full">
            Đóng
          </Button>
          <Button onClick={handleUpload} disabled={!canUpload} className="h-11 w-full gap-1.5">
            {importMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Đang xử lý...
              </>
            ) : (
              <>
                <Upload className="w-4 h-4" />
                Nhập file
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
