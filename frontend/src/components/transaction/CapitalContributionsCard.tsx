import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp } from 'lucide-react';
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
      <div className="space-y-1.5">
        <div className="flex items-center gap-1.5">
          <Skeleton className="h-3 w-3" />
          <Skeleton className="h-3 w-24" />
        </div>
        <div className="flex items-stretch gap-px rounded-xl border border-border/60 bg-border/40 overflow-hidden">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex flex-col items-center gap-1.5 px-4 py-3 bg-card flex-1">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-3 w-14" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!contributions || contributions.length === 0) return null;

  const total = contributions.reduce((s, c) => s + c.total_contribution, 0);
  const items = [
    { label: 'Tổng vốn', value: fmt(total) },
    ...contributions.map(c => ({ label: c.owner, value: fmt(c.total_contribution) })),
  ];

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5 px-0.5">
        <TrendingUp className="h-3 w-3 text-muted-foreground/60 shrink-0" />
        <span className="text-xs font-medium text-muted-foreground/70">Vốn chủ sở hữu</span>
      </div>
      <div className="flex items-stretch gap-px rounded-xl border border-border/60 bg-border/40 overflow-hidden">
        {items.map((item, i) => (
          <div
            key={i}
            className="flex flex-col items-center justify-center gap-1 px-3 py-3 flex-1 min-w-0 bg-card"
          >
            <span className="text-base font-semibold tabular-nums leading-none tracking-tight text-foreground truncate max-w-full">
              {item.value}
            </span>
            <span className="text-xs text-muted-foreground leading-none text-center whitespace-nowrap truncate max-w-full">
              {item.label}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
