// API Configuration
export const API_CONFIG = {
  baseURL: import.meta.env.VITE_API_BASE_URL ?? (import.meta.env.DEV ? 'http://localhost:8080' : ''),
  apiV1: '/api/v1',
  timeout: Number(import.meta.env.VITE_API_TIMEOUT) || 30000,
};

export const API_ENDPOINTS = {
  // Authentication
  auth: {
    login: '/auth/login',
    loginVerifyOtp: '/auth/login/verify',
    loginResendOtp: '/auth/login/resend',
    logout: '/auth/logout',
    me: '/auth/me',
    updateProfile: '/auth/me',
    changePassword: '/auth/change-password',
    google: '/auth/google',
    passwordResetRequest: '/auth/password-reset/request',
    passwordResetConfirm: '/auth/password-reset/confirm',
    zaloResetRequest: '/auth/zalo-reset/request',
    zaloResetConfirm: '/auth/zalo-reset/confirm',
  },

  // Users
  users: {
    base: '/users',
    summary: '/users/summary',
    byId: (id: number) => `/users/${id}`,
    suspend: (id: number) => `/users/${id}/suspend`,
    activate: (id: number) => `/users/${id}/activate`,
    resetPassword: (id: number) => `/users/${id}/reset-password`,
  },

  // Employees
  employees: {
    base: '/employees',
    summary: '/employees/summary',
    missingBankDetails: '/employees/missing-bank-details',
    search: '/employees/search',
    export: '/employees/export',
    exportDetail: (id: number) => `/employees/${id}/export`,
    import: '/employees/import',
    importStatus: (id: string) => `/employees/import/${id}/status`,
    byId: (id: number) => `/employees/${id}`,
    summaryById: (id: number) => `/employees/${id}/summary`,
    byCCCD: (cccd: string) => `/employees/cccd/${cccd}`,
    projects: (id: number) => `/employees/${id}/projects`,
    currentProjects: (id: number) => `/employees/${id}/current-projects`,
    timesheetSummary: (id: number) => `/employees/${id}/timesheets/summary`,
    payroll: (id: number) => `/employees/${id}/payroll`,
    timesheet: (id: number) => `/employees/${id}/timesheet`,
    assignmentStats: (id: number) => `/employees/${id}/assignment-stats`,
    changePassword: (id: number) => `/employees/${id}/change-password`,
    users: (id: number) => `/employees/${id}/users`,
    userAccess: (employeeId: number, userId: number) => `/employees/${employeeId}/users/${userId}`,
  },

  // Banks
  banks: {
    base: '/banks',
    byId: (id: number) => `/banks/${id}`,
  },

  // Projects
  projects: {
    base: '/projects',
    summary: '/projects/summary',
    partnerSummary: '/projects/partner-summary',
    byId: (id: number) => `/projects/${id}`,
    status: (id: number) => `/projects/${id}/status`,
    employees: (id: number) => `/projects/${id}/employees`,
  employeesRemove: (id: number) => `/projects/${id}/employees/remove`,
    assignEmployee: (id: number) => `/projects/${id}/employees`,
    bulkAssign: (id: number) => `/projects/${id}/employees/bulk-assign`,
    timesheets: (id: number) => `/projects/${id}/timesheets`,
    timesheetsImport: (id: number) => `/projects/${id}/timesheets/import`,
    payrate: (id: number) => `/projects/${id}/payrate`,
    payrateById: (projectId: number, payrateId: number) => `/projects/${projectId}/payrate/${payrateId}`,
    payrateActive: (id: number) => `/projects/${id}/payrate/active`,
    financial: (id: number) => `/projects/${id}/financial`,
    revenue: (id: number) => `/projects/${id}/revenue`,
    assignmentStats: (id: number) => `/projects/${id}/assignment-stats`,
    users: (id: number) => `/projects/${id}/users`,
    userAccess: (projectId: number, userId: number) => `/projects/${projectId}/users/${userId}`,
  },

  // Pay rates (global)
  payrates: {
    byId: (id: number) => `/payrates/${id}`,
  },


  // Timesheets
  timesheets: {
    base: '/timesheets',
    preview: '/timesheets/preview',
    summary: '/timesheets/summary',
    cashReadiness: '/timesheets/cash-readiness',
    grouped: '/timesheets/grouped', // Server-side grouped by employee for partner view
    byId: (id: number) => `/timesheets/${id}`,
    approve: (id: number) => `/timesheets/${id}/approve`,
    reject: (id: number) => `/timesheets/${id}/reject`,
    addToPayroll: (id: number) => `/timesheets/${id}/add-to-payroll`,
    bulkApprove: '/timesheets/bulk-approve',
    approveAll: '/timesheets/approve-all',
    bulkReject: '/timesheets/bulk-reject',
    rejectUnpaid: '/timesheets/reject-unpaid',
    bulkReset: '/timesheets/bulk-reset',
    import: '/timesheets/import',
    export: '/timesheets/export',
    uploadEntriesExcel: '/timesheets/upload-entries-excel',
    payrollReport: '/timesheets/payroll/report',
    emailPayrollReport: '/timesheets/payroll/report/send-email',
    uploadSettlementResult: '/timesheets/payroll/upload-settlement-result',
    byProjectAndDate: (projectId: number) => `/timesheets/projects/${projectId}`,
    // New endpoints for bulk approval by project/employee
    approveByProject: (projectId: number) => `/projects/${projectId}/timesheets/approve`,
    approveByEmployee: (employeeId: number) => `/employees/${employeeId}/timesheets/approve`,
    // Edit request endpoints
    requestEdit: (id: number) => `/timesheets/${id}/request-edit`,
    cancelEditRequest: (id: number) => `/timesheets/${id}/request-edit-cancel`,
    editRequests: '/timesheets/edit-requests',
    editRequestById: (id: number) => `/timesheets/edit-requests/${id}`,
    approveEditRequest: (id: number) => `/timesheets/edit-requests/${id}/approve`,
    rejectEditRequest: (id: number) => `/timesheets/edit-requests/${id}/reject`,
    // BCC partner import
    partnerImport: '/timesheets/partner-import',
    partnerImportById: (id: number) => `/timesheets/partner-import/${id}`,
    partnerImportDownload: (id: number) => `/timesheets/partner-import/${id}/download`,
  },

  // Import/Export
  imports: {
    base: '/imports',
    summary: '/imports/summary',
    byBatchId: (batchId: string) => `/imports/${batchId}`,
    cancel: (batchId: string) => `/imports/${batchId}/cancel`,
    retry: (batchId: string) => `/imports/${batchId}/retry`,
    employees: '/imports/employees',
    timesheets: '/imports/timesheets',
    payrollResults: '/imports/payroll-results',
    templates: (type: string) => `/imports/templates/${type}`,
  },

  exports: {
    base: '/exports',
    byType: (type: string) => `/exports/${type}`,
    byId: (exportId: string) => `/exports/${exportId}`,
    download: (exportId: string) => `/exports/${exportId}/download`,
  },

  // Payrolls
  payrolls: {
    base: '/payrolls',
    byId: (id: number) => `/payrolls/${id}`,
    histories: '/payrolls/histories',
    bankTransferHistories: '/payrolls/bank-transfer-histories',
    exportHistories: '/payrolls/histories/export',
    bulkTransferTemplate: '/payrolls/bulk-transfer-template',
    exportBulkTransfer: '/payrolls/export-bulk-transfer',
    /**
     * Stage 1 of the Wallet Bulk Transfer Pipeline — exports a OnePay-API-
     * compatible "Yêu cầu chuyển tiền" .xlsx with a SWIFT code column resolved
     * from each employee's bank record. Output is the canonical input for
     * /wallet/bulk-transfer/upload (Stage 2).
     */
    exportOnePayBulk: '/payrolls/export-onepay-bulk',
    simulateSettlement: '/payrolls/simulate-settlement',
    importBulkTransferResult: '/payrolls/bulk-transfer-result',
    bulkTransferUploadHistories: '/payrolls/bulk-transfer-upload-histories',
    bulkTransferUploadHistoryById: (id: number) => `/payrolls/bulk-transfer-upload-histories/${id}`,
    bulkTransferUploadHistoryExportPdf: (id: number) => `/payrolls/bulk-transfer-upload-histories/${id}/export-pdf`,
    bulkTransferUploadHistoryDownload: (id: number) => `/payrolls/bulk-transfer-upload-histories/${id}/download`,
    autoBulkTransferConfig: '/payrolls/auto-bulk-transfer/config',
    initiateAutoBulkTransfer: '/payrolls/auto-bulk-transfer',
    autoBulkTransferStatus: (batchId: string) => `/payrolls/auto-bulk-transfer/${batchId}/status`,
    /**
     * Mark a list of timesheets as paid externally (e.g. after a 9Pay batch
     * partially fails and admin pays those rows out-of-band).
     * **Backend TODO** — endpoint not implemented yet; calls will return 404
     * until the server side is built. See docs/timesheet-redesign-2026-05-09.md.
     */
    markBulkTransferExternallyPaid: '/payrolls/bulk-transfer-external-mark',
  },

  // Financial
  financial: {
    dashboard: '/financial/dashboard',
    receivables: '/financial/receivables',
    payables: '/financial/payables',
    reports: '/financial/reports',
    reportById: (reportId: string) => `/financial/reports/${reportId}`,
    projectSummary: (projectId: number) => `/financial/projects/${projectId}/summary`,
    cashFlow: '/financial/cash-flow',
  },

  // Attendance
  attendance: {
    admin: {
      list: '/admin/attendances',
      byId: (id: number) => `/admin/attendances/${id}`,
      approve: (id: number) => `/admin/attendances/${id}/approve`,
      reject: (id: number) => `/admin/attendances/${id}/reject`,
    },
    mobile: {
      today: '/mobile/attendance/today',
      checkIn: '/mobile/attendance/check-in',
      checkOut: '/mobile/attendance/check-out',
      cancelCurrent: '/mobile/attendance/cancel-current',
      attemptLog: '/mobile/attendance/attempt-log',
      history: '/mobile/attendance/history',
    },
  },

  // Ledger (New API endpoints)
  ledger: {
    entries: '/ledger/entries',
    entryById: (id: number) => `/ledger/entries/${id}`,
    reverseEntry: (id: number) => `/ledger/entries/${id}/reverse`,
    transactions: '/ledger/transactions',
    balance: '/ledger/balance',
    recalculateBalance: '/ledger/balance/recalculate',
    accountBalance: (account: string) => `/ledger/balance/account/${account}`,
    projectBalance: (projectId: number) => `/ledger/balance/project/${projectId}`,
    cashFlow: '/ledger/cash-flow',
    summary: '/ledger/summary',
    accountsMetadata: '/ledger/accounts/metadata',
    export: '/ledger/export',
    onePayFeeReports: '/ledger/onepay-fee-reports',
  },

  // Wallet disbursement settlement — admin on-demand trigger for the EOD
  // consolidation that groups completed wallet payments into ledger records.
  walletSettlement: {
    run: '/admin/wallet-settlement/run',
  },

  // Wallet Bulk Transfer Pipeline (Stage 2: upload + process).
  // POST /wallet/bulk-transfer/upload parses the .xlsx and enqueues per-row
  // OnePay transfer tasks. GET endpoints return batch state for the
  // progress UI; /kq returns the result .xlsx.
  walletBulkTransfer: {
    upload: '/wallet/bulk-transfer/upload',
    batches: '/wallet/bulk-transfer/batches',
    batchById: (id: number) => `/wallet/bulk-transfer/batches/${id}`,
    batchKQ: (id: number) => `/wallet/bulk-transfer/batches/${id}/kq`,
  },

  // Transactions (Revenue/Expense management)
  transactions: {
    base: '/transactions',
    byId: (id: number) => `/transactions/${id}`,
    pending: '/transactions/pending',
    settle: (id: number) => `/transactions/${id}/settle`,
    reverse: (id: number) => `/transactions/${id}/reverse`,
    export: '/transactions/export',
    metadata: '/transactions/metadata',
  },

  // Assets (Evidence files)
  assets: {
    base: '/assets',
    upload: '/assets/upload',
    byId: (id: number) => `/assets/${id}`,
    download: (id: number) => `/assets/${id}/download`,
    reference: (id: number) => `/assets/${id}/reference`,
  },

  // Lenders
  lenders: {
    base: '/lenders',
    byId: (id: number) => `/lenders/${id}`,
  },

  // Loans
  loans: {
    base: '/loans',
    byId: (id: number) => `/loans/${id}`,
    disburse: (id: number) => `/loans/${id}/disburse`,
    repay: (id: number) => `/loans/${id}/repay`,
    repaySchedule: (id: number) => `/loans/${id}/repay-schedule`,
    schedule: (id: number) => `/loans/${id}/schedule`,
  },

  // Notifications
  notifications: {
    base: '/notifications',
    unread: '/notifications/unread',
    markRead: (id: number) => `/notifications/${id}/read`,
    markAllRead: '/notifications/read-all',
    send: '/notification', // Admin only - send notification to user
  },

  // Push Notifications (Web Push)
  push: {
    vapidKey: '/push/vapid-key',
    subscribe: '/push/subscribe',
    unsubscribe: '/push/unsubscribe',
  },

  // Email
  email: {
    history: '/email/history',
    senders: '/email/senders',
    settleHistory: (id: number) => `/email/history/${id}/settle`,
    uploadSettlement: (id: number) => `/email/history/${id}/upload-settlement`,
    send: '/email/send',
  },

  // Settings (Admin only)
  settings: {
    base: '/settings',
    byId: (id: number) => `/settings/${id}`,
    byKey: (key: string) => `/settings/key/${key}`,
  },

  // Zalo ZNS connection management (Admin only)
  zalo: {
    status: '/admin/zalo',
    credentials: '/admin/zalo/credentials',
    oauthStart: '/admin/zalo/oauth/start',
    oauthCallback: '/admin/zalo/oauth/callback',
    enabled: '/admin/zalo/enabled',
    refresh: '/admin/zalo/refresh',
    test: '/admin/zalo/test',
  },

  // Health Check
  health: {
    base: '../healthz',
  },

  // System Health / API Metrics
  systemHealth: {
    apiSummary: '/metrics/api',
    errorsByUser: '/metrics/errors/by-user',
    latencyTrend: '/metrics/latency/trend',
    slowestEndpoints: '/metrics/latency/slowest',
    recentErrors: '/metrics/errors/recent',
    errorCount: '/metrics/errors/count',
    eventBus: '/metrics/event-bus',
    cache: '/metrics/cache',
    endpointTrend: '/metrics/latency/endpoint-trend',
    topEndpoints: '/metrics/top-endpoints',
    failedLogins: '/metrics/failed-logins',
    failedLoginsByIdentifier: (identifier: string) => `/metrics/failed-logins/${encodeURIComponent(identifier)}`,
    browserPlatformStats: '/metrics/browser-platform-stats',
    browserPlatformUsers: '/metrics/browser-platform-stats/users',
    osStats: '/metrics/os-stats',
    osUsers: '/metrics/os-stats/users',
    browserStats: '/metrics/browser-stats',
    browserUsers: '/metrics/browser-stats/users',
  },

  // Cron Health
  cronHealth: {
    jobs: '/cron-jobs',
    toggle: (name: string) => `/cron-jobs/${name}/toggle`,
  },

  // Audit & Security
  audit: {
    base: '/audit',
    summary: '/audit/summary',
    logs: '/audit/logs',
    logById: (id: number) => `/audit/logs/${id}`,
    userActivity: (userId: number) => `/audit/users/${userId}/activity`,
    export: '/audit/export',
    blacklistedTokens: '/audit/blacklisted-tokens',
    revokeUserTokens: (userId: number) => `/audit/users/${userId}/revoke-tokens`,
    securityReport: '/audit/security-report',
    loginHistory: '/audit/login-history',
    trackEvent: '/audit/track-event',
    complianceReport: '/audit/compliance-report',
    dataLineage: (entityType: string, entityId: number) => `/audit/data-lineage/${entityType}/${entityId}`,
    integrityCheck: '/audit/integrity-check',
    dataRetentionCleanup: '/audit/data-retention-cleanup',
  },

  // Dashboard
  dashboard: {
    base: '/dashboard',
    summary: '/dashboard/summary',
    financialOverview: '/dashboard/financial-overview',
    financial: '/dashboard/financial',
    recentActivities: '/dashboard/recent-activities',
    notifications: '/dashboard/notifications',
    newEmployees: '/dashboard/new-employees',
    salaryDistribution: '/dashboard/salary-distribution',
    historical: '/dashboard/historical',
    monthlyFinancials: '/dashboard/monthly-financials',
    employeeActivity: '/dashboard/employee-activity',
    employeeActivityUsers: '/dashboard/employee-activity/users',
    projectProfitability: '/dashboard/project-profitability',
    projectWeeklyProfit: '/dashboard/project-weekly-profit',
    topPaidEmployees: '/dashboard/top-paid-employees',
    partnerDashboard: '/dashboard/partner',
    bankUsage: '/dashboard/bank-usage',
    bankUsageProjects: '/dashboard/bank-usage/projects',
    partnerEmployeeList: '/dashboard/partner/employees',
    checkInHealth: '/dashboard/check-in-health',
    quotaAnomalies: '/dashboard/quota-anomalies',
  },

  // Admin Attendances
  adminAttendances: {
    failedAttempts: '/admin/attendances/failed-attempts',
  },

  // Employee Self-Service Portal
  employee: {
    profile: '/me',
    updateProfile: '/me',
    updatePassword: '/me/password',
    timesheets: '/me/timesheet',
    summary: '/me/summary',
    // Advance Payment endpoints for employee
    advancePayment: '/me/advance-payment',
    advancePaymentRequest: '/me/advance-payment/request',
    advancePaymentHistory: '/me/advance-payment/history',
    advancePaymentCalculateFee: '/me/advance-payment/calculate-fee',
    advancePaymentCancelRequest: (id: number) => `/me/advance-payment/request/${id}/cancel`,
    // Self-check-in advance flow — dedicated path for check-in-enabled employees
    checkInAdvance: '/me/check-in-advance',
    checkInAdvanceRequest: '/me/check-in-advance/request',
  },

  // Advance Payments (Admin)
  advancePayments: {
    base: '/advance-payments',
    summary: '/advance-payments/summary',
    export: '/advance-payments/export',
    uploadResult: '/advance-payments/upload-result',
    import: '/advance-payments/import',
    importStatus: (id: number) => `/advance-payments/import/${id}`,
    importEmployeeList: '/advance-payments/import-employee-list',
    files: '/advance-payments/files',
    fileDownload: (id: number) => `/advance-payments/files/${id}/download`,
    downloadTemplate: '/advance-payments/template',
    transferHistoryDownload: (id: number) => `/advance-payments/transfer-histories/${id}/download`,
    cancel: (id: number) => `/advance-payments/${id}/cancel`,
    retryDisbursement: (id: number) => `/advance-payments/${id}/retry-disbursement`,
    disbursementStatus: (id: number) => `/advance-payments/${id}/disbursement-status`,
    employees: '/advance-payments/employees',
    exportEmployees: '/advance-payments/employees/export',
    availableMonths: '/advance-payments/available-months',
    reconciliation: {
      sendEmail: '/advance-payments/reconciliation/send-email',
      settle: '/advance-payments/reconciliation/settle',
      export: '/advance-payments/reconciliation/export',
    },
  },

  // Admin: Advance Payment Fee Schedule (Cấu hình phí ứng lương)
  advancePaymentFees: {
    base: '/admin/advance-payment-fees',
    byId: (id: string) => `/admin/advance-payment-fees/${id}`,
  },

  // Admin: Disbursement Fee Schedule (Cấu hình phí giao dịch chi hộ)
  disbursementFees: {
    base: '/admin/disbursement-fees',
    byId: (id: string) => `/admin/disbursement-fees/${id}`,
  },

  // Admin: Manual Disbursement (Chuyển tiền)
  manualDisbursement: {
    base: '/admin/manual-disbursement',
    byTxnId: (txnId: string) => `/admin/manual-disbursement/${txnId}`,
    checkAccount: '/admin/manual-disbursement/check-account',
    reconciliationDownload: '/admin/manual-disbursement/reconciliation/download',
  },

  // Lenders

  // Adv Partner
  advPartner: {
    updateUser: (id: number) => `/adv-partner/users/${id}`,
  },
};

