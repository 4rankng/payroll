/**
 * Utilities for handling Vietnamese text normalization and comparison
 */

/**
 * Normalize Vietnamese text by removing diacritics and converting to lowercase
 * @param str - The string to normalize
 * @returns Normalized string without diacritics
 */
export const normalizeVietnamese = (str: string): string => {
  return str
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '');
};

/**
 * Find a day type from available options, preferring common variants
 * @param availableDayTypes - Array of available day type strings
 * @param preferredTypes - Preferred day type variants to search for (defaults to "ngày thường" variants)
 * @returns The matched day type or the first available day type
 */
export const findPreferredDayType = (
  availableDayTypes: string[],
  preferredTypes: string[] = ['ngay thuong', 'ngày thường']
): string => {
  if (availableDayTypes.length === 0) {
    return '';
  }

  // Try to find a preferred day type
  const preferred = availableDayTypes.find(dayType => {
    const normalized = normalizeVietnamese(dayType);
    return preferredTypes.some(pref =>
      normalized === normalizeVietnamese(pref)
    );
  });

  return preferred || availableDayTypes[0];
};

/**
 * Check if a day type matches any of the preferred variants
 * @param dayType - The day type to check
 * @param preferredTypes - Preferred day type variants
 * @returns True if the day type matches any preferred variant
 */
export const isPreferredDayType = (
  dayType: string,
  preferredTypes: string[] = ['ngay thuong', 'ngày thường']
): boolean => {
  const normalized = normalizeVietnamese(dayType);
  return preferredTypes.some(pref =>
    normalized === normalizeVietnamese(pref)
  );
};
