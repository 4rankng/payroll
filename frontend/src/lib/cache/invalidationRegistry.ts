/**
 * Invalidation Registry
 * Centralized mapping of mutations to their required cache invalidations
 * Supports cross-entity invalidations and context-aware patterns
 */

import { QueryKeys } from '@/lib/queryKeys/index';

/**
 * Context interface for dynamic invalidation patterns
 */
export interface InvalidationContext {
  projectId?: number;
  employeeId?: number;
  timesheetId?: number;
  assignmentId?: number;
  userId?: number;
  assetId?: number;
  data?: { employee_id?: number; project_id?: number; [key: string]: unknown };
}

/**
 * Invalidation pattern types
 */
export type InvalidationPattern =
  | readonly unknown[] // Static query key
  | ((context: InvalidationContext) => readonly unknown[]) // Dynamic query key
  | ((context: InvalidationContext) => readonly unknown[][]) // Multiple dynamic keys
  | string; // Registry group reference

/**
 * Invalidation Groups
 * Reusable collections of related query keys
 */
export const InvalidationGroups = {
  // Project context - all queries related to a specific project
  PROJECT_CONTEXT: (projectId: number) => [
    QueryKeys.projects.detail(projectId),
    QueryKeys.projects.employees(projectId),
    QueryKeys.projects.timesheets(projectId),
    QueryKeys.projects.payrates(projectId),
    QueryKeys.projects.financial(projectId),
    QueryKeys.projects.assignments(projectId),
    QueryKeys.projects.stats(projectId),
    QueryKeys.assignments.byProject(projectId),
    QueryKeys.assignments.projectStats(projectId),
    QueryKeys.payrates.byProject(projectId),
  ],

  // Employee context - all queries related to a specific employee
  EMPLOYEE_CONTEXT: (employeeId: number) => [
    QueryKeys.employees.detail(employeeId),
    QueryKeys.employees.projects(employeeId),
    QueryKeys.employees.timesheets(employeeId),
    QueryKeys.employees.assignments(employeeId),
    QueryKeys.employees.stats(employeeId),
    QueryKeys.employees.payrates(employeeId),
    QueryKeys.assignments.byEmployee(employeeId),
    QueryKeys.assignments.employeeStats(employeeId),
    QueryKeys.payrates.byEmployee(employeeId),
    // Note: current-projects uses a predicate pattern, see InvalidationRegistry mutations
  ],

  // Global statistics and summaries
  GLOBAL_STATS: [
    QueryKeys.dashboard.all,
    QueryKeys.projects.summary(),
    QueryKeys.employees.summary(),
    QueryKeys.timesheets.summary(),
    QueryKeys.projects.partner.summary(),
  ],

  // Project lists and searches
  PROJECT_LISTS: [
    QueryKeys.projects.lists(),
    QueryKeys.projects.search(),
    QueryKeys.projects.assignable(),
    QueryKeys.projects.partner.all(),
  ],

  // Employee lists and searches
  EMPLOYEE_LISTS: [
    QueryKeys.employees.lists(),
    QueryKeys.employees.search(),
    QueryKeys.employees.missingBankDetails(),
  ],

  // Timesheet lists
  TIMESHEET_LISTS: [
    QueryKeys.timesheets.lists(),
    QueryKeys.timesheets.analytics(),
  ],

  // Assignment lists and searches
  ASSIGNMENT_LISTS: [
    QueryKeys.assignments.lists(),
    QueryKeys.assignments.search({}),
  ],
} as const;

/**
 * Main Invalidation Registry
 * Maps mutation types to their required invalidations
 */
