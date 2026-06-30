/**
 * Centralized Query Key Factory
 * Single source of truth for all query keys across the application
 * Provides type-safe, hierarchical query key builders
 */

import type {
  ProjectFilters,
  AssignEmployeeData,
  CreatePayRateData,
} from '@/types/api/project.types';
import type {
  EmployeeFilters,
  CreateEmployeeData,
} from '@/types/api/employee.types';
import type {
  TimesheetFilters,
  CreateTimesheetData,
} from '@/types/api/timesheet.types';
import type {
  ProjectEmployeeListParams,
  EmployeeProjectListParams,
  TimesheetSummaryParams,
  AdvancedAssignmentFilters,
} from '@/types/api/project-employee.types';
import type { PayRateListParams } from '@/types/api/payrate.types';
import type { UserFilters } from '@/services/api/user.service';
import type { AssetFilters } from '@/types/api/financial.types';

// Generic filter interfaces for query parameters
interface ReportParams {
  fromDate?: string;
  toDate?: string;
  project_id?: number;
  employee_id?: number;
  status?: string;
  [key: string]: unknown;
}

interface NotificationFilters {
  status?: 'read' | 'unread';
  type?: string;
  fromDate?: string;
  toDate?: string;
  page?: number;
  pageSize?: number;
}

interface AuditLogFilters {
  action?: string;
  resource?: string;
  user_id?: number;
  fromDate?: string;
  toDate?: string;
  page?: number;
  pageSize?: number;
}

/**
 * Core Query Keys Factory
 * Hierarchical structure with type-safe builders
 */
