/**
 * Normalizes Vietnamese text for search and comparison
 * Removes diacritics (accent marks) and converts to lowercase
 */
const COMBINING_MARKS_REGEX = /[\u0300-\u036f]/g;
const SPECIAL_VIETNAMESE_MAP: Record<string, string> = {
  Đ: 'D',
  đ: 'd',
};

export const stripVietnameseDiacritics = (str: string): string => {
  if (!str) {
    return '';
  }

  return str
    .normalize('NFD')
    .replace(COMBINING_MARKS_REGEX, '')
    .replace(/[Đđ]/g, char => SPECIAL_VIETNAMESE_MAP[char]);
};

export const normalizeVietnamese = (str: string): string => {
  return stripVietnameseDiacritics(str).toLowerCase();
};

/**
 * Checks if a Vietnamese text contains a search term (case-insensitive, diacritic-insensitive)
 */
export const vietnameseIncludes = (text: string, searchTerm: string): boolean => {
  return normalizeVietnamese(text).includes(normalizeVietnamese(searchTerm));
};

/**
 * Compares two Vietnamese strings (case-insensitive, diacritic-insensitive)
 */
export const vietnameseEquals = (str1: string, str2: string): boolean => {
  return normalizeVietnamese(str1) === normalizeVietnamese(str2);
};