export const InvalidationRegistry = {
  // ========== PROJECT MUTATIONS ==========
  'project:create': [
    'PROJECT_LISTS',
    'GLOBAL_STATS',
    QueryKeys.projects.partner.summary(), // Partner summary might change
  ],

  'project:update': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    'PROJECT_LISTS',
    'GLOBAL_STATS',
  ],

  'project:delete': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    'PROJECT_LISTS',
    'EMPLOYEE_LISTS', // Affects employee availability
    'TIMESHEET_LISTS', // Timesheets for this project are affected
    'ASSIGNMENT_LISTS',
    'GLOBAL_STATS',
  ],

  'project:start': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    'PROJECT_LISTS',
    'GLOBAL_STATS',
  ],

  'project:complete': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    'PROJECT_LISTS',
    'GLOBAL_STATS',
    'EMPLOYEE_LISTS', // Affects employee availability
  ],

  'project:cancel': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    'PROJECT_LISTS',
    'GLOBAL_STATS',
    'EMPLOYEE_LISTS', // Affects employee availability
  ],

  // ========== EMPLOYEE MUTATIONS ==========
  'employee:create': [
    'EMPLOYEE_LISTS',
    'GLOBAL_STATS',
  ],

  'employee:update': (context: InvalidationContext) => [
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
    'EMPLOYEE_LISTS',
    'GLOBAL_STATS',
  ],

  'employee:delete': (context: InvalidationContext) => [
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
    'EMPLOYEE_LISTS',
    'PROJECT_LISTS', // Projects this employee was assigned to
    'TIMESHEET_LISTS', // Employee's timesheets
    'ASSIGNMENT_LISTS',
    'GLOBAL_STATS',
  ],

  'employee:import': [
    'EMPLOYEE_LISTS',
    'GLOBAL_STATS',
  ],

  // ========== TIMESHEET MUTATIONS ==========
  'timesheet:create': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    // Add predicate-based invalidation for current-projects
    if (context.data?.employee_id) {
      patterns.push(
        (ctx: InvalidationContext) => [
          ['employees', ctx.data?.employee_id, 'current-projects']
        ] as readonly unknown[]
      );
    }

    return patterns;
  },

  'timesheet:update': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.timesheetId) {
      patterns.push(QueryKeys.timesheets.detail(context.timesheetId));
    }

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        QueryKeys.timesheets.employeeSummary(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    return patterns;
  },

  'timesheet:delete': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        QueryKeys.timesheets.employeeSummary(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    return patterns;
  },

  'timesheet:approve': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.timesheetId) {
      patterns.push(QueryKeys.timesheets.detail(context.timesheetId));
    }

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        QueryKeys.timesheets.employeeSummary(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    return patterns;
  },

  'timesheet:reject': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.timesheetId) {
      patterns.push(QueryKeys.timesheets.detail(context.timesheetId));
    }

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        QueryKeys.timesheets.employeeSummary(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    return patterns;
  },

  'timesheet:reset': (context: InvalidationContext) => {
    const patterns: InvalidationPattern[] = [
      'TIMESHEET_LISTS',
      'GLOBAL_STATS',
    ];

    if (context.timesheetId) {
      patterns.push(QueryKeys.timesheets.detail(context.timesheetId));
    }

    if (context.data?.employee_id) {
      patterns.push(
        QueryKeys.timesheets.byEmployee(context.data.employee_id),
        QueryKeys.timesheets.employeeSummary(context.data.employee_id),
        ...InvalidationGroups.EMPLOYEE_CONTEXT(context.data.employee_id)
      );
    }

    if (context.data?.project_id) {
      patterns.push(
        QueryKeys.timesheets.byProject(context.data.project_id),
        ...InvalidationGroups.PROJECT_CONTEXT(context.data.project_id)
      );
    }

    return patterns;
  },

  'timesheet:bulkApprove': [
    QueryKeys.timesheets.all, // Bulk operations invalidate all timesheet queries
    'GLOBAL_STATS',
  ],

  'timesheet:bulkReject': [
    QueryKeys.timesheets.all, // Bulk operations invalidate all timesheet queries
    'GLOBAL_STATS',
  ],

  'timesheet:bulkReset': [
    QueryKeys.timesheets.all, // Bulk reset affects all timesheet queries
    'GLOBAL_STATS',
  ],

  'timesheet:import': [
    'TIMESHEET_LISTS',
    'PROJECT_LISTS', // Projects might be affected
    'EMPLOYEE_LISTS', // Employee stats might change
    'GLOBAL_STATS',
  ],

  // ========== ASSIGNMENT MUTATIONS ==========
  'assignment:create': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
    'ASSIGNMENT_LISTS',
    'EMPLOYEE_LISTS', // Employee assignment status changes
    'GLOBAL_STATS',
  ],

  'assignment:update': (context: InvalidationContext) => [
    QueryKeys.assignments.detail(context.assignmentId!),
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
    'ASSIGNMENT_LISTS',
  ],

  'assignment:remove': (context: InvalidationContext) => [
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
    'ASSIGNMENT_LISTS',
    'EMPLOYEE_LISTS', // Employee assignment status changes
    'GLOBAL_STATS',
  ],

  // ========== PAY RATE MUTATIONS ==========
  'payrate:create': (context: InvalidationContext) => [
    QueryKeys.payrates.lists(),
    QueryKeys.payrates.byProject(context.projectId!),
    QueryKeys.payrates.byEmployee(context.employeeId!),
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
  ],

  'payrate:update': (context: InvalidationContext) => [
    QueryKeys.payrates.detail(context.data?.id as number),
    QueryKeys.payrates.lists(),
    QueryKeys.payrates.byProject(context.projectId!),
    QueryKeys.payrates.byEmployee(context.employeeId!),
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
  ],

  'payrate:delete': (context: InvalidationContext) => [
    QueryKeys.payrates.lists(),
    QueryKeys.payrates.byProject(context.projectId!),
    QueryKeys.payrates.byEmployee(context.employeeId!),
    ...InvalidationGroups.PROJECT_CONTEXT(context.projectId!),
    ...InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId!),
  ],

  // ========== USER MUTATIONS ==========
  'user:create': [
    QueryKeys.users.lists(),
    'GLOBAL_STATS',
  ],

  'user:update': (context: InvalidationContext) => [
    QueryKeys.users.detail(context.userId!),
    QueryKeys.users.lists(),
    QueryKeys.auth.user(), // Current user might be updated
  ],

  'user:delete': (context: InvalidationContext) => [
    QueryKeys.users.detail(context.userId!),
    QueryKeys.users.lists(),
    'GLOBAL_STATS',
  ],

  // ========== ASSET MUTATIONS ==========
  'asset:create': (context: InvalidationContext) => [
    QueryKeys.assets.lists(),
    QueryKeys.assets.byReference(context.data?.reference_type as string, context.data?.reference_id as number),
  ],

  'asset:update': (context: InvalidationContext) => [
    QueryKeys.assets.detail(context.assetId!),
    QueryKeys.assets.lists(),
    QueryKeys.assets.byReference(context.data?.reference_type as string, context.data?.reference_id as number),
  ],

  'asset:delete': (context: InvalidationContext) => [
    QueryKeys.assets.lists(),
    QueryKeys.assets.byReference(context.data?.reference_type as string, context.data?.reference_id as number),
  ],

  // ========== BANK MUTATIONS ==========
  'bank:create': [
    QueryKeys.banks.lists(),
    QueryKeys.banks.search(),
  ],

  'bank:update': (context: InvalidationContext) => [
    QueryKeys.banks.lists(),
    QueryKeys.banks.search(),
  ],

  'bank:delete': [
    QueryKeys.banks.lists(),
    QueryKeys.banks.search(),
  ],

  // ========== AUTH MUTATIONS ==========
  'auth:login': [
    QueryKeys.auth.user(),
    QueryKeys.auth.permissions(),
    QueryKeys.notifications.unread(),
  ],

  'auth:logout': [
    QueryKeys.auth.all,
  ],

  'auth:refresh': [
    QueryKeys.auth.user(),
    QueryKeys.auth.permissions(),
  ],
} as const;

