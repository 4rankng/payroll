import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  LedgerEntry,
  CreateLedgerEntry,
  LedgerFilters,
  LedgerEntriesResponse,
  OverallBalance,
  AccountBalanceResponse,
  CashFlowSummary,
  CreateTransactionRequest,
  ReversalRequest,
  RecalculateBalanceResponse,
  LedgerSummary,
  LedgerSummaryResponse,
  AccountMetadata,
  AccountMetadataResponse,
  Asset,
} from '@/types/api/financial.types';
import { authManager } from '@/lib/auth';

class LedgerService {
  /**
   * Get ledger entries with filtering and pagination
   */
  async getEntries(filters?: LedgerFilters): Promise<LedgerEntriesResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<LedgerEntriesResponse>(
      `${API_ENDPOINTS.ledger.entries}${queryString}`
    );

    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get single ledger entry by ID
   */
  async getEntry(id: number): Promise<LedgerEntry> {
    const response = await apiClient.get<LedgerEntry>(
      API_ENDPOINTS.ledger.entryById(id)
    );

    const entry = response.data;

    if (!entry) {
      throw new Error('Ledger entry not found');
    }

    return entry;
  }

  /**
   * Create new ledger entry
   */
  async createEntry(data: CreateLedgerEntry): Promise<LedgerEntry> {
    // Create the ledger entry - backend expects array format even for single entries
    const response = await apiClient.post<{
      entries: LedgerEntry[];
      total_entries: number;
      total_debits: number;
      total_credits: number;
    }>(
      API_ENDPOINTS.ledger.entries,
      [data] // Send as array
    );

    if (!response.data || !response.data.entries || response.data.entries.length === 0) {
      throw new Error('Invalid response from server: missing entries data');
    }

    const createdEntry = response.data.entries[0];

    return createdEntry; // Return first entry from array
  }


