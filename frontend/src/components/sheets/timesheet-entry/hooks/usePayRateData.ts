import { useCallback, useRef, useEffect, useMemo } from 'react';
import { useCurrentPayRate, isForbiddenError } from '@/hooks/api/usePayRates';
import type { RateCategory } from '@/types/api/payrate.types';

export const usePayRateData = (projectId: number) => {
  const { data: payRateData, isLoading: isLoadingPayRate, error: payRateError } = useCurrentPayRate(projectId || 0);
  // 403 = no access to this project's payrate (partner IDOR guard) — consumers
  // show a permission message instead of "project has no payrate config".
  const payRateForbidden = isForbiddenError(payRateError);
  const payRateDataRef = useRef(payRateData);

  useEffect(() => {
    payRateDataRef.current = payRateData;
  }, [payRateData]);

  // Helper function to get day types for a specific position
  // Optimized with ref to avoid recreation when payRateData changes
  const getDayTypesForPosition = useCallback((position: string) => {
    const rates = payRateDataRef.current?.rates;
    if (!rates) {
      return [];
    }

    const positionRates = rates[position];
    if (!positionRates || typeof positionRates !== 'object') return [];

    return Object.keys(positionRates);
  }, []); // Empty deps - uses ref

  // Helper function to get hour types for specific position and day type
  const getHourTypesForEntry = useCallback((position: string, dayType: string) => {
    const rates = payRateDataRef.current?.rates;
    if (!rates) {
      return [];
    }

    const positionRates = rates[position];
    if (!positionRates) return [];

    const dayRates = positionRates[dayType];
    if (!dayRates || typeof dayRates !== 'object') return [];

    return Object.keys(dayRates);
  }, []); // Empty deps - uses ref

  const getPayRateForEntry = useCallback((position: string, dayType: string, hourType: string): number => {
    const rates = payRateDataRef.current?.rates;
    if (!rates) {
      return 0;
    }

    const positionRates = rates[position];
    if (!positionRates || typeof positionRates !== 'object') {
      return 0;
    }

    const dayRates = (positionRates as Record<string, RateCategory | number>)[dayType];
    if (!dayRates) {
      return 0;
    }

    // If it's a direct number rate
    if (typeof dayRates === 'number') {
      return dayRates > 0 ? dayRates : 0;
    }

    // If it's an object with hour types
    if (typeof dayRates === 'object') {
      const hourConfig = (dayRates as Record<string, RateCategory | number>)[hourType];

      // Direct rate for this hour type
      if (typeof hourConfig === 'number') {
        return hourConfig > 0 ? hourConfig : 0;
      }

      // If it's a nested object, traverse to find a numeric rate
      if (hourConfig && typeof hourConfig === 'object') {
        const stack: RateCategory[] = [hourConfig as RateCategory];

        while (stack.length > 0) {
          const current = stack.pop() as RateCategory;

          for (const value of Object.values(current)) {
            if (typeof value === 'number') {
              if (value > 0) {
                return value;
              }
              continue;
            }

            if (value && typeof value === 'object') {
              stack.push(value as RateCategory);
            }
          }
        }
      }
    }

    return 0;
  }, []);

  const hasPayRateForEntry = useCallback((position: string, dayType: string, hourType: string) => {
    const rate = getPayRateForEntry(position, dayType, hourType);
    return rate > 0;
  }, [getPayRateForEntry]);

  const isPayRateReady = useMemo(() => {
    if (!payRateData?.rates) {
      return false;
    }

    return Object.keys(payRateData.rates).length > 0;
  }, [payRateData]);

  return {
    payRateData,
    isLoadingPayRate,
    payRateForbidden,
    isPayRateReady,
    getDayTypesForPosition,
    getHourTypesForEntry,
    hasPayRateForEntry,
    getPayRateForEntry
  };
};
