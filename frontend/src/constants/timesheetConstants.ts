/**
 * Constants used in timesheet entry management
 */

/**
 * Preview validation debounce delay in milliseconds
 * Set to 500ms to balance between responsiveness and API load
 */
export const PREVIEW_DEBOUNCE_MS = 500;

/**
 * Preferred day type variants (Vietnamese)
 * Used when auto-selecting default day types
 */
export const PREFERRED_DAY_TYPES = ['ngay thuong', 'ngày thường'] as const;

/**
 * Default values for form initialization
 */
export const DEFAULTS = {
  PROJECT_ID: 0,
  EMPLOYEE_ID: 0,
  HOURS: {},
  DAY_TYPE: '',
  HOUR_TYPE: ''
} as const;

/**
 * Validation cache expiry time in milliseconds
 * Cache entries older than this will be considered stale
 */
export const CACHE_EXPIRY_MS = 5 * 60 * 1000; // 5 minutes
