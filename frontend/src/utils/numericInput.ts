import type { KeyboardEvent, ChangeEvent } from 'react';

/**
 * Sanitizes input value to only allow digits and one decimal point
 */
export const sanitizeNumericInput = (value: string): string => {
  // Remove all non-digit and non-decimal point characters
  value = value.replace(/[^\d.]/g, '');

  // Ensure only one decimal point
  const decimalPoints = value.match(/\./g);
  if (decimalPoints && decimalPoints.length > 1) {
    value = value.replace(/\.+$/, '');
  }

  return value;
};

/**
 * Converts input string to number, treating empty as 0
 */
export const parseNumericInput = (value: string): number => {
  return value === '' ? 0 : parseFloat(value) || 0;
};

/**
 * Handles numeric input change, filtering and parsing the value
 */
export const handleNumericInputChange = (
  e: ChangeEvent<HTMLInputElement>,
  onChange: (value: number) => void
) => {
  const sanitizedValue = sanitizeNumericInput(e.target.value);
  const numValue = parseNumericInput(sanitizedValue);
  onChange(numValue);
};

/**
 * Checks if a key is allowed for numeric input
 */
export const isAllowedKey = (key: string): boolean => {
  const allowedKeys = [
    'Backspace', 'Delete', 'Tab', 'Escape', 'Enter',
    'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown',
    'Home', 'End', // Navigation keys
    '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
    '.', // Decimal point
  ];

  return allowedKeys.includes(key);
};

/**
 * Handles keyboard events for numeric input, preventing invalid characters
 */
export const handleNumericInputKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
  // Allow Ctrl/Cmd + A, C, V, X for select all, copy, paste, cut
  if ((e.ctrlKey || e.metaKey) && ['a', 'c', 'v', 'x'].includes(e.key.toLowerCase())) {
    return;
  }

  // If the key is not allowed, prevent it
  if (!isAllowedKey(e.key)) {
    e.preventDefault();
    return;
  }

  // Prevent multiple decimal points
  if (e.key === '.' && e.currentTarget.value.includes('.')) {
    e.preventDefault();
  }
};

/**
 * Props for numeric input handlers
 */
export interface NumericInputHandlers {
  onChange: (e: ChangeEvent<HTMLInputElement>) => void;
  onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void;
}

/**
 * Creates handlers for numeric input validation
 */
export const createNumericInputHandlers = (
  onChange: (value: number) => void
): NumericInputHandlers => ({
  onChange: (e: ChangeEvent<HTMLInputElement>) => {
    handleNumericInputChange(e, onChange);
  },
  onKeyDown: handleNumericInputKeyDown,
});