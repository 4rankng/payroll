import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  AuditSummaryResponse,
  AuditSummaryParams,
  AuditLogsResponse,
  AuditLogsParams,
  AuditLogDetailsResponse,
  UserActivityResponse,
  UserActivityParams,
  AuditExportResponse,
  AuditExportParams,
  BlacklistedTokensResponse,
  BlacklistedTokensParams,
  RevokeTokensResponse,
  RevokeTokensRequest,
  SecurityReportResponse,
  SecurityReportParams,
  LoginHistoryResponse,
  LoginHistoryParams,
  TrackEventResponse,
  TrackEventRequest,
  ComplianceReportResponse,
  ComplianceReportRequest,
  DataLineageResponse,
  IntegrityCheckResponse,
  IntegrityCheckRequest,
  EntityType
} from '@/types/api/audit.types';

/**
 * Audit & Security API Service
 * Handles audit trails, security monitoring, compliance reporting, and token management
 */
export class AuditService {

  /**
   * Get audit summary statistics
   */
  async getSummary(params: AuditSummaryParams = {}): Promise<AuditSummaryResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<AuditSummaryResponse>(`${API_ENDPOINTS.audit.summary}${queryString}`);
  }

  /**
   * Get paginated audit logs
   */
  async getLogs(params: AuditLogsParams = {}): Promise<AuditLogsResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<AuditLogsResponse>(`${API_ENDPOINTS.audit.logs}${queryString}`);
  }

  /**
   * Get detailed audit log entry
   */
  async getLogDetails(id: number): Promise<AuditLogDetailsResponse> {
    return apiClient.get<AuditLogDetailsResponse>(API_ENDPOINTS.audit.logById(id));
  }

  /**
   * Get user activity history
   */
  async getUserActivity(
    userId: number, 
    params: UserActivityParams = {}
  ): Promise<UserActivityResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<UserActivityResponse>(
      `${API_ENDPOINTS.audit.userActivity(userId)}${queryString}`
    );
  }

  /**
   * Export audit logs
   */
  async exportLogs(params: AuditExportParams = {}): Promise<AuditExportResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<AuditExportResponse>(`${API_ENDPOINTS.audit.export}${queryString}`);
  }

  /**
   * Get blacklisted tokens
   */
  async getBlacklistedTokens(
    params: BlacklistedTokensParams = {}
  ): Promise<BlacklistedTokensResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<BlacklistedTokensResponse>(
      `${API_ENDPOINTS.audit.blacklistedTokens}${queryString}`
    );
  }

  /**
   * Revoke user tokens
   */
  async revokeUserTokens(
    userId: number, 
    data: RevokeTokensRequest
  ): Promise<RevokeTokensResponse> {
    return apiClient.post<RevokeTokensResponse>(
      API_ENDPOINTS.audit.revokeUserTokens(userId),
      data
    );
  }

  /**
   * Get security report
   */
  async getSecurityReport(
    params: SecurityReportParams = {}
  ): Promise<SecurityReportResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<SecurityReportResponse>(
      `${API_ENDPOINTS.audit.securityReport}${queryString}`
    );
  }

  /**
   * Get login history
   */
  async getLoginHistory(
    params: LoginHistoryParams = {}
  ): Promise<LoginHistoryResponse> {
    const queryString = buildQueryString(params);
    return apiClient.get<LoginHistoryResponse>(
      `${API_ENDPOINTS.audit.loginHistory}${queryString}`
    );
  }

  /**
   * Track custom event
   */
  async trackEvent(data: TrackEventRequest): Promise<TrackEventResponse> {
    return apiClient.post<TrackEventResponse>(API_ENDPOINTS.audit.trackEvent, data);
  }

  /**
   * Generate compliance report
   */
  async generateComplianceReport(
    data: ComplianceReportRequest
  ): Promise<ComplianceReportResponse> {
    return apiClient.get<ComplianceReportResponse>(
      `${API_ENDPOINTS.audit.complianceReport}${buildQueryString(data)}`
    );
  }

  /**
   * Get data lineage for entity
   */
  async getDataLineage(
    entityType: EntityType, 
    entityId: number
  ): Promise<DataLineageResponse> {
    return apiClient.get<DataLineageResponse>(
      API_ENDPOINTS.audit.dataLineage(entityType, entityId)
    );
  }

  /**
   * Perform audit log integrity check
   */
  async performIntegrityCheck(
    data: IntegrityCheckRequest
  ): Promise<IntegrityCheckResponse> {
    return apiClient.post<IntegrityCheckResponse>(
      API_ENDPOINTS.audit.integrityCheck,
      data
    );
  }

  /**
   * Perform data retention cleanup (dry run or execute)
   */
  async performDataRetentionCleanup(data: {
    cleanup_type: string;
    retention_days: number;
    dry_run: boolean;
    categories: string[];
  }): Promise<{
    data: {
      cleanup_id: string;
      mode: string;
      retention_cutoff_date: string;
      records_to_cleanup: Record<string, number>;
      estimated_space_freed: string;
      compliance_impact: string;
      warnings: string[];
      next_steps: string[];
      scheduled_at: string;
    };
  }> {
    return apiClient.post(API_ENDPOINTS.audit.dataRetentionCleanup, data);
  }

  /**
   * Download audit export file
   */
  async downloadExport(downloadUrl: string, filename?: string): Promise<void> {
    return apiClient.download(downloadUrl, filename);
  }
}

// Export singleton instance
export const auditService = new AuditService();