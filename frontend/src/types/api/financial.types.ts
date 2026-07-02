// Financial related types

export interface FinancialDashboard {
  totalRevenue: number;
  totalExpenses: number;
  netProfit: number;
  outstandingReceivables: number;
  pendingPayables: number;
  cashBalance: number;
  profitMargin: number;
  monthlyTrend: {
    revenue: number[];
    expenses: number[];
    months: string[];
  };
  topProjects: Array<{
    projectId: number;
    projectName: string;
    revenue: number;
    expenses: number;
    profit: number;
    profitMargin: number;
  }>;
  lastUpdated: string;
}

export interface LedgerEntry {
  id: number;
  account: string; // Updated to allow any account type from API
  party: string;
  description: string;
  debit: number;
  credit: number;
  balance: number; // Running balance (cash - payable + receivable)
  net_amount: number; // Intuitive balance for user display
  date: string;
  url?: string; // Evidence URL - external link or local file path
  asset_id?: number; // ID of uploaded evidence file
  asset?: Asset; // Nested asset object when asset_id is present
  reference?: string; // Reference field for reversals
  project_id?: number; // Optional project association
  created_by: number;
  created_at: string;
  updated_at: string;
  // Additional fields for display
  createdByName?: string;
}

export interface CreateLedgerEntry {
  account: string; // Updated to allow any account type
  party: string;
  description: string;
  debit: number;
  credit: number;
  date: string;
  url?: string; // Evidence URL - external link or local file path
  asset_id?: number; // ID of uploaded evidence file (single asset only)
}

export interface AccountBalance {
  account: string;
  currentBalance: number;
  totalDebit: number;
  totalCredit: number;
  entryCount: number;
  lastTransactionDate: string;
  balanceAsOfDate: string;
}

export interface Receivables {
  totalReceivables: number;
  overdueReceivables: number;
  currentReceivables: number;
  averageDaysOverdue: number;
  receivables: Array<{
    projectId: number;
    projectName: string;
    clientName: string;
    totalOutstanding: number;
    dueDate: string;
    daysOverdue: number;
    invoices: Array<{
      invoiceNumber: string;
      amount: number;
      issueDate: string;
      dueDate: string;
      status: 'pending' | 'paid' | 'overdue';
    }>;
  }>;
}

export interface Payables {
  totalPayables: number;
  overduePayables: number;
  currentPayables: number;
  payables: Array<{
    projectId: number;
    projectName: string;
    totalOutstanding: number;
    employeeCount: number;
    oldestPayableDate: string;
    payrollBatches: Array<{
      batchCode: string;
      amount: number;
      employeeCount: number;
      payDate: string;
      status: 'pending' | 'processing' | 'completed';
    }>;
  }>;
}

export interface CashFlowAnalysis {
  currentCashPosition: number;
  projectedCashFlow: Array<{
    month: string;
    inflow: number;
    outflow: number;
    netFlow: number;
    endingBalance: number;
  }>;
  riskAnalysis: {
    cashRunwayDays: number;
    minimumCashDate: string;
    riskLevel: 'low' | 'medium' | 'high';
    recommendations: string[];
  };
}

export interface FinancialReport {
  reportId: string;
  reportType: 'profit_loss' | 'balance_sheet' | 'cash_flow' | 'project_summary';
  status: 'processing' | 'completed' | 'failed';
  downloadUrl?: string;
  filename?: string;
  fileSize?: number;
  expiresAt?: string;
  createdAt: string;
  completedAt?: string;
  estimatedCompletionTime?: string;
}

export interface GenerateReportRequest {
  dateStart: string;
  dateEnd: string;
  projectIds?: number[];
  includeProjectBreakdown?: boolean;
  includeEmployeeDetails?: boolean;
}

export interface LedgerFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  account?: string;
  party?: string;
  created_by?: number;
  fromDate?: string;
  toDate?: string;
  has_evidence?: boolean; // Filter entries with or without evidence
  project_id?: number;
}

// New API response interfaces
export interface LedgerEntriesResponse {
  status: string;
  data: LedgerEntry[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface OverallBalance {
  balance: number;
}

export interface AccountBalanceResponse {
  account: string;
  balance: number;
}

export interface CashFlowSummary {
  start_date: string;
  end_date: string;
  total_inflow: number;
  total_outflow: number;
  net_cash_flow: number;
  opening_balance: number;
  closing_balance: number;
  by_account: Record<string, {
    total_debits: number;
    total_credits: number;
    net_amount: number;
  }>;
}

export interface CreateTransactionRequest {
  entries: CreateLedgerEntry[];
}

export interface ReversalRequest {
  reason: string;
}

export interface RecalculateBalanceResponse {
  status: string;
  data: null;
  message: string;
}

export interface OnePayFeeReportIssue {
  code: string;
  message: string;
  row?: number;
  reference?: string;
}

export interface OnePayFeeReportSummary {
  merchant_id: string;
  merchant_name: string;
  period_label: string;
  period_from: string;
  period_to: string;
  transaction_count: number;
  fee_per_transaction: number;
  total_fee: number;
  detail_total_amount: number;
  app_recorded_fee_total: number;
  import_reference: string;
}

export interface OnePayFeeImportResponse {
  summary: OnePayFeeReportSummary;
  transaction_id: number;
  transaction_code: string;
  ledger_entry_ids: number[];
  created_at: string;
}

// Asset types for evidence management - matching API spec
export interface Asset {
  id: number;
  filename: string;
  original_filename?: string; // Original filename before processing
  content_type?: string; // MIME type
  file_size?: number; // File size in bytes
  upload_type: 'document' | 'ledger_evidence' | 'bulk_transfer_result' | 'general';
  uploaded_by: number;
  created_at: string;
}

export interface CreateAssetData {
  file: File;
  upload_type: 'document' | 'ledger_evidence' | 'bulk_transfer_result' | 'general';
}

export interface AssetFilters {
  limit?: number;
  offset?: number;
  upload_type?: string;
  uploaded_by?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface AssetsResponse {
  status: string;
  message: string;
  data: Asset[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface AssetResponse {
  status: string;
  message: string;
  data: Asset;
}

// Ledger Summary types - Updated to match new API
export interface OwnerContribution {
  owner: string;
  total_contribution: number;
  percentage: number;
}

export interface LedgerSummary {
  period: {
    from: string;
    to: string;
  };
  totals: {
    opening_balance: number;
    closing_balance: number;
    net_cashflow: number;
  };
  by_account: Record<string, {
    debit: number;
    credit: number;
    net_amount: number;
  }>;
  by_owners?: OwnerContribution[];
}

export interface LedgerSummaryResponse {
  status: string;
  data: LedgerSummary;
  message: string;
}

// Account Metadata types
export interface AccountMetadata {
  value: string;
  label: string;
  category: 'asset' | 'liability' | 'equity' | 'revenue' | 'expense';
  normal_side: 'debit' | 'credit';
}

export interface AccountMetadataResponse {
  status: string;
  data: AccountMetadata[];
  message: string;
}
