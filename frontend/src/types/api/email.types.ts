export interface SendEmailData {
  from?: string;
  recipients: string[];
  cc?: string[];
  bcc?: string[];
  subject: string;
  htmlBody?: string;
  textBody?: string;
  replyTo?: string;
  attachments?: File[];
}

export interface SendEmailResponse {
  success: boolean;
  message: string;
  messageId?: string;
}

export interface EmailHistoryFilters {
  page: number;
  pageSize: number;
}

export interface EmailHistoryPagination {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface EmailHistoryRecord {
  id: number;
  type: string;
  channel: string;
  sender: string;
  senderId?: number;
  senderName?: string;
  senderUsername?: string;
  recipients: Array<{ name?: string; address: string }>;
  subject: string;
  body?: string;
  sentAt: string;
  settledAt?: string | null;
  status: 'SENT' | 'FAILED';
  payrollMeta?: {
    totalAmount: number;
    feePercentage: number;
    feeAmount: number;
    totalWithFee: number;
    reportAtDate: string;
    timesheetIds: number[];
    saoKeAssetId?: number | null;
    cc?: string[];
    bcc?: string[];
  } | null;
}

export interface EmailHistoryResult {
  data: EmailHistoryRecord[];
  pagination: EmailHistoryPagination;
  message?: string;
}

export interface UploadSettlementResponse {
  processedTimesheets: number;
  skippedTimesheets: number;
  settlementsCreated: number;
  settlementAmount: number;
}