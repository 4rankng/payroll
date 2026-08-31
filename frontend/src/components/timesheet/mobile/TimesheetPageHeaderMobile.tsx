import { useState } from 'react';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { Plus, MoreHorizontal, FileText, ArrowRightLeft, History, Upload, FileDown, CheckSquare, X, FileSpreadsheet, FileUp, FileSpreadsheet as TableIcon, Banknote, Trash2, Undo2 } from 'lucide-react';

interface TimesheetPageHeaderMobileProps {
  onAddTimesheet: () => void;
  onApprovedTimesheetsExport: () => void;
  onPayrollReportExport?: () => void;
  // Admin-only
  onBulkTransferExport?: () => void;
  onOnePayExport?: () => void;
  onBulkTransferResultUpload?: () => void;
  onBulkTransferHistory?: () => void;
  onBulkApprove?: () => void;
  /** Admin-only "Bỏ duyệt hết" — reset all approved timesheets to pending. */
  onResetAll?: () => void;
  /** Unified "Chuyển lô" batch-transfer flow — admin only. */
  onChuyenLo?: () => void;
  onBccHistory?: () => void;
  onBccUpload?: () => void;
  onRejectUnpaid?: () => void;
  // Partner-only
  onPaymentHistory?: () => void;
  isApprovedExportPending?: boolean;
  isPayrollReportPending?: boolean;
  isOnePayExportPending?: boolean;
  userRole?: 'admin' | 'partner';
  monthValue: string;
  onMonthChange: (value: string) => void;
}

