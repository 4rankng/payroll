import { useState } from 'react';
import { AlertCircle, CheckCircle2, Loader2, ReceiptText } from 'lucide-react';
import { FileDropZone } from '@/components/advance-payment/FileDropZone';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { OnePayFeeImportResponse, OnePayFeeReportIssue } from '@/types/api/financial.types';
import { formatCurrency } from '@/utils/formatters';

interface OnePayFeeReportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpload: (file: File) => Promise<void>;
  isUploading: boolean;
  result: OnePayFeeImportResponse | null;
  issues: OnePayFeeReportIssue[];
}

export function OnePayFeeReportDialog({
  open,
  onOpenChange,
  onUpload,
  isUploading,
  result,
  issues,
}: OnePayFeeReportDialogProps) {
  const [file, setFile] = useState<File | null>(null);

  const handleSubmit = async () => {
    if (!file || result) return;
    await onUpload(file);
  };

  const hasIssues = issues.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl" title="Nhập phí OnePay" description="Tải file đối soát phí tháng của OnePay">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ReceiptText className="h-5 w-5" />
            Nhập phí OnePay
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <FileDropZone
            file={file}
            onFileChange={setFile}
            accept=".xlsx,.xls"
            inputId="onepay-fee-report-file"
          />

          {hasIssues && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Dữ liệu chưa khớp</AlertTitle>
              <AlertDescription>
                <div className="mt-2 max-h-44 space-y-2 overflow-auto pr-1">
                  {issues.map((issue, index) => (
                    <div key={`${issue.code}-${issue.row ?? 0}-${index}`} className="text-sm">
                      {issue.row ? <span className="font-medium">Dòng {issue.row}: </span> : null}
                      {issue.reference ? <span className="font-mono text-xs">{issue.reference} · </span> : null}
                      {issue.message}
                    </div>
                  ))}
                </div>
              </AlertDescription>
            </Alert>
          )}

          {result && (
            <Alert className="border-emerald-200 bg-emerald-50 text-emerald-900">
              <CheckCircle2 className="h-4 w-4 text-emerald-700" />
              <AlertTitle>Đã tạo chi phí</AlertTitle>
              <AlertDescription>
                <div className="mt-2 grid grid-cols-2 gap-2 text-sm">
                  <span className="text-emerald-700">Kỳ</span>
                  <span className="font-medium">{result.summary.period_label}</span>
                  <span className="text-emerald-700">SLGD</span>
                  <span className="font-medium">{result.summary.transaction_count}</span>
                  <span className="text-emerald-700">Tổng phí</span>
                  <span className="font-medium">{formatCurrency(result.summary.total_fee)}</span>
                  <span className="text-emerald-700">Giao dịch</span>
                  <span className="font-medium">#{result.transaction_id}</span>
                </div>
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Đóng
          </Button>
          {!result && (
            <Button onClick={handleSubmit} disabled={!file || isUploading}>
              {isUploading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              Đối soát và tạo chi phí
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
