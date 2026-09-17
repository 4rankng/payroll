import { InlineStatStrip } from '@/components/shared/InlineStatStrip';
import { Skeleton } from '@/components/ui/skeleton';
import { Calendar, ChevronDown } from 'lucide-react';
import { formatDate } from '@/utils/formatters';
import { useLedgerStatsConfig } from '@/hooks/ledger/useLedgerStatsConfig';
import type { LedgerSummary } from '@/types/api/financial.types';
import { cn } from '@/lib/utils';

interface LedgerSummaryCardProps {
  summary?: LedgerSummary;
  isLoading: boolean;
  className?: string;
}

export function LedgerSummaryCard({ summary, isLoading, className }: LedgerSummaryCardProps) {
  const { mainStatsConfig, accountStatsConfig, isLoading: configLoading } = useLedgerStatsConfig({
    summary,
    isLoading,
  });

  const loading = isLoading || configLoading;

  if (loading) {
    return (
      <div className={cn('space-y-3', className)}>
        <Skeleton className="h-3.5 w-48" />
        <InlineStatStrip items={Array.from({ length: 3 }, () => ({ label: '', value: '' }))} isLoading />
        <InlineStatStrip items={Array.from({ length: 4 }, () => ({ label: '', value: '' }))} isLoading />
      </div>
    );
  }

  if (!summary) return null;

  const mainItems = mainStatsConfig.map(s => ({ label: s.title, value: s.value }));
  const accountItems = accountStatsConfig.map(s => ({ label: s.title, value: s.value }));

  return (
    <div className={cn('space-y-3', className)}>
      {/* Period header */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Calendar className="h-3 w-3" />
        <span className="font-medium">Kỳ báo cáo:</span>
        <span>{formatDate(summary.period.from)} – {formatDate(summary.period.to)}</span>
      </div>

      <div className="space-y-2 lg:hidden">
        <dl className="divide-y divide-border rounded-xl border border-border bg-card px-3">
          {mainItems.map(item => (
            <div key={item.label} className="flex min-h-11 flex-wrap items-center justify-between gap-x-3 gap-y-1 py-2">
              <dt className="text-xs text-muted-foreground">{item.label}</dt>
              <dd className="min-w-0 break-words text-right text-sm font-semibold tabular-nums">{item.value}</dd>
            </div>
          ))}
        </dl>
        {accountItems.length > 0 && (
          <details className="group rounded-xl border border-border bg-card">
            <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-2 px-3 text-xs font-medium [&::-webkit-details-marker]:hidden">
              Số dư theo tài khoản ({accountItems.length})
              <ChevronDown className="h-4 w-4 shrink-0 transition-transform group-open:rotate-180" aria-hidden="true" />
            </summary>
            <dl className="divide-y divide-border border-t border-border px-3">
              {accountItems.map(item => (
                <div key={item.label} className="flex min-h-11 flex-wrap items-center justify-between gap-x-3 gap-y-1 py-2">
                  <dt className="text-xs text-muted-foreground">{item.label}</dt>
                  <dd className="min-w-0 break-words text-right text-sm font-semibold tabular-nums">{item.value}</dd>
                </div>
              ))}
            </dl>
          </details>
        )}
      </div>
      <div className="hidden space-y-3 lg:block">
        {mainItems.length > 0 && <InlineStatStrip items={mainItems} />}
        {accountItems.length > 0 && <InlineStatStrip items={accountItems} />}
      </div>
    </div>
  );
}
