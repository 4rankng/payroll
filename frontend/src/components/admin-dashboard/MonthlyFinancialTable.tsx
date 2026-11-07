import { memo, useMemo, useState } from 'react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { useMonthlyFinancials } from '@/hooks/admin-dashboard/useMonthlyFinancials';
import { formatFullCurrency as formatVND } from '@/utils/formatters';
import {
  ConsoleDelta,
  ConsolePanel,
  ConsoleStatRow,
} from '@/components/shared/console-panel';

function formatMonthLabel(month: string): string {
  try {
    return format(parseISO(`${month}-01`), 'MMM yyyy', { locale: vi });
  } catch {
    return month;
  }
}

interface MonthlyFinancialTableProps {
  className?: string;
}

export const MonthlyFinancialTable = memo(function MonthlyFinancialTable({
  className,
}: MonthlyFinancialTableProps) {
  const { data, isLoading, error } = useMonthlyFinancials('1y');
  const [showAll, setShowAll] = useState(false);

  const sortedMonths = useMemo(
    () => [...(data?.months ?? [])].sort((a, b) => b.month.localeCompare(a.month)),
    [data],
  );

  const visibleMonths = useMemo(
    () => (showAll ? sortedMonths : sortedMonths.slice(0, 5)),
    [sortedMonths, showAll],
  );

  const hasMore = sortedMonths.length > 5;

  // Observed vs previous period: the change beside each profit figure replaces
  // a sentence explaining the trend.
  const profitDelta = useMemo(() => {
    const map = new Map<string, number | null>();
    sortedMonths.forEach((row, index) => {
      const previous = sortedMonths[index + 1];
      map.set(
        row.month,
        previous && previous.fee_earned !== 0
          ? ((row.fee_earned - previous.fee_earned) / Math.abs(previous.fee_earned)) * 100
          : null,
      );
    });
    return map;
  }, [sortedMonths]);

  const latest = sortedMonths[0];
  const latestDelta = latest ? (profitDelta.get(latest.month) ?? null) : null;

  return (
    <ConsolePanel
      kicker="Sổ dòng tiền"
      title="Lịch sử tài chính"
      description="12 tháng gần nhất · so với tháng trước"
      className={className}
    >
      {isLoading ? (
        <div className="space-y-2">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-6 w-full rounded" />
          ))}
        </div>
      ) : error || !data ? (
        <div className="py-4 text-center text-[13px] text-foreground">Không thể tải dữ liệu</div>
      ) : data.months.length === 0 ? (
        <div className="py-4 text-center text-[13px] text-foreground">
          Không có dữ liệu trong 12 tháng gần đây
        </div>
      ) : (
        <>
          <ConsoleStatRow
            stats={[
              {
                label: 'Tích lũy',
                value: formatVND(data.total.fee_earned),
                title: formatVND(data.total.fee_earned),
              },
              {
                label: 'Tháng gần nhất',
                value: latest ? formatVND(latest.fee_earned) : '—',
                title: latest ? formatVND(latest.fee_earned) : undefined,
                hint: <ConsoleDelta value={latestDelta} />,
              },
              {
                label: 'Doanh thu',
                value: latest ? formatVND(latest.billed) : '—',
                title: latest ? formatVND(latest.billed) : undefined,
              },
            ]}
          />

          <div
            className="mt-3 overflow-x-auto focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
            tabIndex={0}
            role="region"
            aria-label="Tài chính theo tháng"
          >
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
                      { label: 'Chi phí', value: row.paid_out },
                      { label: 'Doanh thu', value: row.billed },
                      { label: 'Lợi nhuận', value: row.fee_earned },
                    ].map((cell, index, all) => (
                      <div key={cell.label} className="flex items-baseline justify-between gap-3">
                        <dt className="text-xs text-muted-foreground">{cell.label}</dt>
                        <dd
                          className={cn(
                            'console-value text-xs tabular-nums text-foreground',
                            index === all.length - 1 && 'font-semibold',
                          )}
                        >
                          {formatVND(cell.value)}
                        </dd>
                      </div>
                    ))}
                    <div className="flex items-baseline justify-between gap-3">
                      <dt className="text-xs text-muted-foreground">Δ lợi nhuận</dt>
                      <dd>
                        <ConsoleDelta value={profitDelta.get(row.month) ?? null} />
                      </dd>
                    </div>
                  </dl>
                </li>
              ))}
            </ul>
            {hasMore && (
              <button
                onClick={() => setShowAll((prev) => !prev)}
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
                  <TableHead className="whitespace-nowrap text-left">Tháng</TableHead>
                  <TableHead className="hidden text-right min-[1400px]:table-cell">Chi phí</TableHead>
                  <TableHead className="text-right">Doanh thu</TableHead>
                  <TableHead className="text-right">Lợi nhuận</TableHead>
                  <TableHead className="hidden text-right xl:table-cell">Δ</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleMonths.map((row) => (
                  <TableRow key={row.month}>
                    <TableCell className="whitespace-nowrap font-medium capitalize">
                      {formatMonthLabel(row.month)}
                    </TableCell>
                    <TableCell className="num hidden min-[1400px]:table-cell">
                      {formatVND(row.paid_out)}
                    </TableCell>
                    <TableCell className="num">{formatVND(row.billed)}</TableCell>
                    <TableCell className="num font-semibold">{formatVND(row.fee_earned)}</TableCell>
                    <TableCell className="num hidden xl:table-cell">
                      <ConsoleDelta value={profitDelta.get(row.month) ?? null} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
              {hasMore && (
                <TableFooter className="bg-transparent">
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={5} className="py-2 text-center">
                      <button
                        onClick={() => setShowAll((prev) => !prev)}
                        className="text-xs font-semibold text-foreground active:opacity-70"
                      >
                        {showAll ? 'Thu gọn' : `Xem tất cả (${sortedMonths.length})`}
                      </button>
                    </TableCell>
                  </TableRow>
                </TableFooter>
              )}
            </Table>
          </div>
        </>
      )}
    </ConsolePanel>
  );
});

MonthlyFinancialTable.displayName = 'MonthlyFinancialTable';
