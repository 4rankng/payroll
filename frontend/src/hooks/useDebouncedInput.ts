import { useState, useEffect, useRef, useCallback } from 'react';

/**
 * Manages a debounced search input.
 * - `localValue` drives the <input> directly (no jerk on every keystroke)
 * - `onCommit` fires after `debounceMs` of inactivity
 * - Syncs inward only when `externalValue` is cleared to '' (e.g. "clear filters")
 *
 * Uses a ref for `onCommit` so the debounce timer is never reset by parent re-renders
 * that pass a new function reference.
 */
export function useDebouncedInput(
  externalValue: string,
  onCommit: (value: string) => void,
  debounceMs = 300,
) {
  const [localValue, setLocalValue] = useState(externalValue);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Stable ref — never causes the useCallback below to re-create
  const onCommitRef = useRef(onCommit);
  useEffect(() => { onCommitRef.current = onCommit; }, [onCommit]);

  // Sync inward only on external clear
  useEffect(() => {
    if (externalValue === '') setLocalValue('');
  }, [externalValue]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setLocalValue(value);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => onCommitRef.current(value), debounceMs);
    },
    [debounceMs],
  );

  const handleClear = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setLocalValue('');
    onCommitRef.current('');
  }, []);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  return { localValue, handleChange, handleClear };
}
