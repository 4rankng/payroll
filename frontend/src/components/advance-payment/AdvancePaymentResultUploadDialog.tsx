import { useState, memo, useCallback, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { FileDropZone } from './FileDropZone';
import {
  Upload,
  Loader2,
  CheckCircle,
  XCircle,
  FileDown,
  ArrowUpFromLine,
} from 'lucide-react';
import { FILE_LIMITS } from '@/config/api.config';
import { useUploadBankResult } from '@/hooks/api/useAdvancePayments';
import { useQueryClient } from '@tanstack/react-query';
import { formatDateTime, formatCurrencyFromString } from '@/utils/formatters';
import type { UploadBankResultResponse } from '@/types/api/advance-payment.types';
import { generateAdvancePaymentResultPdf } from '@/utils/pdf/advance-payment-result';

interface AdvancePaymentResultUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const AdvancePaymentResultUploadDialog = memo(
  function AdvancePaymentResultUploadDialog({
    open,
    onOpenChange,
  }: AdvancePaymentResultUploadDialogProps) {
    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [uploadResult, setUploadResult] = useState<UploadBankResultResponse | null>(null);
    const [showResults, setShowResults] = useState(false);
    const [uploadedAtForFilename, setUploadedAtForFilename] = useState<string | null>(null);

    const queryClient = useQueryClient();
    const uploadMutation = useUploadBankResult();

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
      try {
        const formData = new FormData();
        formData.append('file', selectedFile);
        const result = await uploadMutation.mutateAsync(formData);
        if (result.data) {
          setUploadResult(result.data);
          setShowResults(true);
          setUploadedAtForFilename(new Date().toISOString());
        }
        await queryClient.invalidateQueries({ queryKey: ['admin', 'advance-payments'] });
        await queryClient.invalidateQueries({ queryKey: ['employee', 'advance-payment'] });
      } catch {
        // handled in hook
      }
    }, [selectedFile, uploadMutation, queryClient]);

    const handleClose = useCallback(() => {
      if (uploadMutation.isPending) return;
      setSelectedFile(null);
      setUploadResult(null);
      setShowResults(false);
      onOpenChange(false);
    }, [uploadMutation.isPending, onOpenChange]);

    const handleBackToUpload = useCallback(() => {
      setShowResults(false);
      setUploadResult(null);
      setSelectedFile(null);
    }, []);

    const handleDownloadPdf = useCallback(async () => {
      if (!uploadResult) return;
      const src = uploadedAtForFilename || new Date().toISOString();
      const sanitized = src.trim().replace(/[\\/:*?"<>|]/g, '_').replace(/\s+/g, '_');
      await generateAdvancePaymentResultPdf(uploadResult, { fileName: `chuyen_tien_ung_luong_${sanitized}.pdf` });
    }, [uploadResult, uploadedAtForFilename]);

    const canUpload = useMemo(() => selectedFile && !uploadMutation.isPending, [selectedFile, uploadMutation.isPending]);

    // ── Results view ──
    if (showResults && uploadResult) {
      return (
        <Dialog open={open} onOpenChange={onOpenChange}>
          <DialogContent className="flex max-h-[92dvh] max-w-3xl flex-col gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
            <DialogNavyHeader
              title="Kết quả chuyển tiền"
              description="Chi tiết xử lý file Excel chuyển tiền ứng lương"
            />

            <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-5">
              {/* Summary */}
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {[
                  { label: "Tổng giao dịch", value: uploadResult.total_txn, color: "text-foreground" },
                  { label: "Thành công", value: uploadResult.completed_txn, color: "text-emerald-700" },
                  { label: "Thất bại", value: uploadResult.failed_txn, color: "text-destructive" },
                ].map(({ label, value, color }) => (
                  <div key={label} className="rounded-xl border bg-muted/30 px-3 py-2.5 text-center">
                    <p className={`text-xl font-semibold tabular-nums ${color}`}>{value}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
                  </div>
                ))}
              </div>

              {/* Detail grid */}
              <div>
                <p className="text-sm font-semibold mb-2">
                  Chi tiết ({uploadResult.items?.length ?? 0})
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                  {uploadResult.items?.map((detail, index) => {
                    const isPaid = detail.payment_status === 'paid';
                    return (
                      <div
                        key={index}
                        className="rounded-xl border bg-card p-3 space-y-1.5"
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="flex min-w-0 items-center gap-1.5">
                            {isPaid
                              ? <CheckCircle className="w-3.5 h-3.5 text-emerald-700 flex-shrink-0" />
                              : <XCircle className="w-3.5 h-3.5 text-destructive flex-shrink-0" />
                            }
                            <span className="text-xs text-muted-foreground">Dòng {detail.row}</span>
                          </div>
                          <Badge
                            variant={isPaid ? "secondary" : "destructive"}
                            className={isPaid ? "bg-emerald-100 text-emerald-800 text-xs" : "text-xs"}
                          >
                            {isPaid ? "Thành công" : "Thất bại"}
                          </Badge>
                        </div>
                        <p className="break-words text-sm font-medium" title={detail.employee_name}>
                          {detail.employee_name}
                        </p>
                        <p className="break-words text-xs text-muted-foreground">
                          {detail.employee_bank} · {detail.employee_account_number}
                        </p>
                        <p className="break-words text-sm font-semibold">{formatCurrencyFromString(detail.amount)}</p>
                        {detail.paid_at && (
                          <p className="text-xs text-muted-foreground">{formatDateTime(detail.paid_at)}</p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            <DialogFooter className="grid shrink-0 grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-3 sm:px-5">
              <Button variant="outline" onClick={handleBackToUpload} className="h-11 w-full">
                Nhập file khác
              </Button>
              <Button variant="outline" onClick={handleDownloadPdf} className="h-11 w-full gap-1.5">
                <FileDown className="w-4 h-4" />
                Tải PDF
              </Button>
              <Button onClick={handleClose} className="h-11 w-full">Đóng</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      );
    }

    // ── Upload view ──
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[92dvh] max-w-md flex-col gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
          <DialogNavyHeader
            title="Kết quả chuyển tiền"
            description="Tải lên file Excel kết quả chuyển tiền để cập nhật trạng thái"
          />

          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-5">
            <FileDropZone
              file={selectedFile}
              onFileChange={setSelectedFile}
              accept=".xls,.xlsx"
              inputId="result-upload-file-input"
            />
          </div>

          <DialogFooter className="grid shrink-0 grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-[minmax(72px,auto)_1fr] sm:px-5">
            <Button variant="outline" onClick={handleClose} disabled={uploadMutation.isPending} className="h-11 w-full">
              Đóng
            </Button>
            <Button onClick={handleUpload} disabled={!canUpload} className="h-11 w-full gap-1.5">
              {uploadMutation.isPending ? (
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
  },
);
