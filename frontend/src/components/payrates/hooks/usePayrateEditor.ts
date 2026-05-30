import { useMemo } from 'react';
import type { PayrateStructure, DayType, ValidationResult } from '../types';
import { createDefaultPayrateStructure, updateRateValue } from '../types';
import { parseVietnameseCurrency } from '../utils/rateOperations';

export function usePayrateEditor(
  rates: PayrateStructure,
  onChange: (rates: PayrateStructure) => void,
  readOnly: boolean = false
) {
  // Check if original rates are empty BEFORE processing
  const isEmpty = useMemo(() => {
    return !rates || Object.keys(rates).length === 0 || 
           Object.values(rates).every(dayTypes => 
             !dayTypes || Object.keys(dayTypes).length === 0
           );
  }, [rates]);

  // Initialize with default structure if empty and not read-only
  const displayRates = useMemo(() => {
    if (isEmpty && !readOnly) {
      return createDefaultPayrateStructure();
    }
    return rates || {};
  }, [rates, isEmpty, readOnly]);

  const handleRateChange = (position: string, dayType: DayType, hourType: string, value: string) => {
    if (readOnly) return;
    
    const numValue = parseVietnameseCurrency(value);
    const newRates = updateRateValue(displayRates, position, dayType, hourType, numValue);
    onChange(newRates);
  };

  return {
    displayRates,
    handleRateChange,
    isEmpty,
  };
}