import type { ApiError } from '@/services/api/client';
import type { OnePayFeeReportIssue } from '@/types/api/financial.types';

export function extractOnePayFeeIssues(error: unknown): OnePayFeeReportIssue[] {
  const apiError = error as ApiError | undefined;
  const issues = apiError?.details?.issues;
  if (!Array.isArray(issues)) {
    return [];
  }
  return issues.filter((issue): issue is OnePayFeeReportIssue => (
    issue !== null &&
    typeof issue === 'object' &&
    typeof (issue as OnePayFeeReportIssue).message === 'string'
  ));
}
