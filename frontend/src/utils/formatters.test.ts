import { describe, expect, it } from 'vitest';

import { formatFullCurrency, splitCurrencyDisplay } from './formatters';
import { formatProjectCurrency, formatBudgetDisplay } from './projectHelpers';
import { formatVietnameseCurrency } from './vietnamese';

describe('formatFullCurrency', () => {
  it('keeps full digits for million and billion amounts', () => {
    expect(formatFullCurrency(51_400_000)).toBe('51.400.000 đ');
    expect(formatFullCurrency(1_100_000_000)).toBe('1.100.000.000 đ');
  });

  it('supports negative values and symbol-free chart labels without abbreviations', () => {
    expect(formatFullCurrency(-1_250_000)).toBe('-1.250.000 đ');
    expect(formatFullCurrency(1_100_000_000, { showSymbol: false })).toBe('1.100.000.000');
  });
});

describe('full-number currency helpers', () => {
  it('keeps project budgets and Vietnamese currency values unabridged', () => {
    expect(formatBudgetDisplay(1_100_000_000)).toBe(formatProjectCurrency(1_100_000_000));
    expect(formatBudgetDisplay(1_100_000_000)).toBe('1.100.000.000 đ');
    expect(formatVietnameseCurrency(51_400_000)).toBe('51.400.000 đ');
  });
});

describe('splitCurrencyDisplay', () => {
  it('separates trailing VND symbols for secondary KPI typography', () => {
    expect(splitCurrencyDisplay('1.000.000.000 đ')).toEqual({
      amount: '1.000.000.000',
      unit: 'đ',
      isCurrencyUnit: true,
    });
    expect(splitCurrencyDisplay('1.000.000 ₫')).toEqual({
      amount: '1.000.000',
      unit: '₫',
      isCurrencyUnit: true,
    });
  });

  it('preserves non-currency units at their normal typography', () => {
    expect(splitCurrencyDisplay('12', 'người')).toEqual({
      amount: '12',
      unit: 'người',
      isCurrencyUnit: false,
    });
  });
});
