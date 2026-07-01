// Dashboard API Types based on API Documentation

// Dashboard Summary Types (Endpoint 1)
export interface DashboardSummaryParams {
  month?: string; // Format: YYYY-MM (e.g., "2025-04")
}

export interface DashboardSummary {
  total_employees: number;
  total_working_employees: number;
  employees_hired_this_month: number;
  total_paid_salary: number;
  pending_salary_this_month: number;
  paid_salary_this_month: number;
  total_revenue: number;
  total_revenue_this_month: number;
  total_profit: number;
  total_profit_this_month: number;
  total_weekly_salary_employees: number;
  total_monthly_salary_employees: number;
  avg_weekly_salary: number;
  avg_monthly_salary: number;
}

// Financial Overview Types (Endpoint 2)
export interface FinancialOverviewCurrentMonth {
  month: string;
  total_revenue_vnd: number;
  total_expenses_vnd: number;
  net_profit_vnd: number;
  profit_margin_percentage: number;
}

export interface FinancialOverviewGrowth {
  revenue_change_percentage: number;
  expense_change_percentage: number;
  profit_change_percentage: number;
}

export interface FinancialChartDataPoint {
  month: string;
  revenue_vnd: number;
  expenses_vnd: number;
}

export interface FinancialOverviewData {
  period: string;
  year: number;
  current_month: FinancialOverviewCurrentMonth;
  previous_month: FinancialOverviewCurrentMonth;
  growth: FinancialOverviewGrowth;
  chart_data: FinancialChartDataPoint[];
}

export interface FinancialOverviewParams {
  period?: 'month' | 'quarter' | 'year';
  months?: number; // max 24
  year?: number;
}

// Recent Activities Types (Endpoint 3)
export interface RecentActivity {
  id: number;
  /**
   * Mô tả hoạt động (từ API recent-activities)
   * Ví dụ: "Hai Anh đã tạo dự án mới"
   */
  message: string;
  /**
   * Thời gian tạo hoạt động (ISO string)
   */
  created_at: string;
}

export interface RecentActivitiesParams {
  page?: number;
  pageSize?: number;
  /**
   * Trường sắp xếp (mặc định: created_at)
   */
  sortBy?: string;
  /**
   * Thứ tự sắp xếp (asc | desc, mặc định: desc)
   */
  sortOrder?: 'asc' | 'desc';
}

export interface PaginationData {
  page: number;
  pageSize: number;
  totalRecords: number;
  totalPages: number;
}

// System Notifications Types (Endpoint 4)
export interface SystemNotification {
  id: number;
  title: string;
  message: string;
  type: 'error' | 'warning' | 'info' | 'success' | 'approval_required' | 'reminder';
  is_read: boolean;
  action_url: string | null;
  created_at: string;
  icon: string;
}

export interface SystemNotificationsData {
  notifications: SystemNotification[];
  pagination: PaginationData;
  unread_count: number;
}

export interface SystemNotificationsParams {
  pageSize?: number; // max 20
  include_read?: boolean;
}

// New Employees Types (Endpoint 5)
export interface NewEmployeeProject {
  id: number;
  name: string;
  code: string;
}

export interface NewEmployee {
  id: number;
  fullname: string;
  email: string | null;
  cccd: string;
  date_of_birth: string | null;
  status: 'active' | 'inactive';
  current_project: NewEmployeeProject | null;
  created_at: string;
  created_by_name: string;
  avatar: string | null;
}

export interface NewEmployeesParams {
  page?: number;
  pageSize?: number; // max 20
  days?: number;
}

// API Response wrapper types

export interface FinancialOverviewResponse {
  status: 'success' | 'error';
  data: FinancialOverviewData;
  message: string;
}

export interface RecentActivitiesResponse {
  status: 'success' | 'error';
  data: RecentActivity[];
  pagination: PaginationData;
  message: string;
}

export interface SystemNotificationsResponse {
  status: 'success' | 'error';
  data: SystemNotification[];
  pagination: PaginationData;
  unread_count: number;
  message: string;
}

export interface NewEmployeesResponse {
  status: 'success' | 'error';
  data: NewEmployee[];
  pagination: PaginationData;
  message: string;
}

// Financial Chart Types (New endpoint)
export interface FinancialChartDataPoint {
  date: string; // Format depends on period: YYYY-MM-DD, YYYY-WW, YYYY-MM, YYYY-QN, YYYY
  cash_vnd: number;
  receivable_vnd: number;
  payable_vnd: number;
  revenue_vnd: number;
  expenses_vnd: number;
  profit_vnd: number;
  active_employees: number; // total employees assigned to project during the period
}

