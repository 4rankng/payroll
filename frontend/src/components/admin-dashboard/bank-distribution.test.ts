import { describe, expect, it } from 'vitest';

import type { BankUsageItem } from '@/types/api/dashboard.types';

import { buildBankDistribution } from './bank-distribution';

const createBank = (
  bank_name: string,
  employee_count: number,
  transfer_count = employee_count * 2,
  total_paid_vnd = employee_count * 1_000_000,
): BankUsageItem => ({
  bank_name,
  employee_count,
  transfer_count,
  total_paid_vnd,
  percentage: 0,
});

describe('buildBankDistribution', () => {
  it('sorts banks by employee count and calculates their percentage', () => {
    const result = buildBankDistribution(
      [
        createBank('Hàng hải (MSB)', 14),
        createBank('Quân đội (MB)', 58),
        createBank('Ngoại thương Việt Nam (VCB)', 8),
      ],
      100,
    );

    expect(result.map((item) => item.shortName)).toEqual(['MB', 'MSB', 'VCB']);
    expect(result.map((item) => item.percentage)).toEqual([58, 14, 8]);
  });

  it('groups banks after the first five and preserves their financial totals', () => {
    const result = buildBankDistribution(
      [
        createBank('Bank A (A)', 50),
        createBank('Bank B (B)', 20),
        createBank('Bank C (C)', 10),
        createBank('Bank D (D)', 8),
        createBank('Bank E (E)', 6),
        createBank('Bank F (F)', 4, 7, 40_000_000),
        createBank('Bank G (G)', 2, 5, 20_000_000),
      ],
      100,
    );

    expect(result).toHaveLength(6);
    expect(result[5]).toMatchObject({
      name: 'Ngân hàng khác',
      shortName: 'Khác',
      employeeCount: 6,
      percentage: 6,
      transferCount: 12,
      totalPaidVnd: 60_000_000,
    });
  });
});
