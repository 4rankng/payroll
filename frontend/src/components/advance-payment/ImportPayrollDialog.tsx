import { useState, useCallback, useEffect, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
  DialogFooter,
} from "@/components/ui/dialog";
import { FileDropZone } from "./FileDropZone";
import { useImportFlexTemplate } from "@/hooks/api/useAdvancePayments";
import { formatMonthDisplay } from "@/utils/advancePaymentHelpers";
import { FileSpreadsheet, Loader2, CalendarDays, CheckCircle2, Users, FolderKanban, Link } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ImportJobStatus, ImportJobStatusResponse } from "@/types/api/advance-payment.types";

interface ImportPayrollDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const MONTH_OPTIONS = (() => {
  const options: { value: string; label: string }[] = [];
  const now = new Date();
  for (let i = 0; i < 12; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    options.push({ value, label: formatMonthDisplay(value) });
  }
  return options;
})();

export function ImportPayrollDialog({ open, onOpenChange }: ImportPayrollDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [selectedMonth, setSelectedMonth] = useState<string>("");
  const [importProgress, setImportProgress] = useState<number>(0);
  const [importStatus, setImportStatus] = useState<ImportJobStatus | null>(null);
  const [importResult, setImportResult] = useState<ImportJobStatusResponse | null>(null);

  const monthOptions = useMemo(() => MONTH_OPTIONS, []);

  useEffect(() => {
    if (open && !selectedMonth && monthOptions.length > 0) {
      setSelectedMonth(monthOptions[0].value);
    }
  }, [open, selectedMonth, monthOptions]);

  const importTemplateMutation = useImportFlexTemplate({
    onProgress: (percentage) => setImportProgress(percentage),
    onStatusChange: (status) => setImportStatus(status),
  });

  const resetState = useCallback(() => {
    setSelectedFile(null);
    setSelectedMonth("");
    setImportProgress(0);
    setImportStatus(null);
    setImportResult(null);
  }, []);

  const handleImport = useCallback(() => {
    if (!selectedFile || !selectedMonth) return;
    setImportProgress(0);
    setImportStatus("pending");
    setImportResult(null);
    const formData = new FormData();
    formData.append("file", selectedFile);
    formData.append("forMonth", selectedMonth);
    importTemplateMutation.mutate(formData, {
      onSuccess: (response) => {
        setImportResult(response);
      },
      onError: () => {
        onOpenChange(false);
        resetState();
      },
    });
  }, [selectedFile, selectedMonth, importTemplateMutation, onOpenChange, resetState]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    resetState();
  }, [onOpenChange, resetState]);

  const statusLabel = importStatus === "pending"
    ? "Đang chờ xử lý..."
    : importStatus === "processing"
      ? "Đang xử lý..."
      : "Đang nhập...";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md p-0 gap-0 overflow-hidden" hideCloseButton>
        <DialogNavyHeader
          title="Nhập bảng lương"
          description="Nhập file Excel chứa thông tin ứng lương"
        />

        {/* Success state */}
        {importResult ? (
          <>
            <div className="px-5 py-8 flex flex-col items-center gap-5">
              <div className="flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30">
                <CheckCircle2 className="w-8 h-8 text-green-600 dark:text-green-400" />
              </div>
              <div className="text-center space-y-1">
                <p className="font-semibold text-base">Nhập file thành công!</p>
                <p className="text-sm text-muted-foreground">
                  {importResult.filename} · {formatMonthDisplay(importResult.forMonth)}
                </p>
              </div>
              <div className="w-full rounded-xl border bg-muted/30 px-4 py-3 space-y-3">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Tổng số dòng</span>
                  <span className="font-semibold tabular-nums">{importResult.processedRows} / {importResult.totalRows}</span>
                </div>
                {importResult.result && (
                  <>
                    <div className="border-t border-dashed border-border/50" />
                    <div className="grid grid-cols-2 gap-2">
                      <div className="flex items-center gap-2 text-sm">
                        <Users className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Nhân viên mới</span>
                        <span className="ml-auto font-semibold tabular-nums text-green-600 dark:text-green-400">
                          +{importResult.result.employeesCreated}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <Users className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Bỏ qua</span>
                        <span className="ml-auto font-semibold tabular-nums text-muted-foreground">
                          {importResult.result.employeesSkipped}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <FolderKanban className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Dự án mới</span>
                        <span className="ml-auto font-semibold tabular-nums text-green-600 dark:text-green-400">
                          +{importResult.result.projectsCreated}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <FolderKanban className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Bỏ qua</span>
                        <span className="ml-auto font-semibold tabular-nums text-muted-foreground">
                          {importResult.result.projectsSkipped}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <Link className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Phân công mới</span>
                        <span className="ml-auto font-semibold tabular-nums text-green-600 dark:text-green-400">
                          +{importResult.result.assignmentsCreated}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                        <Link className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        <span className="text-muted-foreground">Bỏ qua</span>
                        <span className="ml-auto font-semibold tabular-nums text-muted-foreground">
                          {importResult.result.assignmentsSkipped}
                        </span>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </div>
            <DialogFooter className="px-5 py-4 border-t">
              <Button onClick={handleClose} className="w-full">
                Đóng
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            {/* Body */}
            <div className="px-5 py-4 space-y-5">
              {/* Month selector */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CalendarDays className="h-4 w-4 text-muted-foreground" />
                  <p className="text-sm font-semibold">Tháng ứng lương</p>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {monthOptions.slice(0, 2).map((month) => (
                    <button
                      key={month.value}
                      type="button"
                      onClick={() => setSelectedMonth(month.value)}
                      className={cn(
                        "h-8 px-3 rounded-xl text-sm font-medium border transition-colors",
                        selectedMonth === month.value
                          ? "bg-primary text-primary-foreground border-primary"
                          : "bg-background text-foreground border-border hover:bg-muted"
                      )}
                    >
                      {month.label}
                    </button>
                  ))}
                </div>
              </div>

              <div className="border-t border-dashed border-border/60" />

              {/* File upload */}
              <div className="space-y-2">
                <p className="text-sm font-semibold">File Excel</p>
                <div className="rounded-xl border bg-muted/30 px-4 py-3 space-y-2">
                  <p className="text-xs font-medium text-foreground">Định dạng cột:</p>
                  <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs">
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">B</span><span className="text-muted-foreground">CCCD</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">C</span><span className="text-muted-foreground">Dự án / Khách hàng</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">D</span><span className="text-muted-foreground">Vị trí</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">E</span><span className="text-muted-foreground">Số điện thoại</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">F</span><span className="text-muted-foreground">Họ và tên</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">G</span><span className="text-muted-foreground">Tên tài khoản NH</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">H</span><span className="text-muted-foreground">Số tài khoản NH</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">I</span><span className="text-muted-foreground">Tên ngân hàng</span></div>
                    <div className="flex gap-2"><span className="font-semibold text-primary w-4">J</span><span className="text-muted-foreground">Hạn mức được ứng <span className="text-muted-foreground/60">(để trống hoặc 0 = chỉ nhập NV)</span></span></div>
                  </div>
                </div>
                <FileDropZone
                  file={selectedFile}
                  onFileChange={setSelectedFile}
                  inputId="import-file-input"
                />
              </div>

              {/* Progress */}
              {importTemplateMutation.isPending && (
                <div className="space-y-2 px-3 py-2.5 rounded-xl bg-muted/50 border">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">{statusLabel}</span>
                    <span className="font-medium tabular-nums">{importProgress}%</span>
                  </div>
                  <Progress value={importProgress} className="h-1.5" />
                </div>
              )}
            </div>

            {/* Footer */}
            <DialogFooter className="px-5 py-4 border-t flex flex-row gap-2">
              <Button
                variant="outline"
                onClick={handleClose}
                disabled={importTemplateMutation.isPending}
                className="min-w-[72px]"
              >
                Hủy
              </Button>
              <Button
                onClick={handleImport}
                disabled={!selectedFile || !selectedMonth || importTemplateMutation.isPending}
                className="flex-1 gap-1.5"
              >
                {importTemplateMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Đang xử lý...
                  </>
                ) : (
                  <>
                    <FileSpreadsheet className="w-4 h-4" />
                    Nhập
                  </>
                )}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
