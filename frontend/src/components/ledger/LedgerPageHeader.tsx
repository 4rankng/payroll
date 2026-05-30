import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { PageHeader } from '@/components/shared/PageHeader';
import { Plus, RotateCcw, ArrowLeftRight, ChevronDown, ChevronUp, BarChart3, Mail, History, ChevronDownIcon, Wallet } from 'lucide-react';
import { CashFlowChart } from './CashFlowChart';
import { ledgerService } from '@/services/api/ledger.service';

interface LedgerPageHeaderProps {
  overallBalance: number | undefined;
  netCashFlow: number | undefined;
  isLoadingBalance: boolean;
  isLoadingCashFlow: boolean;
  onAddEntry: () => void;
  onAddDoubleEntry?: () => void;
  onRecalculateBalance?: () => void;
  isRecalculating?: boolean;
  onSendSaoKePayroll: () => void;
  onSendSaoKeAdvance: () => void;
  onViewSaoKeHistory: () => void;
  isSendingSaoKe?: boolean;
}

export function LedgerPageHeader({
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
}: LedgerPageHeaderProps) {
  const [showChart, setShowChart] = useState(false);

  return (
    <div className="space-y-4">
      <PageHeader
        title="Sổ Cái"
        description="Quản lý sổ cái và giao dịch"
        actions={[
          ...(onRecalculateBalance ? [{
            label: 'Tính lại số dư',
            onClick: onRecalculateBalance,
            icon: RotateCcw,
            variant: 'outline' as const,
            disabled: isRecalculating,
            className: isRecalculating ? 'opacity-70' : '',
          }] : []),
          {
            label: 'Thêm Giao Dịch',
            onClick: onAddEntry,
            icon: Plus,
            variant: 'default' as const,
          },
        ]}
      >
        <div className="flex items-center rounded-xl border border-border divide-x divide-border overflow-hidden">
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
      </PageHeader>

      {/* Collapsible Cash Flow Chart */}
      <Collapsible open={showChart} onOpenChange={setShowChart}>
        <CollapsibleContent>
          <CashFlowChart />
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
