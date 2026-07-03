import { useState } from 'react';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import {
  Plus, MoreHorizontal, FileUp, History, Mail, Loader2, ArrowRightLeft,
  FileDown, FileText, CreditCard,
} from 'lucide-react';

interface TransactionPageHeaderMobileProps {
  onAddTransaction: () => void;
  onImportTransactions: () => void;
  onViewHistory: () => void;
  /** Upload settlement result (desktop: SettlementResultUploadDialog) */
  onUploadSettlement?: () => void;
  /** Import OnePay fee report (desktop: OnePayFeeReportDialog) */
  onImportOnePayFee?: () => void;
  /** Export Bảng công sao-ke (desktop: ExportSaoKeDialog) */
  onExportSaoKePayroll?: () => void;
  /** Export Ứng lương sao-ke (desktop: AdvancePaymentExportDialog) */
  onExportSaoKeAdvance?: () => void;
  /** Email payroll sao-ke */
  onSendStatementEmail?: () => void;
  /** Email advance-payment sao-ke (desktop: AdvancePaymentEmailDialog) */
  onSendAdvanceEmail?: () => void;
  onViewSaoKeHistory?: () => void;
  isSendingStatementEmail?: boolean;
  isSendingAdvanceEmail?: boolean;
  isImportingOnePayFee?: boolean;
}

