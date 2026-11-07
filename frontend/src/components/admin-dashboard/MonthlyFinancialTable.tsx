import { memo, useMemo, useState } from 'react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from "@/lib/utils";
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
    <Card variant="console" className="flex flex-col">
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
              <span className="console-value text-sm font-bold tabular-nums text-foreground">
                {formatVND(data.total.fee_earned)}
              </span>
            </div>
            {/* Phones: one stacked block per month — three money columns cannot
                be read side by side below md, they only produce sideways scroll. */}
            <ul className="divide-y divide-border/20 md:hidden">
              {visibleMonths.map((row) => (
                <li key={row.month} className="py-2">
                  <p className="text-xs font-semibold capitalize text-foreground">
                    {formatMonthLabel(row.month)}
                  </p>
                  <dl className="mt-1 space-y-0.5">
                    {[
                      { label: "Chi phí", value: row.paid_out },
                      { label: "Doanh thu", value: row.billed },
                      { label: "Lợi nhuận", value: row.fee_earned },
                    ].map((cell, index, all) => (
                      <div
                        key={cell.label}
                        className="flex items-baseline justify-between gap-3"
                      >
                        <dt className="text-xs text-muted-foreground">
                          {cell.label}
                        </dt>
                        <dd
                          className={cn(
                            "console-value text-xs tabular-nums text-foreground",
                            index === all.length - 1 && "font-semibold",
                          )}
                        >
                          {formatVND(cell.value)}
                        </dd>
                      </div>
                    ))}
                  </dl>
                </li>
              ))}
            </ul>
            {hasMore && (
              <button
                onClick={() => setShowAll(prev => !prev)}
                className="w-full py-2 text-xs font-semibold text-foreground active:opacity-70 md:hidden"
              >
                {showAll ? 'Thu gọn' : `Xem tất cả (${sortedMonths.length})`}
              </button>
            )}

            {/* md+: 36px console rows fit without a horizontal rail; the
                secondary "Chi phí" column returns at lg. */}
            <Table variant="console" className="hidden md:table">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-left">Tháng</TableHead>
                  <TableHead className="hidden text-right lg:table-cell">Chi phí</TableHead>
                  <TableHead className="text-right">Doanh thu</TableHead>
                  <TableHead className="text-right">Lợi nhuận</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleMonths.map((row) => (
                  <TableRow key={row.month}>
                    <TableCell className="font-medium capitalize">{formatMonthLabel(row.month)}</TableCell>
                    <TableCell className="num hidden lg:table-cell">{formatVND(row.paid_out)}</TableCell>
                    <TableCell className="num">{formatVND(row.billed)}</TableCell>
                    <TableCell className="num font-semibold">{formatVND(row.fee_earned)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
              {hasMore && (
                <TableFooter className="bg-transparent">
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={4} className="py-2 text-center">
                      <button
                        onClick={() => setShowAll(prev => !prev)}
                        className="text-xs font-semibold active:opacity-70 text-foreground"
                      >
                        {showAll ? 'Thu gọn' : `Xem tất cả (${sortedMonths.length})`}
                      </button>
                    </TableCell>
                  </TableRow>
                </TableFooter>
              )}
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
});

MonthlyFinancialTable.displayName = 'MonthlyFinancialTable';
