/**
 * Centralized badge styling system
 * Uses index-based styling for consistent colors across different item types
 */

// Style arrays for different badge types
const POSITION_STYLES = [
  'bg-gray-50 text-gray-700 border-gray-200',      // Index 0
  'bg-blue-50 text-blue-700 border-blue-200',      // Index 1
  'bg-teal-50 text-teal-700 border-teal-200',      // Index 2
  'bg-green-50 text-green-700 border-green-200',   // Index 3
  'bg-orange-50 text-orange-700 border-orange-200', // Index 4
  'bg-pink-50 text-pink-700 border-pink-200',      // Index 5
  'bg-cyan-50 text-cyan-700 border-cyan-200',      // Index 6
] as const;

const DAY_TYPE_STYLES = [
  'bg-blue-50 text-blue-700 border-blue-300',      // ngày thường
  'bg-orange-50 text-orange-700 border-orange-300', // ngày nghỉ
  'bg-red-50 text-red-700 border-red-300',         // ngày lễ
] as const;

const HOUR_TYPE_STYLES = [
  'bg-emerald-50 text-emerald-700 border-emerald-200', // Index 0
  'bg-slate-50 text-slate-700 border-slate-200',       // Index 1
  'bg-amber-50 text-amber-700 border-amber-200',       // Index 2
  'bg-cyan-50 text-cyan-700 border-cyan-200',          // Index 3
  'bg-teal-50 text-teal-700 border-teal-200',          // Index 4
  'bg-rose-50 text-rose-700 border-rose-200',          // Index 5
  'bg-lime-50 text-lime-700 border-lime-200',          // Index 6
] as const;

/**
 * Get position badge style based on position index in the DEFAULT_POSITIONS array
 */
export function getPositionBadgeStyle(positionIndex: number): string {
  return POSITION_STYLES[positionIndex % POSITION_STYLES.length];
}

/**
 * Get day type badge style based on day type index in the COMMON_DAY_TYPES array
 */
export function getDayTypeBadgeStyle(dayTypeIndex: number): string {
  return DAY_TYPE_STYLES[dayTypeIndex % DAY_TYPE_STYLES.length];
}

/**
 * Get hour type badge style based on hour type index
 */
export function getHourTypeBadgeStyle(hourTypeIndex: number): string {
  return HOUR_TYPE_STYLES[hourTypeIndex % HOUR_TYPE_STYLES.length];
}

/**
 * Helper function to get index of an item in an array
 */
export function getItemIndex<T>(items: readonly T[], item: T): number {
  const index = items.indexOf(item);
  return index === -1 ? 0 : index; // Default to 0 if not found
}