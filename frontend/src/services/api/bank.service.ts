import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Bank,
  BankFilters,
  BankSearchParams,
  BanksResponse,
  BankSearchResponse,
  CreateBankRequest,
  UpdateBankRequest
} from '@/types/api/bank.types';

class BankService {
  /**
   * Search banks with autocomplete
   */
  async searchBanks(params: BankSearchParams): Promise<BanksResponse> {
    // Set default pagination values - initial load of 20 records
    const searchParams = {
      ...params,
      page: params.page || 1,
      pageSize: params.pageSize || 20,
    };

    const queryString = buildQueryString(searchParams);
    const response = await apiClient.get<Bank[]>(
      `${API_ENDPOINTS.banks.base}${queryString}`
    );

    // Extract data from the nested response structure
    const banks = response.data || [];
    const pagination = response.pagination || {
      page: searchParams.page,
      pageSize: searchParams.pageSize,
      totalPages: 1,
      totalRecords: banks.length
    };

    return {
      data: banks,
      pagination: pagination
    };
  }

  /**
   * Get paginated list of banks
   */
  async getBanks(filters?: BankFilters): Promise<BanksResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<Bank[]>(
      `${API_ENDPOINTS.banks.base}${queryString}`
    );

    return {
      data: response.data || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: 100,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Get single bank by ID
   */
  async getBankById(id: number): Promise<Bank> {
    const response = await apiClient.get<Bank>(
      API_ENDPOINTS.banks.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new bank (ADMIN only)
   */
  async createBank(data: CreateBankRequest): Promise<Bank> {
    const response = await apiClient.post<Bank>(
      API_ENDPOINTS.banks.base,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Update bank (ADMIN only)
   */
  async updateBank(id: number, data: UpdateBankRequest): Promise<Bank> {
    const response = await apiClient.put<Bank>(
      API_ENDPOINTS.banks.byId(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete bank (ADMIN only)
   */
  async deleteBank(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.banks.byId(id));
  }
}

export const bankService = new BankService();