// Rate limit configuration
export const RATE_LIMITS = {
  login: {
    maxAttempts: 5,
    windowMs: 15 * 60 * 1000, // 15 minutes
  },
  passwordChange: {
    maxAttempts: 3,
    windowMs: 60 * 60 * 1000, // 1 hour
  },
  logout: {
    maxRequests: 10,
    windowMs: 60 * 1000, // 1 minute
  },
};

// File upload limits
export const FILE_LIMITS = {
  // Excel import files - XLS format prioritized
  excel: {
    maxSize: 10 * 1024 * 1024, // 10MB
    allowedFormats: ['.xls', '.xlsx'],
  },
  // Evidence files (images, documents)
  evidence: {
    maxSize: 10 * 1024 * 1024, // 10MB
    allowedFormats: [
      // Images
      '.jpg', '.jpeg', '.png', '.gif',
      // Documents
      '.pdf', '.doc', '.docx', '.xls', '.xlsx', '.txt'
    ],
    allowedTypes: [
      // Images
      'image/jpeg', 'image/jpg', 'image/png', 'image/gif',
      // Documents
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'text/plain'
    ],
  },
  // Email attachments
  emailAttachment: {
    maxFiles: 5,
    maxSize: 10 * 1024 * 1024, // 10MB per file
    allowedFormats: [
      '.pdf', '.xlsx', '.xls', '.doc', '.docx',
      '.png', '.jpg', '.jpeg', '.gif', '.txt', '.csv'
    ],
    allowedTypes: [
      // Images
      'image/jpeg', 'image/jpg', 'image/png', 'image/gif',
      // Documents
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      // Text/CSV
      'text/plain',
      'text/csv',
      'application/csv'
    ],
  },
};

// Pagination defaults
export const PAGINATION_DEFAULTS = {
  pageSize: 20,
  maxPageSize: 100,
};

// Retry configuration
export const RETRY_CONFIG = {
  maxRetries: 3,
  baseDelay: 1000, // 1 second
  maxDelay: 10000, // 10 seconds
  backoffFactor: 2,
  retryableStatuses: [408, 500, 502, 503, 504],
  retryableErrors: ['ECONNABORTED', 'ECONNRESET', 'ECONNREFUSED', 'ENETDOWN', 'ENETUNREACH', 'EHOSTDOWN', 'EHOSTUNREACH', 'EPIPE'],
};
