// Payroll and payment history related types

export interface PaymentHistory {
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  project_id: number;
  project_name: string;
  position: string;
  total_paid_amount: number;
  paid_date: string; // ISO 8601 format (UTC)
}

export interface PaymentHistoryFilters {
  page?: number;
  pageSize?: number;
  sortBy?: 'paid_date' | 'employee_name' | 'amount' | 'project_name';
  sortOrder?: 'asc' | 'desc';
  fromDate?: string; // YYYY-MM-DD
  toDate?: string; // YYYY-MM-DD
  projectId?: number[];
  employeeId?: number[];
  position?: string;
  search?: string; // Search by employee name or CCCD
}

export interface PaymentHistoryListResponse {
  status: string;
  message: string;
  data: PaymentHistory[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}
