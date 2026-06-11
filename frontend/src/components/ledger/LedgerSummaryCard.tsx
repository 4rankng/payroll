import { InlineStatStrip } from '@/components/shared/InlineStatStrip';
import { Skeleton } from '@/components/ui/skeleton';
import { Calendar } from 'lucide-react';
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

      {mainItems.length > 0 && <InlineStatStrip items={mainItems} />}
      {accountItems.length > 0 && <InlineStatStrip items={accountItems} />}
    </div>
  );
}
