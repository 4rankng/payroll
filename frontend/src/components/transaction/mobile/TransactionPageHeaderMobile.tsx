import { useState } from 'react';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Plus, MoreHorizontal, FileUp, History, Mail, Loader2, ArrowRightLeft } from 'lucide-react';

interface TransactionPageHeaderMobileProps {
  onAddTransaction: () => void;
  onImportTransactions: () => void;
  onViewHistory: () => void;
  onSendStatementEmail?: () => void;
  onViewSaoKeHistory?: () => void;
  isSendingStatementEmail?: boolean;
}

export function TransactionPageHeaderMobile({
  onAddTransaction,
  onImportTransactions,
  onViewHistory,
  onSendStatementEmail,
  onViewSaoKeHistory,
  isSendingStatementEmail = false,
}: TransactionPageHeaderMobileProps) {
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const close = () => setShowActionsSheet(false);

  return (
    <MobilePageHeader
      title="Giao Dịch"
      subtitle="Quản lý thu chi"
      icon={ArrowRightLeft}
      sticky={false}
      bordered={false}
      actions={
        <div className="flex items-center gap-2 shrink-0">
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
            <SheetContent side="bottom" className="h-auto">
              <SheetHeader>
                <SheetTitle>Tùy chọn</SheetTitle>
              </SheetHeader>
              <div className="space-y-1 py-3" style={{ paddingBottom: 'max(12px, calc(12px + env(safe-area-inset-bottom)))' }}>
                {onSendStatementEmail && (
                  <button
                    className="w-full flex items-center gap-3 px-2 py-3 rounded-xl hover:bg-accent transition-colors touch-manipulation text-left disabled:opacity-50"
                    onClick={() => { onSendStatementEmail(); close(); }}
                    disabled={isSendingStatementEmail}
                  >
                    {isSendingStatementEmail ? (
                      <Loader2 className="h-5 w-5 text-muted-foreground shrink-0 animate-spin" />
                    ) : (
                      <Mail className="h-5 w-5 text-muted-foreground shrink-0" />
                    )}
                    <span className="text-sm font-medium">
                      {isSendingStatementEmail ? 'Đang gửi sao kê...' : 'Gửi sao kê'}
                    </span>
                  </button>
                )}
                {onViewSaoKeHistory && (
                  <button
                    className="w-full flex items-center gap-3 px-2 py-3 rounded-xl hover:bg-accent transition-colors touch-manipulation text-left"
                    onClick={() => { onViewSaoKeHistory(); close(); }}
                  >
                    <History className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Đối soát</span>
                  </button>
                )}
                <button
                  className="w-full flex items-center gap-3 px-2 py-3 rounded-xl hover:bg-accent transition-colors touch-manipulation text-left"
                  onClick={() => { onImportTransactions(); close(); }}
                >
                  <FileUp className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="text-sm font-medium">Nhập KQ chuyển lô</span>
                </button>
                <button
                  className="w-full flex items-center gap-3 px-2 py-3 rounded-xl hover:bg-accent transition-colors touch-manipulation text-left"
                  onClick={() => { onViewHistory(); close(); }}
                >
                  <History className="h-5 w-5 text-muted-foreground shrink-0" />
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
