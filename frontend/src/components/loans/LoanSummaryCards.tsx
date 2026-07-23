import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { Skeleton } from '@/components/ui/skeleton';
import { formatVND } from '@/utils/loanHelpers';
import { DollarSign, TrendingDown, Percent, CreditCard, type LucideIcon } from 'lucide-react';

interface LoansSummary {
  total_borrowed: number;
  total_outstanding: number;
  total_interest_paid: number;
  active_loans_count: number;
}

interface LoanSummaryCardsProps {
  summary?: LoansSummary;
  isLoading?: boolean;
}

// Compose the shared KpiHeroCard (used across admin/partner dashboards) instead
// of a loans-only watermark clone, so every summary strip stays consistent.
export function LoanSummaryCards({ summary, isLoading }: LoanSummaryCardsProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="rounded-xl border border-border/40 bg-card px-4 py-3 shadow-soft">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="mt-3 h-6 w-24" />
          </div>
        ))}
      </div>
    );
  }

  if (!summary) return null;

  const cards: Array<{
    title: string;
    value: string | number;
    icon: LucideIcon;
    color: 'blue' | 'rose' | 'amber' | 'teal';
  }> = [
    { title: 'Tổng vay', value: formatVND(summary.total_borrowed), icon: DollarSign, color: 'blue' },
    { title: 'Dư nợ hiện tại', value: formatVND(summary.total_outstanding), icon: TrendingDown, color: 'rose' },
    { title: 'Lãi đã trả', value: formatVND(summary.total_interest_paid), icon: Percent, color: 'amber' },
    { title: 'Khoản vay', value: summary.active_loans_count, icon: CreditCard, color: 'teal' },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      {cards.map(({ title, value, icon, color }) => (
        <KpiHeroCard key={title} label={title} value={value} icon={icon} color={color} className="h-full" />
      ))}
    </div>
  );
}
