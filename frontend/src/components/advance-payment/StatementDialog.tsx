import { useState, useCallback, useMemo, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
  DialogFooter,
} from "@/components/ui/dialog";
import { FileDropZone } from "./FileDropZone";
import { useUploadAndSettleReconciliation } from "@/hooks/api/useAdvancePaymentReconciliation";
import { apiClient } from "@/services/api/client";
import { API_ENDPOINTS } from "@/config/api.config";
import { showErrorNotification } from "@/utils/error-handler";
import { formatMonthDisplay } from "@/utils/advancePaymentHelpers";
import { Download, ScanLine, Loader2, CalendarDays } from "lucide-react";
import { cn } from "@/lib/utils";

interface StatementDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  forMonth?: string;
  downloadOnly?: boolean;
}

export function StatementDialog({ open, onOpenChange, forMonth: forMonthProp, downloadOnly = false }: StatementDialogProps) {
  const [selectedStatementMonth, setSelectedStatementMonth] = useState<string>("");
  const [selectedStatementFile, setSelectedStatementFile] = useState<File | null>(null);
  const uploadAndSettleMutation = useUploadAndSettleReconciliation();

  const monthOptions = useMemo(() => {
    const options: { value: string; label: string }[] = [];
    const now = new Date();
    for (let i = 0; i < 12; i++) {
      const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
      const value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
      options.push({ value, label: formatMonthDisplay(value) });
    }
    return options;
  }, []);

  useEffect(() => {
    if (open && monthOptions.length > 0) {
      if (forMonthProp) {
        setSelectedStatementMonth(forMonthProp);
      } else if (!selectedStatementMonth) {
        const currentDay = new Date().getDate();
        const defaultMonthIndex = currentDay <= 8 ? 1 : 0;
        setSelectedStatementMonth(monthOptions[defaultMonthIndex]?.value || "");
      }
    }
  }, [open, forMonthProp, selectedStatementMonth, monthOptions]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setSelectedStatementMonth("");
    setSelectedStatementFile(null);
  }, [onOpenChange]);

  const handleUpload = useCallback(() => {
    if (!selectedStatementFile || !selectedStatementMonth) return;
    const formData = new FormData();
    formData.append("file", selectedStatementFile);
    formData.append("forMonth", selectedStatementMonth);
    uploadAndSettleMutation.mutate(formData, {
      onSettled: () => {
        onOpenChange(false);
        setSelectedStatementMonth("");
        setSelectedStatementFile(null);
      },
    });
  }, [selectedStatementFile, selectedStatementMonth, uploadAndSettleMutation, onOpenChange]);

  const handleDownload = useCallback(() => {
    if (!selectedStatementMonth) return;
    apiClient.download(
      `${API_ENDPOINTS.advancePayments.reconciliation.export}?forMonth=${selectedStatementMonth}`,
      `sao_ke_thanh_toan_${selectedStatementMonth}.xlsx`,
    ).catch((error) => { showErrorNotification(error); });
  }, [selectedStatementMonth]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
        <DialogNavyHeader
          title="Đối soát & Sao kê"
          description="Tải xuống hoặc tải lên file sao kê thanh toán"
        />

        {/* Body */}
        <div className="px-5 py-4 space-y-5">
          {/* Month selector */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <CalendarDays className="h-4 w-4 text-muted-foreground" />
              <p className="text-sm font-semibold">Chọn tháng</p>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {monthOptions.slice(0, 4).map((month) => (
                <button
                  key={month.value}
                  type="button"
                  onClick={() => setSelectedStatementMonth(month.value)}
                  className={cn(
                    "h-8 px-3 rounded-xl text-sm font-medium border transition-colors",
                    selectedStatementMonth === month.value
                      ? "bg-primary text-primary-foreground border-primary"
                      : "bg-card text-foreground border-border hover:bg-muted"
                  )}
                >
                  {month.label}
                </button>
              ))}
            </div>
          </div>

          <div className="border-t border-dashed border-border/60" />

          {/* Download action */}
          <div className="flex items-center justify-between gap-3 py-0.5">
            <div>
              <p className="text-sm font-medium">Tải xuống sao kê</p>
              <p className="text-xs text-muted-foreground mt-0.5">Xuất file Excel cho tháng đã chọn</p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleDownload}
              disabled={!selectedStatementMonth}
              className="gap-1.5 flex-shrink-0"
            >
              <Download className="h-3.5 w-3.5" />
              Tải xuống
            </Button>
          </div>

          {!downloadOnly && (
            <>
              <div className="border-t border-dashed border-border/60" />

              <div className="space-y-2">
                <div>
                  <p className="text-sm font-medium">Tải lên file đối soát</p>
                  <p className="text-xs text-muted-foreground mt-0.5">.PDF, .XLSX, .XLS</p>
                </div>
                <FileDropZone
                  file={selectedStatementFile}
                  onFileChange={setSelectedStatementFile}
                  accept=".pdf,.xlsx,.xls"
                  inputId="statement-file-input"
                />
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <DialogFooter className="px-5 py-4 border-t flex flex-row gap-2">
          <Button variant="outline" onClick={handleClose} className="min-w-[72px]">
            Đóng
          </Button>
          {!downloadOnly && (
            <Button
              onClick={handleUpload}
              disabled={!selectedStatementFile || !selectedStatementMonth || uploadAndSettleMutation.isPending}
              className="flex-1 gap-1.5"
            >
              {uploadAndSettleMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <ScanLine className="w-4 h-4" />
                  Đối soát & Xử lý
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
