import { dateToString } from '@/utils/dateHelpers';

export function getInitialDateRange() {
  const now = new Date();
  const oneYearAgo = new Date(now.getFullYear() - 1, now.getMonth(), 1);
  const endOfCurrentMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return {
    fromDate: dateToString(oneYearAgo),
    toDate: dateToString(endOfCurrentMonth),
  };
}
