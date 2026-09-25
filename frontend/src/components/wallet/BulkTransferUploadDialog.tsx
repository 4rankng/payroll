/**
 * BulkTransferUploadDialog — Stage 2 upload entry point of the Wallet
 * Bulk Transfer Pipeline.
 *
 * Flow:
 *   select     → user picks the .xlsx exported by Stage 1 (OnePay export)
 *   uploading  → mutation streams upload progress through onProgress
 *   result     → success summary + CTA to open the live progress view
 *
 * The dialog follows the same Dialog/DialogNavyHeader/DialogFooter pattern
 * as BulkTransferResultUploadDialog. We deliberately keep the upload
 * progress UI minimal — the per-row status lives in BulkTransferProgress,
 * which is opened via onViewProgress.
 */
import { memo, useCallback, useEffect, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { FileDropZone } from '@/components/advance-payment/FileDropZone';
import { FileSpreadsheet, Loader2, Upload, CheckCircle2, AlertCircle } from 'lucide-react';
import { useUploadWalletBulkTransfer } from '@/hooks/api/useWalletBulkTransfer';
import { validateFile, formatFileSize } from '@/utils/file-upload';
import { formatCurrency } from '@/utils/formatters';
import type { WalletBulkUploadResponse } from '@/types/wallet-bulk-transfer';

interface BulkTransferUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called when user clicks "Xem tiến độ" on the success screen. */
  onViewProgress?: (batchId: number) => void;
}

type Phase = 'select' | 'uploading' | 'result';

