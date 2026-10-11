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

class ChatService {
  async createConversation(title = ''): Promise<ChatConversation> {
    const response = await apiClient.post<ChatEnvelope<ChatConversation>>(
      API_ENDPOINTS.chat.conversations,
      title ? { title } : {}
    );
    return response.data.data;
  }

  async listConversations(page = 1, pageSize = 20): Promise<ChatConversation[]> {
    const response = await apiClient.get<ChatEnvelope<ChatConversation[]>>(
      `${API_ENDPOINTS.chat.conversations}${buildQueryString({ page, pageSize })}`
    );
    const raw = response.data?.data;
    return Array.isArray(raw) ? raw : [];
  }

  async getConversation(id: number, messagePage = 1, messagePageSize = 50): Promise<ChatConversationDetail> {
    const response = await apiClient.get<ChatEnvelope<ChatConversationDetail>>(
      `${API_ENDPOINTS.chat.conversations}/${id}${buildQueryString({ messagePage, messagePageSize })}`
    );
    return response.data.data;
  }

  async uploadAttachment(file: File): Promise<number> {
    const form = new FormData();
    form.append('file', file);
    const response = await apiClient.post<ChatEnvelope<{ id: number }>>(
      API_ENDPOINTS.chat.attachments,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return response.data.data.id;
  }

  async sendMessage(id: number, content: string, attachmentIds: number[] = []): Promise<ChatConversationDetail> {
    const response = await apiClient.post<ChatEnvelope<ChatConversationDetail>>(
      `${API_ENDPOINTS.chat.conversations}/${id}/messages`,
      { content, attachmentIds }
    );
    return response.data.data;
  }
}

export const chatService = new ChatService();
