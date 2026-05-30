import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Lender,
  LenderWithSummary,
  CreateLenderRequest,
  UpdateLenderRequest,
  LenderFilters,
  Loan,
  CreateLoanRequest,
  DisburseLoanRequest,
  DisburseLoanResponse,
  RepayLoanRequest,
  RepayLoanResponse,
  RepayScheduleRequest,
  RepayScheduleResponse,
  UpdateLoanRequest,
  LoanFilters,
  ScheduleItem,
} from '@/types/api/loan.types';

class LoanService {
  // ============================================================================
  // Lenders API
  // ============================================================================

  /**
   * Get paginated list of lenders
   */
  async getLenders(filters?: LenderFilters): Promise<ApiResponse<Lender[]>> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<Lender[]>(
      `${API_ENDPOINTS.lenders.base}${queryString}`
    );
    return response;
  }

  /**
   * Get single lender by ID with financial summary
   */
  async getLenderById(id: number): Promise<LenderWithSummary> {
    const response = await apiClient.get<LenderWithSummary>(
      API_ENDPOINTS.lenders.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new lender
   */
  async createLender(data: CreateLenderRequest): Promise<ApiResponse<Lender>> {
    const response = await apiClient.post<Lender>(
      API_ENDPOINTS.lenders.base,
      data
    );
    return response;
  }

  /**
   * Update existing lender
   */
  async updateLender(id: number, data: UpdateLenderRequest): Promise<ApiResponse<Lender>> {
    const response = await apiClient.put<Lender>(
      API_ENDPOINTS.lenders.byId(id),
      data
    );
    return response;
  }

  /**
   * Delete lender (soft delete)
   */
  async deleteLender(id: number): Promise<ApiResponse<null>> {
    const response = await apiClient.delete<null>(
      API_ENDPOINTS.lenders.byId(id)
    );
    return response;
  }

  // ============================================================================
  // Loans API
  // ============================================================================

  /**
   * Get paginated list of loans
   */
  async getLoans(filters?: LoanFilters): Promise<ApiResponse<Loan[]>> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<Loan[]>(
      `${API_ENDPOINTS.loans.base}${queryString}`
    );
    // Assume backend adheres to spec: data is Loan[] and pagination is top-level
    return response;
  }

  /**
   * Get single loan by ID
   */
  async getLoanById(id: number): Promise<Loan> {
    const response = await apiClient.get<Loan>(
      API_ENDPOINTS.loans.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new loan (with optional immediate disbursement)
   */
  async createLoan(data: CreateLoanRequest): Promise<ApiResponse<Loan>> {
    const response = await apiClient.post<Loan>(
      API_ENDPOINTS.loans.base,
      data
    );
    return response;
  }

  /**
   * Disburse a loan
   */
  async disburseLoan(id: number, data: DisburseLoanRequest): Promise<ApiResponse<DisburseLoanResponse>> {
    const response = await apiClient.post<DisburseLoanResponse>(
      API_ENDPOINTS.loans.disburse(id),
      data
    );
    return response;
  }

  /**
   * Repay principal (full or partial)
   */
  async repayPrincipal(id: number, data: RepayLoanRequest): Promise<ApiResponse<RepayLoanResponse>> {
    const response = await apiClient.post<RepayLoanResponse>(
      API_ENDPOINTS.loans.repay(id),
      data
    );
    return response;
  }

  /**
   * Process scheduled payment for custom schedule loans
   */
  async repaySchedule(id: number, data: RepayScheduleRequest): Promise<ApiResponse<RepayScheduleResponse>> {
    const response = await apiClient.post<RepayScheduleResponse>(
      API_ENDPOINTS.loans.repaySchedule(id),
      data
    );
    return response;
  }

  /**
   * Update loan metadata (description, payment day)
   */
  async updateLoan(id: number, data: UpdateLoanRequest): Promise<ApiResponse<Loan>> {
    const response = await apiClient.patch<Loan>(
      API_ENDPOINTS.loans.byId(id),
      data
    );
    return response;
  }

  /**
   * Get loan payment schedule
   */
  async getLoanSchedule(id: number): Promise<ScheduleItem[]> {
    const response = await apiClient.get<ScheduleItem[]>(
      API_ENDPOINTS.loans.schedule(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete loan (soft delete, only if not disbursed)
   */
  async deleteLoan(id: number): Promise<ApiResponse<null>> {
    const response = await apiClient.delete<null>(
      API_ENDPOINTS.loans.byId(id)
    );
    return response;
  }
}

export const loanService = new LoanService();
