import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { PageHeader } from '@/components/shared/PageHeader';
import {
  Plus,
  RotateCcw,
  BarChart3,
  Mail,
  History,
  ReceiptText,
  ChevronDownIcon,
  Wallet,
  GitMerge,
  MoreHorizontal,
  Loader2,
  TrendingUp,
  Zap,
} from 'lucide-react';
import { CashFlowChart } from './CashFlowChart';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/utils/formatters';

interface LedgerPageHeaderProps {
  overallBalance?: number;
  netCashFlow?: number;
  isLoadingBalance?: boolean;
  isLoadingCashFlow?: boolean;
  onAddEntry: () => void;
  onAddDoubleEntry?: () => void;
  onRecalculateBalance?: () => void;
  isRecalculating?: boolean;
  onSendSaoKePayroll?: () => void;
  onSendSaoKeAdvance?: () => void;
  onViewSaoKeHistory?: () => void;
  onImportOnePayFeeReport?: () => void;
  isSendingSaoKe?: boolean;
  onRunWalletSettlement?: () => void;
  isRunningWalletSettlement?: boolean;
}

export function LedgerPageHeader({
  overallBalance,
  netCashFlow,
  isLoadingBalance = false,
  isLoadingCashFlow = false,
  onAddEntry,
  onAddDoubleEntry,
  onRecalculateBalance,
  isRecalculating = false,
  onSendSaoKePayroll,
  onSendSaoKeAdvance,
  onViewSaoKeHistory,
  onImportOnePayFeeReport,
  isSendingSaoKe = false,
  onRunWalletSettlement,
  isRunningWalletSettlement = false,
}: LedgerPageHeaderProps) {
  const [showChart, setShowChart] = useState(false);
  const [showActionsSheet, setShowActionsSheet] = useState(false);

  // Balance cards render only on mobile (<sm); the desktop layout shows balance elsewhere.
  const isPositive = (netCashFlow ?? 0) >= 0;

  // Secondary actions (live in the `...` Sheet on mobile, inline on desktop)
  const secondaryActions = [
    onAddDoubleEntry && {
      key: 'double-entry',
      label: 'Nhập bút toán kép',
      icon: GitMerge,
      onClick: onAddDoubleEntry,
    },
    onRunWalletSettlement && {
      key: 'wallet-settlement',
      label: isRunningWalletSettlement ? 'Đang chốt lương...' : 'Chốt lương chuyển tiền',
      icon: Zap,
      onClick: onRunWalletSettlement,
      disabled: isRunningWalletSettlement,
    },
    onRecalculateBalance && {
      key: 'recalc',
      label: isRecalculating ? 'Đang tính lại...' : 'Tính lại số dư',
      icon: RotateCcw,
      onClick: onRecalculateBalance,
      disabled: isRecalculating,
    },
    {
      key: 'chart',
      label: showChart ? 'Ẩn biểu đồ' : 'Xem biểu đồ',
      icon: BarChart3,
      onClick: () => setShowChart((s) => !s),
    },
    onSendSaoKePayroll && {
      key: 'saoke-payroll',
      label: 'Email sao kê lương',
      icon: Mail,
      onClick: onSendSaoKePayroll,
      disabled: isSendingSaoKe,
    },
    onSendSaoKeAdvance && {
      key: 'saoke-advance',
      label: 'Email sao kê ứng lương',
      icon: Wallet,
      onClick: onSendSaoKeAdvance,
      disabled: isSendingSaoKe,
    },
    onViewSaoKeHistory && {
      key: 'saoke-history',
      label: 'Lịch sử sao kê',
      icon: History,
      onClick: onViewSaoKeHistory,
    },
    onImportOnePayFeeReport && {
      key: 'onepay-fee',
      label: 'Nhập phí OnePay',
      icon: ReceiptText,
      onClick: onImportOnePayFeeReport,
    },
  ].filter(Boolean) as Array<{ key: string; label: string; icon: typeof Plus; onClick: () => void; disabled?: boolean }>;

  return (
    <div className="space-y-4">
      <PageHeader
        title="Sổ Cái"
        actions={[
          {
            label: 'Thêm Giao Dịch',
            onClick: onAddEntry,
            icon: Plus,
            variant: 'default' as const,
          },
        ]}
      >
        {/* Secondary action cluster:
            - Desktop (sm+): all actions inline in a divided button group, with
              "Gửi sao kê" as a DropdownMenu (Bảng công / Ứng lương)
            - Mobile (<sm): hidden — surfaced through the `...` Sheet below */}
        <div className="hidden sm:flex items-center rounded-xl border border-border divide-x divide-border overflow-hidden">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="rounded-none border-0 gap-1.5"
                disabled={isSendingSaoKe}
              >
                <Mail className="w-4 h-4" />
                {isSendingSaoKe ? 'Đang gửi...' : 'Gửi sao kê'}
                <ChevronDownIcon className="w-3 h-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onSendSaoKePayroll}>
                <BarChart3 className="w-4 h-4 mr-2" />
                Bảng công
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onSendSaoKeAdvance}>
                <Wallet className="w-4 h-4 mr-2" />
                Ứng lương
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            variant="ghost"
            className="rounded-none border-0 gap-1.5"
            onClick={onViewSaoKeHistory}
          >
            <History className="w-4 h-4" />
            Đối soát
          </Button>
          {onImportOnePayFeeReport && (
            <Button
              variant="ghost"
              className="rounded-none border-0 gap-1.5"
              onClick={onImportOnePayFeeReport}
            >
              <ReceiptText className="w-4 h-4" />
              Phí OnePay
            </Button>
          )}
          {onRunWalletSettlement && (
            <Button
              variant="ghost"
              className="rounded-none border-0 gap-1.5"
              onClick={onRunWalletSettlement}
              disabled={isRunningWalletSettlement}
            >
              {isRunningWalletSettlement ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Zap className="w-4 h-4" />
              )}
              Chốt lương
            </Button>
          )}
        </div>

        {/* Mobile-only `...` Sheet for secondary actions */}
        <Sheet open={showActionsSheet} onOpenChange={setShowActionsSheet}>
          <SheetTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="h-11 w-11 shrink-0 touch-manipulation sm:hidden"
              aria-label="Tùy chọn khác"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="h-auto">
            <SheetHeader><SheetTitle>Tùy chọn</SheetTitle></SheetHeader>
            <div className="space-y-2 mt-4 pb-4">
              {secondaryActions.map((action) => {
                const Icon = action.icon;
                return (
                  <Button
                    key={action.key}
                    onClick={() => { action.onClick(); setShowActionsSheet(false); }}
                    variant="outline"
                    disabled={action.disabled}
                    className="w-full justify-start h-12 gap-3"
                  >
                    {action.key === 'recalc' && isRecalculating ? (
                      <Loader2 className={cn("h-5 w-5 text-muted-foreground animate-spin")} />
                    ) : (
                      <Icon className="h-5 w-5 text-muted-foreground" />
                    )}
                    {action.label}
                  </Button>
                );
              })}
            </div>
          </SheetContent>
        </Sheet>
      </PageHeader>

      {/* Mobile-only balance + net cash-flow cards (restores the display lost when the
          mobile-only header was consolidated into this shared component). */}
      <div className="sm:hidden grid grid-cols-2 gap-3">
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
          'rounded-xl border p-3.5 shadow-sm',
          isPositive ? 'border-emerald-200 bg-emerald-50' : 'border-red-200 bg-red-50',
        )}>
          <div className="flex items-center gap-1 mb-1">
            <TrendingUp className={cn('h-3 w-3', isPositive ? 'text-emerald-700' : 'text-red-600')} />
            <p className={cn('text-xs', isPositive ? 'text-emerald-700' : 'text-red-700')}>Dòng tiền ròng</p>
          </div>
          <p className={cn('text-base font-bold tabular-nums', isPositive ? 'text-emerald-800' : 'text-red-800')}>
            {isLoadingCashFlow ? (
              <span className="text-sm opacity-60">Đang tải...</span>
            ) : netCashFlow !== undefined ? (
              `${isPositive ? '+' : ''}${formatCurrency(netCashFlow)}`
            ) : '—'}
          </p>
        </div>
      </div>

      {/* Collapsible Cash Flow Chart */}
      <Collapsible open={showChart} onOpenChange={setShowChart}>
        <CollapsibleContent>
          <CashFlowChart />
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