export const BulkTransferUploadDialog = memo(function BulkTransferUploadDialog({
  open,
  onOpenChange,
  onViewProgress,
}: BulkTransferUploadDialogProps) {
  const [file, setFile] = useState<File | null>(null);
  const [phase, setPhase] = useState<Phase>('select');
  const [progress, setProgress] = useState(0);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [result, setResult] = useState<WalletBulkUploadResponse | null>(null);

  const uploadMutation = useUploadWalletBulkTransfer();

  // Reset internal state whenever the dialog is closed so a reopen starts
  // clean (mirrors CreateManualDisbursementDialog).
  useEffect(() => {
    if (!open) {
      const t = setTimeout(() => {
        setFile(null);
        setPhase('select');
        setProgress(0);
        setValidationError(null);
        setResult(null);
        uploadMutation.reset();
      }, 200);
      return () => clearTimeout(t);
    }
  }, [open, uploadMutation]);

  const handleFileChange = useCallback((next: File | null) => {
    setValidationError(null);
    if (!next) {
      setFile(null);
      return;
    }
    // 10 MiB cap — must match backend MaxBulkUploadBytes; validateFile uses
    // FILE_LIMITS.excel (10 MB) by default so we pass it explicitly.
    const v = validateFile(next, {
      maxSize: 10 * 1024 * 1024,
      allowedFormats: ['.xlsx', '.xls'],
    });
    if (!v.isValid) {
      setValidationError(v.error ?? 'File không hợp lệ');
      setFile(null);
      return;
    }
    setFile(next);
  }, []);

  const handleUpload = useCallback(async () => {
    if (!file) return;
    setPhase('uploading');
    setProgress(0);
    try {
      const res = await uploadMutation.mutateAsync({
        file,
        onProgress: (p) => setProgress(p),
      });
      setResult(res);
      setPhase('result');
    } catch {
      // Surface the error inline and let the user retry without closing.
      // The mutation hook already toasts the message via showErrorNotification.
      setPhase('select');
    }
  }, [file, uploadMutation]);

  const handleClose = useCallback(() => {
    if (phase === 'uploading') return; // don't dismiss mid-upload
    onOpenChange(false);
  }, [phase, onOpenChange]);

  const handleViewProgress = useCallback(() => {
    if (result && onViewProgress) {
      onViewProgress(result.batch_id);
    }
    onOpenChange(false);
  }, [result, onViewProgress, onOpenChange]);

  const handleUploadAnother = useCallback(() => {
    setFile(null);
    setResult(null);
    setValidationError(null);
    setProgress(0);
    setPhase('select');
    uploadMutation.reset();
  }, [uploadMutation]);

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(true) : handleClose())}>
      <DialogContent
        className="flex max-h-[92dvh] max-w-2xl flex-col gap-0 overflow-hidden"
        contentPadding="none"
        hideCloseButton
      >
        <DialogNavyHeader
          title={
            <span className="flex items-center gap-2">
              <Upload className="h-4 w-4 text-white/70" />
              Tải File
            </span>
          }
          description="Tải lên file Yêu cầu chuyển tiền (.xlsx) đã xuất từ Bảng công để xử lý."
        />

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-5 sm:px-6">
          {phase === 'select' && (
            <>
              <FileDropZone
                file={file}
                onFileChange={handleFileChange}
                accept=".xlsx,.xls"
                inputId="bulk-transfer-upload-input"
              />
              {validationError && (
                <p className="flex items-center gap-1.5 text-xs text-destructive">
                  <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                  {validationError}
                </p>
              )}
              {file && (
                <div className="rounded-xl border border-border/60 bg-muted/40 p-3">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <FileSpreadsheet className="h-3.5 w-3.5 text-primary" />
                    <span>{formatFileSize(file.size)}</span>
                  </div>
                </div>
              )}
              <p className="text-xs leading-snug text-muted-foreground">
                File tối đa 10MB. Hệ thống sẽ tạo một lô chuyển tiền và xử lý từng dòng
                bất đồng bộ qua OnePay.
              </p>
            </>
          )}

          {phase === 'uploading' && (
            <div className="space-y-4 py-4">
              <div className="flex items-center gap-3">
                <Loader2 className="h-5 w-5 animate-spin text-primary" />
                <div className="min-w-0">
                  <p className="text-sm font-medium text-foreground">
                    Đang tải lên file...
                  </p>
                  <p className="truncate text-xs text-muted-foreground">{file?.name}</p>
                </div>
              </div>
              <Progress value={progress} className="h-2" />
              <p className="text-right text-xs tabular-nums text-muted-foreground">
                {progress}%
              </p>
            </div>
          )}

          {phase === 'result' && result && (
            <div className="space-y-4 py-2">
              <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-3">
                <CheckCircle2 className="h-5 w-5 shrink-0 text-emerald-700" />
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-emerald-900">
                    Đã tải lên lô chuyển tiền thành công
                  </p>
                  <p className="text-xs text-emerald-800">
                    Lô #{result.batch_id} đang được xử lý.
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-2 rounded-xl border border-border/60 bg-muted/40 p-3 sm:grid-cols-3">
                <SummaryStat
                  label="Tổng số dòng"
                  value={result.total_count.toLocaleString('vi-VN')}
                />
                <SummaryStat
                  label="Tổng tiền"
                  value={formatCurrency(result.transfer_amount)}
                />
                <SummaryStat
                  label="Phí ước tính"
                  value={formatCurrency(result.estimated_fee_total)}
                />
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="grid shrink-0 grid-cols-1 gap-2 border-t px-4 py-4 sm:grid-cols-2 sm:gap-4 sm:px-6">
          {phase === 'select' && (
            <>
              <Button variant="outline" onClick={handleClose} className="h-11 w-full">
                Đóng
              </Button>
              <Button
                onClick={handleUpload}
                disabled={!file || uploadMutation.isPending}
                className="h-11 w-full"
              >
                {uploadMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Đang xử lý...
                  </>
                ) : (
                  <>
                    <Upload className="h-4 w-4" />
                    Tải File
                  </>
                )}
              </Button>
            </>
          )}

          {phase === 'uploading' && (
            <Button disabled className="h-11 w-full sm:col-span-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Đang tải lên...
            </Button>
          )}

          {phase === 'result' && (
            <>
              <Button
                variant="outline"
                onClick={handleUploadAnother}
                className="h-11 w-full"
              >
                Tải file khác
              </Button>
              {onViewProgress ? (
                <Button onClick={handleViewProgress} className="h-11 w-full">
                  Xem tiến độ
                </Button>
              ) : (
                <Button onClick={handleClose} className="h-11 w-full">
                  Đóng
                </Button>
              )}
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});

interface SummaryStatProps {
  label: string;
  value: string;
}

function SummaryStat({ label, value }: SummaryStatProps) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {label}
      </p>
      <p className="mt-0.5 break-words font-financial text-sm font-bold tabular-nums text-foreground">
        {value}
      </p>
    </div>
  );
}

export default BulkTransferUploadDialog;
