// Shared filter types

export interface FilterOption {
  label: string;
  value: string | number;
}

export interface DateRangeFilter {
  startDate?: string;
  endDate?: string;
}

export interface SelectFilter {
  value?: string | number;
  options: FilterOption[];
}

export interface MultiSelectFilter {
  values: (string | number)[];
  options: FilterOption[];
}

export interface SearchFilter {
  query: string;
  placeholder?: string;
}

export interface FilterState {
  search?: SearchFilter;
  dateRange?: DateRangeFilter;
  status?: SelectFilter;
  category?: SelectFilter;
  multiSelect?: MultiSelectFilter;
  [key: string]: unknown;
}

export interface FilterProps<T = FilterState> {
  filters: T;
  onFiltersChange: (filters: T) => void;
  loading?: boolean;
  resetFilters?: () => void;
}

export interface FilterResult<T = unknown> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}