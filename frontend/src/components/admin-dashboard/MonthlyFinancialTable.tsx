import { memo, useMemo, useState } from 'react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useMonthlyFinancials } from '@/hooks/admin-dashboard/useMonthlyFinancials';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

function formatMonthLabel(month: string): string {
  try {
    return format(parseISO(`${month}-01`), 'MMM yyyy', { locale: vi });
  } catch {
    return month;
  }
}

export const MonthlyFinancialTable = memo(() => {
  const { data, isLoading, error } = useMonthlyFinancials('1y');
  const [showAll, setShowAll] = useState(false);

  const sortedMonths = useMemo(
    () => [...(data?.months ?? [])].sort((a, b) => b.month.localeCompare(a.month)),
    [data]
  );

  const visibleMonths = useMemo(
    () => showAll ? sortedMonths : sortedMonths.slice(0, 5),
    [sortedMonths, showAll]
  );

  const hasMore = sortedMonths.length > 5;

  return (
    <Card className="flex flex-col shadow-none">
      <CardContent className="p-0 pb-1">
        {isLoading ? (
          <div className="px-3 py-3 space-y-2">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-6 w-full rounded" />
            ))}
          </div>
        ) : error || !data ? (
          <div className="px-3 py-6 text-center text-xs text-foreground">
            Không thể tải dữ liệu
          </div>
        ) : data.months.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-foreground">
            Không có dữ liệu trong 12 tháng gần đây
          </div>
        ) : (
          <div className="overflow-x-auto focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring" tabIndex={0} role="region" aria-label="Tài chính theo tháng">
            {/* Total profit summary banner */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-border/40">
              <span className="text-xs font-medium text-foreground">Lợi nhuận tích lũy</span>
              <span className="text-sm font-bold tabular-nums text-foreground">
                {formatVND(data.total.fee_earned)}
              </span>
            </div>
            <table className="w-full min-w-[640px] text-xs">
              <thead>
                <tr className="border-b border-border/40">
                  <th className="text-left px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-foreground">Tháng</th>
                  <th className="text-right px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-foreground">Chi phí</th>
                  <th className="text-right px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-foreground">Doanh thu</th>
                  <th className="text-right px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-foreground">Lợi nhuận</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/20">
                {visibleMonths.map((row) => (
                  <tr key={row.month} className="transition-colors">
                    <td className="px-3 py-1.5 font-medium capitalize text-foreground">{formatMonthLabel(row.month)}</td>
                    <td className="px-3 py-1.5 text-right tabular-nums text-foreground">{formatVND(row.paid_out)}</td>
                    <td className="px-3 py-1.5 text-right tabular-nums text-foreground">{formatVND(row.billed)}</td>
                    <td className="px-3 py-1.5 text-right font-semibold tabular-nums text-foreground">
                      {formatVND(row.fee_earned)}
                    </td>
                  </tr>
                ))}
              </tbody>
              {hasMore && (
                <tfoot>
                  <tr>
                    <td colSpan={4} className="py-2 text-center">
                      <button
                        onClick={() => setShowAll(prev => !prev)}
                        className="text-xs font-semibold active:opacity-70 text-foreground"
                      >
                        {showAll ? 'Thu gọn' : `Xem tất cả (${sortedMonths.length})`}
                      </button>
                    </td>
                  </tr>
                </tfoot>
              )}
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
});

MonthlyFinancialTable.displayName = 'MonthlyFinancialTable';
