import { apiClient, buildQueryString, ApiResponse, createIdempotencyKey } from './client';
import type { LedgerEntry, Asset } from '@/types/api/financial.types';
import { API_ENDPOINTS } from '@/config/api.config';
import { authManager } from '@/lib/auth';

// Dynamic types - validated against backend metadata
export type TransactionType = string;
export type TransactionStatus = string;
export type SettlementMethod = 'cash' | 'payable' | 'receivable';

// Metadata types
export interface TransactionMetadataItem {
  type: string;
  label: string;
}

export interface TransactionMetadata {
  transaction_types: TransactionMetadataItem[];
  statuses: TransactionMetadataItem[];
}

export interface Transaction {
  id: number;
  description: string;
  transaction_code: string;
  transaction_type: TransactionType;
  amount: number;
  settled_amount?: number;
  remaining_amount?: number;
  pending_amount?: number;
  party: string;
  // For capital transactions, the contributing user's ID
  user_id?: number;
  status: TransactionStatus;
  settlement_method: SettlementMethod;
  settled_at?: string;
  settled_by?: number;
  reference_txn_id?: number;
  reversed_transaction_id?: number;
  url?: string;
  asset_id?: number;
  asset?: Asset | null;
  settlements?: Settlement[];
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface CreateTransactionRequest {
  description: string;
  transaction_type: TransactionType;
  amount: number;
  party: string;
  // Optional: for capital contribution, the contributing user's ID
  user_id?: number;
  status: TransactionStatus;
  url?: string;
  asset_id?: number;
}

export interface Settlement {
  id: number;
  transaction_id: number;
  amount: number;
  settlement_date: string;
  proof_url?: string;
  proof_asset_id?: number;
  proof_asset?: Asset | null;
  payment_method?: string;
  notes?: string;
  created_by: number;
  created_at: string;
}

export interface SettleTransactionRequest {
  amount: number;
  settlement_date: string;
  proof_url?: string;
  proof_asset_id?: number;
  payment_method?: string;
  notes?: string;
}

export interface TransactionListResponse {
  data: Transaction[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface TransactionWithLedgerResponse {
  transaction: Transaction;
  ledger_entries: LedgerEntry[];
}

export interface SettleTransactionResponse {
  transaction: Transaction;
  settlement: Settlement;
  ledger_entries: LedgerEntry[];
}

export interface UpdateTransactionEvidenceRequest {
  url?: string;
  asset_id?: number;
}

export type UpdateTransactionEvidenceResponse = ApiResponse<Transaction>;

export interface TransactionFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: string;
  transaction_type?: TransactionType;
  status?: TransactionStatus; // settled or pending
  party?: string;
  search?: string;
  fromDate?: string; // YYYY-MM-DD format
  toDate?: string; // YYYY-MM-DD format
  created_by?: number;
}

const TRANSACTION_STATUS_LABEL_OVERRIDES: Record<string, string> = {
  settled: 'Đã TT',
  pending: 'Chờ TT',
  partially_settled: 'TT thiếu',
};

export interface SendPayrollReportEmailRequest {
  reportAtDate: string;
  recipients: string[];
  cc?: string[];
  bcc?: string[];
}

class TransactionService {
  /**
   * Create a new transaction
   */
  async createTransaction(data: CreateTransactionRequest): Promise<TransactionWithLedgerResponse> {
    const response = await apiClient.post<{ data: TransactionWithLedgerResponse }>(
      API_ENDPOINTS.transactions.base,
      data
    );

    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Get transaction by ID
   */
  async getTransaction(id: number): Promise<Transaction> {
    const response = await apiClient.get<Transaction>(
      API_ENDPOINTS.transactions.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * List transactions with filters
   */
  async listTransactions(filters?: TransactionFilters): Promise<TransactionListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<Transaction[]>(
      `${API_ENDPOINTS.transactions.base}${queryString}`
    );

    if (!response.pagination) {
      throw new Error('API response missing expected data');
    }
    return {
      data: response.data || [],
      pagination: response.pagination,
    };
  }

  /**
   * Get pending transactions
   */
  async getPendingTransactions(filters?: TransactionFilters): Promise<TransactionListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<{
      status: string;
      data: Transaction[];
      message: string;
      pagination: {
        page: number;
        pageSize: number;
        totalPages: number;
        totalRecords: number;
      };
    }>(
      `${API_ENDPOINTS.transactions.pending}${queryString}`
    );

    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return {
      data: response.data.data,
      pagination: response.data.pagination
    };
  }

  /**
   * Settle a pending transaction
   */
  async settleTransaction(id: number, data: SettleTransactionRequest): Promise<SettleTransactionResponse> {
    const response = await apiClient.post<{ data: SettleTransactionResponse }>(
      API_ENDPOINTS.transactions.settle(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Export transactions via API
   */
  async exportTransactions(fromDate?: string, toDate?: string, transactionType?: TransactionType): Promise<void> {
    // Build query parameters
    const params = new URLSearchParams();
    if (fromDate) params.append('fromDate', fromDate);
    if (toDate) params.append('toDate', toDate);
    if (transactionType) params.append('transaction_type', transactionType);

    const queryString = params.toString() ? `?${params.toString()}` : '';

    // Use apiClient.download which handles blob responses correctly
    await apiClient.download(
      `${API_ENDPOINTS.transactions.export}${queryString}`,
      `lich_su_giao_dich_${Date.now()}.xlsx`
    );
  }

  /**
   * Send payroll report email via timesheet endpoint
   */
  async sendPayrollReportEmail(payload: SendPayrollReportEmailRequest): Promise<void> {
    await apiClient.post(API_ENDPOINTS.timesheets.emailPayrollReport, payload, {
      headers: {
        'Idempotency-Key': createIdempotencyKey('payroll-report-email'),
      },
    });
  }

  /**
   * Format currency for display
   */
  formatCurrency(amount: number): string {
    return new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency: 'VND',
    }).format(amount);
  }

  /**
   * Get transaction metadata
   */
  async getTransactionMetadata(): Promise<TransactionMetadata> {
    // Prevent unauthorized roles from calling the endpoint
    try {
      const role = authManager.getUserRole();
      if (role !== 'admin') {
        return { transaction_types: [], statuses: [] };
      }
    } catch {
      return { transaction_types: [], statuses: [] };
    }

    const response = await apiClient.get<{ status: string; data: TransactionMetadata; message: string }>(
      API_ENDPOINTS.transactions.metadata
    );
    // API returns { status: "success", data: {...}, message: "..." }
    // apiClient.get returns axios response, so we need response.data.data
    return response.data as unknown as TransactionMetadata || { transaction_types: [], statuses: [] };
  }

  /**
   * Get transaction type display name
   * Uses metadata for display, falls back to type value if not found
   */
  getTransactionTypeDisplay(type: TransactionType, metadata?: TransactionMetadata): string {
    if (metadata) {
      const found = metadata.transaction_types.find(t => t.type === type);
      if (found) return found.label;
    }
    return type;
  }

  /**
   * Get status display name
   * Uses metadata for display, falls back to status value if not found
   */
  getStatusDisplay(status: TransactionStatus, metadata?: TransactionMetadata): string {
    const override = TRANSACTION_STATUS_LABEL_OVERRIDES[status];
    if (override) {
      return override;
    }

    if (metadata) {
      const found = metadata.statuses.find(s => s.type === status);
      if (found) return found.label;
    }
    return status;
  }

  /**
   * Validate transaction type against metadata
   */
  isValidTransactionType(type: string, metadata: TransactionMetadata): boolean {
    return metadata.transaction_types.some(t => t.type === type);
  }

  /**
   * Validate status against metadata
   */
  isValidStatus(status: string, metadata: TransactionMetadata): boolean {
    return metadata.statuses.some(s => s.type === status);
  }

  /**
   * Update transaction evidence (url/asset only)
   */
  async updateTransactionEvidence(
    id: number,
    payload: UpdateTransactionEvidenceRequest
  ): Promise<UpdateTransactionEvidenceResponse> {
    const hasUrl = typeof payload.url === 'string' && payload.url.trim().length > 0;
    const hasAsset = typeof payload.asset_id === 'number' && Number.isFinite(payload.asset_id);

    if (!hasUrl && !hasAsset) {
      throw new Error('Cần cung cấp ít nhất URL hoặc tệp chứng từ');
    }

    const requestBody: UpdateTransactionEvidenceRequest = {};

    if (hasUrl) {
      requestBody.url = payload.url!.trim();
    }

    if (hasAsset) {
      requestBody.asset_id = payload.asset_id;
    }

    const response = await apiClient.put<UpdateTransactionEvidenceResponse>(
      API_ENDPOINTS.transactions.byId(id),
      requestBody
    );

    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Reverse a transaction
   */
  async reverseTransaction(id: number, reason?: string): Promise<unknown> {
    const response = await apiClient.post<{ data: unknown }>(
      API_ENDPOINTS.transactions.reverse(id),
      { reason }
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }
}

export const transactionService = new TransactionService();

export const getTransactionRemainingAmount = (transaction: Transaction): number => {
  return transaction.pending_amount ?? transaction.remaining_amount ?? transaction.amount;
};
