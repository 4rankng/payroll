import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  MoreHorizontal,
  FileSpreadsheet,
  FileDown,
  ArrowRightLeft,
  Users,
  Loader2,
  History,
  Mail,
  FileText,
} from "lucide-react";

interface AdvancePaymentPageHeaderMobileProps {
  onImportPayroll: () => void;
  onExportBatch: () => void;
  isExportBatchPending?: boolean;
  onUploadResult: () => void;
  onFileHistory: () => void;
  onViewEmployees: () => void;
  onStatement?: () => void;
  onEmail?: () => void;
  isEmailPending?: boolean;
  hideUploadResult?: boolean;
  hideFileHistory?: boolean;
}

export function AdvancePaymentPageHeaderMobile({
  onImportPayroll,
  onExportBatch,
  isExportBatchPending = false,
  onUploadResult,
  onFileHistory,
  onViewEmployees,
  onStatement,
  onEmail,
  isEmailPending = false,
  hideUploadResult = false,
  hideFileHistory = false,
}: AdvancePaymentPageHeaderMobileProps) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);

  return (
    <div className="flex items-center justify-between gap-2">
      <div className="min-w-0">
        <h1 className="text-base font-semibold text-foreground tracking-tight leading-snug">Ứng lương</h1>
        <p className="text-xs text-muted-foreground mt-0.5">
          Quản lý yêu cầu ứng lương
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Button
          onClick={onImportPayroll}
          size="sm"
          className="h-9 px-3 touch-manipulation"
        >
          <FileSpreadsheet className="h-4 w-4 mr-1" />
          Nhập
        </Button>

        <Button
          variant="outline"
          size="sm"
          className="h-9 px-3 touch-manipulation"
          onClick={onViewEmployees}
          aria-label="Danh sách nhân viên"
        >
          <Users className="h-4 w-4" />
        </Button>

        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="h-9 w-9 touch-manipulation"
              aria-label="Tùy chọn"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </SheetTrigger>
          <SheetContent side="top" className="h-auto px-4 pb-3" style={{ paddingTop: "max(16px, calc(16px + env(safe-area-inset-top)))" }}>
            <SheetHeader className="mb-2">
              <SheetTitle>Tùy chọn</SheetTitle>
            </SheetHeader>

            <div className="space-y-0.5">
              {/* ── Transfer group ── */}
              <Button
                variant="ghost"
                className="w-full justify-start gap-3 h-12 px-2"
                onClick={() => { onExportBatch(); close(); }}
                disabled={isExportBatchPending}
              >
                {isExportBatchPending
                  ? <Loader2 className="h-5 w-5 text-muted-foreground shrink-0 animate-spin" />
                  : <FileDown className="h-5 w-5 text-muted-foreground shrink-0" />
                }
                <span className="text-sm font-medium">Chuyển lô</span>
              </Button>

              {!hideUploadResult && (
                <Button
                  variant="ghost"
                  className="w-full justify-start gap-3 h-12 px-2"
                  onClick={() => { onUploadResult(); close(); }}
                >
                  <ArrowRightLeft className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Nhập KQ</span>
                </Button>
              )}

              {onStatement && (
                <Button
                  variant="ghost"
                  className="w-full justify-start gap-3 h-12 px-2"
                  onClick={() => { onStatement(); close(); }}
                >
                  <FileText className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Sao kê</span>
                </Button>
              )}

              {onEmail && (
                <Button
                  variant="ghost"
                  className="w-full justify-start gap-3 h-12 px-2"
                  onClick={() => { onEmail(); close(); }}
                  disabled={isEmailPending}
                >
                  {isEmailPending
                    ? <Loader2 className="h-5 w-5 text-muted-foreground shrink-0 animate-spin" />
                    : <Mail className="h-5 w-5 text-muted-foreground shrink-0" />
                  }
                  <span className="text-sm font-medium">Email sao kê</span>
                </Button>
              )}

              <div className="h-px bg-border mx-2 my-1" />

              {/* ── History ── */}
              {!hideFileHistory && (
                <Button
                  variant="ghost"
                  className="w-full justify-start gap-3 h-12 px-2"
                  onClick={() => { onFileHistory(); close(); }}
                >
                  <History className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Lịch sử file</span>
                </Button>
              )}
            </div>

            {/* Drag handle at bottom signals the sheet can be dismissed */}
            <div className="flex justify-center pt-3 pb-1">
              <div className="w-10 h-1 rounded-full bg-muted-foreground/25" />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  );
}
