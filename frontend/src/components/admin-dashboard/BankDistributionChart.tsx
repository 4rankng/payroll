import { memo } from 'react';

import { formatFullCurrency } from '@/utils/formatters';

import type { BankDistributionItem } from './bank-distribution';

interface BankDistributionChartProps {
  items: BankDistributionItem[];
  totalEmployees: number;
}

interface BankDistributionRowProps {
  item: BankDistributionItem;
  rank: number;
}

const BankDistributionRow = memo(function BankDistributionRow({
  item,
  rank,
}: BankDistributionRowProps) {
  return (
    <li className="rounded-lg border border-border/60 bg-muted/20 p-2.5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-2.5">
          <span className="pt-0.5 text-[11px] font-semibold tabular-nums text-muted-foreground">
            {String(rank).padStart(2, '0')}
          </span>
          <span
            className="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: item.color }}
            aria-hidden="true"
          />
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-foreground">{item.name}</p>
            <div className="mt-0.5 flex flex-wrap items-center gap-x-2 text-xs tabular-nums text-muted-foreground">
              <span>{item.employeeCount.toLocaleString('vi-VN')} NV</span>
              {item.transferCount > 0 ? (
                <span>{item.transferCount.toLocaleString('vi-VN')} lần</span>
              ) : null}
              {item.totalPaidVnd > 0 ? (
                <span>{formatFullCurrency(item.totalPaidVnd)}</span>
              ) : null}
            </div>
          </div>
        </div>
        <span className="shrink-0 text-sm font-semibold tabular-nums text-foreground">
          {item.percentage}%
        </span>
      </div>

      <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted" aria-hidden="true">
        <div
          className="h-full rounded-full"
          style={{ width: `${item.percentage}%`, backgroundColor: item.color }}
        />
      </div>
    </li>
  );
});

export const BankDistributionChart = memo(function BankDistributionChart({
  items,
  totalEmployees,
}: BankDistributionChartProps) {
  const leadingBank = items[0];
  const distributionSummary = items
    .map((item) => `${item.shortName} ${item.percentage}%`)
    .join(', ');

  return (
    <div>
      <div className="flex flex-col gap-2.5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs font-medium text-muted-foreground">Tổng nhân sự</p>
          <p className="mt-1 flex items-baseline gap-1.5 text-2xl font-semibold tracking-tight text-foreground">
            <span className="tabular-nums">{totalEmployees.toLocaleString('vi-VN')}</span>
            <span className="text-sm font-medium tracking-normal text-muted-foreground">
              nhân viên
            </span>
          </p>
        </div>

        {leadingBank ? (
          <div className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-background px-2.5 py-1.5 sm:min-w-44">
            <div className="min-w-0">
              <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                Ngân hàng phổ biến nhất
              </p>
              <p className="truncate text-sm font-medium text-foreground">{leadingBank.shortName}</p>
            </div>
            <span className="text-lg font-semibold tabular-nums text-foreground">
              {leadingBank.percentage}%
            </span>
          </div>
        ) : null}
      </div>

      <div
        className="mt-3 flex h-2.5 overflow-hidden rounded-full bg-muted ring-1 ring-inset ring-border/40"
        role="img"
        aria-label={`Phân bổ ngân hàng nhận lương: ${distributionSummary}`}
      >
        {items.map((item) => (
          <span
            key={item.shortName}
            className="h-full border-r border-background/80 last:border-r-0"
            style={{ width: `${item.percentage}%`, backgroundColor: item.color }}
            aria-hidden="true"
          />
        ))}
      </div>

      <ol className="mt-3 grid gap-1.5 sm:grid-cols-2">
        {items.map((item, index) => (
          <BankDistributionRow key={item.shortName} item={item} rank={index + 1} />
        ))}
      </ol>
    </div>
  );
});
