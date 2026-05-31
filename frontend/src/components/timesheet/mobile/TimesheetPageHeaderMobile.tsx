import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Plus, MoreHorizontal, FileText, ArrowRightLeft, History, Upload, FileDown, CheckSquare, X, FileSpreadsheet, FileUp } from 'lucide-react';

interface TimesheetPageHeaderMobileProps {
  onAddTimesheet: () => void;
  onApprovedTimesheetsExport: () => void;
  onPayrollReportExport: () => void;
  // Admin-only
  onBulkTransferExport?: () => void;
  onBulkTransferResultUpload?: () => void;
  onBulkTransferHistory?: () => void;
  onBulkApprove?: () => void;
  onBccHistory?: () => void;
  onBccUpload?: () => void;
  // Partner-only
  onPaymentHistory?: () => void;
  isApprovedExportPending?: boolean;
  isPayrollReportPending?: boolean;
  userRole?: 'admin' | 'partner';
}

export function TimesheetPageHeaderMobile({
  onAddTimesheet,
  onApprovedTimesheetsExport,
  onPayrollReportExport,
  onBulkTransferExport,
  onBulkTransferResultUpload,
  onBulkTransferHistory,
  onBulkApprove,
  onBccHistory,
  onBccUpload,
  onPaymentHistory,
  isApprovedExportPending = false,
  isPayrollReportPending = false,
  userRole = 'admin',
}: TimesheetPageHeaderMobileProps) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);

  const description = userRole === 'partner'
    ? 'Theo dõi và quản lý bảng công'
    : 'Theo dõi và duyệt bảng công';

  return (
    <div className="flex items-center justify-between gap-2">
      <div className="min-w-0">
        <h1 className="text-base font-semibold text-foreground tracking-tight leading-snug">Bảng công</h1>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Button onClick={onAddTimesheet} className="touch-manipulation">
          <Plus />
          Nhập
        </Button>

        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="icon" className="touch-manipulation" aria-label="Thêm tùy chọn">
              <MoreHorizontal />
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="h-auto">
            <SheetHeader><SheetTitle>Tùy chọn</SheetTitle></SheetHeader>
            <div className="space-y-1 py-3" style={{ paddingBottom: "max(12px, calc(12px + env(safe-area-inset-bottom)))" }}>

              {/* Exports */}
              <Button
                variant="ghost"
                className="w-full justify-start h-auto px-2 py-3"
                onClick={() => { onApprovedTimesheetsExport(); close(); }}
                disabled={isApprovedExportPending}
              >
                <FileText className="h-5 w-5 text-muted-foreground shrink-0" />
                <span className="text-sm font-medium">{isApprovedExportPending ? 'Đang xuất...' : 'Xuất bảng công đã duyệt'}</span>
              </Button>

              <Button
                variant="ghost"
                className="w-full justify-start h-auto px-2 py-3"
                onClick={() => { onPayrollReportExport(); close(); }}
                disabled={isPayrollReportPending}
              >
                <FileDown className="h-5 w-5 text-muted-foreground shrink-0" />
                <span className="text-sm font-medium">{isPayrollReportPending ? 'Đang xuất...' : 'Xuất sao kê lương'}</span>
              </Button>

              {onBulkTransferExport && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onBulkTransferExport(); close(); }}
                >
                  <ArrowRightLeft className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Xuất file chuyển lô</span>
                </Button>
              )}

              {/* Imports / uploads */}
              {onBulkTransferResultUpload && (
                <>
                  <div className="h-px bg-border mx-1 my-1" />
                  <Button
                    variant="ghost"
                    className="w-full justify-start h-auto px-2 py-3"
                    onClick={() => { onBulkTransferResultUpload(); close(); }}
                  >
                    <Upload className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Nhập KQ chuyển lô</span>
                  </Button>
                </>
              )}

              {/* Actions */}
              {(onBulkApprove || onBulkTransferHistory || onPaymentHistory) && (
                <div className="h-px bg-border mx-1 my-1" />
              )}

              {onBulkApprove && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onBulkApprove(); close(); }}
                >
                  <CheckSquare className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Duyệt hết</span>
                </Button>
              )}

              {onBulkTransferHistory && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onBulkTransferHistory(); close(); }}
                >
                  <History className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Lịch sử chuyển lô</span>
                </Button>
              )}

              {onBccUpload && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onBccUpload(); close(); }}
                >
                  <FileUp className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Tải lên BCC</span>
                </Button>
              )}

              {onBccHistory && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onBccHistory(); close(); }}
                >
                  <FileSpreadsheet className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Lịch sử BCC</span>
                </Button>
              )}

              {onPaymentHistory && (
                <Button
                  variant="ghost"
                  className="w-full justify-start h-auto px-2 py-3"
                  onClick={() => { onPaymentHistory(); close(); }}
                >
                  <History className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Lịch sử trả lương</span>
                </Button>
              )}

              {/* Close button at bottom */}
              <div className="h-px bg-border mx-1 my-1" />
              <Button
                variant="ghost"
                className="w-full justify-start h-auto px-2 py-3 text-muted-foreground"
                onClick={close}
              >
                <X className="h-5 w-5 shrink-0" />
                <span className="text-sm font-medium">Đóng</span>
              </Button>
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  );
}
