import type { HealthLevel } from './types';

export const HEALTH_META: Record<HealthLevel, { label: string; dot: string; pill: string }> = {
  healthy: {
    label: 'Bình Thường',
    dot: 'bg-utility-success-500',
    pill: 'bg-utility-success-50 text-fg-success-primary border-utility-success-200',
  },
  degraded: {
    label: 'Cảnh Báo',
    dot: 'bg-utility-warning-500',
    pill: 'bg-utility-warning-50 text-fg-warning-primary border-utility-warning-200',
  },
  critical: {
    label: 'Nghiêm Trọng',
    dot: 'bg-utility-error-500',
    pill: 'bg-utility-error-50 text-fg-error-primary border-utility-error-200',
  },
};
