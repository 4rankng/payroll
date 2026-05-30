import { useState, useRef, useEffect, useCallback } from 'react';
import { usePreviewTimesheets } from '@/hooks/api/useTimesheets';
import { PREVIEW_DEBOUNCE_MS } from '@/constants/timesheetConstants';
import {
  generateEntryKey,
  transformEntriesToAPI,
  entriesEqual
} from '@/utils/timesheetTransformers';
import { hasEntryChangedFromOriginal } from './utils/entryChange';
import type { TimesheetEntry, MultiTimesheetFormData } from '../types/multi-timesheet.types';

// Entry validation cache interface
interface EntryValidationCache {
  [entryKey: string]: {
    isValid: boolean;
    errors: string[];
    timestamp: number;
    entry: TimesheetEntry;
  };
}

// Per-employee error map: employeeId -> error messages
// employeeId 0 = global errors not tied to a specific employee
type EmployeeErrorMap = Map<number, string[]>;

// Per-row error map: "employeeId|date" -> error messages
type RowErrorMap = Map<string, string[]>;

export const usePreviewValidation = (formData: MultiTimesheetFormData) => {
  const previewMutation = usePreviewTimesheets();

  const [entryValidationCache, setEntryValidationCache] = useState<EntryValidationCache>({});
  const [allEntriesValid, setAllEntriesValid] = useState<boolean>(false);
  const [employeeErrors, setEmployeeErrors] = useState<EmployeeErrorMap>(new Map());
  const [rowErrors, setRowErrors] = useState<RowErrorMap>(new Map());
  const [hasTriggeredPreview, setHasTriggeredPreview] = useState<boolean>(false);
  const previewTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const formDataRef = useRef(formData);
  const entryValidationCacheRef = useRef(entryValidationCache);

  useEffect(() => {
    formDataRef.current = formData;
    entryValidationCacheRef.current = entryValidationCache;
  }, [formData, entryValidationCache]);

  useEffect(() => {
    return () => {
      if (previewTimeoutRef.current) {
        clearTimeout(previewTimeoutRef.current);
        previewTimeoutRef.current = null;
      }
    };
  }, []);

  const triggerPreview = useCallback(() => {
    if (previewTimeoutRef.current) {
      clearTimeout(previewTimeoutRef.current);
    }

    previewTimeoutRef.current = setTimeout(async () => {
      try {
        const currentFormData = formDataRef.current;
        const currentCache = entryValidationCacheRef.current;

        const entriesToValidate: TimesheetEntry[] = [];
        const updatedCache: EntryValidationCache = { ...currentCache };

        currentFormData.entries.forEach(entry => {
          if (!hasEntryChangedFromOriginal(entry)) return;
          if (!entry.employeeId || !entry.date || !entry.projectId) return;

          // Deletion intent — auto-valid
          if (entry.originalValues && (!entry.hours || Object.keys(entry.hours).length === 0 || Object.values(entry.hours).every(v => v === 0))) {
            const key = generateEntryKey(entry);
            updatedCache[key] = { isValid: true, errors: [], timestamp: Date.now(), entry: { ...entry } };
            return;
          }

          if (!entry.hours || Object.keys(entry.hours).length === 0) return;

          const key = generateEntryKey(entry);
          const cached = currentCache[key];
          // Re-validate if: not cached, entry changed, or previously cached as invalid
          // (invalid entries may have been cached incorrectly due to co-entry errors)
          if (!cached || !cached.isValid || !entriesEqual(cached.entry, entry)) {
            entriesToValidate.push(entry);
          }
        });

        // Clear cache for removed entries
        const currentEntryKeys = new Set(currentFormData.entries.map(generateEntryKey));
        Object.keys(currentCache).forEach(key => {
          if (!currentEntryKeys.has(key)) delete updatedCache[key];
        });

        if (entriesToValidate.length === 0) {
          setEmployeeErrors(new Map());
          setRowErrors(new Map());
          setHasTriggeredPreview(false);
          setEntryValidationCache(updatedCache);

          const hasInvalidCached = currentFormData.entries.some(entry => {
            if (!hasEntryChangedFromOriginal(entry)) return false;
            if (entry.originalValues && (!entry.hours || Object.keys(entry.hours).length === 0 || Object.values(entry.hours).every(v => v === 0))) return false;
            if (!entry.hours || Object.keys(entry.hours).length === 0) return false;
            const key = generateEntryKey(entry);
            const cached = currentCache[key];
            return !cached || !cached.isValid;
          });

          setAllEntriesValid(!hasInvalidCached);
          return;
        }

        setHasTriggeredPreview(true);

        const apiEntries = transformEntriesToAPI(entriesToValidate);
        const result = await previewMutation.mutateAsync(apiEntries);

        if (result) {
          const hasErrors = Array.isArray(result.errors) && result.errors.length > 0;

          if (!hasErrors) {
            setEmployeeErrors(new Map());
            setRowErrors(new Map());

            entriesToValidate.forEach(entry => {
              const key = generateEntryKey(entry);
              updatedCache[key] = { isValid: true, errors: [], timestamp: Date.now(), entry: { ...entry } };
            });

            const hasInvalid = currentFormData.entries.some(entry => {
              if (!hasEntryChangedFromOriginal(entry)) return false;
              if (entry.originalValues && (!entry.hours || Object.keys(entry.hours).length === 0 || Object.values(entry.hours).every(v => v === 0))) return false;
              if (!entry.hours || Object.keys(entry.hours).length === 0) return false;
              const key = generateEntryKey(entry);
              const cached = updatedCache[key];
              return !cached || !cached.isValid;
            });

            setAllEntriesValid(!hasInvalid);
          } else {
            // Group errors by employeeId (for accordion banner) and by employeeId+date (for row highlight)
            const errorMap: EmployeeErrorMap = new Map();
            const rowErrorMap: RowErrorMap = new Map();
            for (const err of result.errors!) {
              const id = err.employeeId ?? 0;
              if (!errorMap.has(id)) errorMap.set(id, []);
              errorMap.get(id)!.push(err.message);

              if (err.date) {
                const rowKey = `${id}|${err.date}`;
                if (!rowErrorMap.has(rowKey)) rowErrorMap.set(rowKey, []);
                rowErrorMap.get(rowKey)!.push(err.message);
              }
            }
            setEmployeeErrors(errorMap);
            setRowErrors(rowErrorMap);

            // Build a set of errored employee+date pairs so only those entries are marked invalid
            const erroredRowKeys = new Set(result.errors!
              .filter(err => err.employeeId && err.date)
              .map(err => `${err.employeeId}|${err.date}`)
            );
            entriesToValidate.forEach(entry => {
              const key = generateEntryKey(entry);
              const rowKey = `${entry.employeeId}|${entry.date}`;
              const isValid = !erroredRowKeys.has(rowKey);
              updatedCache[key] = { isValid, errors: [], timestamp: Date.now(), entry: { ...entry } };
            });

            const hasInvalidAfterErrors = currentFormData.entries.some(entry => {
              if (!hasEntryChangedFromOriginal(entry)) return false;
              if (entry.originalValues && (!entry.hours || Object.keys(entry.hours).length === 0 || Object.values(entry.hours).every(v => v === 0))) return false;
              if (!entry.hours || Object.keys(entry.hours).length === 0) return false;
              const key = generateEntryKey(entry);
              const cached = updatedCache[key];
              return !cached || !cached.isValid;
            });
            setAllEntriesValid(!hasInvalidAfterErrors);
          }
        } else {
          setEmployeeErrors(new Map([[0, ['Lỗi không xác định khi kiểm tra dữ liệu']]]));
          setRowErrors(new Map());
          entriesToValidate.forEach(entry => {
            const key = generateEntryKey(entry);
            updatedCache[key] = { isValid: false, errors: [], timestamp: Date.now(), entry: { ...entry } };
          });
          setAllEntriesValid(false);
        }

        setEntryValidationCache(updatedCache);
      } catch (error) {
        console.error('Preview validation failed:', error);
        setAllEntriesValid(false);
      }
    }, PREVIEW_DEBOUNCE_MS);
  }, [previewMutation]);

  const isPreviewLoading = previewMutation.isPending;

  /** Returns errors for a specific row (employeeId + date) */
  const getErrorsForRow = useCallback((employeeId: number, date: string): string[] | null => {
    if (!hasTriggeredPreview) return null;
    const key = `${employeeId}|${date}`;
    const errs = rowErrors.get(key);
    return errs && errs.length > 0 ? errs : null;
  }, [rowErrors, hasTriggeredPreview]);

  /** Returns errors for a specific employee, or global errors (employeeId=0) if no id given */
  const getErrorsForEmployee = useCallback((employeeId: number): string[] | null => {
    if (!hasTriggeredPreview) return null;
    const errs = employeeErrors.get(employeeId);
    return errs && errs.length > 0 ? errs : null;
  }, [employeeErrors, hasTriggeredPreview]);

  /** Returns all errors flattened — used for global banner / save-button tooltip */
  const getValidationErrors = useCallback((): string[] | null => {
    if (!hasTriggeredPreview || employeeErrors.size === 0) return null;
    const all: string[] = [];
    employeeErrors.forEach(msgs => all.push(...msgs));
    return all.length > 0 ? all : null;
  }, [employeeErrors, hasTriggeredPreview]);

  const resetValidation = useCallback(() => {
    setEntryValidationCache({});
    setAllEntriesValid(false);
    setEmployeeErrors(new Map());
    setRowErrors(new Map());
    setHasTriggeredPreview(false);
  }, []);

  return {
    triggerPreview,
    isPreviewLoading,
    allEntriesValid,
    getValidationErrors,
    getErrorsForEmployee,
    getErrorsForRow,
    resetValidation
  };
};
