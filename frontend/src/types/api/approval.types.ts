/** Approval-request engine types — mirror backend handlers/approval DTOs. */

export type ApprovalStatus =
  | 'pending'
  | 'approved'
  | 'executed'
  | 'rejected'
  | 'rescheduled'
  | 'escalated'
  | 'expired'
  | 'cancelled';

export type ApprovalType = 'send_sao_ke_email' | 'send_payroll_email' | 'send_reminder' | 'void_timesheet';

export interface ApprovalRequest {
  id: number;
  type: ApprovalType;
  type_label: string;
  status: ApprovalStatus;
  status_label: string;
  payload?: Record<string, unknown>;
  decision_note?: string | null;
  outcome?: string | null;
  retry_count: number;
  scheduled_at?: string | null;
  requester_id: number;
  created_at: string;
}

export interface ApprovalListFilters {
  statuses?: ApprovalStatus[];
  page?: number;
  pageSize?: number;
}

export interface ApprovalPagination {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface ApprovalListResult {
  items: ApprovalRequest[];
  pagination: ApprovalPagination | null;
}

export interface FileApprovalData {
  type: ApprovalType;
  params: Record<string, unknown>;
}
