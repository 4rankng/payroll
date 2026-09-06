import { Plus, BarChart3, Mail, History, ChevronDownIcon, Wallet, Download, ReceiptText, Zap, Loader2, MoreHorizontal, ClipboardCheck, BookOpen } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { PageHeader } from '@/components/shared/PageHeader';

interface TransactionPageHeaderProps {
  onAddTransaction: () => void;
  onSendSaoKePayroll: () => void;
  onSendSaoKeAdvance: () => void;
  onViewSaoKeHistory: () => void;
  onExportSaoKePayroll: () => void;
  onExportSaoKeAdvance: () => void;
  onImportOnePayFeeReport: () => void;
  isSendingSaoKe?: boolean;
  onRunWalletSettlement?: () => void;
  isRunningWalletSettlement?: boolean;
  onSimulateSettlement?: () => void;
}

export function TransactionPageHeader({
  onAddTransaction,
  onSendSaoKePayroll,
  onSendSaoKeAdvance,
  onViewSaoKeHistory,
  onExportSaoKePayroll,
  onExportSaoKeAdvance,
  onImportOnePayFeeReport,
  isSendingSaoKe = false,
  onRunWalletSettlement,
  isRunningWalletSettlement = false,
  onSimulateSettlement,
}: TransactionPageHeaderProps) {
  return (
    <PageHeader title="Sổ Cái" description="Quản lý thu chi và dòng tiền" icon={BookOpen}>
      <div className="hidden max-w-full flex-wrap items-center justify-end gap-2 min-[1800px]:flex">
        <div className="flex max-w-full items-center divide-x divide-border overflow-hidden rounded-xl border border-border">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="rounded-none border-0 gap-1.5"
              >
                <Download className="w-4 h-4" />
                Xuất sao kê
                <ChevronDownIcon className="w-3 h-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onExportSaoKePayroll}>
                <BarChart3 className="w-4 h-4 mr-2" />
                Bảng công
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onExportSaoKeAdvance}>
                <Wallet className="w-4 h-4 mr-2" />
                Ứng lương
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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
        </div>
        <Button variant="outline" onClick={onImportOnePayFeeReport}>
          <ReceiptText className="w-4 h-4" />
          Phí OnePay
        </Button>
        {onSimulateSettlement && (
          <Button variant="outline" onClick={onSimulateSettlement}>
            <ClipboardCheck className="w-4 h-4" />
            Mô phỏng đối soát
          </Button>
        )}
        {onRunWalletSettlement && (
          <Button
            variant="outline"
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
        <Button onClick={onAddTransaction}>
          <Plus />
          Thêm
        </Button>
      </div>

      <div className="flex items-center gap-2 min-[1800px]:hidden">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="h-9 w-9 shrink-0"
              aria-label="Tùy chọn khác"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-64">
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Download className="mr-2 h-4 w-4" />
                Xuất sao kê
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuItem onClick={onExportSaoKePayroll}>
                  <BarChart3 className="mr-2 h-4 w-4" />
                  Bảng công
                </DropdownMenuItem>
                <DropdownMenuItem onClick={onExportSaoKeAdvance}>
                  <Wallet className="mr-2 h-4 w-4" />
                  Ứng lương
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            <DropdownMenuSub>
              <DropdownMenuSubTrigger disabled={isSendingSaoKe}>
                {isSendingSaoKe ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Mail className="mr-2 h-4 w-4" />
                )}
                {isSendingSaoKe ? 'Đang gửi...' : 'Gửi sao kê'}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuItem onClick={onSendSaoKePayroll}>
                  <BarChart3 className="mr-2 h-4 w-4" />
                  Bảng công
                </DropdownMenuItem>
                <DropdownMenuItem onClick={onSendSaoKeAdvance}>
                  <Wallet className="mr-2 h-4 w-4" />
                  Ứng lương
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onViewSaoKeHistory}>
              <History className="mr-2 h-4 w-4" />
              Đối soát
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onImportOnePayFeeReport}>
              <ReceiptText className="mr-2 h-4 w-4" />
              Phí OnePay
            </DropdownMenuItem>
            {onSimulateSettlement && (
              <DropdownMenuItem className="lg:hidden" onClick={onSimulateSettlement}>
                <ClipboardCheck className="mr-2 h-4 w-4" />
                Mô phỏng đối soát
              </DropdownMenuItem>
            )}
            {onRunWalletSettlement && (
              <DropdownMenuItem
                onClick={onRunWalletSettlement}
                disabled={isRunningWalletSettlement}
              >
                {isRunningWalletSettlement ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Zap className="mr-2 h-4 w-4" />
                )}
                {isRunningWalletSettlement ? 'Đang chốt lương...' : 'Chốt lương'}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        {onSimulateSettlement && (
          <Button
            data-testid="compact-settlement-simulation-button"
            variant="outline"
            onClick={onSimulateSettlement}
            className="hidden h-9 lg:inline-flex"
          >
            <ClipboardCheck className="h-4 w-4" />
            Mô phỏng đối soát
          </Button>
        )}

        <Button onClick={onAddTransaction} size="sm" className="h-9 px-3 text-sm">
          <Plus className="h-4 w-4" />
          Thêm
        </Button>
      </div>
    </PageHeader>
  );
}
