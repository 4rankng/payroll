import {
  FileSpreadsheet,
  FileDown,
  ArrowRightLeft,
  Loader2,
  History,
  Download,
  ScanFace,
} from "lucide-react";

interface AdvancePaymentActionButtonsProps {
  onImportPayroll: () => void;
  onExportBatch: () => void;
  onUploadResult: () => void;
  isExportBatchPending?: boolean;
  onHistory: () => void;
  hideUploadResult?: boolean;
  hideHistory?: boolean;
  hideExportBatch?: boolean;
  onExportList?: () => void;
  isExportListPending?: boolean;
  onCheckIn?: () => void;
}

export function AdvancePaymentActionButtons({
  onImportPayroll,
  onExportBatch,
  onUploadResult,
  isExportBatchPending = false,
  onHistory,
  hideUploadResult = false,
  hideHistory = false,
  hideExportBatch = false,
  onExportList,
  isExportListPending = false,
  onCheckIn,
}: AdvancePaymentActionButtonsProps) {
  return (
    <div className="flex flex-wrap gap-1.5 items-center">
      {/* ── Import group ── */}
      <div className="flex items-center gap-px rounded-xl border border-border overflow-hidden shadow-sm hover:shadow-md transition-shadow">
        <button onClick={onImportPayroll} className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all">
          <FileSpreadsheet className="h-4 w-4 shrink-0" />
          <span className="hidden sm:inline">Nhập bảng lương</span>
          <span className="sm:hidden">Nhập BL</span>
        </button>
        {onExportList && (
          <button onClick={onExportList} disabled={isExportListPending} className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all border-l border-border disabled:opacity-50 disabled:pointer-events-none">
            {isExportListPending
              ? <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
              : <Download className="h-4 w-4 shrink-0" />
            }
            <span className="hidden sm:inline">Xuất DS</span>
            <span className="sm:hidden">Xuất</span>
          </button>
        )}
      </div>

      {/* ── Transfer group ── */}
      {!hideExportBatch && (
        <div className="flex items-center gap-px rounded-xl border border-border overflow-hidden shadow-sm hover:shadow-md transition-shadow">
          <button onClick={onExportBatch} disabled={isExportBatchPending} className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all disabled:opacity-50 disabled:pointer-events-none">
            {isExportBatchPending
              ? <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
              : <FileDown className="h-4 w-4 shrink-0" />
            }
            Chuyển lô
          </button>
          {!hideUploadResult && (
            <button onClick={onUploadResult} className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all border-l border-border">
              <ArrowRightLeft className="h-4 w-4 shrink-0" />
              Nhập KQ
            </button>
          )}
        </div>
      )}

      {/* ── Check-in config button ── */}
      {onCheckIn && (
        <button onClick={onCheckIn} className="inline-flex items-center gap-1.5 h-9 px-4 rounded-xl border border-border bg-card/90 backdrop-blur-sm shadow-sm hover:shadow-md text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all">
          <ScanFace className="h-4 w-4 shrink-0" />
          <span className="hidden sm:inline">Điểm danh</span>
          <span className="sm:hidden">ĐD</span>
        </button>
      )}

      {/* ── History button ── */}
      {!hideHistory && (
        <button onClick={onHistory} className="inline-flex items-center gap-1.5 h-9 px-4 rounded-xl border border-border bg-card/90 backdrop-blur-sm shadow-sm hover:shadow-md text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all">
          <History className="h-4 w-4 shrink-0" />
          <span className="hidden sm:inline">Lịch sử file</span>
          <span className="sm:hidden">Lịch sử</span>
        </button>
      )}
    </div>
  );
}