/**
 * Get invalidation patterns for a mutation type
 */
export function getInvalidationPatterns(
  mutationType: keyof typeof InvalidationRegistry,
  context: InvalidationContext = {}
): InvalidationPattern[] {
  const patterns = InvalidationRegistry[mutationType];

  if (typeof patterns === 'function') {
    return (patterns as (ctx: InvalidationContext) => InvalidationPattern[])(context);
  }

  return patterns as unknown as InvalidationPattern[];
}

/**
 * Resolve group references to actual query keys
 */
export function resolveGroupReference(groupName: string, context: InvalidationContext = {}): readonly unknown[] {
  switch (groupName) {
    case 'PROJECT_CONTEXT':
      return context.projectId ? InvalidationGroups.PROJECT_CONTEXT(context.projectId) : [];
    case 'EMPLOYEE_CONTEXT':
      return context.employeeId ? InvalidationGroups.EMPLOYEE_CONTEXT(context.employeeId) : [];
    case 'GLOBAL_STATS':
      return InvalidationGroups.GLOBAL_STATS;
    case 'PROJECT_LISTS':
      return InvalidationGroups.PROJECT_LISTS;
    case 'EMPLOYEE_LISTS':
      return InvalidationGroups.EMPLOYEE_LISTS;
    case 'TIMESHEET_LISTS':
      return InvalidationGroups.TIMESHEET_LISTS;
    case 'ASSIGNMENT_LISTS':
      return InvalidationGroups.ASSIGNMENT_LISTS;
    default:
      console.warn(`Unknown invalidation group: ${groupName}`);
      return [];
  }
}
