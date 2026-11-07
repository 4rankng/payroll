import { memo } from 'react';

import { ConsoleDonut } from '@/components/shared/console-charts';
import { ConsoleStatRow } from '@/components/shared/console-panel';
import { formatCompactVND } from '@/utils/formatters';

import type { BankDistributionItem } from './bank-distribution';

interface BankDistributionChartProps {
  items: BankDistributionItem[];
  totalEmployees: number;
}

/**
 * Console donut for bank salary distribution: one 10px ring (top-5 banks +
 * Khác), centre total paid, stacked legend with per-bank transactions, and
 * three compact stat chips. The old headline block and progress bars are gone
 * — the donut and the chips carry the same facts without repeating them.
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
      <ConsoleDonut
        segments={segments}
        centreValue={formatCompactVND(totalPaidVnd)}
        centreLabel="TỔNG CHI TRẢ"
      />
      <ConsoleStatRow
        className="mt-3"
        stats={[
          {
            label: 'Nhân sự',
            value: totalEmployees.toLocaleString('vi-VN'),
            hint: 'có tài khoản nhận lương',
          },
          {
            label: 'Dẫn đầu',
            value: items[0]?.shortName ?? '—',
            hint: items[0] ? `${items[0].percentage}% nhân sự` : undefined,
          },
          {
            label: 'Ngân hàng',
            value: items.length.toLocaleString('vi-VN'),
            hint: 'trong phân bổ',
          },
        ]}
      />
    </div>
  );
});