  /**
   * Reverse ledger entry (creates offsetting entry)
   */
  async reverseEntry(id: number, reason: string): Promise<LedgerEntry> {
    const response = await apiClient.post<{ status: string; data: LedgerEntry; message: string }>(
      `${API_ENDPOINTS.ledger.entryById(id)}/reverse`,
      { reason }
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }


  /**
   * Create balanced double-entry transaction
   */
  async createTransaction(data: CreateTransactionRequest): Promise<{
    entries: LedgerEntry[];
    total_entries: number;
    total_debits: number;
    total_credits: number;
  }> {
    const response = await apiClient.post<{
      status: string;
      data: {
        entries: LedgerEntry[];
        total_entries: number;
        total_debits: number;
        total_credits: number;
      };
      message: string;
    }>(API_ENDPOINTS.ledger.transactions, data);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Get overall balance
   */
  async getOverallBalance(): Promise<number> {
    const response = await apiClient.get<{ balance: number }>(
      API_ENDPOINTS.ledger.balance
    );

    return response.data?.balance || 0;
  }

  /**
   * Get balance by account type
   */
  async getAccountBalance(account: string): Promise<AccountBalanceResponse> {
    const response = await apiClient.get<{ status: string; data: AccountBalanceResponse; message: string }>(
      API_ENDPOINTS.ledger.accountBalance(account)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Get balance by project
   */
  async getProjectBalance(projectId: number): Promise<number> {
    const response = await apiClient.get<{ status: string; data: OverallBalance; message: string }>(
      API_ENDPOINTS.ledger.projectBalance(projectId)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data.balance;
  }

  /**
   * Get cash flow summary for a date range
   */
  async getCashFlowSummary(fromDate: string, toDate: string): Promise<CashFlowSummary> {
    const queryString = buildQueryString({ fromDate, toDate });
    const response = await apiClient.get<{ status: string; data: CashFlowSummary; message: string }>(
      `${API_ENDPOINTS.ledger.cashFlow}${queryString}`
    );

    return response.data || {
      start_date: fromDate,
      end_date: toDate,
      total_inflow: 0,
      total_outflow: 0,
      net_cash_flow: 0,
      opening_balance: 0,
      closing_balance: 0,
      by_account: {},
      by_project: {}
    };
  }

  /**
   * Get ledger summary for a date range with optional project filter
   * Updated to match new API: GET /api/v1/ledger/summary?from=YYYY-MM-DD&to=YYYY-MM-DD&project_id=123
   * When both from and to are empty, returns all-time summary
   */
  async getLedgerSummary(from: string, to: string, projectId?: number): Promise<LedgerSummary> {
    const params: Record<string, string | number> = {};

    if (from) {
      params.from = from;
    }
    if (to) {
      params.to = to;
    }
    if (projectId) {
      params.project_id = projectId;
    }

    const queryString = buildQueryString(params);
    const fullUrl = `${API_ENDPOINTS.ledger.summary}${queryString}`;

    const response = await apiClient.get<LedgerSummaryResponse>(fullUrl);

    // The API returns { status: "success", data: {...}, message: "..." }
    const apiData = response.data;

    if (!apiData) {
      console.warn('No API response data received');
      return {
        period: { from, to },
        totals: {
          opening_balance: 0,
          closing_balance: 0,
          net_cashflow: 0,
        },
        by_account: {},
      };
    }

    // Check if response is wrapped in API format
    if (apiData.status && apiData.data) {
      // Wrapped format: { status: "success", data: {...}, message: "..." }
      if (apiData.status !== 'success') {
        console.warn('API returned non-success status:', apiData.status);
        return {
          period: { from, to },
          totals: {
            opening_balance: 0,
            closing_balance: 0,
            net_cashflow: 0,
          },
          by_account: {},
        };
      }
      const result = apiData.data;

      return result;
    } else {
      // Direct format: { period: {...}, totals: {...}, by_account: {...} }
      const result = apiData as unknown as LedgerSummary;

      return result;
    }
  }

  /**
   * Recalculate all running balances
   */
  async recalculateBalances(): Promise<void> {
    await apiClient.post<RecalculateBalanceResponse>(
      `${API_ENDPOINTS.ledger.balance}/recalculate`
    );
  }

  /**
   * Get account metadata from API
   */
  async getAccountMetadata(): Promise<AccountMetadata[]> {
    // Prevent unauthorized roles from calling the endpoint
    try {
      const role = authManager.getUserRole();
      if (role !== 'admin') {
        return [];
      }
    } catch {
      // If auth cannot be resolved, be safe and do not call the API
      return [];
    }

    const response = await apiClient.get<{ status: string; data: AccountMetadata[]; message: string }>(
      API_ENDPOINTS.ledger.accountsMetadata
    );
    // API returns { status: "success", data: [...], message: "..." }
    // apiClient.get returns axios response, so we need response.data.data
    return response.data || [];
  }

  /**
   * Validate ledger entry
   */
  validateEntry(entry: CreateLedgerEntry): { isValid: boolean; errors: string[] } {
    const errors: string[] = [];

    // Either debit or credit must be greater than zero, but not both
    if (entry.debit === 0 && entry.credit === 0) {
      errors.push('Số tiền nợ hoặc có phải lớn hơn 0');
    }
    if (entry.debit > 0 && entry.credit > 0) {
      errors.push('Không thể có cả nợ và có trong cùng một bút toán');
    }

    // Validate account type - now more flexible
    if (!entry.account?.trim()) {
      errors.push('Loại tài khoản là bắt buộc');
    }

    // Validate required fields
    if (!entry.party?.trim()) {
      errors.push('Đối tượng là bắt buộc');
    }
    if (!entry.description?.trim()) {
      errors.push('Diễn giải là bắt buộc');
    }

    // Validate date format and not in future
    const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
    if (!dateRegex.test(entry.date)) {
      errors.push('Ngày phải có định dạng YYYY-MM-DD');
    } else {
      const entryDate = new Date(entry.date);
      const today = new Date();
      today.setHours(23, 59, 59, 999); // End of today

      if (entryDate > today) {
        errors.push('Ngày không thể trong tương lai');
      }
    }

    // Evidence is optional - no validation needed for URL or asset_id

    return {
      isValid: errors.length === 0,
      errors,
    };
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
   * Format currency for display in short form (for charts)
   */
  formatCurrencyShort(amount: number): string {
    return new Intl.NumberFormat('vi-VN').format(amount) + ' đ';
  }

  /**
   * Get account type display name (fallback for static data)
   */
  getAccountDisplayName(account: string): string {
    const accountNames = {
      cash: 'Tiền mặt',
      receivable: 'Phải thu',
      payable: 'Phải trả',
      revenue: 'Doanh thu',
      expense: 'Chi phí',
    };
    return accountNames[account as keyof typeof accountNames] || account;
  }

  /**
   * Get account display name from metadata array
   */
  getAccountDisplayNameFromMetadata(account: string, metadata: AccountMetadata[]): string {
    const accountMeta = metadata.find(meta => meta.value === account);
    return accountMeta?.label || this.getAccountDisplayName(account);
  }

  /**
   * Get common account type options for forms (fallback for static data)
   */
  getAccountOptions() {
    return [
      { value: 'cash', label: 'Tiền mặt' },
      { value: 'receivable', label: 'Phải thu' },
      { value: 'payable', label: 'Phải trả' },
    ];
  }

  /**
   * Convert metadata to options format for form components
   */
  getAccountOptionsFromMetadata(metadata: AccountMetadata[]) {
    return metadata.map(meta => ({
      value: meta.value,
      label: meta.label,
      category: meta.category,
      normal_side: meta.normal_side,
    }));
  }


  /**
   * Check if ledger entry has evidence (URL or asset)
   */
  hasEvidence(entry: LedgerEntry): boolean {
    return !!(entry.url?.trim() || entry.asset_id);
  }

  /**
   * Get evidence count for display
   */
  getEvidenceCount(entry: LedgerEntry): number {
    let count = 0;
    if (entry.url?.trim()) count++;
    if (entry.asset_id) count++;
    return count;
  }

  /**
   * Get evidence display information for UI
   */
  getEvidenceDisplay(entry: LedgerEntry): {
    hasEvidence: boolean;
    type: 'url' | 'asset' | 'both' | 'none';
    url?: string;
    asset?: Asset;
    asset_id?: number;
  } {
    const hasUrl = !!(entry.url?.trim());
    const hasAsset = !!(entry.asset_id); // Check only asset_id, not nested asset

    if (hasUrl && hasAsset) {
      return { hasEvidence: true, type: 'both', url: entry.url, asset: entry.asset, asset_id: entry.asset_id };
    } else if (hasUrl) {
      return { hasEvidence: true, type: 'url', url: entry.url };
    } else if (hasAsset) {
      return { hasEvidence: true, type: 'asset', asset: entry.asset, asset_id: entry.asset_id };
    } else {
      return { hasEvidence: false, type: 'none' };
    }
  }

  /**
   * Validate evidence URL
   */
  validateEvidenceUrl(url: string): { isValid: boolean; error?: string } {
    if (!url.trim()) {
      return { isValid: true }; // URL is optional
    }

    try {
      new URL(url);
      return { isValid: true };
    } catch {
      return {
        isValid: false,
        error: 'URL không hợp lệ. Vui lòng nhập URL đầy đủ (bao gồm http:// hoặc https://)',
      };
    }
  }
}

export const ledgerService = new LedgerService();
