import { useState } from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { formatDate } from '@/utils/formatters';
import { useLedgerStatsConfig } from '@/hooks/ledger/useLedgerStatsConfig';
import type { LedgerSummary } from '@/types/api/financial.types';

interface LedgerSummaryCardMobileProps {
  summary?: LedgerSummary;
  isLoading: boolean;
  className?: string;
}

export function LedgerSummaryCardMobile({ summary, isLoading, className }: LedgerSummaryCardMobileProps) {
  const [showDetails, setShowDetails] = useState(false);

  const { mainStatsConfig, accountStatsConfig, isLoading: configLoading } = useLedgerStatsConfig({
    summary,
    isLoading,
  });

  if (isLoading || configLoading) {
    return (
      <div className={className}>
        <div className="grid grid-cols-2 gap-2 mb-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-10 rounded-xl" />
      </div>
    );
  }

  if (!summary) return null;

  return (
    <div className={className}>
      <p className="text-xs text-muted-foreground mb-3">
        Kỳ: {formatDate(summary.period.from)} – {formatDate(summary.period.to)}
      </p>

      {/* Main stats — 2-col grid */}
      <div className="grid grid-cols-2 gap-2 mb-3">
        {mainStatsConfig.map((stat, index) => {
          const Icon = stat.icon;
          return (
            <div key={index} className="rounded-xl border border-border bg-card p-3 shadow-sm">
              <div className="flex items-center gap-1.5 mb-1">
                {Icon && <Icon className="h-3.5 w-3.5 text-muted-foreground shrink-0" />}
                <p className="text-xs text-muted-foreground truncate">{stat.title}</p>
              </div>
              <p className="text-sm font-bold text-foreground tabular-nums truncate">{stat.value}</p>
            </div>
          );
        })}
      </div>

      {/* Account breakdown — collapsible */}
      {accountStatsConfig.length > 0 && (
        <Collapsible open={showDetails} onOpenChange={setShowDetails}>
          <CollapsibleTrigger asChild>
            <Button
              variant="ghost"
              className="w-full flex items-center justify-between h-9 px-3 bg-muted/50 hover:bg-muted rounded-xl text-sm font-medium text-foreground"
            >
              Chi tiết tài khoản ({accountStatsConfig.length})
              {showDetails ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="grid grid-cols-2 gap-2 pt-2">
              {accountStatsConfig.map((stat, index) => {
                const Icon = stat.icon;
                return (
                  <div key={index} className="rounded-xl border border-border bg-card p-2.5 shadow-sm">
                    <div className="flex items-center gap-1 mb-1">
                      {Icon && <Icon className="h-3 w-3 text-muted-foreground shrink-0" />}
                      <p className="text-xs text-muted-foreground truncate">{stat.title}</p>
                    </div>
                    <p className="text-xs font-bold text-foreground tabular-nums truncate">{stat.value}</p>
                  </div>
                );
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  );
}
