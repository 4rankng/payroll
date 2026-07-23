import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/utils/formatters';
import type { LedgerSummary } from '@/types/api/financial.types';
import { useMemo, ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface SupportingMetric {
  label: string;
  value: string;
  valueClassName?: string;
}

function SupportingMetrics({
  title,
  items,
}: {
  title: string;
  items: SupportingMetric[];
}) {
  return (
    <article className="overflow-hidden rounded-2xl border border-border/70 bg-card">
      <div className="border-b border-border/60 px-5 py-4">
        <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          {title}
        </h3>
      </div>
      <dl className="divide-y divide-border/50">
        {items.map((item) => (
          <div
            key={item.label}
            className="flex min-h-14 items-center justify-between gap-4 px-5 py-3"
          >
            <dt className="text-sm text-muted-foreground">{item.label}</dt>
            <dd
              className={cn(
                'min-w-0 break-words text-right font-financial text-base font-semibold tabular-nums text-foreground',
                item.valueClassName,
              )}
            >
              {item.value}
            </dd>
          </div>
        ))}
      </dl>
    </article>
  );
}

function SummarySkeleton({ showCapital }: { showCapital: boolean }) {
  return (
    <div
      className="grid gap-3 lg:grid-cols-2"
      aria-busy="true"
      aria-label="Đang tải tổng quan tài chính"
    >
      <div className="overflow-hidden rounded-2xl border border-border/70 bg-card">
        <div className="space-y-3 px-5 py-5 sm:px-6 sm:py-6">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-8 w-52" />
        </div>
        <div className="grid grid-cols-1 border-t border-border/60 sm:grid-cols-3 sm:divide-x sm:divide-border/60">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="space-y-2 border-b border-border/60 px-5 py-4 last:border-b-0 sm:border-b-0">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-5 w-28" />
            </div>
          ))}
        </div>
      </div>
      <div className="overflow-hidden rounded-2xl border border-border/70 bg-card">
        <div className="border-b border-border/60 px-5 py-4">
          <Skeleton className="h-3 w-20" />
        </div>
        {Array.from({ length: 3 }).map((_, rowIndex) => (
          <div key={rowIndex} className="flex min-h-14 items-center justify-between border-b border-border/50 px-5 py-3 last:border-b-0">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-4 w-28" />
          </div>
        ))}
      </div>
      {showCapital && (
        <div className="overflow-hidden rounded-2xl border border-border/70 bg-card lg:col-span-2">
          <div className="border-b border-border/60 px-5 py-4">
            <Skeleton className="h-3 w-28" />
          </div>
          <div className="grid grid-cols-1 min-[420px]:grid-cols-3">
            {Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className="space-y-2 border-b border-border/50 px-5 py-4 last:border-b-0 min-[420px]:border-b-0 min-[420px]:border-r min-[420px]:last:border-r-0">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-5 w-32" />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface TransactionSummaryCardProps {
  ledgerSummary?: LedgerSummary;
  isLoading: boolean;
  className?: string;
  renderCapitalCard?: () => ReactNode;
}

export function TransactionSummaryCard({ ledgerSummary, isLoading, className, renderCapitalCard }: TransactionSummaryCardProps) {
  const liabilityItems = useMemo<SupportingMetric[]>(() => {
    if (!ledgerSummary) return [];
    const { by_account } = ledgerSummary;
    const payable = by_account.payable?.net_amount ?? 0;
    return [
      { label: 'Phải thu', value: formatCurrency(by_account.receivable?.net_amount ?? 0) },
      { label: 'Phải trả', value: formatCurrency(payable), valueClassName: payable < 0 ? 'text-red-600' : undefined },
      { label: 'Vay nợ',   value: formatCurrency(by_account.loan?.net_amount ?? 0) },
    ];
  }, [ledgerSummary]);

  const performance = useMemo(() => {
    if (!ledgerSummary) return null;
    const { by_account } = ledgerSummary;
    const revenue = by_account.revenue?.net_amount ?? 0;
    const expense = by_account.expense?.net_amount ?? 0;
    const cash    = by_account.cash?.net_amount ?? 0;
    return {
      cash: formatCurrency(cash),
      revenue: formatCurrency(revenue),
      expense: formatCurrency(expense),
      profit: formatCurrency(revenue - expense),
    };
  }, [ledgerSummary]);

  if (isLoading) {
    return (
      <div className={cn(className)}>
        <SummarySkeleton showCapital={Boolean(renderCapitalCard)} />
      </div>
    );
  }

  if (!ledgerSummary || !performance) return null;

  return (
    <section
      aria-label="Tổng quan tài chính"
      className={cn(
        'grid gap-3 lg:grid-cols-2',
        className,
      )}
    >
      <h2 className="sr-only">Tổng quan tài chính</h2>
      <article className="overflow-hidden rounded-2xl border border-border/70 bg-card">
        <div className="px-5 py-5 sm:px-6 sm:py-6">
          <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Thanh khoản hiện tại
          </h3>
          <p className="mt-2 break-words font-financial text-2xl font-semibold tabular-nums tracking-tight text-primary sm:text-3xl">
            {performance.cash}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">Tiền mặt khả dụng trên sổ cái</p>
        </div>

        <dl className="grid grid-cols-1 border-t border-border/60 sm:grid-cols-3 sm:divide-x sm:divide-border/60">
          {[
            { label: 'Doanh thu', value: performance.revenue },
            { label: 'Chi phí', value: performance.expense },
            { label: 'Lợi nhuận', value: performance.profit, highlight: true },
          ].map((item) => (
            <div
              key={item.label}
              className="border-b border-border/60 px-5 py-4 last:border-b-0 sm:border-b-0"
            >
              <dt className="text-xs text-muted-foreground">{item.label}</dt>
              <dd
                className={cn(
                  'mt-1 break-words font-financial text-base font-semibold tabular-nums text-foreground',
                  item.highlight && 'text-primary',
                )}
              >
                {item.value}
              </dd>
            </div>
          ))}
        </dl>
      </article>

      {liabilityItems.length > 0 && (
        <SupportingMetrics title="Công nợ" items={liabilityItems} />
      )}
      {renderCapitalCard && (
        <div className="lg:col-span-2">{renderCapitalCard()}</div>
      )}
    </section>
  );
}
