import type { HealthLevel } from './types';
import { HEALTH_META } from './constants';

export function computeHealthMeta(
  summaryData: { count: number; error?: { total?: number }; latency: { avg: number } }[] | undefined,
  recentErrors: unknown[] | undefined,
): { level: HealthLevel; meta: (typeof HEALTH_META)[HealthLevel] } | null {
  if (!summaryData) return null;
  const total = summaryData.reduce((s, d) => s + d.count, 0);
  const errors = summaryData.reduce((s, d) => s + (d.error?.total ?? 0), 0);
  const errorRate = total > 0 ? (errors / total) * 100 : 0;
  const avgLat = total > 0
    ? summaryData.reduce((s, d) => s + d.latency.avg * d.count, 0) / total
    : 0;
  let level: HealthLevel = 'healthy';
  if (errorRate > 5 || avgLat > 500) level = 'critical';
  else if (errorRate > 1 || avgLat > 200 || (recentErrors?.length ?? 0) > 10) level = 'degraded';
  return { level, meta: HEALTH_META[level] };
}
