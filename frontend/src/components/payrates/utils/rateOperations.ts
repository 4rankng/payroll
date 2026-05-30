// Utility functions for rate operations like copying rates between positions
import type { PayrateStructure, DayType } from '../types';

/**
 * Copy rates from one position to another
 */
export function copyRatesFromPosition(
  rates: PayrateStructure,
  fromPosition: string,
  toPosition: string
): PayrateStructure {
  if (fromPosition === toPosition) return rates;

  const newRates = { ...rates };
  newRates[toPosition] = { ...newRates[fromPosition] };
  return newRates;
}

/**
 * Parse Vietnamese number format (remove dots and commas, parse as integer)
 */
export function parseVietnameseCurrency(value: string): number {
  const cleanValue = value.replace(/[^\d]/g, '');
  return cleanValue ? parseInt(cleanValue, 10) : 0;
}