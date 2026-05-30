export function getDefaultDateRange(): { startDate: string; endDate: string } {
  const today = new Date();

/**
   * Business rules:
   * - Day 4–10  → range = 1–7   (current month)
   * - Day 11–17 → range = 8–14  (current month)
   * - Day 18–24 → range = 15–21 (current month)
   * - Day 25–end of month → range = 22–28 (current month)
   *
   * For day = < 4 → range = 22–28 (last month)
   */

  // Determine which calendar month we should base the range on
  const isFirstThreeDays = today.getDate() < 4;
  const referenceDate = isFirstThreeDays
    ? new Date(today.getFullYear(), today.getMonth(), 0) // last day of previous month
    : today;

  const currentDay = referenceDate.getDate();
  const year = referenceDate.getFullYear();
  const month = referenceDate.getMonth(); // 0-indexed

  let startDay: number;
  let endDay: number;

  // Determine which range to use based on current day
  if (currentDay >= 4 && currentDay <= 10) {
    startDay = 1;
    endDay = 7;
  } else if (currentDay >= 11 && currentDay <= 17) {
    startDay = 8;
    endDay = 14;
  } else if (currentDay >= 18 && currentDay <= 24) {
    startDay = 15;
    endDay = 21;
  } else {
    // Day 25–end of month (or days 1-3 of next month mapped to previous month)
    startDay = 22;
    endDay = 28;
  }

  // Format as YYYY-MM-DD
  const startDate = new Date(year, month, startDay);
  const endDate = new Date(year, month, endDay);

  const formatDate = (date: Date): string => {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
  };

  return {
    startDate: formatDate(startDate),
    endDate: formatDate(endDate)
  };
}
