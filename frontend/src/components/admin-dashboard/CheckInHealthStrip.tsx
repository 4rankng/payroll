import { memo, useCallback, useMemo, useState } from 'react';
import {
  Activity,
  AlertTriangle,
  type LucideIcon,
} from 'lucide-react';

import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';
import { cn } from '@/lib/utils';
import { useCheckInHealth } from '@/hooks/api/useDashboard';
import { HealthDrilldownSheet } from './HealthDrilldownSheet';
import type { CheckInHealthResponse } from '@/types/api/dashboard.types';

/**
 * Which drill-down sheet (if any) should be open.
 * - `failed-attempts`: list of failed check-in / check-out attempts; optional category filter.
 * - `quota-anomaly`: list of quota rows for the given anomaly type.
 */
type AttendanceListTarget = {
  type: 'attendance-list';
  label: string;
  status?: 'checked_in' | 'orphaned' | 'rejected';
  successfulCheckout?: boolean;
  zeroEarning?: boolean;
  emptyLabel: string;
};

export type HealthDrilldownTarget =
  | { type: 'failed-attempts'; category?: string; attemptType?: 'check_in' | 'check_out' }
  | { type: 'quota-anomaly'; anomalyType: string }
  | AttendanceListTarget
  | { type: 'successful-checkouts' }
  | null;

interface CheckInHealthStripProps {
  month?: string;
  /** Optional className wrapper for layout tuning */
  className?: string;
}

/**
 * Two-section health strip for the admin dashboard.
 *
 *   A. "Lỗi cần xử lý" — every anomaly across attendance, quota, and advance
 *      requests, rendered only when non-zero (each row keeps its drill-down).
 *      A single green all-clear row stands in when nothing is wrong.
 *   B. "Hoạt động" — the period's throughput figures: successful checkouts,
 *      advance request completion, salary credited, max advanceable amount.
 */
