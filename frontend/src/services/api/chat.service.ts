import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

export interface ChatConversation {
  id: number;
  user_id: number;
  role: string;
  title: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  id: number;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface ChatToolCall {
  id: number;
  message_id?: number | null;
  tool_name: string;
  arguments: string;
  outcome: string;
  result_summary: string;
  duration_ms: number;
}

export interface ChatConversationDetail {
  conversation: ChatConversation;
  messages: ChatMessage[];
  tool_calls: ChatToolCall[];
  total_messages: number;
}

interface ChatEnvelope<T> {
  status: string;
  data: T;
  message?: string;
}


// The repo's apiClient resolves to the response body, and some endpoints wrap
// payloads in a {status, data} envelope while others return them bare —
// normalize once here so callers never guess the shape.
function unwrap<T>(body: unknown): T {
  if (body && typeof body === 'object' && 'data' in (body as Record<string, unknown>)) {
    return ((body as Record<string, unknown>).data as T) ?? (body as T);
  }
  return body as T;
}

class ChatService {
  async createConversation(title = ''): Promise<ChatConversation> {
    const response = await apiClient.post<ChatEnvelope<ChatConversation>>(
      API_ENDPOINTS.chat.conversations,
      title ? { title } : {}
    );
    return unwrap<ChatConversation>(response.data);
  }

  async listConversations(page = 1, pageSize = 20): Promise<ChatConversation[]> {
    const response = await apiClient.get<ChatEnvelope<ChatConversation[]>>(
      `${API_ENDPOINTS.chat.conversations}${buildQueryString({ page, pageSize })}`
    );
    const raw = unwrap<ChatConversation[]>(response.data);
    return Array.isArray(raw) ? raw : [];
  }

  async getConversation(id: number, messagePage = 1, messagePageSize = 50): Promise<ChatConversationDetail> {
    const response = await apiClient.get<ChatEnvelope<ChatConversationDetail>>(
      `${API_ENDPOINTS.chat.conversations}/${id}${buildQueryString({ messagePage, messagePageSize })}`
    );
    return unwrap<ChatConversationDetail>(response.data);
  }

  async uploadAttachment(file: File): Promise<number> {
    const form = new FormData();
    form.append('file', file);
    const response = await apiClient.post<ChatEnvelope<{ id: number }>>(
      API_ENDPOINTS.chat.attachments,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return unwrap<{ id: number }>(response.data).id;
  }

  async sendMessage(id: number, content: string, attachmentIds: number[] = []): Promise<ChatConversationDetail> {
    const response = await apiClient.post<ChatEnvelope<ChatConversationDetail>>(
      `${API_ENDPOINTS.chat.conversations}/${id}/messages`,
      { content, attachmentIds }
    );
    return unwrap<ChatConversationDetail>(response.data);
  }
}

export const chatService = new ChatService();
