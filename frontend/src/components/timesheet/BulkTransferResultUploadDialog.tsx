import { useState, memo, useCallback, useMemo, useEffect } from 'react';
import { Dialog, DialogContent, DialogNavyHeader, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { FileUpload } from '@/components/ui/FileUpload';
import { Upload, Loader2, CheckCircle, XCircle, FileSpreadsheet, X, FileDown, Clock3 } from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useImportBulkTransferResult } from '@/hooks/api/usePayrolls';
import { useQueryClient } from '@tanstack/react-query';
import type { BulkTransferResultResponse } from '@/services/api/bulk-transfer.service';
import { generateBulkTransferResultPdf } from '@/utils/pdf/bulk-transfer-result';
import { dismissPendingExport, getMostRecentPendingExport, type PendingExport } from '@/utils/timesheet/pendingExports';

interface BulkTransferResultUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const BulkTransferResultUploadDialog = memo(function BulkTransferResultUploadDialog({
  open,
  onOpenChange
}: BulkTransferResultUploadDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadResult, setUploadResult] = useState<BulkTransferResultResponse | null>(null);
  const [showResults, setShowResults] = useState(false);
  const [uploadedAtForFilename, setUploadedAtForFilename] = useState<string | null>(null);
  const [pendingHint, setPendingHint] = useState<PendingExport | null>(null);

  // Reload most-recent pending export hint each time dialog opens
  useEffect(() => {
    if (!open) return;
    setPendingHint(getMostRecentPendingExport());
  }, [open]);

  const queryClient = useQueryClient();
  const importMutation = useImportBulkTransferResult();

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
      const result = await importMutation.mutateAsync(selectedFile);
      setUploadResult(result);
      setShowResults(true);
      setUploadedAtForFilename(new Date().toISOString());

      // Clear pending-export hint — the export→upload cycle is now complete
      if (pendingHint) {
        dismissPendingExport(pendingHint.id);
        setPendingHint(null);
      }

      // Refresh timesheet, transaction, ledger, and upload history data after successful upload
      await queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      await queryClient.invalidateQueries({ queryKey: ['timesheet-summary'] });
      await queryClient.invalidateQueries({ queryKey: ['transactions'] });
      await queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      await queryClient.invalidateQueries({ queryKey: ['bulk-transfer-upload-histories'] });
    } catch {
      // Error is already handled in the mutation hook
    }
  }, [selectedFile, importMutation, queryClient, pendingHint]);

  // Handle dialog close
  const handleClose = useCallback(() => {
    if (importMutation.isPending) return;

    setSelectedFile(null);
    setUploadResult(null);
    setShowResults(false);
    onOpenChange(false);
  }, [importMutation.isPending, onOpenChange]);

  // Handle back to upload
  const handleBackToUpload = useCallback(() => {
    setShowResults(false);
    setUploadResult(null);
    setSelectedFile(null);
  }, []);

  // Handle download PDF
  const handleDownloadPdf = useCallback(async () => {
    if (!uploadResult) return;
    const src = uploadedAtForFilename || new Date().toISOString();
    const sanitized = src
      .toString()
      .trim()
      .replace(/[\\/:*?"<>|]/g, '_')
      .replace(/\s+/g, '_');
    const fileName = `chuyen_tien_${sanitized}.pdf`;
    await generateBulkTransferResultPdf(uploadResult, { fileName });
  }, [uploadResult, uploadedAtForFilename]);

  // Validation
  const canUpload = useMemo(() => {
    return selectedFile && !importMutation.isPending;
  }, [selectedFile, importMutation.isPending]);

  // Format date to Vietnamese format
  const formatDate = (dateString: string) => {
    if (!dateString) return '';
    try {
      const date = new Date(dateString);
      return date.toLocaleString('vi-VN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  // Format currency
  const formatCurrency = (amount: string) => {
    if (!amount) return '0 VND';
    const numValue = Number(amount.replace(/,/g, ''));
    return isNaN(numValue) ? amount : `${numValue.toLocaleString('vi-VN')} VND`;
  };

  // Results display helpers
  const getStatusIcon = useCallback((status: string) => {
    return status === 'paid' ? (
      <CheckCircle className="w-4 h-4 text-green-500" />
    ) : (
      <XCircle className="w-4 h-4 text-red-500" />
    );
  }, []);

  const getStatusBadge = useCallback((status: string) => {
    return status === 'paid' ? (
      <Badge variant="secondary" className="bg-green-100 text-green-800">
        Thành công
      </Badge>
    ) : (
      <Badge variant="destructive">
        Thất bại
      </Badge>
    );
  }, []);

  if (showResults && uploadResult) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-7xl max-h-[90vh] p-0 gap-0 flex flex-col overflow-hidden" hideCloseButton>
          <DialogNavyHeader
            title="Kết quả chuyển tiền"
            description="Chi tiết kết quả xử lý file Excel chuyển tiền hàng loạt."
          />

          <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
            {/* Summary */}
            <div className="flex items-center gap-6 p-4 bg-muted/50 rounded-xl border">
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-foreground">{uploadResult.total_txn}</span>
                <span className="ml-2">Tổng số bản ghi</span>
              </div>
              <div className="w-px h-6 bg-slate-200"></div>
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-emerald-600">{uploadResult.completed_txn}</span>
                <span className="ml-2">Thành công</span>
              </div>
              <div className="w-px h-6 bg-slate-200"></div>
              <div className="typography-body-medium text-muted-foreground">
                <span className="typography-data-large font-semibold text-red-600">{uploadResult.failed_txn}</span>
                <span className="ml-2">Thất bại</span>
              </div>
            </div>

            {/* Details */}
            <div className="space-y-2">
              <Label className="typography-label-large font-medium">Chi tiết xử lý ({uploadResult.items.length})</Label>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 max-h-[500px] overflow-y-auto pr-2">
                {uploadResult.items.map((detail, index) => (
                  <div key={index} className="border rounded-xl p-3 bg-card hover:shadow-sm transition-shadow">
                    <div className="flex items-center gap-2 mb-2">
                      {getStatusIcon(detail.payment_status)}
                      <span className="typography-body-small font-medium text-muted-foreground">Dòng {detail.row}</span>
                      {getStatusBadge(detail.payment_status)}
                    </div>
                    <div className="space-y-1 typography-body-small">
                      <div className="font-medium text-foreground truncate" title={detail.employee_name}>
                        {detail.employee_name}
                      </div>
                      <div className="text-muted-foreground">
                        {detail.employee_bank} - {detail.employee_account_number}
                      </div>
                      {detail.employee_cccd && (
                        <div className="text-muted-foreground">CCCD: {detail.employee_cccd}</div>
                      )}
                      <div className="font-medium text-foreground pt-1">
                        {formatCurrency(detail.amount)}
                      </div>
                      {detail.paid_at && (
                        <div className="text-muted-foreground typography-body-small">
                          {formatDate(detail.paid_at)}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <DialogFooter className="px-6 py-4 border-t grid grid-cols-3 gap-4">
            <Button
              variant="outline"
              onClick={handleBackToUpload}
              className="w-full"
            >
              Nhập file khác
            </Button>
            <Button
              variant="secondary"
              onClick={handleDownloadPdf}
              className="w-full"
            >
              <FileDown className="w-4 h-4 mr-2" />
              Tải PDF
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl p-0 gap-0 flex flex-col overflow-hidden" hideCloseButton>
        <DialogNavyHeader
          title="Kết quả chuyển tiền"
          description="Tải lên file Excel chứa kết quả chuyển tiền để cập nhật trạng thái thanh toán."
        />

        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {/* Pending Export Hint Banner — Stage A → Stage B handoff hint */}
          {pendingHint && (
            <div className="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50/60 p-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-blue-100 text-blue-700">
                <Clock3 className="h-4 w-4" />
              </div>
              <div className="flex-1 min-w-0 space-y-0.5">
                <p className="text-sm font-medium text-blue-900">
                  Bạn vừa xuất file chuyển lô{' '}
                  <span className="text-blue-700">
                    {(() => {
                      const minutesAgo = Math.max(1, Math.round((Date.now() - Date.parse(pendingHint.exportedAt)) / 60000));
                      if (minutesAgo < 60) return `${minutesAgo} phút trước`;
                      const hoursAgo = Math.round(minutesAgo / 60);
                      if (hoursAgo < 24) return `${hoursAgo} giờ trước`;
                      const daysAgo = Math.round(hoursAgo / 24);
                      return `${daysAgo} ngày trước`;
                    })()}
                  </span>
                </p>
                <p className="text-xs text-blue-800">
                  {pendingHint.forMonth
                    ? `Kỳ: ${pendingHint.forMonth}`
                    : pendingHint.fromDate && pendingHint.toDate
                      ? `Kỳ: ${pendingHint.fromDate} → ${pendingHint.toDate}`
                      : 'Upload file kết quả chuyển khoản (.xlsx) để cập nhật trạng thái thanh toán.'}
                </p>
              </div>
              <button
                type="button"
                onClick={() => {
                  dismissPendingExport(pendingHint.id);
                  setPendingHint(null);
                }}
                className="text-blue-500 hover:text-blue-700 transition-colors"
                aria-label="Bỏ qua nhắc nhở"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* File Upload Section */}
          <div className="space-y-4">
            <Label className="typography-body-medium">
              FILE EXCEL <span className="text-red-500">*</span>
            </Label>

            {!selectedFile ? (
              <FileUpload
                onFilesSelected={handleFilesSelected}
                accept=".xls,.xlsx"
                multiple={false}
                maxFiles={1}
                validateFile={validateExcelFile}
                disabled={importMutation.isPending}
              >
                <div className="flex flex-col items-center justify-center text-center space-y-2 py-6">
                  <div className="w-12 h-12 rounded-full flex items-center justify-center bg-muted">
                    <Upload className="w-6 h-6 text-muted-foreground" />
                  </div>

                  <div className="space-y-1">
                    <p className="typography-body-medium text-foreground">
                      Tải lên file Excel kết quả chuyển tiền
                    </p>
                    <p className="typography-body-small text-muted-foreground">
                      Kéo thả hoặc{" "}
                      <span className="text-blue-600 hover:text-blue-700 font-medium">
                        chọn file
                      </span>
                    </p>
                    <p className="typography-body-small text-gray-400">
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
                      <FileSpreadsheet className="w-5 h-5 text-green-600" />
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
                    disabled={importMutation.isPending}
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>


        </div>

        <DialogFooter className="px-6 py-4 border-t grid grid-cols-2 gap-4">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={importMutation.isPending}
            className="w-full"
          >
            Đóng
          </Button>
          <Button
            onClick={handleUpload}
            disabled={!canUpload}
            className="w-full"
          >
            {importMutation.isPending ? (
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
