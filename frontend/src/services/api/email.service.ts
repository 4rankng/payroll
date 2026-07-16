import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  EmailHistoryFilters,
  EmailHistoryResult,
  EmailHistoryPagination,
  EmailHistoryRecord,
  EmailSenderOption,
  SendEmailData,
  SendEmailResponse,
  UploadSettlementResponse,
} from '@/types/api/email.types';

class EmailService {
  async sendCustomEmail(data: SendEmailData): Promise<SendEmailResponse> {
    // Always use multipart so attachments and no-attachment messages share the
    // same backend contract.
    const formData = new FormData();

    // Add optional from field
    if (data.from) {
      formData.append('from', data.from);
    }

    // Add recipients (can be multiple)
    data.recipients.forEach(recipient => {
      formData.append('recipients', recipient);
    });

    // Add cc if provided
    if (data.cc && data.cc.length > 0) {
      data.cc.forEach(cc => {
        formData.append('cc', cc);
      });
    }

    // Add bcc if provided
    if (data.bcc && data.bcc.length > 0) {
      data.bcc.forEach(bcc => {
        formData.append('bcc', bcc);
      });
    }

    // Add subject
    formData.append('subject', data.subject);

    // Add htmlBody if provided
    if (data.htmlBody) {
      formData.append('htmlBody', data.htmlBody);
    }

    // Add textBody if provided
    if (data.textBody) {
      formData.append('textBody', data.textBody);
    }

    // Add replyTo if provided
    if (data.replyTo) {
      formData.append('replyTo', data.replyTo);
    }

    // Add attachments
    (data.attachments ?? []).forEach(file => {
      formData.append('attachments', file);
    });

    const response = await apiClient.upload<{ message_id: string }>(API_ENDPOINTS.email.send, formData);
    return {
      success: response.status === 'success',
      message: response.message ?? '',
      messageId: response.data?.message_id,
    };
  }

  async getAvailableSenders(): Promise<EmailSenderOption[]> {
    const response = await apiClient.get<EmailSenderOption[]>(API_ENDPOINTS.email.senders);
    return response.data ?? [];
  }
  async settleFromEmailHistory(id: number): Promise<void> {
    await apiClient.post(API_ENDPOINTS.email.settleHistory(id), {});
  }

  async uploadSettlement(id: number, file: File): Promise<UploadSettlementResponse> {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.upload(API_ENDPOINTS.email.uploadSettlement(id), formData) as unknown as Promise<UploadSettlementResponse>;
  }

  async uploadStandaloneSettlement(file: File): Promise<unknown> {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.upload(API_ENDPOINTS.timesheets.uploadSettlementResult, formData);
  }

  async getEmailHistory(filters?: EmailHistoryFilters): Promise<EmailHistoryResult> {
    const queryString = filters
      ? buildQueryString({
          page: filters.page,
          pageSize: filters.pageSize,
        })
      : '';

    const response = await apiClient.get<EmailHistoryRecord[]>(
      `${API_ENDPOINTS.email.history}${queryString}`
    );

    const pagination: EmailHistoryPagination = response.pagination ?? {
      page: filters?.page ?? 1,
      pageSize: filters?.pageSize ?? (response.data?.length ?? 0),
      totalPages: response.data ? 1 : 0,
      totalRecords: response.data?.length ?? 0,
    };

    const data = response.data ?? [];

    return {
      data,
      pagination,
      message: response.message,
    };
  }
}

export const emailService = new EmailService();