export interface FinancialChartParams {
  period: 'day' | 'week' | 'month' | 'quarter' | 'year';
}

export interface FinancialChartResponse {
  status: 'success' | 'error';
  data: FinancialChartDataPoint[];
  message: string;
}

// Salary Distribution Types
export interface SalaryDistributionSummary {
  count: number;
  mean: number;
  median: number;
  min: number;
  max: number;
  std_dev: number;
}

export interface SalaryDistributionData {
  values: number[];
  summary: SalaryDistributionSummary;
}

export type SalaryDistributionCycleKey = 'weekly' | 'monthly' | 'flexible';

export interface SalaryDistributionGroupedData {
  weekly?: SalaryDistributionData;
  monthly?: SalaryDistributionData;
  flexible?: SalaryDistributionData;
}

export interface SalaryDistributionParams {
  fromDate?: string;
  toDate?: string;
  month?: string;
}

export interface SalaryDistributionMeta {
  period: {
    from: string;
    to: string;
  };
}

export interface SalaryDistributionResponse {
  success: boolean;
  data: SalaryDistributionGroupedData;
  meta: SalaryDistributionMeta;
}

// Employee Activity Stats
export interface EmployeeActivityStats {
  active_weekly: number;
  active_monthly: number;
  active_flexible: number;
  active_total: number;
}

// Active employee user in activity list
export interface ActiveEmployeeUser {
  user_id: number;
  employee_id: number;
  fullname: string;
  username: string;
  last_login: string | null;
  role: string;
  payment_schedule: string;
}
// Projection Types
export interface WeeklyPayData {
  date: string;
  paid_amount: number;
  paid_employees: number;
  avg_pay_per_employee: number;
}

export interface RevenueData {
  date: string;
  total_revenue: number;
  total_expenses: number;
  profit: number;
}

export interface CapitalData {
  date: string;
  amount: number;
}

export interface HistoricalData {
  weekly_pay: WeeklyPayData[];
  revenue: RevenueData[];
  capital: CapitalData[];
}

export interface ProjectionsData {
  paid_employees: {
    '30d': number;
    '60d': number;
    '90d': number;
  };
  weekly_pay: {
    '30d': number;
    '60d': number;
    '90d': number;
  };
  capital: {
    '30d': number;
    '60d': number;
    '90d': number;
  };
  profit: {
    '30d': number;
    '60d': number;
    '90d': number;
  };
  revenue: {
    '30d': number;
    '60d': number;
    '90d': number;
  };
}

export interface ProjectionResponseData {
  date: string;
  historical: HistoricalData;
  projections: ProjectionsData;
}

export interface ProjectionResponse {
  status: 'success' | 'error';
  data: ProjectionResponseData;
  message: string;
}

export interface HistoricalResponseData {
  date: string;
  historical: HistoricalData;
}

export interface HistoricalResponse {
  status: 'success' | 'error';
  data: HistoricalResponseData;
  message: string;
}

// Project Profitability Types
export interface ProjectProfitabilityItem {
  rank: number;
  project_id: number;
  project_name: string;
  client_name: string;
  employee_count: number;
  total_received_vnd: number;
  total_payout_vnd: number;
  net_profit_vnd: number;
  profit_margin_percent: number;
  status: string;
}

export interface ProjectProfitabilityResponse {
  projects: ProjectProfitabilityItem[];
  total_count: number;
}

// Project Weekly Profit Types
export interface ProjectWeeklySeries {
  project_id: number;
  project_name: string;
  total_profit: number;
  data: number[];
}

export interface ProjectWeeklyProfitResponse {
  days: string[];
  series: ProjectWeeklySeries[];
}

// Monthly Financial Summary Types
export type MonthlyFinancialsPeriod = '3m' | '6m' | '1y';

export interface MonthlyFinancialRow {
  month: string;     // "2026-01"
  paid_out: number;  // expenses
  billed: number;    // revenue
  fee_earned: number; // profit
}

export interface MonthlyFinancialsResponse {
  period: MonthlyFinancialsPeriod;
  months: MonthlyFinancialRow[];
  total: MonthlyFinancialRow;
}

// Top Paid Employees Types
export interface TopPaidEmployeeItem {
  rank: number;
  employee_id: number;
  employee_name: string;
  total_paid_vnd: number;
  is_active: boolean;
  last_timesheet_date: string;
}

