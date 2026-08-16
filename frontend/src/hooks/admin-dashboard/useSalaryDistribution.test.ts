import { describe, expect, it } from 'vitest';

import { buildFixedBins, SALARY_BINS } from './useSalaryDistribution';

describe('salary distribution labels', () => {
  it('uses full VND thresholds without abbreviated suffixes', () => {
    expect(SALARY_BINS.map((bin) => bin.label)).toEqual([
      '< 1.500.000 ₫',
      '1.500.000 – 2.500.000 ₫',
      '2.500.000 – 3.500.000 ₫',
      '> 3.500.000 ₫',
    ]);

    expect(buildFixedBins([1_000_000, 2_000_000, 3_000_000, 4_000_000])).toEqual(
      SALARY_BINS.map((bin) => ({ label: bin.label, count: 1 })),
    );
  });
});
