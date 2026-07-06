import { apiClient } from './client';
import type {
  WalletBalance,
  WalletTopup,
  CreateWalletTopupRequest,
  WalletTopupListResponse,
  WalletPayment,
  WalletPaymentListResponse,
  WalletPaymentFilter,
  UnifiedTransaction,
  UnifiedTransactionListResponse,
  UnifiedTransactionFilter,
  ReconciliationJob,
  SyncBalanceResponse,
  AdjustBalanceRequest,
  WalletDemandForecastResponse,
} from '../../types/api/wallet.types';

const WALLET_BASE_PATH = '/wallet';

export const walletService = {
  // ── Balance ──────────────────────────────────────────────────────────────
  async getBalance(): Promise<WalletBalance> {
    const r = await apiClient.get<WalletBalance>(`${WALLET_BASE_PATH}/balance`);
    return r.data!;
  },

  async getDemandForecast(): Promise<WalletDemandForecastResponse> {
    const r = await apiClient.get<WalletDemandForecastResponse>(
      `${WALLET_BASE_PATH}/demand-forecast`,
    );
    return r.data!;
  },

  // ── Topups ───────────────────────────────────────────────────────────────
  async createTopup(data: CreateWalletTopupRequest): Promise<WalletTopup> {
    const r = await apiClient.post<WalletTopup>(`${WALLET_BASE_PATH}/topups`, data);
    return r.data!;
  },

  async getTopupById(id: number): Promise<WalletTopup> {
    const r = await apiClient.get<WalletTopup>(`${WALLET_BASE_PATH}/topups/${id}`);
    return r.data!;
  },

  async listTopups(params?: {
    bank_ref?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    page_size?: number;
  }): Promise<WalletTopupListResponse> {
    const r = await apiClient.get<WalletTopup[]>(`${WALLET_BASE_PATH}/topups`, { params });
    return {
      data: r.data ?? [],
      total: r.pagination?.totalRecords ?? 0,
      page: r.pagination?.page ?? 1,
      page_size: r.pagination?.pageSize ?? 20,
    };
  },

  // ── Payments ─────────────────────────────────────────────────────────────
  async getPaymentById(id: number): Promise<WalletPayment> {
    const r = await apiClient.get<WalletPayment>(`${WALLET_BASE_PATH}/payments/${id}`);
    return r.data!;
  },

  async listPayments(filter?: WalletPaymentFilter): Promise<WalletPaymentListResponse> {
    const r = await apiClient.get<WalletPayment[]>(`${WALLET_BASE_PATH}/payments`, { params: filter });
    return {
      data: r.data ?? [],
      total: r.pagination?.totalRecords ?? 0,
      page: r.pagination?.page ?? 1,
      page_size: r.pagination?.pageSize ?? 20,
    };
  },

  // ── Unified transactions ────────────────────────────────────────────────
  async getTransactions(
    params?: UnifiedTransactionFilter,
  ): Promise<UnifiedTransactionListResponse> {
    const r = await apiClient.get<UnifiedTransaction[]>(`${WALLET_BASE_PATH}/transactions`, {
      params,
    });
    return {
      data: r.data ?? [],
      total: r.pagination?.totalRecords ?? 0,
      page: r.pagination?.page ?? params?.page ?? 1,
      page_size: r.pagination?.pageSize ?? params?.page_size ?? 20,
    };
  },

  // ── Reconciliation ──────────────────────────────────────────────────────
  async uploadReconciliation(file: File): Promise<{ job_id: string }> {
    const formData = new FormData();
    formData.append('file', file);
    const r = await apiClient.post<{ job_id: string }>(
      `${WALLET_BASE_PATH}/reconcile/upload`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return r.data!;
  },

  async getReconciliationJobStatus(jobId: string): Promise<ReconciliationJob> {
    const r = await apiClient.get<ReconciliationJob>(
      `${WALLET_BASE_PATH}/reconcile/jobs/${jobId}`,
    );
    return r.data ?? { id: jobId, status: 'completed', total_rows: 0, matched: 0, unmatched: 0 };
  },

  async exportReconciliationReport(month: string): Promise<Blob> {
    const response = await apiClient.get<Blob>(
      `${WALLET_BASE_PATH}/reconcile/export`,
      { params: { month }, responseType: 'blob' },
    );
    return response.data!;
  },

  async autoReconcile(startDate: string, endDate: string): Promise<{ job_id: string }> {
    const r = await apiClient.post<{ job_id: string }>(
      `${WALLET_BASE_PATH}/reconcile/auto`,
      null,
      { params: { start_date: startDate, end_date: endDate } },
    );
    return r.data!;
  },

  async resolvePayment(id: number, action: 'complete' | 'fail' | 'reverse', reason?: string): Promise<WalletPayment> {
    const r = await apiClient.post<WalletPayment>(
      `${WALLET_BASE_PATH}/payments/${id}/resolve`,
      { action, reason },
    );
    return r.data!;
  },

  // ── Break transaction CSV download ──────────────────────────────────
  downloadBreakTransactionsCsv(job: ReconciliationJob): void {
    const BOM = '﻿';
    const csvEscape = (v: string) => `"${v.replace(/"/g, '""')}"`;
    const header = (job.headers ?? []).map(csvEscape).join(',');
    const rows = (job.raw_rows ?? []).map((r) => r.map(csvEscape).join(','));
    const csv = BOM + header + '\n' + rows.join('\n');

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `break-transactions-${job.id}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  },

  // ── Balance sync ────────────────────────────────────────────────────────
  async syncBalance(): Promise<SyncBalanceResponse> {
    const r = await apiClient.post<SyncBalanceResponse>(`${WALLET_BASE_PATH}/balance/sync`);
    return r.data!;
  },

  async adjustBalance(data: AdjustBalanceRequest): Promise<WalletTopup> {
    const r = await apiClient.post<WalletTopup>(`${WALLET_BASE_PATH}/balance/adjust`, data);
    return r.data!;
  },
};
