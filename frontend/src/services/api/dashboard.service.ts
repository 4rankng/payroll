import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  DashboardSummary,
  DashboardSummaryParams,
  FinancialOverviewResponse,
  FinancialOverviewParams,
  FinancialChartResponse,
  FinancialChartParams,
  FinancialChartDataPoint,
  RecentActivitiesResponse,
  RecentActivitiesParams,
  NewEmployeesResponse,
  NewEmployeesParams,
  SalaryDistributionResponse,
  SalaryDistributionParams,
  HistoricalResponse,
  EmployeeActivityStats,
  ActiveEmployeeUser,
  ProjectProfitabilityResponse,
  ProjectWeeklyProfitResponse,
  MonthlyFinancialsPeriod,
  MonthlyFinancialsResponse,
  TopPaidEmployeesParams,
  TopPaidEmployeesResponse,
  PartnerDashboardParams,
  PartnerDashboardData,
  BankUsageParams,
  BankUsageResponse,
  BankUsageAllProjectsResponse,
  PartnerEmployeeListParams,
  PartnerEmployeeListResponse,
} from '@/types/api/dashboard.types';

class DashboardService {

  /**
   * Get Dashboard Summary
   * Returns summary statistics including employee counts, salary information, and profit metrics
   */
  async getDashboardSummary(params?: DashboardSummaryParams): Promise<DashboardSummary> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<DashboardSummary>(
      `${API_ENDPOINTS.dashboard.summary}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get Financial Overview
   * Returns revenue and expense data for financial charts
   */
  async getFinancialOverview(params?: FinancialOverviewParams): Promise<FinancialOverviewResponse> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<FinancialOverviewResponse['data']>(`${API_ENDPOINTS.dashboard.financialOverview}${queryString}`);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return {
      status: response.status || 'success',
      data: response.data,
      message: response.message || 'Financial overview retrieved successfully'
    };
  }

  /**
   * Get Financial Chart Data
   * Returns financial data for charts based on period (day/week/month/quarter/year)
   */
  async getFinancialChart(params: FinancialChartParams): Promise<FinancialChartResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<FinancialChartResponse>(`${API_ENDPOINTS.dashboard.financial}${queryString}`);
    if (!response.data) {
      return { status: 'success' as const, data: [], message: 'No data' };
    }
    const data = response.data;
    return {
      status: (data.status as 'success' | 'error') || 'success',
      data: data.data || [],
      message: data.message || response.message || 'Financial chart data retrieved successfully'
    };
  }

  /**
   * Get Recent Activities
   * Returns feed of recent system activities
   */
  async getRecentActivities(params?: RecentActivitiesParams): Promise<RecentActivitiesResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<RecentActivitiesResponse>(`${API_ENDPOINTS.dashboard.recentActivities}${queryString}`);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    const data = response.data;
    return {
      status: (data.status as 'success' | 'error') || 'success',
      data: data.data,
      pagination: data.pagination,
      message: data.message || response.message || 'Recent activities retrieved successfully'
    };
  }

  /**
   * Get New Employees
   * Returns recently added employees
   */
  async getNewEmployees(params?: NewEmployeesParams): Promise<NewEmployeesResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<NewEmployeesResponse>(`${API_ENDPOINTS.dashboard.newEmployees}${queryString}`);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    const data = response.data;
    return {
      status: (data.status as 'success' | 'error') || 'success',
      data: data.data,
      pagination: data.pagination,
      message: data.message || response.message || 'New employees retrieved successfully'
    };
  }

  /**
   * Get Salary Distribution
   * Returns salary distribution data grouped by cycle with optional date range filtering
   * @param params Optional date range or month filter
   */
  async getSalaryDistribution(params?: SalaryDistributionParams): Promise<SalaryDistributionResponse> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.rawRequest<SalaryDistributionResponse>({
      method: 'GET',
      url: `${API_ENDPOINTS.dashboard.salaryDistribution}${queryString}`,
    });
    return response;
  }

  
  /**
   * Get Historical Data
   * Returns historical data for employees, payroll, capital, revenue, and profit
   */
  async getHistorical(): Promise<HistoricalResponse> {
    const response = await apiClient.get<HistoricalResponse>(API_ENDPOINTS.dashboard.historical);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    const data = response.data;
    return {
      status: (data.status as 'success' | 'error') || 'success',
      data: data.data,
      message: data.message || response.message || 'Lấy dữ liệu lịch sử thành công'
    };
  }

  /**
   * Get Employee Activity Stats
   * Returns employee login activity counts grouped by payment schedule for a given month
   */
  async getEmployeeActivityStats(month?: string): Promise<EmployeeActivityStats> {
    const qs = month ? `?month=${month}` : '';
    const response = await apiClient.get<EmployeeActivityStats>(
      `${API_ENDPOINTS.dashboard.employeeActivity}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get Active Employees By Schedule
   * Returns list of users who logged in within the selected month for a given schedule type
   */
  async getActiveEmployeesBySchedule(schedule: 'weekly' | 'monthly' | 'flexible', month?: string): Promise<ActiveEmployeeUser[]> {
    const params = new URLSearchParams({ schedule });
    if (month) params.set('month', month);
    const response = await apiClient.get<ActiveEmployeeUser[]>(
      `${API_ENDPOINTS.dashboard.employeeActivityUsers}?${params.toString()}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getProjectProfitability(): Promise<ProjectProfitabilityResponse> {
    const response = await apiClient.get<ProjectProfitabilityResponse>(
      API_ENDPOINTS.dashboard.projectProfitability
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getProjectWeeklyProfit(days?: number): Promise<ProjectWeeklyProfitResponse> {
    const qs = days ? `?days=${days}` : '';
    const response = await apiClient.get<ProjectWeeklyProfitResponse>(
      `${API_ENDPOINTS.dashboard.projectWeeklyProfit}${qs}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getMonthlyFinancials(period: MonthlyFinancialsPeriod = '3m'): Promise<MonthlyFinancialsResponse> {
    const response = await apiClient.get<MonthlyFinancialsResponse>(
      `${API_ENDPOINTS.dashboard.monthlyFinancials}?period=${period}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getTopPaidEmployees(params?: TopPaidEmployeesParams): Promise<TopPaidEmployeesResponse> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<TopPaidEmployeesResponse>(
      `${API_ENDPOINTS.dashboard.topPaidEmployees}${queryString}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getPartnerDashboard(params?: PartnerDashboardParams): Promise<PartnerDashboardData> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<PartnerDashboardData>(
      `${API_ENDPOINTS.dashboard.partnerDashboard}${queryString}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getPartnerEmployeeList(params: PartnerEmployeeListParams): Promise<PartnerEmployeeListResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<PartnerEmployeeListResponse>(
      `${API_ENDPOINTS.dashboard.partnerEmployeeList}${queryString}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getBankUsage(params?: BankUsageParams): Promise<BankUsageResponse> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<BankUsageResponse>(
      `${API_ENDPOINTS.dashboard.bankUsage}${queryString}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }

  async getBankUsageAllProjects(): Promise<BankUsageAllProjectsResponse> {
    const response = await apiClient.get<BankUsageAllProjectsResponse>(
      API_ENDPOINTS.dashboard.bankUsageProjects
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }
}

export const dashboardService = new DashboardService();
