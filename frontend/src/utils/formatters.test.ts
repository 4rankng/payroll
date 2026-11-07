import { describe, expect, it } from 'vitest';

import { formatCompactVND, formatFullCurrency, splitCurrencyDisplay } from './formatters';
import { formatProjectCurrency, formatBudgetDisplay } from './projectHelpers';
import { formatVietnameseCurrency } from './vietnamese';

describe('formatFullCurrency', () => {
  it('keeps full digits for million and billion amounts', () => {
    expect(formatFullCurrency(51_400_000)).toBe('51.400.000 ₫');
    expect(formatFullCurrency(1_100_000_000)).toBe('1.100.000.000 ₫');
  });

  it('supports negative values and symbol-free chart labels without abbreviations', () => {
    expect(formatFullCurrency(-1_250_000)).toBe('-1.250.000 ₫');
    expect(formatFullCurrency(1_100_000_000, { showSymbol: false })).toBe('1.100.000.000');
  });
});

describe('full-number currency helpers', () => {
  it('keeps project budgets and Vietnamese currency values unabridged', () => {
    expect(formatBudgetDisplay(1_100_000_000)).toBe(formatProjectCurrency(1_100_000_000));
    expect(formatBudgetDisplay(1_100_000_000)).toBe('1.100.000.000 ₫');
    expect(formatVietnameseCurrency(51_400_000)).toBe('51.400.000 ₫');
  });
});

describe('splitCurrencyDisplay', () => {
  it('separates trailing VND symbols for secondary KPI typography', () => {
    expect(splitCurrencyDisplay('1.000.000.000 đ')).toEqual({
      amount: '1.000.000.000',
      unit: '₫',
      isCurrencyUnit: true,
    });
    expect(splitCurrencyDisplay('1.000.000 ₫')).toEqual({
      amount: '1.000.000',
      unit: '₫',
      isCurrencyUnit: true,
    });
  });

  it('keeps non-currency units separate for secondary KPI typography', () => {
    expect(splitCurrencyDisplay('12', 'người')).toEqual({
      amount: '12',
      unit: 'người',
      isCurrencyUnit: false,
    });
    expect(splitCurrencyDisplay('12', ' đ')).toEqual({
      amount: '12',
      unit: '₫',
      isCurrencyUnit: true,
    });
  });
});

describe('formatCompactVND', () => {
  it('returns an em dash for null, undefined and NaN', () => {
    expect(formatCompactVND(null)).toBe('—');
    expect(formatCompactVND(undefined)).toBe('—');
    expect(formatCompactVND(Number.NaN)).toBe('—');
  });

  it('keeps sub-thousand figures exact', () => {
    expect(formatCompactVND(0)).toBe('0');
    expect(formatCompactVND(999)).toBe('999');
  });

  it('abbreviates thousands with ng', () => {
    expect(formatCompactVND(1_000)).toBe('1 ng');
    expect(formatCompactVND(850_000)).toBe('850 ng');
  });

  it('abbreviates millions with one-decimal tr', () => {
    expect(formatCompactVND(1_000_000)).toBe('1 tr');
    expect(formatCompactVND(45_200_000)).toBe('45,2 tr');
  });

  it('abbreviates billions with two-decimal tỷ', () => {
    expect(formatCompactVND(1_000_000_000)).toBe('1 tỷ');
    expect(formatCompactVND(1_030_000_000)).toBe('1,03 tỷ');
  });
});