export function TransactionPageHeaderMobile({
  onAddTransaction,
  onImportTransactions,
  onViewHistory,
  onUploadSettlement,
  onImportOnePayFee,
  onExportSaoKePayroll,
  onExportSaoKeAdvance,
  onSendStatementEmail,
  onSendAdvanceEmail,
  onViewSaoKeHistory,
  isSendingStatementEmail = false,
  isSendingAdvanceEmail = false,
  isImportingOnePayFee = false,
}: TransactionPageHeaderMobileProps) {
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const close = () => setShowActionsSheet(false);

  const actionClass =
    'w-full flex items-center gap-3 px-2 py-3 rounded-xl hover:bg-accent transition-colors touch-manipulation text-left disabled:opacity-50';

  return (
    <MobilePageHeader
      title="Giao Dịch"
      subtitle="Quản lý thu chi"
      icon={ArrowRightLeft}
      sticky={false}
      bordered={false}
      actions={
        <div className="flex shrink-0 items-center gap-2">
          <Button
            onClick={onAddTransaction}
            size="sm"
            className="h-11 px-4 touch-manipulation"
            aria-label="Thêm giao dịch"
          >
            <Plus className="h-4 w-4" />
            Thêm
          </Button>

          <Sheet open={showActionsSheet} onOpenChange={setShowActionsSheet}>
            <SheetTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                className="h-11 w-11 touch-manipulation"
                aria-label="Tùy chọn khác"
              >
                <MoreHorizontal />
              </Button>
            </SheetTrigger>
            <SheetContent
              side="bottom"
              className="h-auto max-h-[80dvh] overflow-y-auto rounded-t-[28px] border-[hsl(var(--surface-border))] bg-white p-0"
            >
              <div className="mx-auto mb-3 mt-3 h-1 w-10 rounded-full bg-slate-300" />
              <SheetHeader>
                <SheetTitle>Tùy chọn</SheetTitle>
              </SheetHeader>
              <div
                className="space-y-1 py-3"
                style={{ paddingBottom: 'max(12px, calc(12px + env(safe-area-inset-bottom)))' }}
              >
                {/* Exports — sao-ke */}
                {(onExportSaoKePayroll || onExportSaoKeAdvance) && (
                  <div className="px-3 pb-1 pt-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                    Xuất sao kê
                  </div>
                )}
                {onExportSaoKePayroll && (
                  <button className={actionClass} onClick={() => { onExportSaoKePayroll(); close(); }}>
                    <FileDown className="h-5 w-5 shrink-0 text-muted-foreground" />
                    <span className="text-sm font-medium">Xuất sao kê Bảng công</span>
                  </button>
                )}
                {onExportSaoKeAdvance && (
                  <button className={actionClass} onClick={() => { onExportSaoKeAdvance(); close(); }}>
                    <FileDown className="h-5 w-5 shrink-0 text-muted-foreground" />
                    <span className="text-sm font-medium">Xuất sao kê Ứng lương</span>
                  </button>
                )}

                {/* Emails */}
                {(onSendStatementEmail || onSendAdvanceEmail) && (
                  <div className="h-px bg-border mx-1 my-1" />
                )}
                {onSendStatementEmail && (
                  <button
                    className={actionClass}
                    onClick={() => { onSendStatementEmail(); close(); }}
                    disabled={isSendingStatementEmail}
                  >
                    {isSendingStatementEmail ? (
                      <Loader2 className="h-5 w-5 shrink-0 animate-spin text-muted-foreground" />
                    ) : (
                      <Mail className="h-5 w-5 shrink-0 text-muted-foreground" />
                    )}
                    <span className="text-sm font-medium">
                      {isSendingStatementEmail ? 'Đang gửi...' : 'Gửi sao kê Bảng công'}
                    </span>
                  </button>
                )}
                {onSendAdvanceEmail && (
                  <button
                    className={actionClass}
                    onClick={() => { onSendAdvanceEmail(); close(); }}
                    disabled={isSendingAdvanceEmail}
                  >
                    {isSendingAdvanceEmail ? (
                      <Loader2 className="h-5 w-5 shrink-0 animate-spin text-muted-foreground" />
                    ) : (
                      <Mail className="h-5 w-5 shrink-0 text-muted-foreground" />
                    )}
                    <span className="text-sm font-medium">
                      {isSendingAdvanceEmail ? 'Đang gửi...' : 'Gửi sao kê Ứng lương'}
                    </span>
                  </button>
                )}

                {/* Imports */}
                {(onImportTransactions || onUploadSettlement || onImportOnePayFee) && (
                  <div className="h-px bg-border mx-1 my-1" />
                )}
                <button className={actionClass} onClick={() => { onImportTransactions(); close(); }}>
                  <FileUp className="h-5 w-5 shrink-0 text-muted-foreground" />
                  <span className="text-sm font-medium">Nhập KQ chuyển lô</span>
                </button>
                {onUploadSettlement && (
                  <button className={actionClass} onClick={() => { onUploadSettlement(); close(); }}>
                    <FileUp className="h-5 w-5 shrink-0 text-muted-foreground" />
                    <span className="text-sm font-medium">Nhập kết quả đối soát</span>
                  </button>
                )}
                {onImportOnePayFee && (
                  <button
                    className={actionClass}
                    onClick={() => { onImportOnePayFee(); close(); }}
                    disabled={isImportingOnePayFee}
                  >
                    {isImportingOnePayFee ? (
                      <Loader2 className="h-5 w-5 shrink-0 animate-spin text-muted-foreground" />
                    ) : (
                      <CreditCard className="h-5 w-5 shrink-0 text-muted-foreground" />
                    )}
                    <span className="text-sm font-medium">
                      {isImportingOnePayFee ? 'Đang nhập...' : 'Nhập phí OnePay'}
                    </span>
                  </button>
                )}

                {/* History */}
                {(onViewSaoKeHistory || onViewHistory) && (
                  <div className="h-px bg-border mx-1 my-1" />
                )}
                {onViewSaoKeHistory && (
                  <button className={actionClass} onClick={() => { onViewSaoKeHistory(); close(); }}>
                    <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
                    <span className="text-sm font-medium">Đối soát sao kê</span>
                  </button>
                )}
                <button className={actionClass} onClick={() => { onViewHistory(); close(); }}>
                  <History className="h-5 w-5 shrink-0 text-muted-foreground" />
                  <span className="text-sm font-medium">Lịch sử chuyển lô</span>
                </button>
              </div>
            </SheetContent>
          </Sheet>
        </div>
      }
    />
  );
}
