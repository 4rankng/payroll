import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format, startOfMonth, lastDayOfMonth, subMonths } from 'date-fns';
import { dashboardService } from '@/services/api/dashboard.service';

export function useSalaryDistribution(enabled = true) {
  const { fromDate, toDate } = useMemo(() => {
    const now = new Date();
    return {
      fromDate: format(startOfMonth(subMonths(now, 2)), 'yyyy-MM-dd'),
      toDate: format(lastDayOfMonth(now), 'yyyy-MM-dd'),
    };
  }, []);

  const query = useQuery({
    queryKey: ['dashboard', 'salary-distribution', fromDate, toDate],
    queryFn: () => dashboardService.getSalaryDistribution({ fromDate, toDate }),
    enabled,
  });

  return {
    ...query,
    fromDate,
    toDate,
  };
}

/** Cycle labels in Vietnamese */
export const CYCLE_LABELS: Record<string, string> = {
  weekly: 'Theo Tuần',
  monthly: 'Theo Tháng',
  flexible: 'Linh Hoạt',
};

/** Fixed salary bin definitions */
export const SALARY_BINS = [
  { label: '< 1.500.000 đ', min: -Infinity, max: 1_499_999 },
  { label: '1.500.000 – 2.500.000 đ', min: 1_500_000, max: 2_499_999 },
  { label: '2.500.000 – 3.500.000 đ', min: 2_500_000, max: 3_499_999 },
  { label: '> 3.500.000 đ', min: 3_500_000, max: Infinity },
] as const;

/** Bin salary values into fixed ranges */
export function buildFixedBins(values: number[]) {
  if (!values.length) return [];

  return SALARY_BINS.map((bin) => ({
    label: bin.label,
    count: values.filter((v) => v >= bin.min && v <= bin.max).length,
  })).filter((b) => b.count > 0);
}

/** Extract non-empty cycle sections from grouped data */
export function getCycleSections(data: Record<string, unknown> | undefined) {
  if (!data) return [];
  const sections: Array<{ cycle: string; label: string; values: number[]; summary: { count: number; mean: number; median: number; min: number; max: number; std_dev: number } }> = [];

  // Only show weekly and monthly — flexible employees are on a separate advance payment flow
  const allowedCycles = ['weekly', 'monthly'];

  for (const [cycle, cycleData] of Object.entries(data)) {
    if (!allowedCycles.includes(cycle)) continue;
    const typed = cycleData as { values?: number[]; summary?: { count: number; mean: number; median: number; min: number; max: number; std_dev: number } } | undefined;
    if (typed?.values && typed.values.length > 0 && typed.summary) {
      sections.push({
        cycle,
        label: CYCLE_LABELS[cycle] || cycle,
        values: typed.values,
        summary: typed.summary,
      });
    }
  }

  return sections;
}
