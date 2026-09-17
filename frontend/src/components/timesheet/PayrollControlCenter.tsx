import { memo } from 'react';
import { CashReadinessCard } from './CashReadinessCard';
import { PayrollMetricCard } from './PayrollMetricCard';
import { formatCurrency } from '@/utils/formatters';
import { cn } from '@/lib/utils';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';
import type { TimesheetSummary } from '@/types/api/timesheet.types';

/** Filter values a metric can apply. Subset of the management hook's status union. */
export type MetricFilter =
  | 'pending_approval'
  | 'pending_payment'
  | 'approved'
  | 'paid'
  | 'all';

interface MetricDef {
  key: string;
  label: string;
  value: string;
  primary?: boolean;
  mobileSpan?: boolean;
  className?: string;
  filter: MetricFilter;
}

interface PayrollControlCenterProps {
  cashReadiness: { data?: CashReadinessResponse; isLoading: boolean; isError: boolean };
  stats: {
    summary?: TimesheetSummary;
    isLoading: boolean;
  };
  /** Current status filter applied to the table — drives KPI selection. */
  activeFilter: string;
  onFilterChange: (filter: MetricFilter) => void;
  className?: string;
}

const fmtInt = (n: number | null | undefined): string => (n ?? 0).toLocaleString('vi-VN');

/**
 * Unified "Payroll Control Center" — one panel pairing the cash-readiness
 * forecast (left, 5 cols) with the operational KPI grid (right, 7 cols).
 *
 * KPI selection is derived from `activeFilter` (the same state the table's
 * status dropdown uses), so the two never desync. Clicking a tile toggles
 * that filter.
 */
export const PayrollControlCenter = memo(function PayrollControlCenter({
  cashReadiness,
  stats,
  activeFilter,
  onFilterChange,
  className,
}: PayrollControlCenterProps) {
  const { summary, isLoading: statsLoading } = stats;

  const metrics: MetricDef[] = [
    {
      key: 'pending_approval',
      label: 'Chờ duyệt',
      value: fmtInt(summary?.pendingApproval),
      filter: 'pending_approval',
    },
    {
      key: 'pending_payment_count',
      label: 'Nhân viên chờ thanh toán',
      value: fmtInt(summary?.pendingEmployees),
      filter: 'pending_payment',
    },
    {
      key: 'pending_amount',
      label: 'Chờ thanh toán',
      value: formatCurrency(summary?.pendingPaymentAmount ?? 0),
      primary: true,
      mobileSpan: true,
      filter: 'pending_payment',
    },
    {
      key: 'approved',
      label: 'Đã duyệt',
      value: fmtInt(summary?.approvedEntries),
      filter: 'approved',
      className: 'col-span-2 min-[480px]:col-span-1',
    },
    {
      key: 'paid',
      label: 'Đã thanh toán',
      value: formatCurrency(summary?.paidAmount ?? 0),
      className: 'col-span-2 min-[480px]:col-span-1 lg:col-span-3',
      filter: 'paid',
    },
  ];

  const handleClick = (m: MetricDef) => {
    onFilterChange(activeFilter === m.filter ? 'all' : m.filter);
  };

  return (
    <section
      aria-label="Trung tâm điều khiển bảng công"
      className={cn(
        'rounded-xl border border-border bg-card p-3 shadow-[var(--shadow-soft)] sm:p-4',
        className,
      )}
    >
      {/* treasury-grid — the fading seam between the forecast zone and the KPI
          zone replaces the old hard lg:border-r divider. */}
      <div className="treasury-grid grid grid-cols-1 gap-3 lg:grid-cols-12 lg:gap-0">
        {/* Cash readiness — 5 columns. Zone ownership: this panel owns the
            forecast numbers (reserve, interval, reliability); the KPI grid owns
            the operational counts. No metric appears in both zones. */}
        <div className="lg:col-span-5 lg:pr-5">
          <CashReadinessCard
            data={cashReadiness.data}
            isLoading={cashReadiness.isLoading}
            isError={cashReadiness.isError}
          />
        </div>

        {/* Operational KPIs — 7 columns */}
        <div className="lg:col-span-7 lg:pl-5">
          <div className="mb-2 flex items-center justify-between gap-2">
            <h2 className="font-display text-[13px] font-bold tracking-tight text-foreground">
              Chỉ số vận hành
            </h2>
            <span className="hidden text-[11px] text-muted-foreground sm:inline">
              Chọn một chỉ số để lọc danh sách
            </span>
          </div>

          <div className="grid auto-rows-fr grid-cols-2 gap-2 lg:grid-cols-4">
            {metrics.map((m) => (
              <PayrollMetricCard
                key={m.key}
                label={m.label}
                value={m.value}
                primary={m.primary}
                mobileSpan={m.mobileSpan}
                className={m.className}
                selected={activeFilter === m.filter}
                isLoading={statsLoading}
                onClick={() => handleClick(m)}
              />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
});
