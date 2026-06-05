import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { auditService } from '@/services/api/audit.service';
import type {
  AuditSummaryParams,
  AuditLogsParams,
  UserActivityParams,
  AuditExportParams,
  BlacklistedTokensParams,
  SecurityReportParams,
  LoginHistoryParams,
  RevokeTokensRequest,
  TrackEventRequest,
  ComplianceReportRequest,
  IntegrityCheckRequest,
  EntityType
} from '@/types/api/audit.types';

// ========== AUDIT SUMMARY ==========

export function useAuditSummary(params: AuditSummaryParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getSummary>>['data']>({
    queryKey: ['audit', 'summary', params] as const,
    queryFn: async () => (await auditService.getSummary(params)).data,
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

// ========== AUDIT LOGS ==========

export function useAuditLogs(params: AuditLogsParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getLogs>>['data']>({
    queryKey: ['audit', 'logs', params] as const,
    queryFn: async () => (await auditService.getLogs(params)).data,
    placeholderData: (prev) => prev,
  });
}

export function useAuditLogDetails(id: number, enabled = true) {
  return useQuery<Awaited<ReturnType<typeof auditService.getLogDetails>>['data']>({
    queryKey: ['audit', 'logs', id] as const,
    queryFn: async () => (await auditService.getLogDetails(id)).data,
    enabled,
  });
}

// ========== USER ACTIVITY ==========

export function useUserActivity(userId: number, params: UserActivityParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getUserActivity>>['data']>({
    queryKey: ['audit', 'users', userId, 'activity', params] as const,
    queryFn: async () => (await auditService.getUserActivity(userId, params)).data,
    placeholderData: (prev) => prev,
  });
}

// ========== BLACKLISTED TOKENS ==========

export function useBlacklistedTokens(params: BlacklistedTokensParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getBlacklistedTokens>>['data']>({
    queryKey: ['audit', 'blacklisted-tokens', params] as const,
    queryFn: async () => (await auditService.getBlacklistedTokens(params)).data,
    placeholderData: (prev) => prev,
  });
}

// ========== SECURITY REPORT ==========

export function useSecurityReport(params: SecurityReportParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getSecurityReport>>['data']>({
    queryKey: ['audit', 'security-report', params] as const,
    queryFn: async () => (await auditService.getSecurityReport(params)).data,
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}

// ========== LOGIN HISTORY ==========

export function useLoginHistory(params: LoginHistoryParams = {}) {
  return useQuery<Awaited<ReturnType<typeof auditService.getLoginHistory>>['data']>({
    queryKey: ['audit', 'login-history', params] as const,
    queryFn: async () => (await auditService.getLoginHistory(params)).data,
    placeholderData: (prev) => prev,
  });
}

// ========== DATA LINEAGE ==========

export function useDataLineage(
  entityType: EntityType, 
  entityId: number, 
  enabled = true
) {
  return useQuery({
    queryKey: ['audit', 'data-lineage', entityType, entityId],
    queryFn: async () => (await auditService.getDataLineage(entityType, entityId)).data,
    enabled,
    gcTime: 60 * 60 * 1000, // 1 hour
  });
}

// ========== MUTATIONS ==========

export function useRevokeUserTokens() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ userId, data }: { userId: number; data: RevokeTokensRequest }) =>
      auditService.revokeUserTokens(userId, data),
    onSuccess: () => {
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: ['audit', 'blacklisted-tokens'] });
      queryClient.invalidateQueries({ queryKey: ['audit', 'security-report'] });
    },
  });
}

export function useTrackEvent() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: TrackEventRequest) => auditService.trackEvent(data),
    onSuccess: () => {
      // Invalidate audit logs to show new event
      queryClient.invalidateQueries({ queryKey: ['audit', 'logs'] });
      queryClient.invalidateQueries({ queryKey: ['audit', 'summary'] });
    },
  });
}

export function useGenerateComplianceReport() {
  return useMutation({
    mutationFn: (data: ComplianceReportRequest) =>
      auditService.generateComplianceReport(data),
  });
}

export function usePerformIntegrityCheck() {
  return useMutation({
    mutationFn: (data: IntegrityCheckRequest) =>
      auditService.performIntegrityCheck(data),
  });
}

export function useExportAuditLogs() {
  return useMutation({
    mutationFn: (params: AuditExportParams) => auditService.exportLogs(params),
  });
}

export function useDataRetentionCleanup() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      cleanup_type: string;
      retention_days: number;
      dry_run: boolean;
      categories: string[];
    }) => auditService.performDataRetentionCleanup(data),
    onSuccess: () => {
      // Invalidate audit queries if not dry run
      queryClient.invalidateQueries({ queryKey: ['audit'] });
    },
  });
}

// ========== CONVENIENCE HOOKS ==========

/**
 * Hook for real-time audit monitoring
 * Automatically refetches data at regular intervals
 */
export function useAuditMonitoring(interval = 30000) { // 30 seconds
  const summaryQuery = useQuery({
    queryKey: ['audit', 'summary', 'monitoring'],
    queryFn: () => auditService.getSummary(),
    refetchInterval: interval,
    refetchIntervalInBackground: true,
  });

  const securityQuery = useQuery({
    queryKey: ['audit', 'security-report', 'monitoring'],
    queryFn: () => auditService.getSecurityReport(),
    refetchInterval: interval * 2, // Less frequent for security reports
    refetchIntervalInBackground: true,
  });

  return {
    summary: summaryQuery,
    security: securityQuery,
    isMonitoring: !summaryQuery.isError && !securityQuery.isError,
  };
}

/**
 * Hook for paginated audit logs with search
 */
export function usePaginatedAuditLogs(
  initialParams: AuditLogsParams = {},
  pageSize = 20
) {
  const [params, setParams] = useState({ 
    pageSize, 
    page: 1, 
    ...initialParams 
  });

  const query = useAuditLogs(params);

  const goToPage = (page: number) => {
    setParams(prev => ({ ...prev, page }));
  };

  const updateFilters = (newFilters: Partial<AuditLogsParams>) => {
    setParams(prev => ({ ...prev, ...newFilters, page: 1 }));
  };

  return {
    ...query,
    params,
    goToPage,
    updateFilters,
    currentPage: params.page,
    pageSize: params.pageSize,
  };
}

/**
 * Hook for user activity tracking
 */
export function useCurrentUserActivity() {
  const [userId, setUserId] = useState<number | null>(null);

  // Get current user ID (you might need to implement this based on your auth system)
  useEffect(() => {
    // This would typically come from your auth context
    const currentUser = JSON.parse(localStorage.getItem('currentUser') || 'null');
    if (currentUser?.id) {
      setUserId(currentUser.id);
    }
  }, []);

  return useUserActivity(userId!, { page: 1, pageSize: 10 });
  // enabled is handled by queryFn only running when userId is set
}