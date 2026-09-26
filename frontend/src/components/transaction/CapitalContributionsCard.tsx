import { Skeleton } from '@/components/ui/skeleton';
import { transactionService } from '@/services/api/transaction.service';
import type { OwnerContribution } from '@/types/api/financial.types';
import { cn } from '@/lib/utils';

interface CapitalContributionsCardProps {
  contributions?: OwnerContribution[];
  isLoading: boolean;
}

/** Compact stacked capital card for the ledger overview's right column. */
export function CapitalContributionsCard({ contributions, isLoading }: CapitalContributionsCardProps) {
  const fmt = (n: number) => transactionService.formatCurrency(n);

  if (isLoading) {
    return (
      <div className="h-full overflow-hidden rounded-xl border border-border/70 bg-card" aria-busy="true" aria-label="Đang tải vốn chủ sở hữu">
        <div className="border-b border-border/60 px-4 py-2.5">
          <Skeleton className="h-3 w-28" />
        </div>
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="border-b border-border/50 px-4 py-2.5 last:border-b-0">
            <Skeleton className="h-3 w-24" />
          </div>
        ))}
      </div>
    );
  }

  if (!contributions || contributions.length === 0) return null;

  const total = contributions.reduce((s, c) => s + c.total_contribution, 0);
  return (
    <article className="h-full overflow-hidden rounded-xl border border-border/70 bg-card">
      <div className="border-b border-border/60 px-4 py-2.5">
        <h3 className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          Vốn chủ sở hữu
        </h3>
      </div>
      <dl>
        {[
          { owner: 'Tổng vốn', total_contribution: total, total: true },
          ...contributions.map((c) => ({ ...c, total: false })),
        ].map(({ owner, total_contribution, total: isTotal }) => (
          <div key={owner} className="flex items-center justify-between gap-3 border-b border-border/50 px-4 py-2.5 last:border-b-0">
            <dt className="text-xs text-muted-foreground">{owner}</dt>
            <dd className={cn(
              'break-words text-right font-financial text-sm font-semibold tabular-nums text-foreground',
              isTotal && 'text-base text-primary',
            )}>
              {fmt(total_contribution)}
            </dd>
          </div>
        ))}
      </dl>
    </article>
  );
}
