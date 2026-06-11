import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';
import { transactionService } from '@/services/api/transaction.service';
import { formatCurrency } from '@/utils/formatters';
import type { LedgerSummary } from '@/types/api/financial.types';
import { useMemo, ReactNode } from 'react';
import { cn } from '@/lib/utils';
import { Landmark, TrendingUp } from 'lucide-react';
import { Masonry } from 'masonic';

// ── masonry ────────────────────────────────────────────────────────────────

interface SummaryCardData {
  id: string;
  icon?: React.ComponentType<{ className?: string }>;
  label: string;
  isLoading?: boolean;
  items: InlineStatItem[];
  customNode?: ReactNode;
}

const SummaryMasonryCard = ({ data }: { index: number; data: SummaryCardData; width: number }) => {
  if (data.customNode) return <div className="w-full">{data.customNode}</div>;
  const Icon = data.icon;
  return (
    <div className="w-full space-y-2">
      {Icon && (
        <div className="flex items-center gap-2 px-0.5">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/5 border border-primary/10">
            <Icon className="h-3.5 w-3.5 text-primary/70" />
          </div>
          <span className="text-xs font-bold uppercase tracking-wider text-foreground">{data.label}</span>
        </div>
      )}
      <InlineStatStrip items={data.items} isLoading={data.isLoading} direction="vertical" />
    </div>
  );
};

interface TransactionSummaryCardProps {
  ledgerSummary?: LedgerSummary;
  isLoading: boolean;
  className?: string;
  renderCapitalCard?: () => ReactNode;
}

export function TransactionSummaryCard({ ledgerSummary, isLoading, className, renderCapitalCard }: TransactionSummaryCardProps) {
  const liabilityItems = useMemo<InlineStatItem[]>(() => {
    if (!ledgerSummary) return [];
    const { by_account } = ledgerSummary;
    const payable = by_account.payable?.net_amount ?? 0;
    return [
      { label: 'Phải thu', value: formatCurrency(by_account.receivable?.net_amount ?? 0) },
      { label: 'Phải trả', value: formatCurrency(payable), valueClassName: payable < 0 ? 'text-red-600' : undefined },
      { label: 'Vay nợ',   value: formatCurrency(by_account.loan?.net_amount ?? 0) },
    ];
  }, [ledgerSummary]);

  const performanceItems = useMemo<InlineStatItem[]>(() => {
    if (!ledgerSummary) return [];
    const { by_account } = ledgerSummary;
    const revenue = by_account.revenue?.net_amount ?? 0;
    const expense = by_account.expense?.net_amount ?? 0;
    const cash    = by_account.cash?.net_amount ?? 0;
    return [
      { label: 'Tiền mặt',        value: formatCurrency(cash),            highlight: true },
      { label: 'Doanh thu',       value: formatCurrency(revenue) },
      { label: 'Chi phí',         value: formatCurrency(expense) },
      { label: 'Lợi nhuận',       value: formatCurrency(revenue - expense) },
    ];
  }, [ledgerSummary]);

  const loadingItems = useMemo<SummaryCardData[]>(() => {
    const emptyItems = (n: number) => Array.from({ length: n }, () => ({ label: '', value: '' }));
    const items: SummaryCardData[] = [
      { id: 'liability', icon: Landmark, label: 'Công nợ', isLoading: true, items: emptyItems(3) },
      { id: 'performance', icon: TrendingUp, label: 'Hiệu quả', isLoading: true, items: emptyItems(4) },
    ];
    if (renderCapitalCard) {
      items.push({ id: 'capital', icon: Landmark, label: 'Vốn', isLoading: true, items: emptyItems(3) });
    }
    return items;
  }, [renderCapitalCard]);

  const loadedItems = useMemo<SummaryCardData[]>(() => {
    const items: SummaryCardData[] = [];
    if (liabilityItems.length > 0) {
      items.push({ id: 'liability', icon: Landmark, label: 'Công nợ', items: liabilityItems });
    }
    if (performanceItems.length > 0) {
      items.push({ id: 'performance', icon: TrendingUp, label: 'Hiệu quả', items: performanceItems });
    }
    if (renderCapitalCard) {
      items.push({ id: 'capital', label: 'Vốn', items: [], customNode: renderCapitalCard() });
    }
    return items;
  }, [liabilityItems, performanceItems, renderCapitalCard]);

  if (isLoading) {
    return (
      <div className={cn(className)}>
        <Masonry
          items={loadingItems}
          render={SummaryMasonryCard}
          columnWidth={350}
          columnGutter={16}
          rowGutter={16}
          maxColumnCount={2}
          overscanBy={Infinity}
          itemKey={data => data.id}
        />
      </div>
    );
  }

  if (!ledgerSummary) return null;

  return (
    <div className={cn(className)}>
      <Masonry
        items={loadedItems}
        render={SummaryMasonryCard}
        columnWidth={350}
        columnGutter={16}
        rowGutter={16}
        maxColumnCount={2}
        overscanBy={Infinity}
        itemKey={data => data.id}
      />
    </div>
  );
}
