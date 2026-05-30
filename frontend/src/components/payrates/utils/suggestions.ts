// Utility functions for getting suggested positions and hour types
import type { PayrateStructure } from '../types';
import { getPositionsFromRates, getAllHourTypes } from '../types';
import { DEFAULT_POSITIONS, COMMON_HOUR_TYPES } from '@/types/api/payrate.types';

/**
 * Get suggested positions that are not already in the rates
 */
export function getSuggestedPositions(rates: PayrateStructure): string[] {
  const positions = getPositionsFromRates(rates);
  return DEFAULT_POSITIONS.filter(pos => !positions.includes(pos));
}

/**
 * Get suggested hour types that are not already in the rates
 */
export function getSuggestedHourTypes(rates: PayrateStructure): string[] {
  const hourTypes = getAllHourTypes(rates);
  const commonHourTypes = [
    ...COMMON_HOUR_TYPES.CATEGORIES,
    ...COMMON_HOUR_TYPES.TIME_RANGES
  ];
  return commonHourTypes.filter(hour => !hourTypes.includes(hour));
}
