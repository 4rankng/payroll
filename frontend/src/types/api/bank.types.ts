// Bank related types

export interface Bank {
  id: number;
  branch_name: string;
  created_at?: string;
  updated_at?: string;
}

export interface BankFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  search?: string;
}

export interface BankSearchParams {
  search: string;
  page?: number;
  pageSize?: number;
}

export interface BanksResponse {
  data: Bank[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface BankSearchResponse {
  data: {
    banks: Bank[];
  };
}

export interface CreateBankRequest {
  branch_name: string;
}

export interface UpdateBankRequest {
  branch_name?: string;
}