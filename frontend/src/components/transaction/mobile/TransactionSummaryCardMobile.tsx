import { Skeleton } from '@/components/ui/skeleton';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { useState } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { transactionService } from '@/services/api/transaction.service';
import type { LedgerSummary } from '@/types/api/financial.types';

interface TransactionSummaryCardMobileProps {
  ledgerSummary?: LedgerSummary;
  isLoading: boolean;
}

export function TransactionSummaryCardMobile({ ledgerSummary, isLoading }: TransactionSummaryCardMobileProps) {
  const [showMore, setShowMore] = useState(false);

  const fmt = (amount: number) => transactionService.formatCurrency(amount);
  const fmtDate = (d: string) => {
    try { return format(new Date(d), 'dd/MM/yyyy', { locale: vi }); } catch { return d; }
  };

  if (isLoading) {
    return (
      <div className="space-y-2">
        <div className="grid grid-cols-2 gap-2">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-xl" />)}
        </div>
        <Skeleton className="h-10 rounded-xl" />
      </div>
    );
  }

  if (!ledgerSummary) return null;

  const { by_account, period } = ledgerSummary;
  const revenue = by_account.revenue?.net_amount ?? 0;
  const expense = by_account.expense?.net_amount ?? 0;
  const profit = revenue - expense;

  const primaryStats = [
    { label: 'Doanh thu', value: fmt(revenue), positive: true },
    { label: 'Chi phí', value: fmt(expense), positive: false },
    { label: 'Lợi nhuận', value: fmt(profit), positive: profit >= 0 },
    { label: 'Tiền mặt', value: fmt(by_account.cash?.net_amount ?? 0), neutral: true },
  ];

  const secondaryStats = [
    { label: 'Phải thu', value: fmt(by_account.receivable?.net_amount ?? 0) },
    { label: 'Phải trả', value: fmt(by_account.payable?.net_amount ?? 0) },
    { label: 'Vay nợ', value: fmt(by_account.loan?.net_amount ?? 0) },
    { label: 'Vốn CSH', value: fmt(by_account.equity?.net_amount ?? 0) },
  ];

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        Kỳ: {fmtDate(period.from)} – {fmtDate(period.to)}
      </p>

      {/* Primary stats grid */}
      <div className="grid grid-cols-2 gap-2">
        {primaryStats.map((stat, i) => (
          <div key={i} className="rounded-xl border border-border bg-card p-3 shadow-sm">
            <p className="text-xs text-muted-foreground mb-1">{stat.label}</p>
            <p className={`text-sm font-bold tabular-nums ${
              stat.neutral ? 'text-foreground' :
              stat.positive ? 'text-emerald-700' : 'text-red-600'
            }`}>
              {stat.value}
            </p>
          </div>
        ))}
      </div>

      {/* Secondary stats — collapsible */}
      <Collapsible open={showMore} onOpenChange={setShowMore}>
        <CollapsibleTrigger asChild>
          <Button
            variant="ghost"
            className="w-full flex items-center justify-between h-9 px-3 bg-muted/50 hover:bg-muted rounded-xl text-sm font-medium text-foreground"
          >
            Công nợ & Vốn
            {showMore ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="grid grid-cols-2 gap-2 pt-2">
            {secondaryStats.map((stat, i) => (
              <div key={i} className="rounded-xl border border-border bg-card p-2.5 shadow-sm">
                <p className="text-xs text-muted-foreground mb-0.5">{stat.label}</p>
                <p className="text-xs font-bold text-foreground tabular-nums">{stat.value}</p>
              </div>
            ))}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
