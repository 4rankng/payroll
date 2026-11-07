import { memo } from 'react';

import { ConsoleDonut } from '@/components/shared/console-charts';
import { formatCompactVND } from '@/utils/formatters';

import type { BankDistributionItem } from './bank-distribution';

interface BankDistributionChartProps {
  items: BankDistributionItem[];
  totalEmployees: number;
}

/**
 * Console donut for bank salary distribution: one 10px ring (top-5 banks +
 * Khác), centre total paid, stacked legend with per-bank transactions,
 * compact amount, and whole-percent share of headcount.
 */
export const BankDistributionChart = memo(function BankDistributionChart({
  items,
  totalEmployees,
}: BankDistributionChartProps) {
  const totalPaidVnd = items.reduce((total, item) => total + item.totalPaidVnd, 0);
  const segments = items.map((item) => ({
    label: item.shortName,
    value: item.employeeCount,
    display: `${item.transferCount} gd · ${formatCompactVND(item.totalPaidVnd)}`,
    color: item.color,
  }));

  return (
    <div>
      <div className="flex flex-col gap-2.5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="console-kicker">Tổng nhân sự</p>
          <p className="mt-1 flex items-baseline gap-1.5 text-2xl font-semibold tracking-tight text-foreground">
            <span className="console-value">{totalEmployees.toLocaleString('vi-VN')}</span>
            <span className="text-sm font-medium tracking-normal text-muted-foreground">
              nhân viên
            </span>
          </p>
        </div>

        {items[0] ? (
          <div className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-card px-2.5 py-1.5 sm:min-w-44">
            <div className="min-w-0">
              <p className="console-kicker">Ngân hàng phổ biến nhất</p>
              <p className="truncate text-sm font-medium text-foreground">{items[0].shortName}</p>
            </div>
            <span className="console-value shrink-0 text-lg font-semibold text-foreground">
              {items[0].percentage}%
            </span>
          </div>
        ) : null}
      </div>

      <ConsoleDonut
        className="mt-3"
        segments={segments}
        centreValue={formatCompactVND(totalPaidVnd)}
        centreLabel="TỔNG CHI TRẢ"
      />
    </div>
  );
});
