import { memo, useCallback, useMemo, useState } from 'react';
import {
  Activity,
  CreditCard,
  Wallet,
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
 * Compact 3-section health strip for the admin dashboard.
 *
 * Sections:
 *   A. Check-in / Check-out throughput + anomalies
 *   B. Quota invariant health
 *   C. Advance request pipeline
 *
 * Anomaly tiles (healthy = 0) render green when 0, red/amber when > 0.
 * Throughput tiles render in the default neutral style.
 * Clicking an anomaly tile opens a drill-down sheet (rendered internally).
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

  // ── Section A: Check-in / Check-out ─────────────────────────────────────
  const sectionA = useMemo(() => buildSectionItems(
    data,
    isLoading,
    [
      { key: 'failed_check_in_today', label: 'Lỗi vào làm', anomaly: true, drilldown: () => openFailedAttempts(undefined, 'check_in') },
      { key: 'failed_check_out_today', label: 'Lỗi tan ca', anomaly: true, drilldown: () => openFailedAttempts(undefined, 'check_out') },
      { key: 'open_checked_in', label: 'Đang chấm công', anomaly: true, drilldown: () => openAttendanceList({ type: 'attendance-list', label: 'Đang chấm công', status: 'checked_in', emptyLabel: 'Không có ca đang chấm công trong tháng này' }) },
      { key: 'orphaned', label: 'Thiếu dữ liệu ghép cặp', anomaly: true, drilldown: () => openAttendanceList({ type: 'attendance-list', label: 'Thiếu dữ liệu ghép cặp', status: 'orphaned', emptyLabel: 'Không có ca thiếu dữ liệu ghép cặp trong tháng này' }) },
      { key: 'auto_rejected_today', label: 'Tự động huỷ', anomaly: true, drilldown: () => openAttendanceList({ type: 'attendance-list', label: 'Tự động huỷ', status: 'rejected', emptyLabel: 'Không có ca tự động huỷ trong tháng này' }) },
      { key: 'completed_zero_earning_today', label: 'Ca không có lương', anomaly: true, drilldown: () => openAttendanceList({ type: 'attendance-list', label: 'Ca không có lương', zeroEarning: true, emptyLabel: 'Không có ca không có lương trong tháng này' }) },
      { key: 'successful_checkouts_today', label: 'Chấm công ra thành công', anomaly: false, drilldown: () => openSuccessfulCheckouts() },
    ],
    'Không có lỗi chấm công',
  ), [data, isLoading, openAttendanceList, openFailedAttempts, openSuccessfulCheckouts]);

  // ── Section B: Quota ────────────────────────────────────────────────────
  const sectionB = useMemo(() => buildSectionItems(
    data,
    isLoading,
    [
      { key: 'quota_invariant_drift', label: 'Sai lệch hạn mức', anomaly: true, drilldown: () => openQuotaAnomaly('drift') },
      { key: 'missing_quota_rows', label: 'Thiếu dòng hạn mức', anomaly: true, drilldown: () => openQuotaAnomaly('missing') },
      { key: 'stale_quota_after_disable', label: 'Hạn mức cũ còn tồn tại', anomaly: true, drilldown: () => openQuotaAnomaly('stale') },
      { key: 'quota_salary_this_month', label: 'Lương tháng này', anomaly: false },
      { key: 'quota_max_adv_this_month', label: 'Số tiền ứng tối đa', anomaly: false },
    ],
    'Không có sai lệch hạn mức',
  ), [data, isLoading, openQuotaAnomaly]);

  // ── Section C: Advance requests ─────────────────────────────────────────
  const sectionC = useMemo(() => {
    if (!data) return [];
    const stuck = data.requests_stuck_pending ?? 0;
    const failed = data.requests_failed_today ?? 0;
    const completed = data.requests_completed_today ?? 0;
    const total = data.requests_total_today ?? 0;

    const items: InlineStatItem[] = [];
    if (stuck + failed === 0) {
      items.push({ label: 'Không có yêu cầu lỗi', value: 0, valueClassName: 'text-financial-positive' });
    } else {
      if (stuck > 0) items.push({ label: 'Đang chờ quá lâu', value: stuck, highlight: true, valueClassName: 'text-financial-negative', onClick: () => openFailedAttempts('stuck_pending') });
      if (failed > 0) items.push({ label: 'Lỗi hôm nay', value: failed, highlight: true, valueClassName: 'text-financial-negative', onClick: () => openFailedAttempts('request_failed') });
    }

    // Same number ⇒ one row says it all; differ ⇒ show both.
    if (completed === total) {
      items.push({ label: 'Hoàn tất/Tổng cộng', value: `${completed}/${total}` });
    } else {
      items.push({ label: 'Hoàn tất', value: completed });
      items.push({ label: 'Tổng cộng', value: total });
    }
    return items;
  }, [data, openFailedAttempts]);

  return (
    <div className={cn('space-y-3', className)}>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
        <SectionCard
          title="Chấm công"
          icon={Activity}
          items={sectionA}
          isLoading={isLoading}
        />
        <SectionCard
          title="Hạn mức ứng lương"
          icon={Wallet}
          items={sectionB}
          isLoading={isLoading}
        />
        <SectionCard
          title="Yêu cầu ứng lương"
          icon={CreditCard}
          items={sectionC}
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

interface ItemSpec {
  key: keyof CheckInHealthResponse;
  label: string;
  anomaly: boolean;
  drilldown?: () => void;
}

/**
 * Build a condensed InlineStatItem[] for a section:
 *   - anomaly rows render only when non-zero (they carry drill-downs, so they
 *     appear exactly when admins need to click them)
 *   - when every anomaly row is zero, one green "all clear" row stands in
 *   - throughput rows always render
 */
function buildSectionItems(
  data: CheckInHealthResponse | undefined,
  _isLoading: boolean,
  specs: ItemSpec[],
  cleanLabel: string,
): InlineStatItem[] {
  if (!data) return [];

  const items: InlineStatItem[] = [];
  let anomaliesShown = 0;
  for (const { key, label, anomaly, drilldown } of specs) {
    const value = (data[key] as number) ?? 0;
    const isAnomalyHit = anomaly && value > 0;
    if (anomaly && value === 0) continue;
    if (isAnomalyHit) anomaliesShown++;

    items.push({
      label,
      value,
      highlight: isAnomalyHit,
      valueClassName: cn(isAnomalyHit && 'text-financial-negative'),
      onClick: drilldown,
    } satisfies InlineStatItem);
  }

  if (anomaliesShown === 0) {
    items.unshift({
      label: cleanLabel,
      value: 0,
      valueClassName: 'text-financial-positive',
    });
  }
  return items;
}

// 4-row placeholder so the vertical skeleton renders while loading
const LOADING_PLACEHOLDER_ITEMS: InlineStatItem[] = Array.from({ length: 4 }, (_, i) => ({
  label: `.${i}`,
  value: 0,
}));
