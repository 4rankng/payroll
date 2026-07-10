import { memo } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  CheckSquare,
  DollarSign,
  Users,
  Wallet,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { CashReadinessCard } from './CashReadinessCard';
import { PayrollMetricCard, type MetricCategory } from './PayrollMetricCard';
import { formatCurrency } from '@/utils/formatters';
import { cn } from '@/lib/utils';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';
import type { TimesheetSummaryResponse } from '@/types/api/timesheet.types';

/** Filter values a metric can apply. Subset of the management hook's status union. */
export type MetricFilter =
  | 'pending_approval'
  | 'pending_payment'
  | 'approved'
  | 'paid'
  | 'rejected'
  | 'all';

interface MetricDef {
  key: string;
  label: string;
  value: string;
  icon: LucideIcon;
  category: MetricCategory;
  supportText?: string;
  primary?: boolean;
  mobileSpan?: boolean;
  filter: MetricFilter;
}

interface PayrollControlCenterProps {
  cashReadiness: { data?: CashReadinessResponse; isLoading: boolean; isError: boolean };
  stats: {
    summary?: TimesheetSummaryResponse;
    /** Pending edit-request count (the "Yêu cầu sửa" figure). */
    editCount: number;
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
  const { summary, editCount, isLoading: statsLoading } = stats;
  const rejected = summary?.rejectedEntries ?? 0;

  const metrics: MetricDef[] = [
    {
      key: 'pending_approval',
      label: 'Chờ duyệt',
      value: fmtInt(summary?.pendingApproval),
      icon: AlertCircle,
      category: 'action',
      filter: 'pending_approval',
    },
    {
      key: 'pending_payment_count',
      label: 'Nhân viên chờ thanh toán',
      value: fmtInt(summary?.pendingEmployees),
      icon: Users,
      category: 'action',
      filter: 'pending_payment',
    },
    {
      key: 'pending_amount',
      label: 'Chờ thanh toán',
      value: formatCurrency(summary?.pendingPaymentAmount ?? 0),
      icon: Wallet,
      category: 'action',
      primary: true,
      mobileSpan: true,
      filter: 'pending_payment',
    },
    {
      key: 'approved',
      label: 'Đã duyệt',
      value: fmtInt(summary?.approvedEntries),
      icon: CheckSquare,
      category: 'done',
      filter: 'approved',
    },
    {
      key: 'paid',
      label: 'Đã thanh toán',
      value: fmtInt(summary?.paidEntries),
      icon: DollarSign,
      category: 'done',
      filter: 'paid',
    },
    {
      key: 'issue',
      label: 'Có vấn đề',
      value: fmtInt(rejected + editCount),
      icon: AlertTriangle,
      category: 'issue',
      supportText: `${fmtInt(editCount)} yêu cầu sửa · ${fmtInt(rejected)} bị loại`,
      mobileSpan: true,
      filter: 'rejected',
    },
  ];

  const handleClick = (m: MetricDef) => {
    onFilterChange(activeFilter === m.filter ? 'all' : m.filter);
  };

  return (
    <section
      aria-label="Trung tâm điều khiển bảng công"
      className={cn(
        'rounded-xl border border-border bg-card p-4 shadow-[var(--shadow-soft)] sm:p-5',
        className,
      )}
    >
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-12 lg:gap-6">
        {/* Cash readiness — 5 columns */}
        <div className="lg:col-span-5 lg:border-r lg:border-border lg:pr-6">
          <CashReadinessCard
            data={cashReadiness.data}
            isLoading={cashReadiness.isLoading}
            isError={cashReadiness.isError}
          />
        </div>

        {/* Operational KPIs — 7 columns */}
        <div className="lg:col-span-7">
          <div className="mb-2.5 flex items-center justify-between gap-2">
            <h2 className="font-display text-[13px] font-bold tracking-tight text-foreground">
              Chỉ số vận hành
            </h2>
            <span className="hidden text-[11px] text-muted-foreground sm:inline">
              Chọn một chỉ số để lọc danh sách
            </span>
          </div>

          <div className="grid auto-rows-fr grid-cols-2 gap-2.5 sm:grid-cols-3 sm:gap-3">
            {metrics.map((m) => (
              <PayrollMetricCard
                key={m.key}
                label={m.label}
                value={m.value}
                icon={m.icon}
                category={m.category}
                supportText={m.supportText}
                primary={m.primary}
                mobileSpan={m.mobileSpan}
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
