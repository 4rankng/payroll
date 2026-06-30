import { memo, useCallback, useMemo, useState } from 'react';
import {
  Activity,
  CreditCard,
  ShieldCheck,
  Wallet,
  type LucideIcon,
} from 'lucide-react';

import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useCheckInHealth } from '@/hooks/api/useDashboard';
import { HealthDrilldownSheet } from './HealthDrilldownSheet';
import type { CheckInHealthResponse } from '@/types/api/dashboard.types';

/**
 * Which drill-down sheet (if any) should be open.
 * - `failed-attempts`: list of failed check-in / check-out attempts; optional category filter.
 * - `quota-anomaly`: list of quota rows for the given anomaly type.
 */
export type HealthDrilldownTarget =
  | { type: 'failed-attempts'; category?: string }
  | { type: 'quota-anomaly'; anomalyType: string }
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
  const openFailedAttempts = useCallback((category?: string) => {
    setDrilldown({ type: 'failed-attempts', category });
  }, []);

  const openQuotaAnomaly = useCallback((anomalyType: string) => {
    setDrilldown({ type: 'quota-anomaly', anomalyType });
  }, []);

  // ── Section A: Check-in / Check-out ─────────────────────────────────────
  const sectionA = useMemo(() => buildSectionItems(
    data,
    isLoading,
    [
      { key: 'failed_attempts_today', label: 'Thất bại', anomaly: true, drilldown: () => openFailedAttempts() },
      { key: 'open_checked_in', label: 'Đang chấm công', anomaly: true, drilldown: () => openFailedAttempts('open') },
      { key: 'orphaned', label: 'Thiếu dữ liệu ghép cặp', anomaly: true, drilldown: () => openFailedAttempts('orphaned') },
      { key: 'auto_rejected_today', label: 'Tự huỷ', anomaly: true, drilldown: () => openFailedAttempts('auto_rejected') },
      { key: 'completed_zero_earning_today', label: 'Ca không có lương', anomaly: true, drilldown: () => openFailedAttempts('zero_earning') },
      { key: 'successful_checkouts_today', label: 'Chấm công ra thành công', anomaly: false },
    ],
  ), [data, isLoading, openFailedAttempts]);

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
  ), [data, isLoading, openQuotaAnomaly]);

  // ── Section C: Advance requests ─────────────────────────────────────────
  const sectionC = useMemo(() => buildSectionItems(
    data,
    isLoading,
    [
      { key: 'requests_stuck_pending', label: 'Đang chờ quá lâu', anomaly: true, drilldown: () => openFailedAttempts('stuck_pending') },
      { key: 'requests_failed_today', label: 'Lỗi hôm nay', anomaly: true, drilldown: () => openFailedAttempts('request_failed') },
      { key: 'requests_completed_today', label: 'Hoàn tất', anomaly: false },
      { key: 'requests_total_today', label: 'Tổng cộng', anomaly: false },
    ],
  ), [data, isLoading, openFailedAttempts]);

  return (
    <div className={cn('space-y-2', className)}>
      <div className="flex items-center gap-2 px-0.5">
        <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
          <ShieldCheck className="h-3.5 w-3.5 text-primary/70" />
        </div>
        <span className="text-xs font-bold uppercase tracking-wider text-foreground">
          Sức khoẻ chấm công / ứng lương
        </span>
      </div>

      <div className="grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-3">
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
    <div className={cn('space-y-2', className)}>
      <div className="flex items-center gap-2 px-0.5">
        <Icon className="h-3.5 w-3.5 text-primary/70" />
        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </span>
      </div>
      <InlineStatStrip
        items={isLoading ? LOADING_PLACEHOLDER_ITEMS : items}
        isLoading={isLoading}
        variant="premium"
        direction="vertical"
      />
    </div>
  );
});

interface ItemSpec {
  key: keyof CheckInHealthResponse;
  label: string;
  anomaly: boolean;
  drilldown?: () => void;
}

/**
 * Build an InlineStatItem[] for a section. Anomaly tiles:
 *   - value 0 → green-muted styling
 *   - value > 0 → red/amber with highlight + onClick to open drill-down
 * Throughput tiles: default neutral styling, no onClick.
 */
function buildSectionItems(
  data: CheckInHealthResponse | undefined,
  _isLoading: boolean,
  specs: ItemSpec[],
): InlineStatItem[] {
  if (!data) return [];

  return specs.map(({ key, label, anomaly, drilldown }) => {
    const value = (data[key] as number) ?? 0;
    const isHealthyZero = anomaly && value === 0;
    const isAnomalyHit = anomaly && value > 0;

    return {
      label,
      value,
      highlight: isAnomalyHit,
      valueClassName: cn(
        isHealthyZero && 'text-financial-positive',
        isAnomalyHit && 'text-financial-negative',
      ),
      onClick: drilldown,
    } satisfies InlineStatItem;
  });
}

// 4-row placeholder so the vertical skeleton renders while loading
const LOADING_PLACEHOLDER_ITEMS: InlineStatItem[] = Array.from({ length: 4 }, (_, i) => ({
  label: `.${i}`,
  value: 0,
}));
