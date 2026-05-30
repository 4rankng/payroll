import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  FinancialDashboard,
  LedgerEntry,
  CreateLedgerEntry,
  AccountBalance,
  Receivables,
  Payables,
  CashFlowAnalysis,
  FinancialReport,
  GenerateReportRequest,
  LedgerFilters,
} from '@/types/api/financial.types';

class FinancialService {
  /**
   * Get financial dashboard overview
   */
  async getDashboard(params?: {
    project_id?: number;
    fromDate?: string;
    toDate?: string;
  }): Promise<FinancialDashboard> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<FinancialDashboard>(
      `${API_ENDPOINTS.financial.dashboard}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get ledger entries
   */
  async getLedgerEntries(filters?: LedgerFilters) {
    const queryString = filters ? buildQueryString(filters as Record<string, unknown>) : '';
    const response = await apiClient.get<LedgerEntry[]>(
      `${API_ENDPOINTS.ledger.entries}${queryString}`
    );
    return response;
  }

  /**
   * Get single ledger entry
   */
  async getLedgerEntry(id: number): Promise<LedgerEntry> {
    const response = await apiClient.get<LedgerEntry>(
      API_ENDPOINTS.ledger.entryById(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new ledger entry
   */
  async createLedgerEntry(data: CreateLedgerEntry): Promise<LedgerEntry> {
    const response = await apiClient.post<LedgerEntry>(
      API_ENDPOINTS.ledger.entries,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Update ledger entry
   */
  async updateLedgerEntry(id: number, data: Partial<CreateLedgerEntry>): Promise<LedgerEntry> {
    const response = await apiClient.put<LedgerEntry>(
      API_ENDPOINTS.ledger.entryById(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete ledger entry
   */
  async deleteLedgerEntry(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.ledger.entryById(id));
  }

  /**
   * Get account balance
   */
  async getAccountBalance(
    account: string,
    params?: { project_id?: number; date?: string }
  ): Promise<AccountBalance> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<AccountBalance>(
      `${API_ENDPOINTS.ledger.accountBalance(account)}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get project financial summary
   */
  async getProjectFinancialSummary(projectId: number) {
    const response = await apiClient.get(
      API_ENDPOINTS.financial.projectSummary(projectId)
    );
    return response.data;
  }

  /**
   * Get receivables report
   */
  async getReceivables(params?: {
    project_id?: number;
    overdue_only?: boolean;
    days_overdue?: number;
  }): Promise<Receivables> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<Receivables>(
      `${API_ENDPOINTS.financial.receivables}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get payables report
   */
  async getPayables(params?: {
    project_id?: number;
    overdue_only?: boolean;
  }): Promise<Payables> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<Payables>(
      `${API_ENDPOINTS.financial.payables}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Generate financial report
   */
  async generateReport(
    reportType: 'profit_loss' | 'balance_sheet' | 'cash_flow' | 'project_summary',
    data: GenerateReportRequest,
    format: 'excel' | 'pdf' | 'csv' = 'excel'
  ): Promise<FinancialReport> {
    const queryString = buildQueryString({ report_type: reportType, format });
    const response = await apiClient.post<FinancialReport>(
      `${API_ENDPOINTS.financial.reports}${queryString}`,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get report status
   */
  async getReportStatus(reportId: string): Promise<FinancialReport> {
    const response = await apiClient.get<FinancialReport>(
      API_ENDPOINTS.financial.reportById(reportId)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Download financial report
   */
  async downloadReport(reportId: string): Promise<void> {
    const report = await this.getReportStatus(reportId);
    if (report.downloadUrl && report.filename) {
      await apiClient.download(report.downloadUrl, report.filename);
    } else {
      throw new Error('Report not ready for download');
    }
  }

  /**
   * Get cash flow analysis
   */
  async getCashFlowAnalysis(params?: {
    project_id?: number;
    months_ahead?: number;
  }): Promise<CashFlowAnalysis> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<CashFlowAnalysis>(
      `${API_ENDPOINTS.financial.cashFlow}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Validate ledger entry
   */
  validateLedgerEntry(entry: CreateLedgerEntry): { isValid: boolean; errors: string[] } {
    const errors: string[] = [];

    // Either debit or credit must be greater than zero, but not both
    if (entry.debit === 0 && entry.credit === 0) {
      errors.push('Either debit or credit must be greater than zero');
    }
    if (entry.debit > 0 && entry.credit > 0) {
      errors.push('Cannot have both debit and credit in the same entry');
    }

    // Validate account type
    const validAccounts = ['capital', 'revenue', 'expense', 'salary', 'partner_payment'];
    if (!validAccounts.includes(entry.account)) {
      errors.push('Invalid account type');
    }

    // Validate date format
    const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
    if (!dateRegex.test(entry.date)) {
      errors.push('Date must be in YYYY-MM-DD format');
    }

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
   * Calculate profit margin
   */
  calculateProfitMargin(revenue: number, expenses: number): number {
    if (revenue === 0) return 0;
    return ((revenue - expenses) / revenue) * 100;
  }
}

export const financialService = new FinancialService();
