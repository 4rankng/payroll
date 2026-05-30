export interface ApprovalItem {
  id: string;
  type: 'timesheet' | 'payrate' | 'payroll' | 'employee';
  title: string;
  description: string;
  submittedBy: {
    name: string;
    role: string;
    avatar?: string;
  };
  submittedAt: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'pending' | 'approved' | 'rejected' | 'reviewing';
  details: Record<string, unknown>;
  attachments?: number;
  comments?: number;
  dueDate?: string;
}

export type ApprovalStatus = ApprovalItem['status'];
export type ApprovalPriority = ApprovalItem['priority'];
export type ApprovalType = ApprovalItem['type'];
