import { useState, memo, useCallback, useMemo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';
import { FileUpload } from '@/components/ui/FileUpload';
import { Upload, Loader2, CheckCircle, XCircle, FileSpreadsheet, X, AlertCircle } from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useUploadEntriesExcel } from '@/hooks/api/useTimesheets';
import type { UploadEntriesExcelResult } from '@/types/api/timesheet.types';

interface TimesheetImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const TimesheetImportDialog = memo(function TimesheetImportDialog({
  open,
  onOpenChange
}: TimesheetImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadResult, setUploadResult] = useState<UploadEntriesExcelResult | null>(null);
  const [showResults, setShowResults] = useState(false);

  const uploadMutation = useUploadEntriesExcel();

  // File validation for Excel files
  const validateExcelFile = useCallback((file: File) => {
    const allowedTypes = [
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ];
    const allowedExtensions = ['.xls', '.xlsx'];
    const hasValidType = allowedTypes.includes(file.type);
    const hasValidExtension = allowedExtensions.some(ext => file.name.toLowerCase().endsWith(ext));

    if (!hasValidType && !hasValidExtension) {
      return {
        isValid: false,
        error: 'Chỉ chấp nhận file Excel (.xls, .xlsx)'
      };
    }

    if (file.size > FILE_LIMITS.excel.maxSize) {
      return {
        isValid: false,
        error: `File quá lớn. Kích thước tối đa: ${FILE_LIMITS.excel.maxSize / 1024 / 1024}MB`
      };
    }

    return { isValid: true };
  }, []);

  // Handle file selection
  const handleFilesSelected = useCallback((files: File[]) => {
    if (files.length > 0) {
      setSelectedFile(files[0]);
    }
  }, []);

  // Handle upload
  const handleUpload = useCallback(async () => {
    if (!selectedFile) return;

    try {
      const result = await uploadMutation.mutateAsync({ file: selectedFile });
      setUploadResult(result);
      setShowResults(true);
    } catch (error) {
      // Error is already handled in the mutation hook
    }
  }, [selectedFile, uploadMutation]);

  // Handle dialog close
  const handleClose = useCallback(() => {
    if (uploadMutation.isPending) return;

    setSelectedFile(null);
    setUploadResult(null);
    setShowResults(false);
    onOpenChange(false);
  }, [uploadMutation.isPending, onOpenChange]);

  // Handle back to upload
  const handleBackToUpload = useCallback(() => {
    setShowResults(false);
    setUploadResult(null);
    setSelectedFile(null);
  }, []);

  // Validation
  const canUpload = useMemo(() => {
    return selectedFile && !uploadMutation.isPending;
  }, [selectedFile, uploadMutation.isPending]);

  // Results display helpers
  const getStatusIcon = useCallback((hasError: boolean) => {
    return hasError ? (
      <XCircle className="w-4 h-4 text-red-600" />
    ) : (
      <CheckCircle className="w-4 h-4 text-green-700" />
    );
  }, []);

  const getStatusBadge = useCallback((hasError: boolean) => {
    return hasError ? (
      <Badge variant="destructive">
        Lỗi
      </Badge>
    ) : (
      <Badge variant="secondary" className="bg-green-100 text-green-800">
        Thành công
      </Badge>
    );
  }, []);

  // Results view
  if (showResults && uploadResult) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-4xl max-h-[90vh] shadow-none">
          <DialogHeader>
            <DialogTitle>Kết quả nhập công</DialogTitle>
            <DialogDescription>
              Chi tiết kết quả xử lý file Excel nhập công.
            </DialogDescription>
          </DialogHeader>

          <Separator />

          <div className="space-y-6">
            {/* Summary */}
            <div className="flex items-center gap-6 p-4 bg-muted/50 rounded-xl border">
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-emerald-700">{uploadResult.created_count}</span>
                <span className="ml-2">Tạo mới</span>
              </div>
              <div className="w-px h-6 bg-slate-200"></div>
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-muted-foreground">{uploadResult.skipped_count}</span>
                <span className="ml-2">Bỏ qua</span>
              </div>
              <div className="w-px h-6 bg-slate-200"></div>
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-red-600">{uploadResult.error_count}</span>
                <span className="ml-2">Lỗi</span>
              </div>
            </div>

            {/* Errors (if any) */}
            {uploadResult.failed_entries && uploadResult.failed_entries.length > 0 && (
              <div className="space-y-2">
                <Label className="typography-label-large font-medium">
                  Các bản ghi lỗi ({uploadResult.failed_entries.length})
                </Label>
                <div className="max-h-[400px] overflow-y-auto pr-2 space-y-3">
                  {uploadResult.failed_entries.map((entry, index) => (
                    <div key={index} className="border rounded-xl p-4 bg-red-50 border-red-200">
                      <div className="flex items-start gap-3">
                        <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
                        <div className="flex-1 space-y-2">
                          <div className="flex items-center gap-2">
                            <span className="typography-body-small font-medium text-muted-foreground">
                              Dòng {entry.Index}
                            </span>
                            {getStatusBadge(true)}
                          </div>
                          <p className="typography-body-small text-red-800 font-medium">
                            {entry.Error}
                          </p>
                          <div className="typography-body-small text-muted-foreground space-y-1 pt-2 border-t border-red-200">
                            <div><strong>Dự án ID:</strong> {entry.Request.project_id}</div>
                            <div><strong>Nhân viên ID:</strong> {entry.Request.employee_id}</div>
                            <div><strong>Ngày:</strong> {entry.Request.date}</div>
                            <div><strong>Số giờ:</strong> {entry.Request.hours_worked}</div>
                            <div><strong>Loại ca:</strong> {entry.Request.hour_type}</div>
                            <div><strong>Loại ngày:</strong> {entry.Request.day_type}</div>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Success message when no errors */}
            {uploadResult.error_count === 0 && (
              <div className="flex items-center gap-3 p-4 bg-green-50 border border-green-200 rounded-xl">
                <CheckCircle className="w-5 h-5 text-green-700" />
                <p className="typography-body-medium text-green-800">
                  Đã nhập thành công {uploadResult.created_count} bản ghi công.
                </p>
              </div>
            )}
          </div>

          <DialogFooter className="grid grid-cols-2 gap-4">
            <Button
              variant="outline"
              onClick={handleBackToUpload}
              className="w-full"
            >
              Nhập file khác
            </Button>
            <Button
              onClick={handleClose}
              className="w-full"
            >
              Đóng
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // Upload view
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl shadow-none">
        <DialogHeader>
          <DialogTitle>Nhập công Excel</DialogTitle>
          <DialogDescription>
            Tải lên file Excel chứa dữ liệu công để nhập vào hệ thống.
          </DialogDescription>
        </DialogHeader>

        <Separator />

        <div className="space-y-6">
          {/* File Upload Section */}
          <div className="space-y-4">
            <Label className="typography-body-medium">
              FILE EXCEL <span className="text-red-600">*</span>
            </Label>

            {!selectedFile ? (
              <FileUpload
                onFilesSelected={handleFilesSelected}
                accept=".xls,.xlsx"
                multiple={false}
                maxFiles={1}
                validateFile={validateExcelFile}
                disabled={uploadMutation.isPending}
              >
                <div className="flex flex-col items-center justify-center text-center space-y-2 py-6">
                  <div className="w-12 h-12 rounded-full flex items-center justify-center bg-muted">
                    <Upload className="w-6 h-6 text-muted-foreground" />
                  </div>

                  <div className="space-y-1">
                    <p className="typography-body-medium text-foreground">
                      Tải lên file Excel nhập công
                    </p>
                    <p className="typography-body-small text-muted-foreground">
                      Kéo thả hoặc{" "}
                      <span className="text-blue-600 hover:text-blue-700 font-medium">
                        chọn file
                      </span>
                    </p>
                    <p className="typography-body-small text-gray-500">
                      Chỉ chấp nhận file Excel (.xls, .xlsx) - Tối đa 10MB
                    </p>
                  </div>
                </div>
              </FileUpload>
            ) : (
              <div className="border rounded-xl p-4 bg-muted/50">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded bg-green-100 flex items-center justify-center">
                      <FileSpreadsheet className="w-5 h-5 text-green-700" />
                    </div>
                    <div>
                      <p className="typography-body-medium font-medium text-foreground">
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
                    disabled={uploadMutation.isPending}
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>



        </div>

        <DialogFooter className="grid grid-cols-2 gap-4">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={uploadMutation.isPending}
            className="w-full"
          >
            Đóng
          </Button>
          <Button
            onClick={handleUpload}
            disabled={!canUpload}
            className="w-full"
          >
            {uploadMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang xử lý...
              </>
            ) : (
              <>
                <Upload className="w-4 h-4 mr-2" />
                Nhập file
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
