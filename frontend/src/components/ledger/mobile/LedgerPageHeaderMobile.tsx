import { useState } from 'react';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible';
import { BookOpen, Plus, RotateCcw, BarChart3, MoreHorizontal, TrendingUp, GitMerge, Mail, History, Loader2 } from 'lucide-react';
import { CashFlowChart } from '../CashFlowChart';
import { formatCurrency } from '@/utils/formatters';
import { cn } from '@/lib/utils';

interface LedgerPageHeaderMobileProps {
  overallBalance: number | undefined;
  netCashFlow: number | undefined;
  isLoadingBalance: boolean;
  isLoadingCashFlow: boolean;
  onAddEntry: () => void;
  onAddDoubleEntry?: () => void;
  onRecalculateBalance?: () => void;
  isRecalculating?: boolean;
  onSendSaoKePayroll?: () => void;
  onSendSaoKeAdvance?: () => void;
  onViewSaoKeHistory?: () => void;
  isSendingSaoKe?: boolean;
}

export function LedgerPageHeaderMobile({
  overallBalance,
  netCashFlow,
  isLoadingBalance,
  isLoadingCashFlow,
  onAddEntry,
  onAddDoubleEntry,
  onRecalculateBalance,
  isRecalculating = false,
  onSendSaoKePayroll,
  onSendSaoKeAdvance,
  onViewSaoKeHistory,
  isSendingSaoKe = false,
}: LedgerPageHeaderMobileProps) {
  const [showChart, setShowChart] = useState(false);
  const [showActionsSheet, setShowActionsSheet] = useState(false);

  const isPositive = (netCashFlow ?? 0) >= 0;

  return (
    <div className="space-y-3">
      {/* Header row */}
      <MobilePageHeader
        title="Sổ Cái"
        subtitle="Quản lý sổ cái và giao dịch"
        icon={BookOpen}
        sticky={false}
        bordered={false}
        actions={
          <div className="flex items-center gap-2 shrink-0">
            <Button onClick={onAddEntry} size="sm" className="h-9 btn-admin-primary touch-manipulation">
              <Plus className="h-4 w-4 mr-1" />
              Thêm
            </Button>
            <Sheet open={showActionsSheet} onOpenChange={setShowActionsSheet}>
              <SheetTrigger asChild>
                <Button variant="outline" size="icon" className="h-9 w-9 touch-manipulation">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </SheetTrigger>
              <SheetContent side="bottom" className="h-auto">
                <SheetHeader><SheetTitle>Tùy chọn</SheetTitle></SheetHeader>
                <div className="space-y-2 mt-4 pb-4">
                  {onAddDoubleEntry && (
                    <Button
                      onClick={() => { onAddDoubleEntry(); setShowActionsSheet(false); }}
                      variant="outline"
                      className="w-full justify-start h-12 gap-3"
                    >
                      <GitMerge className="h-5 w-5 text-muted-foreground" />
                      Nhập bút toán kép
                    </Button>
                  )}
                  {onRecalculateBalance && (
                    <Button
                      onClick={() => { onRecalculateBalance(); setShowActionsSheet(false); }}
                      variant="outline"
                      disabled={isRecalculating}
                      className="w-full justify-start h-12 gap-3"
                    >
                      <RotateCcw className={cn("h-5 w-5 text-muted-foreground", isRecalculating && "animate-spin")} />
                      {isRecalculating ? 'Đang tính...' : 'Tính lại số dư'}
                    </Button>
                  )}
                  <Button
                    onClick={() => { setShowChart(!showChart); setShowActionsSheet(false); }}
                    variant="outline"
                    className="w-full justify-start h-12 gap-3"
                  >
                    <BarChart3 className="h-5 w-5 text-muted-foreground" />
                    {showChart ? 'Ẩn biểu đồ' : 'Xem biểu đồ'}
                  </Button>
                  {onSendSaoKePayroll && (
                    <Button
                      onClick={() => { onSendSaoKePayroll(); setShowActionsSheet(false); }}
                      variant="outline"
                      disabled={isSendingSaoKe}
                      className="w-full justify-start h-12 gap-3"
                    >
                      {isSendingSaoKe
                        ? <Loader2 className="h-5 w-5 text-muted-foreground animate-spin" />
                        : <Mail className="h-5 w-5 text-muted-foreground" />
                      }
                      Email sao kê lương
                    </Button>
                  )}
                  {onSendSaoKeAdvance && (
                    <Button
                      onClick={() => { onSendSaoKeAdvance(); setShowActionsSheet(false); }}
                      variant="outline"
                      disabled={isSendingSaoKe}
                      className="w-full justify-start h-12 gap-3"
                    >
                      {isSendingSaoKe
                        ? <Loader2 className="h-5 w-5 text-muted-foreground animate-spin" />
                        : <Mail className="h-5 w-5 text-muted-foreground" />
                      }
                      Email sao kê ứng lương
                    </Button>
                  )}
                  {onViewSaoKeHistory && (
                    <Button
                      onClick={() => { onViewSaoKeHistory(); setShowActionsSheet(false); }}
                      variant="outline"
                      className="w-full justify-start h-12 gap-3"
                    >
                      <History className="h-5 w-5 text-muted-foreground" />
                      Lịch sử sao kê
                    </Button>
                  )}
                </div>
              </SheetContent>
            </Sheet>
          </div>
        }
      />

      {/* Balance cards */}
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-xl border border-border bg-card p-3.5 shadow-sm">
          <p className="text-xs text-muted-foreground mb-1">Tổng số dư</p>
          <p className="text-base font-bold text-foreground tabular-nums">
            {isLoadingBalance ? (
              <span className="text-muted-foreground text-sm">Đang tải...</span>
            ) : overallBalance !== undefined ? (
              formatCurrency(overallBalance)
            ) : '—'}
          </p>
        </div>
        <div className={cn(
          "rounded-xl border p-3.5 shadow-sm",
          isPositive ? "border-emerald-200 bg-emerald-50" : "border-red-200 bg-red-50"
        )}>
          <div className="flex items-center gap-1 mb-1">
            <TrendingUp className={cn("h-3 w-3", isPositive ? "text-emerald-600" : "text-red-600")} />
            <p className={cn("text-xs", isPositive ? "text-emerald-700" : "text-red-700")}>Dòng tiền ròng</p>
          </div>
          <p className={cn("text-base font-bold tabular-nums", isPositive ? "text-emerald-800" : "text-red-800")}>
            {isLoadingCashFlow ? (
              <span className="text-sm opacity-60">Đang tải...</span>
            ) : netCashFlow !== undefined ? (
              `${isPositive ? '+' : ''}${formatCurrency(netCashFlow)}`
            ) : '—'}
          </p>
        </div>
      </div>

      {/* Cash flow chart */}
      <Collapsible open={showChart} onOpenChange={setShowChart}>
        <CollapsibleContent>
          <div className="mt-1">
            <CashFlowChart />
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