function CheckInHealthStripImpl({ month, className }: CheckInHealthStripProps) {
  const { data, isLoading } = useCheckInHealth(month);
  const [drilldown, setDrilldown] = useState<HealthDrilldownTarget>(null);

  const closeDrilldown = useCallback(() => setDrilldown(null), []);

  // ── Click handlers ──────────────────────────────────────────────────────
  const openFailedAttempts = useCallback((category?: string, attemptType?: 'check_in' | 'check_out') => {
    setDrilldown({ type: 'failed-attempts', category, attemptType });
  }, []);

  const openQuotaAnomaly = useCallback((anomalyType: string) => {
    setDrilldown({ type: 'quota-anomaly', anomalyType });
  }, []);

  const openAttendanceList = useCallback((target: AttendanceListTarget) => {
    setDrilldown(target);
  }, []);

  const openSuccessfulCheckouts = useCallback(() => {
    openAttendanceList({
      type: 'attendance-list',
      label: 'Chấm công ra thành công',
      successfulCheckout: true,
      emptyLabel: 'Không có ai chấm công ra thành công trong tháng này',
    });
  }, [openAttendanceList]);

  // ── Section A: every anomaly, one card ─────────────────────────────────
  const issues = useMemo(() => {
    if (!data) return [];
    const anomaly = (
      value: number,
      label: string,
      onClick?: () => void,
    ): InlineStatItem => ({
      label,
      value,
      highlight: value > 0,
      valueClassName: cn(value > 0 && 'text-financial-negative'),
      onClick,
    });

    const items: InlineStatItem[] = [
      anomaly(data.failed_check_in_today ?? 0, 'Lỗi vào làm', () => openFailedAttempts(undefined, 'check_in')),
      anomaly(data.failed_check_out_today ?? 0, 'Lỗi tan ca', () => openFailedAttempts(undefined, 'check_out')),
      anomaly(data.open_checked_in ?? 0, 'Đang chấm công', () => openAttendanceList({ type: 'attendance-list', label: 'Đang chấm công', status: 'checked_in', emptyLabel: 'Không có ca đang chấm công trong tháng này' })),
      anomaly(data.orphaned ?? 0, 'Thiếu dữ liệu ghép cặp', () => openAttendanceList({ type: 'attendance-list', label: 'Thiếu dữ liệu ghép cặp', status: 'orphaned', emptyLabel: 'Không có ca thiếu dữ liệu ghép cặp trong tháng này' })),
      anomaly(data.auto_rejected_today ?? 0, 'Tự động huỷ', () => openAttendanceList({ type: 'attendance-list', label: 'Tự động huỷ', status: 'rejected', emptyLabel: 'Không có ca tự động huỷ trong tháng này' })),
      anomaly(data.completed_zero_earning_today ?? 0, 'Ca không có lương', () => openAttendanceList({ type: 'attendance-list', label: 'Ca không có lương', zeroEarning: true, emptyLabel: 'Không có ca không có lương trong tháng này' })),
      anomaly(data.quota_invariant_drift ?? 0, 'Sai lệch hạn mức', () => openQuotaAnomaly('drift')),
      anomaly(data.missing_quota_rows ?? 0, 'Thiếu dòng hạn mức', () => openQuotaAnomaly('missing')),
      anomaly(data.stale_quota_after_disable ?? 0, 'Hạn mức cũ còn tồn tại', () => openQuotaAnomaly('stale')),
      anomaly(data.requests_stuck_pending ?? 0, 'Đang chờ quá lâu', () => openFailedAttempts('stuck_pending')),
      anomaly(data.requests_failed_today ?? 0, 'Lỗi yêu cầu hôm nay', () => openFailedAttempts('request_failed')),
    ].filter((item) => (item.value as number) > 0);

    if (items.length === 0) {
      items.push({ label: 'Không có lỗi', value: 0, valueClassName: 'text-financial-positive' });
    }
    return items;
  }, [data, openAttendanceList, openFailedAttempts, openQuotaAnomaly]);

  // ── Section B: throughput figures ──────────────────────────────────────
  const activity = useMemo(() => {
    if (!data) return [];
    const completed = data.requests_completed_today ?? 0;
    const total = data.requests_total_today ?? 0;

    const items: InlineStatItem[] = [
      { label: 'Chấm công ra thành công', value: data.successful_checkouts_today ?? 0, onClick: openSuccessfulCheckouts },
    ];

    // Same number ⇒ one row says it all; differ ⇒ show both.
    if (completed === total) {
      items.push({ label: 'Yêu cầu ứng hoàn tất/tổng', value: `${completed}/${total}` });
    } else {
      items.push({ label: 'Yêu cầu ứng hoàn tất', value: completed });
      items.push({ label: 'Yêu cầu ứng tổng cộng', value: total });
    }
    items.push({ label: 'Lương tháng này', value: data.quota_salary_this_month ?? 0 });
    items.push({ label: 'Số tiền ứng tối đa', value: data.quota_max_adv_this_month ?? 0 });
    return items;
  }, [data, openSuccessfulCheckouts]);

  return (
    <div className={cn('space-y-3', className)}>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <SectionCard
          title="Lỗi cần xử lý"
          icon={AlertTriangle}
          items={issues}
          isLoading={isLoading}
        />
        <SectionCard
          title="Hoạt động"
          icon={Activity}
          items={activity}
          isLoading={isLoading}
        />
      </div>

      <HealthDrilldownSheet
        target={drilldown}
        month={month}
        periodStart={data?.period_start}
        periodEnd={data?.period_end}
        onClose={closeDrilldown}
      />
    </div>
  );
}

export const CheckInHealthStrip = memo(CheckInHealthStripImpl);

// ─── Helpers ──────────────────────────────────────────────────────────────

interface SectionCardProps {
  title: string;
  icon: LucideIcon;
  items: InlineStatItem[];
  isLoading: boolean;
  className?: string;
}

const SectionCard = memo(function SectionCard({ title, icon: Icon, items, isLoading, className }: SectionCardProps) {
  return (
    <section className={cn('ct-card ct-card-border min-w-0 border-border/80 bg-card text-card-foreground shadow-none', className)}>
      <div className="ct-card-body min-w-0 gap-3 p-3 sm:p-4">
        <div className="flex min-w-0 items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-primary/15 bg-primary/5 text-primary">
            <Icon className="h-4 w-4" aria-hidden="true" />
          </div>
          <h3 className="ct-card-title min-w-0 text-sm font-semibold leading-snug text-foreground">
            {title}
          </h3>
        </div>
        <InlineStatStrip
          items={isLoading ? LOADING_PLACEHOLDER_ITEMS : items}
          isLoading={isLoading}
          variant="premium"
          direction="vertical"
        />
      </div>
    </section>
  );
});

// 4-row placeholder so the vertical skeleton renders while loading
const LOADING_PLACEHOLDER_ITEMS: InlineStatItem[] = Array.from({ length: 4 }, (_, i) => ({
  label: `.${i}`,
  value: 0,
}));