export function TimesheetPageHeaderMobile({
  onAddTimesheet,
  onApprovedTimesheetsExport,
  onPayrollReportExport,
  onBulkTransferExport,
  onOnePayExport,
  onBulkTransferResultUpload,
  onBulkTransferHistory,
  onBulkApprove,
  onResetAll,
  onChuyenLo,
  onBccHistory,
  onBccUpload,
  onRejectUnpaid,
  onPaymentHistory,
  isApprovedExportPending = false,
  isPayrollReportPending = false,
  isOnePayExportPending = false,
  userRole = 'admin',
  monthValue,
  onMonthChange,
}: TimesheetPageHeaderMobileProps) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);

  const description = userRole === 'partner'
    ? 'Theo dõi và quản lý bảng công'
    : 'Theo dõi và duyệt bảng công';

  return (
    <section className="ct-card overflow-hidden rounded-2xl border border-border/80 bg-card shadow-[0_12px_32px_-28px_hsl(var(--foreground)/0.45)]">
      <MobilePageHeader
        title="Bảng công"
        subtitle={description}
        icon={TableIcon}
        sticky={false}
        bordered={false}
        className="bg-transparent"
        actionsLayout="stacked"
        actions={
          <div className="flex w-full min-w-0 items-center gap-2 sm:w-auto">
            <Button
              onClick={onAddTimesheet}
              className="h-11 min-w-0 flex-1 rounded-xl px-4 shadow-sm touch-manipulation sm:flex-none"
            >
              <Plus className="mr-1 h-4 w-4" />
              Nhập
            </Button>

            {onChuyenLo && userRole === 'admin' && (
              <Button
                variant="outline"
                onClick={onChuyenLo}
                className="h-11 min-w-0 flex-[1.25] rounded-xl border-primary/20 bg-primary/[0.04] px-3 text-primary shadow-none hover:bg-primary/[0.08] hover:text-primary touch-manipulation sm:flex-none"
              >
                <ArrowRightLeft className="mr-1 h-4 w-4" />
                Chuyển lô
              </Button>
            )}

            <Sheet open={open} onOpenChange={setOpen}>
              <SheetTrigger asChild>
                <Button variant="outline" size="icon" className="h-11 w-11 shrink-0 rounded-xl border-border bg-background shadow-none touch-manipulation" aria-label="Thêm tùy chọn">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </SheetTrigger>
              <SheetContent side="bottom" className="h-auto max-h-[80dvh] overflow-y-auto rounded-t-3xl border-[hsl(var(--surface-border))] bg-white px-4 pt-3">
              <div className="mx-auto mb-3 h-1 w-10 rounded-full bg-slate-300" />
              <SheetHeader><SheetTitle>Tùy chọn</SheetTitle></SheetHeader>
              <div className="space-y-1 py-3" style={{ paddingBottom: "max(12px, calc(12px + env(safe-area-inset-bottom)))" }}>

                {/* Exports */}
                <Button
                  variant="ghost"
                  className="w-full min-h-11 justify-start h-auto px-2 py-3"
                  onClick={() => { onApprovedTimesheetsExport(); close(); }}
                  disabled={isApprovedExportPending}
                >
                  <FileText className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">{isApprovedExportPending ? 'Đang xuất...' : 'Xuất bảng công đã duyệt'}</span>
                </Button>

                {onPayrollReportExport && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onPayrollReportExport(); close(); }}
                    disabled={isPayrollReportPending}
                  >
                    <FileDown className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">
                      {isPayrollReportPending
                        ? 'Đang xuất...'
                        : userRole === 'partner'
                          ? 'Xuất sao kê'
                          : 'Xuất sao kê lương'}
                    </span>
                  </Button>
                )}

                {onBulkTransferExport && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBulkTransferExport(); close(); }}
                  >
                    <ArrowRightLeft className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Xuất file chuyển lô</span>
                  </Button>
                )}

                {onOnePayExport && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onOnePayExport(); close(); }}
                    disabled={isOnePayExportPending}
                  >
                    <Banknote className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">
                      {isOnePayExportPending ? 'Đang xuất...' : 'Chuyển OnePay'}
                    </span>
                  </Button>
                )}

                {/* Imports / uploads */}
                {onBulkTransferResultUpload && (
                  <>
                    <div className="h-px bg-border mx-1 my-1" />
                    <Button
                      variant="ghost"
                      className="w-full min-h-11 justify-start h-auto px-2 py-3"
                      onClick={() => { onBulkTransferResultUpload(); close(); }}
                    >
                      <Upload className="h-5 w-5 text-muted-foreground shrink-0" />
                      <span className="text-sm font-medium">Nhập KQ chuyển lô</span>
                    </Button>
                  </>
                )}

                {/* Actions */}
                {(onBulkApprove || onResetAll || onBulkTransferHistory || onPaymentHistory || (onRejectUnpaid && userRole === 'admin')) && (
                  <div className="h-px bg-border mx-1 my-1" />
                )}

                {onBulkApprove && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBulkApprove(); close(); }}
                  >
                    <CheckSquare className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Duyệt hết</span>
                  </Button>
                )}

                {onResetAll && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onResetAll(); close(); }}
                  >
                    <Undo2 className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Bỏ duyệt hết</span>
                  </Button>
                )}

                {onBulkTransferHistory && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBulkTransferHistory(); close(); }}
                  >
                    <History className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Lịch sử chuyển lô</span>
                  </Button>
                )}

                {onBccUpload && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBccUpload(); close(); }}
                  >
                    <FileUp className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Tải lên BCC</span>
                  </Button>
                )}

                {onBccHistory && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBccHistory(); close(); }}
                  >
                    <FileSpreadsheet className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Lịch sử BCC</span>
                  </Button>
                )}

                {onRejectUnpaid && userRole === 'admin' && (
                  <Button
                    variant="ghost"
                    className="h-auto min-h-11 w-full justify-start px-2 py-3 text-destructive hover:text-destructive"
                    onClick={() => { onRejectUnpaid(); close(); }}
                  >
                    <Trash2 className="h-5 w-5 shrink-0" />
                    <span className="text-sm font-medium">Loại công</span>
                  </Button>
                )}

                {onPaymentHistory && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
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
                  className="w-full min-h-11 justify-start h-auto px-2 py-3 text-muted-foreground"
                  onClick={close}
                >
                  <X className="h-5 w-5 shrink-0" />
                  <span className="text-sm font-medium">Đóng</span>
                </Button>
              </div>
              </SheetContent>
            </Sheet>
          </div>
        }
      />

      <div className="flex items-center gap-3 border-t border-border/70 bg-muted/25 px-3 py-2.5 sm:px-4">
        <span className="hidden shrink-0 text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground sm:inline">
          Kỳ công
        </span>
        <TimesheetMonthSelector
          value={monthValue}
          onChange={onMonthChange}
          className="min-w-0 flex-1 sm:flex-none"
        />
      </div>
    </section>
  );
}
