import type { ImportError } from '@/types/api/timesheet.types';

/**
 * Parse the JSON-encoded error_detail string from a PartnerImportFile.
 * Returns an empty array if the detail is missing, empty, or invalid JSON.
 */
export function parseImportErrors(detail?: string | null): ImportError[] {
  if (!detail) return [];
  try {
    return JSON.parse(detail);
  } catch {
    return [];
  }
}
