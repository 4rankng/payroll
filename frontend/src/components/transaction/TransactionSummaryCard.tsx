import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency, formatFullCurrency } from '@/utils/formatters';
import type { LedgerSummary } from '@/types/api/financial.types';
import { useMemo, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface SupportingMetric {
  label: string;
  value: string;
  full?: string;
  valueClassName?: string;
}

interface KpiTile {
  label: string;
  value: string;
  /** Exact figure for compact values (title + aria-label). */
  full?: string;
  sub?: string;
  emphasis?: boolean;
  wide?: boolean;
  valueClassName?: string;
}

function KpiTileView({ tile }: { tile: KpiTile }) {
  return (
    <article className={cn('console-panel px-3 py-2.5', tile.wide && 'sm:col-span-2')}>
      <h3 className="console-kicker">
        {tile.label}
      </h3>
      <p
        className={cn('console-value mt-1.5 break-words font-display text-xl font-semibold tracking-tight', tile.emphasis ? 'text-primary' : 'text-foreground', tile.valueClassName)}
        title={tile.full}
        aria-label={tile.full}
      >
        {tile.value}
      </p>
      {tile.emphasis && tile.sub && (
        <p className="mt-0.5 text-[10px] text-muted-foreground">{tile.sub}</p>
      )}
    </article>
  );
}

function SummarySkeleton({ showCapital }: { showCapital: boolean }) {
  return (
    <div className="grid gap-3" aria-busy="true" aria-label="Đang tải tổng quan tài chính">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        <Skeleton className="h-[76px] rounded-lg sm:col-span-2" />
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[76px] rounded-lg" />
        ))}
      </div>
      <div className="grid gap-3 lg:grid-cols-3">
        <Skeleton className="h-[116px] rounded-lg lg:col-span-2" />
        <Skeleton className="h-[116px] rounded-lg" />
      </div>
      {showCapital && <div className="hidden" aria-hidden="true" />}
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
      { label: 'Phải thu', value: formatFullCurrency(by_account.receivable?.net_amount ?? 0), full: formatCurrency(by_account.receivable?.net_amount ?? 0) },
      { label: 'Phải trả', value: formatFullCurrency(payable), full: formatCurrency(payable), valueClassName: payable < 0 ? 'text-financial-negative' : undefined },
      { label: 'Vay nợ', value: formatFullCurrency(by_account.loan?.net_amount ?? 0), full: formatCurrency(by_account.loan?.net_amount ?? 0) },
    ];
  }, [ledgerSummary]);

  const perf = useMemo(() => {
    if (!ledgerSummary) return null;
    const { by_account } = ledgerSummary;
    const revenue = by_account.revenue?.net_amount ?? 0;
    const expense = by_account.expense?.net_amount ?? 0;
    const cash = by_account.cash?.net_amount ?? 0;
    return {
      cash: formatFullCurrency(cash),
      cashFull: formatCurrency(cash),
      revenue: formatFullCurrency(revenue),
      revenueFull: formatCurrency(revenue),
      expense: formatFullCurrency(expense),
      expenseFull: formatCurrency(expense),
      profit: formatFullCurrency(revenue - expense),
      profitFull: formatCurrency(revenue - expense),
      margin: revenue > 0 ? `${((revenue - expense) / revenue * 100).toLocaleString('vi-VN', { maximumFractionDigits: 1 })}%` : '—',
    };
  }, [ledgerSummary]);

  if (isLoading) {
    return (
      <div className={cn(className)}>
        <SummarySkeleton showCapital={Boolean(renderCapitalCard)} />
      </div>
    );
  }

  if (!ledgerSummary || !perf) return null;

  // KPI strip: cash position leads (2 tiles wide, emphasized); the Công nợ
  // rows become tiles so receivables/payables sit beside cash where they
  // drive action.
  const kpis: KpiTile[] = [
    { label: 'Thanh khoản hiện tại', value: perf.cash, full: perf.cashFull, sub: 'Tiền mặt khả dụng trên sổ cái', emphasis: true, wide: true },
    ...liabilityItems,
  ];

  const perfStats = [
    { label: 'Doanh thu', value: perf.revenue, full: perf.revenueFull },
    { label: 'Chi phí', value: perf.expense, full: perf.expenseFull },
    { label: 'Lợi nhuận', value: perf.profit, full: perf.profitFull, highlight: true },
    { label: 'Biên LN', value: perf.margin },
  ];

  return (
    <section aria-label="Tổng quan tài chính" className={cn('space-y-3', className)}>
      <h2 className="sr-only">Tổng quan tài chính</h2>

      {/* KPI strip — cash (2 tiles wide) + Phải thu / Phải trả / Vay nợ */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {kpis.map((tile) => (
          <KpiTileView key={tile.label} tile={tile} />
        ))}
      </div>

      {/* P&L story (2/3) + capital structure (1/3) */}
      <div className="grid gap-3 lg:grid-cols-3">
        <article className="console-panel lg:col-span-2">
          <div className="console-panel-header">
            <h3 className="console-kicker">
              Hiệu quả kinh doanh
            </h3>
            <p className="ml-auto text-[10px] text-muted-foreground">
              Doanh thu − Chi phí trên sổ cái trong kỳ
            </p>
          </div>
          <dl className="grid grid-cols-2 sm:grid-cols-4 sm:divide-x sm:divide-border/60">
            {perfStats.map((item) => (
              <div key={item.label} className="px-4 py-3">
                <dt className="text-xs text-muted-foreground">{item.label}</dt>
                <dd
                  title={item.full}
                  aria-label={item.full}
                  className={cn(
                    'console-value mt-1 break-words font-financial text-base font-semibold tracking-tight text-foreground',
                    item.highlight && 'text-primary',
                  )}
                >
                  {item.value}
                </dd>
              </div>
            ))}
          </dl>
        </article>

        <div className="lg:col-span-1">{renderCapitalCard?.()}</div>
      </div>
    </section>
  );
}
