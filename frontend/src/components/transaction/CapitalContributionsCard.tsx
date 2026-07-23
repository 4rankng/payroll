import { Skeleton } from '@/components/ui/skeleton';
import { transactionService } from '@/services/api/transaction.service';
import type { OwnerContribution } from '@/types/api/financial.types';

interface CapitalContributionsCardProps {
  contributions?: OwnerContribution[];
  isLoading: boolean;
}

export function CapitalContributionsCard({ contributions, isLoading }: CapitalContributionsCardProps) {
  const fmt = (n: number) => transactionService.formatCurrency(n);

  if (isLoading) {
    return (
      <div
        className="overflow-hidden rounded-2xl border border-border/70 bg-card"
        aria-busy="true"
        aria-label="Đang tải vốn chủ sở hữu"
      >
        <div className="border-b border-border/60 px-5 py-4">
          <Skeleton className="h-3 w-28" />
        </div>
        <div className="grid grid-cols-1 min-[420px]:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="space-y-2 border-b border-border/50 px-5 py-4 last:border-b-0 min-[420px]:border-b-0 min-[420px]:border-r min-[420px]:last:border-r-0">
              <Skeleton className="h-3 w-14" />
              <Skeleton className="h-5 w-32" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!contributions || contributions.length === 0) return null;

  const total = contributions.reduce((s, c) => s + c.total_contribution, 0);
  return (
    <article className="overflow-hidden rounded-2xl border border-border/70 bg-card">
      <div className="border-b border-border/60 px-5 py-4">
        <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          Vốn chủ sở hữu
        </h3>
      </div>
      <dl className="grid grid-cols-1 min-[420px]:grid-cols-3">
        {[
          { owner: 'Tổng vốn', total_contribution: total },
          ...contributions,
        ].map((contribution, index) => (
          <div
            key={contribution.owner}
            className="min-w-0 border-b border-border/50 px-5 py-4 last:border-b-0 min-[420px]:border-b-0 min-[420px]:border-r min-[420px]:last:border-r-0"
          >
            <dt className="truncate text-xs text-muted-foreground">{contribution.owner}</dt>
            <dd
              className={
                index === 0
                  ? 'mt-1 break-words font-financial text-xl font-semibold tabular-nums tracking-tight text-foreground'
                  : 'mt-1 break-words font-financial text-base font-semibold tabular-nums text-foreground'
              }
            >
              {fmt(contribution.total_contribution)}
            </dd>
          </div>
        ))}
      </dl>
    </article>
  );
}