export interface TopPaidEmployeesParams {
  month?: string; // YYYY-MM, optional; omit for all-time
  limit?: number;
}

export interface TopPaidEmployeesResponse {
  employees: TopPaidEmployeeItem[];
  period: string; // "all_time" or "YYYY-MM"
}

// Partner Dashboard Types
export interface PartnerDashboardParams {
  month?: string; // YYYY-MM
}

export interface PartnerDashboardWoWChange {
  current_week: number;
  previous_week: number;
  change_amount: number;
  change_pct: number;
}

export interface PartnerDashboardMoMChange {
  current_month: number;
  previous_month: number;
  change_amount: number;
  change_pct: number;
}

export interface PartnerDashboardData {
  active_employees: number;
  dropped_employees: number;
  paid_employees: number;
  total_paid_vnd: number;
  wow_paid_employees: PartnerDashboardWoWChange;
  wow_paid_amount: PartnerDashboardWoWChange;
  mom_paid_employees: PartnerDashboardMoMChange;
  mom_paid_amount: PartnerDashboardMoMChange;
  top_paid_employees: TopPaidEmployeeItem[];
  period: string;
}

// Bank Usage Types
export interface BankUsageItem {
  bank_name: string;
  employee_count: number;
  total_paid_vnd: number;
  transfer_count: number;
  percentage: number;
}

export interface BankUsageParams {
  project_id?: number;
}

export interface BankUsageResponse {
  banks: BankUsageItem[];
  total_employees: number;
  total_paid_vnd: number;
  total_transfers: number;
  project_id?: number;
  project_name?: string;
}

export interface ProjectBankUsageItem {
  project_id: number;
  project_name: string;
  banks: BankUsageItem[];
  total_employees: number;
}

export interface BankUsageAllProjectsResponse {
  projects: ProjectBankUsageItem[];
  overall: BankUsageResponse;
}

// Partner Employee List Types
export type PartnerEmployeeListType = 'active' | 'dropped' | 'paid';

export interface PartnerEmployeeDetailItem {
  employee_id: number;
  employee_name: string;
  mobile: string;
  cccd: string;
  project_name: string;
  last_timesheet_date: string;
  total_paid_vnd: number;
  last_paid_vnd: number;
  last_paid_date: string;
  is_active: boolean;
}

export interface PartnerEmployeeListParams {
  type: PartnerEmployeeListType;
  month?: string;
}

export interface PartnerEmployeeListResponse {
  employees: PartnerEmployeeDetailItem[];
  type: PartnerEmployeeListType;
  total: number;
}

// ========== Check-in Health Metrics ==========

export interface FailedAttemptCategoryCount {
  category: string;
  count: number;
}

export interface CheckInHealthResponse {
  report_date?: string;
  period_start?: string;
  period_end?: string;
  // A. Check-in / Check-out
  failed_attempts_by_category: FailedAttemptCategoryCount[];
  failed_check_in_today: number;
  failed_check_out_today: number;
  open_checked_in: number;
  orphaned: number;
  auto_rejected_today: number;
  completed_zero_earning_today: number;
  successful_checkouts_today: number;
  // B. Quota
  quota_invariant_drift: number;
  missing_quota_rows: number;
  stale_quota_after_disable: number;
  quota_salary_this_month: number;
  quota_max_adv_this_month: number;
  // C. Advance requests
  requests_stuck_pending: number;
  requests_failed_today: number;
  requests_completed_today: number;
  requests_total_today: number;
}

export interface QuotaAnomalyRow {
  employee_id: number;
  project_id: number;
  for_month: string;
  salary: number;
  max_adv_amount: number;
  expected_max: number;
  reason: string;
  employee_name?: string;
  project_name?: string;
}

// ========== Admin Failed Attempts ==========

export interface AdminFailedAttempt {
  id: number;
  employee_id: number;
  employee_name?: string;
  attempt_type: string;
  reason_category: string;
  project_id: number;
  lat?: number | null;
  lng?: number | null;
  accuracy?: number | null;
  gps_at?: string | null;
  nearest_checkpoint_name?: string | null;
  nearest_checkpoint_distance_meters?: number | null;
  geofence_radius_meters?: number | null;
  error_message?: string | null;
  created_at: string;
}

export interface PaginatedFailedAttemptsResponse {
  data: AdminFailedAttempt[];
  total: number;
}
