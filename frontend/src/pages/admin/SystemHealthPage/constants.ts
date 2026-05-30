import type { HealthLevel } from './types';

export const HEALTH_META: Record<HealthLevel, { label: string; dot: string; pill: string }> = {
  healthy: {
    label: 'Bình Thường',
    dot: 'bg-emerald-500',
    pill: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  },
  degraded: {
    label: 'Cảnh Báo',
    dot: 'bg-amber-500',
    pill: 'bg-amber-50 text-amber-700 border-amber-200',
  },
  critical: {
    label: 'Nghiêm Trọng',
    dot: 'bg-red-500',
    pill: 'bg-red-50 text-red-700 border-red-200',
  },
};
