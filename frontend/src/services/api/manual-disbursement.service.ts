// Admin "Chuyển tiền" (Manual Disbursement) API client.
// All endpoints require admin role; the backend Casbin rule
// `p, admin, /api/*, *, allow` covers them.

import { apiClient, type ApiResponse } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  BankInfoResponse,
  InitiateManualDisbursementRequest,
  EmployeeAccountLookupRequest,
  EmployeeAccountLookupResponse,
  ManualDisbursementResponse,
  VerifyAccountRequest,
  VerifyAccountResponse,
  WalletBalanceInfo,
} from "@/types/api/manual-disbursement.types";

class ManualDisbursementService {
  initiate(
    body: InitiateManualDisbursementRequest,
  ): Promise<ApiResponse<ManualDisbursementResponse>> {
    return apiClient.post<ManualDisbursementResponse>(
      API_ENDPOINTS.manualDisbursement.base,
      body,
    );
  }

  status(txnId: string): Promise<ApiResponse<ManualDisbursementResponse>> {
    return apiClient.get<ManualDisbursementResponse>(
      API_ENDPOINTS.manualDisbursement.byTxnId(txnId),
    );
  }

  list(limit = 20): Promise<ApiResponse<ManualDisbursementResponse[]>> {
    return apiClient.get<ManualDisbursementResponse[]>(
      `${API_ENDPOINTS.manualDisbursement.base}?limit=${limit}`,
    );
  }

  verifyAccount(
    body: VerifyAccountRequest,
  ): Promise<ApiResponse<VerifyAccountResponse>> {
    return apiClient.post<VerifyAccountResponse>(
      API_ENDPOINTS.manualDisbursement.checkAccount,
      body,
    );
  }

  checkEmployeeAccount(
    body: EmployeeAccountLookupRequest,
  ): Promise<ApiResponse<EmployeeAccountLookupResponse>> {
    return apiClient.post<EmployeeAccountLookupResponse>(
      API_ENDPOINTS.manualDisbursement.employeeAccountCheck,
      body,
    );
  }

  banks(): Promise<ApiResponse<BankInfoResponse[]>> {
    return apiClient.get<BankInfoResponse[]>(
      `${API_ENDPOINTS.manualDisbursement.base}/banks`,
    );
  }

  // Synchronously fetches the reconciliation CSV for [dateFrom,dateTo]
  // and triggers a browser download. Both dates are DD/MM/YYYY strings.
  // The backend blocks while it round-trips the provider's two-step export API
  // (typically 1-3 seconds) — the caller should render a spinner.
  balance(): Promise<ApiResponse<WalletBalanceInfo>> {
    return apiClient.get<WalletBalanceInfo>(
      `${API_ENDPOINTS.manualDisbursement.base}/balance`,
    );
  }

  async downloadReconciliation(dateFrom: string, dateTo: string): Promise<void> {
    const params = new URLSearchParams({ date_from: dateFrom, date_to: dateTo });
    const url = `${API_ENDPOINTS.manualDisbursement.reconciliationDownload}?${params.toString()}`;
    await apiClient.download(url);
  }
}

export const manualDisbursementService = new ManualDisbursementService();
