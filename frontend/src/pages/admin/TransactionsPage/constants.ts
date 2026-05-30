import { getInitialDateRange } from './utils';

const { fromDate, toDate } = getInitialDateRange();

export const SORT_FIELD_MAP: Record<string, string> = {
  created_at: 'created_at',
  description: 'description',
  transaction_type: 'transaction_type',
  party: 'party',
  amount: 'amount',
  status: 'status',
};

export const DEFAULT_FILTERS = {
  page: 1,
  pageSize: 50,
  sortBy: 'created_at',
  sortOrder: 'desc',
  fromDate,
  toDate,
};
