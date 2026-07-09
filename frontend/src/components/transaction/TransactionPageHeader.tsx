import { Plus, BarChart3, Mail, History, ChevronDownIcon, Wallet, Download, ReceiptText, Zap, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
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
}: TransactionPageHeaderProps) {
  return (
    <PageHeader title="Sổ Cái" description="Quản lý thu chi và dòng tiền">
      <div className="flex items-center gap-2">
        <div className="flex items-center rounded-xl border border-border divide-x divide-border overflow-hidden">
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
    </PageHeader>
  );
}