export const QueryKeys = {
  // ========== PROJECTS ==========
  projects: {
    all: ['projects'] as const,
    lists: () => [...QueryKeys.projects.all, 'list'] as const,
    list: (filters?: ProjectFilters) => [...QueryKeys.projects.lists(), filters] as const,
    search: (params?: { search: string; pageSize?: number; status?: string }) =>
      [...QueryKeys.projects.all, 'search', params] as const,
    details: () => [...QueryKeys.projects.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.projects.details(), id] as const,
    summary: () => [...QueryKeys.projects.all, 'summary'] as const,
    assignable: () => [...QueryKeys.projects.all, 'assignable'] as const,

    // Project relationships
    employees: (id: number, filters?: AdvancedAssignmentFilters) =>
      [...QueryKeys.projects.all, id, 'employees', filters] as const,
    timesheets: (id: number, filters?: TimesheetFilters) =>
      [...QueryKeys.projects.all, id, 'timesheets', filters] as const,
    payrates: (id: number) => [...QueryKeys.projects.all, id, 'payrates'] as const,
    financial: (id: number) => [...QueryKeys.projects.all, id, 'financial'] as const,
    assignments: (id: number) => [...QueryKeys.projects.all, id, 'assignments'] as const,
    stats: (id: number) => [...QueryKeys.projects.all, id, 'stats'] as const,

    // Partner projects
    partner: {
      all: () => [...QueryKeys.projects.all, 'partner'] as const,
      summary: () => [...QueryKeys.projects.partner.all(), 'summary'] as const,
      list: (filters?: ProjectFilters) => [...QueryKeys.projects.partner.all(), 'list', filters] as const,
    },

    users: (id: number) => [...QueryKeys.projects.all, id, 'users'] as const,
  },

  // ========== EMPLOYEES ==========
  employees: {
    all: ['employees'] as const,
    lists: () => [...QueryKeys.employees.all, 'list'] as const,
    list: (filters?: EmployeeFilters) => [...QueryKeys.employees.lists(), filters] as const,
    search: (params?: { search: string; pageSize?: number }) =>
      [...QueryKeys.employees.all, 'search', params] as const,
    details: () => [...QueryKeys.employees.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.employees.details(), id] as const,
    summary: () => [...QueryKeys.employees.all, 'summary'] as const,

    // Employee relationships
    projects: (id: number, params?: EmployeeProjectListParams) =>
      [...QueryKeys.employees.all, id, 'projects', params] as const,
    timesheets: (id: number, filters?: TimesheetFilters) =>
      [...QueryKeys.employees.all, id, 'timesheets', filters] as const,
    assignments: (id: number) => [...QueryKeys.employees.all, id, 'assignments'] as const,
    stats: (id: number) => [...QueryKeys.employees.all, id, 'stats'] as const,
    payrates: (id: number) => [...QueryKeys.employees.all, id, 'payrates'] as const,
    missingBankDetails: (filters?: EmployeeFilters) => [...QueryKeys.employees.all, 'missing-bank-details', filters] as const,
    timesheet: (id: number, filters?: TimesheetFilters) =>
      [...QueryKeys.employees.all, id, 'timesheet', filters] as const,
    employeeSummary: (id: number) =>
      [...QueryKeys.employees.all, id, 'employee-summary'] as const,
    currentProjects: (id: number, filters?: Record<string, unknown>) =>
      [...QueryKeys.employees.all, id, 'current-projects', filters] as const,
    payroll: (id: number, filters?: Record<string, unknown>) =>
      [...QueryKeys.employees.all, id, 'payroll', filters] as const,
    infiniteList: (filters?: Omit<EmployeeFilters, 'page'>) =>
      [...QueryKeys.employees.all, 'infinite-list', filters] as const,
    byCCCD: (cccd: string) =>
      [...QueryKeys.employees.all, 'by-cccd', cccd] as const,
    users: (employeeId: number) =>
      [...QueryKeys.employees.all, employeeId, 'users'] as const,
  },

  // ========== TIMESHEETS ==========
  timesheets: {
    all: ['timesheets'] as const,
    lists: () => [...QueryKeys.timesheets.all, 'list'] as const,
    list: (filters?: TimesheetFilters) => [...QueryKeys.timesheets.lists(), filters] as const,
    details: () => [...QueryKeys.timesheets.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.timesheets.details(), id] as const,
    summary: (params?: { project_id?: number; employee_id?: number; fromDate?: string; toDate?: string; }) =>
      [...QueryKeys.timesheets.all, 'summary', params] as const,

    // Filtered timesheets
    byProject: (projectId: number, filters?: TimesheetFilters) =>
      [...QueryKeys.timesheets.all, 'byProject', projectId, filters] as const,
    byEmployee: (employeeId: number, filters?: TimesheetFilters) =>
      [...QueryKeys.timesheets.all, 'byEmployee', employeeId, filters] as const,
    byProjectAndDate: (projectId: number, date: string) =>
      [...QueryKeys.timesheets.all, 'project', projectId, 'date', date] as const,
    employeeSummary: (employeeId: number, params?: { fromDate?: string; toDate?: string; }) =>
      ['employees', employeeId, 'timesheet-summary', params] as const,

    // Timesheet analytics
    analytics: () => [...QueryKeys.timesheets.all, 'analytics'] as const,
    reports: (params?: ReportParams) => [...QueryKeys.timesheets.all, 'reports', params] as const,
  },

  // ========== PROJECT EMPLOYEES / ASSIGNMENTS ==========
  assignments: {
    all: ['assignments'] as const,
    lists: () => [...QueryKeys.assignments.all, 'list'] as const,
    details: () => [...QueryKeys.assignments.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.assignments.details(), id] as const,

    // Assignment relationships
    timesheetSummary: (id: number, params?: TimesheetSummaryParams) =>
      [...QueryKeys.assignments.all, id, 'timesheet-summary', params] as const,
    history: (id: number) => [...QueryKeys.assignments.all, id, 'history'] as const,

    // Assignment queries
    search: (filters: AdvancedAssignmentFilters) => [...QueryKeys.assignments.all, 'search', filters] as const,
    byProject: (projectId: number, params?: ProjectEmployeeListParams) =>
      [...QueryKeys.assignments.all, 'byProject', projectId, params] as const,
    byEmployee: (employeeId: number, params?: EmployeeProjectListParams) =>
      [...QueryKeys.assignments.all, 'byEmployee', employeeId, params] as const,
    check: (employeeId: number, projectId: number) =>
      [...QueryKeys.assignments.all, 'check', employeeId, projectId] as const,

    // Assignment stats
    projectStats: (projectId: number) =>
      [...QueryKeys.assignments.all, 'projectStats', projectId] as const,
    employeeStats: (employeeId: number) =>
      [...QueryKeys.assignments.all, 'employeeStats', employeeId] as const,
  },

  // ========== PAY RATES ==========
  payrates: {
    all: ['payrates'] as const,
    lists: () => [...QueryKeys.payrates.all, 'list'] as const,
    list: (filters?: PayRateListParams) => [...QueryKeys.payrates.lists(), filters] as const,
    details: () => [...QueryKeys.payrates.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.payrates.details(), id] as const,

    // Pay rate relationships
    byProject: (projectId: number) => [...QueryKeys.payrates.all, 'byProject', projectId] as const,
    byEmployee: (employeeId: number) => [...QueryKeys.payrates.all, 'byEmployee', employeeId] as const,
  },

  // ========== USERS ==========
  users: {
    all: ['users'] as const,
    lists: () => [...QueryKeys.users.all, 'list'] as const,
    list: (filters?: UserFilters) => [...QueryKeys.users.lists(), filters] as const,
    infiniteLists: () => [...QueryKeys.users.all, 'infinite-list'] as const,
    infiniteList: (filters?: Omit<UserFilters, 'page'>) => [...QueryKeys.users.infiniteLists(), filters] as const,
    details: () => [...QueryKeys.users.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.users.details(), id] as const,
    profile: () => [...QueryKeys.users.all, 'profile'] as const,
    summary: () => [...QueryKeys.users.all, 'summary'] as const,
    activities: (id: number, days?: number) => [...QueryKeys.users.all, id, 'activities', days] as const,
  },

  // ========== ASSETS ==========
  assets: {
    all: ['assets'] as const,
    lists: () => [...QueryKeys.assets.all, 'list'] as const,
    list: (filters?: AssetFilters) => [...QueryKeys.assets.lists(), filters] as const,
    details: () => [...QueryKeys.assets.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.assets.details(), id] as const,
    byReference: (referenceType: string, referenceId: number) =>
      [...QueryKeys.assets.all, 'reference', referenceType, referenceId] as const,
  },

  // ========== BANKS ==========
  banks: {
    all: ['banks'] as const,
    lists: () => [...QueryKeys.banks.all, 'list'] as const,
    list: () => [...QueryKeys.banks.all, 'list'] as const,
    details: () => [...QueryKeys.banks.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.banks.details(), id] as const,
    search: () => [...QueryKeys.banks.all, 'search'] as const,
  },

  // ========== NOTIFICATIONS ==========
  notifications: {
    all: ['notifications'] as const,
    lists: () => [...QueryKeys.notifications.all, 'list'] as const,
    list: (filters?: NotificationFilters) => [...QueryKeys.notifications.lists(), filters] as const,
    unread: () => [...QueryKeys.notifications.all, 'unread'] as const,
  },

  // ========== DASHBOARD ==========
  dashboard: {
    all: ['dashboard'] as const,
    stats: () => [...QueryKeys.dashboard.all, 'stats'] as const,
    summary: (month?: string) => [...QueryKeys.dashboard.all, 'summary', month] as const,
    charts: (type?: string) => [...QueryKeys.dashboard.all, 'charts', type] as const,
    employeeActivity: () => [...QueryKeys.dashboard.all, 'employee-activity'] as const,
    employeeActivityUsers: (schedule: string) => [...QueryKeys.dashboard.all, 'employee-activity-users', schedule] as const,
    projectProfitability: () => [...QueryKeys.dashboard.all, 'project-profitability'] as const,
    projectWeeklyProfit: (weeks?: number) => [...QueryKeys.dashboard.all, 'project-weekly-profit', weeks] as const,
    topPaidEmployees: (month?: string, limit?: number) => [...QueryKeys.dashboard.all, 'top-paid-employees', month, limit] as const,
    partnerDashboard: (month?: string) => [...QueryKeys.dashboard.all, 'partner-dashboard', month] as const,
    bankUsage: (projectId?: number) => [...QueryKeys.dashboard.all, 'bank-usage', projectId] as const,
    bankUsageAllProjects: () => [...QueryKeys.dashboard.all, 'bank-usage-projects'] as const,
    partnerEmployeeList: (type: string, month?: string) => [...QueryKeys.dashboard.all, 'partner-employee-list', type, month] as const,
    checkInHealth: (month?: string) => [...QueryKeys.dashboard.all, 'check-in-health', month] as const,
    quotaAnomalies: (type?: string, month?: string) => [...QueryKeys.dashboard.all, 'quota-anomalies', type, month] as const,
    failedAttempts: (params?: Record<string, unknown>) => [...QueryKeys.dashboard.all, 'failed-attempts', params] as const,
  },

  // ========== AUDIT ==========
  audit: {
    all: ['audit'] as const,
    logs: (filters?: AuditLogFilters) => [...QueryKeys.audit.all, 'logs', filters] as const,
    logDetail: (id: number) => [...QueryKeys.audit.all, 'logs', id] as const,
  },

  // ========== SETTINGS ==========
  settings: {
    all: ['settings'] as const,
    lists: () => [...QueryKeys.settings.all, 'list'] as const,
    list: (filters?: Record<string, unknown>) => [...QueryKeys.settings.lists(), filters] as const,
    details: () => [...QueryKeys.settings.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.settings.details(), id] as const,
    byKey: (key: string) => [...QueryKeys.settings.all, 'key', key] as const,
  },

  // ========== LENDERS ==========
  lenders: {
    all: ['lenders'] as const,
    lists: () => [...QueryKeys.lenders.all, 'list'] as const,
    list: (filters?: Record<string, unknown>) => [...QueryKeys.lenders.lists(), filters] as const,
    details: () => [...QueryKeys.lenders.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.lenders.details(), id] as const,
  },

  // ========== LOANS ==========
  loans: {
    all: ['loans'] as const,
    lists: () => [...QueryKeys.loans.all, 'list'] as const,
    list: (filters?: Record<string, unknown>) => [...QueryKeys.loans.lists(), filters] as const,
    details: () => [...QueryKeys.loans.all, 'detail'] as const,
    detail: (id: number) => [...QueryKeys.loans.details(), id] as const,
    schedule: (id: number) => [...QueryKeys.loans.all, id, 'schedule'] as const,
  },

  // ========== CRON HEALTH ==========
  cronHealth: {
    all: ['cron-health'] as const,
    jobs: () => [...QueryKeys.cronHealth.all, 'jobs'] as const,
  },

  // ========== SYSTEM HEALTH ==========
  systemHealth: {
    all: ['system-health'] as const,
    apiSummary: (days?: number, sortBy?: string) => [...QueryKeys.systemHealth.all, 'api-summary', days, sortBy] as const,
    errorsByUser: (days: number) => [...QueryKeys.systemHealth.all, 'errors-by-user', days] as const,
    latencyTrend: (days: number, groupBy: string) => [...QueryKeys.systemHealth.all, 'latency-trend', days, groupBy] as const,
    slowestEndpoints: (days: number) => [...QueryKeys.systemHealth.all, 'slowest-endpoints', days] as const,
    recentErrors: (days: number) => [...QueryKeys.systemHealth.all, 'recent-errors', days] as const,
    errorCount: (days: number) => [...QueryKeys.systemHealth.all, 'error-count', days] as const,
    eventBus: () => [...QueryKeys.systemHealth.all, 'event-bus'] as const,
    cache: () => [...QueryKeys.systemHealth.all, 'cache'] as const,
    topEndpoints: (days: number) => [...QueryKeys.systemHealth.all, 'top-endpoints', days] as const,
    endpointTrend: (endpointId: number, days: number, groupBy: string) => [...QueryKeys.systemHealth.all, 'endpoint-trend', endpointId, days, groupBy] as const,
    failedLogins: (days: number) => [...QueryKeys.systemHealth.all, 'failed-logins', days] as const,
    failedLoginsByIdentifier: (identifier: string, days: number) => [...QueryKeys.systemHealth.all, 'failed-logins-detail', identifier, days] as const,
    browserPlatformStats: (days: number) => [...QueryKeys.systemHealth.all, 'browser-platform-stats', days] as const,
    browserPlatformUsers: (browser: string, platform: string, days: number) => [...QueryKeys.systemHealth.all, 'browser-platform-users', browser, platform, days] as const,
    osStats: (days: number) => [...QueryKeys.systemHealth.all, 'os-stats', days] as const,
    osUsers: (osFamily: string, days: number) => [...QueryKeys.systemHealth.all, 'os-users', osFamily, days] as const,
    browserStats: (days: number) => [...QueryKeys.systemHealth.all, 'browser-stats', days] as const,
    browserUsers: (browserFamily: string, days: number) => [...QueryKeys.systemHealth.all, 'browser-users', browserFamily, days] as const,
  },

  // ========== ADVANCE PAYMENTS ==========
  advancePayments: {
    employee: {
      info: ['employee', 'advance-payment', 'info'] as const,
      checkInAdvanceInfo: ['employee', 'check-in-advance', 'info'] as const,
      history: (filters?: { page?: number; pageSize?: number }) =>
        ['employee', 'advance-payment', 'history', filters] as const,
    },
    admin: {
      list: (filters?: Record<string, unknown>) =>
        ['admin', 'advance-payments', 'list', filters] as const,
      summary: (params?: { fromDate?: string; toDate?: string }) =>
        ['admin', 'advance-payments', 'summary', params] as const,
      files: ['admin', 'advance-payments', 'files'] as const,
      flexPayEmployees: (filters?: Record<string, unknown>) =>
        ['admin', 'advance-payments', 'flex-pay-employees', filters] as const,
      availableMonths: ['admin', 'advance-payments', 'available-months'] as const,
      feeSchedules: ['admin', 'advance-payments', 'fee-schedules'] as const,
      disbursementFeeSchedules: ['admin', 'advance-payments', 'disbursement-fee-schedules'] as const,
    },
  },

  // ========== MANUAL DISBURSEMENT (admin "Chuyển tiền" page) ==========
  manualDisbursement: {
    list: (limit: number) => ['admin', 'manual-disbursement', 'list', limit] as const,
    detail: (txnId: string) => ['admin', 'manual-disbursement', 'detail', txnId] as const,
  },

  // ========== AUTH ==========
  auth: {
    all: ['auth'] as const,
    user: () => [...QueryKeys.auth.all, 'user'] as const,
    permissions: () => [...QueryKeys.auth.all, 'permissions'] as const,
  },
} as const;

/**
 * Query Key Utilities
 */
export const QueryKeyUtils = {
  /**
   * Get all query keys for a specific entity
   */
  getEntityKeys: (entity: keyof typeof QueryKeys) => (QueryKeys[entity] as { all: unknown }).all,

  /**
   * Check if a query key matches a pattern
   */
  matchesPattern: (queryKey: readonly unknown[], pattern: readonly unknown[]): boolean => {
    if (pattern.length > queryKey.length) return false;
    return pattern.every((part, index) => part === queryKey[index]);
  },

  /**
   * Get all keys that start with a pattern
   */
  getKeysStartingWith: (pattern: readonly unknown[]) => pattern,
} as const;

/**
 * Legacy query key exports for backward compatibility
 * These will be gradually replaced
 */
export {
  projectEmployeesKey,
  projectDetailKey,
  employeeProjectsKey,
  employeeAssignmentCheckKey,
  projectAssignmentStatsKey,
  employeeAssignmentStatsKey,
  shouldInvalidateForProjectEmployeeChange
} from '@/lib/queryKeys';
