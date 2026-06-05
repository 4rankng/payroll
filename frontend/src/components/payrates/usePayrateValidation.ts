import { useState, useCallback, useMemo } from 'react';
import {
  PayrateConfig,
  validatePayrateStructure
} from './types';

export interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  field: string;
  message: string;
  path?: string[];
  type: 'error';
}

export interface ValidationWarning {
  field: string;
  message: string;
  path?: string[];
  type: 'warning';
}

export function usePayrateValidation() {
  const [lastValidation, setLastValidation] = useState<ValidationResult>({
    valid: true,
    errors: [],
    warnings: []
  });

  const validateWithEnhancements = useCallback((config: Partial<PayrateConfig>): ValidationResult => {
    const errors: ValidationError[] = [];
    const warnings: ValidationWarning[] = [];

    // Basic validation first
    if (config.rates) {
      const basicValidation = validatePayrateStructure(config.rates);
      basicValidation.errors.forEach(error => {
        errors.push({
          field: 'general',
          message: error,
          type: 'error'
        });
      });
    }

    // Enhanced validation with specific field targeting
    if (config.fromDate && config.toDate) {
      const fromDate = new Date(config.fromDate);
      const toDate = new Date(config.toDate);
      const daysDiff = (toDate.getTime() - fromDate.getTime()) / (1000 * 3600 * 24);

      if (daysDiff > 365) {
        warnings.push({
          field: 'toDate',
          message: 'Thời gian hiệu lực hơn 1 năm có thể cần được xem xét lại',
          type: 'warning'
        });
      }

      if (daysDiff < 30) {
        warnings.push({
          field: 'toDate',
          message: 'Thời gian hiệu lực ngắn hạn (< 1 tháng) có thể gây bất tiện',
          type: 'warning'
        });
      }
    }

    // Check for potential rate inconsistencies if rates exist
    if (config.rates) {
      const flatRates = flattenRates(config.rates);
      if (flatRates.length > 1) {
        const minRate = Math.min(...flatRates);
        const maxRate = Math.max(...flatRates);
        const ratio = maxRate / minRate;

        if (ratio > 10) {
          warnings.push({
            field: 'rates',
            message: `Chênh lệch mức lương quá lớn (${ratio.toFixed(1)}x). Hãy xem xét lại tính hợp lý.`,
            type: 'warning'
          });
        }

        // Check for suspiciously low rates
        if (minRate < 15000) {
          warnings.push({
            field: 'rates',
            message: `Mức lương thấp nhất (${minRate.toLocaleString('vi-VN')} đ) có thể thấp hơn mức tối thiểu.`,
            type: 'warning'
          });
        }
      }

      // Check for empty rates
      if (Object.keys(config.rates).length === 0) {
        errors.push({
          field: 'rates',
          message: 'Chưa có mức lương nào được cấu hình',
          type: 'error'
        });
      }
    }

    const result: ValidationResult = {
      valid: errors.length === 0,
      errors,
      warnings
    };

    setLastValidation(result);
    return result;
  }, []);

  const getFieldErrors = useCallback((fieldName: string): ValidationError[] => {
    return lastValidation.errors.filter(error => error.field === fieldName || error.field.startsWith(`${fieldName}.`));
  }, [lastValidation.errors]);

  const getFieldWarnings = useCallback((fieldName: string): ValidationWarning[] => {
    return lastValidation.warnings.filter(warning => warning.field === fieldName || warning.field.startsWith(`${fieldName}.`));
  }, [lastValidation.warnings]);

  const hasFieldErrors = useCallback((fieldName: string): boolean => {
    return getFieldErrors(fieldName).length > 0;
  }, [getFieldErrors]);

  const hasFieldWarnings = useCallback((fieldName: string): boolean => {
    return getFieldWarnings(fieldName).length > 0;
  }, [getFieldWarnings]);

  const summary = useMemo(() => ({
    totalErrors: lastValidation.errors.length,
    totalWarnings: lastValidation.warnings.length,
    isValid: lastValidation.valid,
    hasIssues: lastValidation.errors.length > 0 || lastValidation.warnings.length > 0
  }), [lastValidation]);

  return {
    validate: validateWithEnhancements,
    lastValidation,
    getFieldErrors,
    getFieldWarnings,
    hasFieldErrors,
    hasFieldWarnings,
    summary
  };
}

// Helper function to flatten rates
function flattenRates(category: Record<string, unknown>): number[] {
  const rates: number[] = [];

  function traverse(obj: Record<string, unknown> | undefined): void {
    if (!obj) return;
    Object.values(obj).forEach(value => {
      if (typeof value === 'number') {
        rates.push(value);
      } else if (typeof value === 'object' && value !== null) {
        traverse(value);
      }
    });
  }

  traverse(category);
  return rates;
}
