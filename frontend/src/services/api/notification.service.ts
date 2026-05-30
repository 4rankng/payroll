import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Notification,
  NotificationFilters,
  PaginatedNotificationResponse,
  SendNotificationRequest,
} from '@/types/api/notification.types';

class NotificationService {
  /**
   * Get user notifications with pagination
   */
  async getNotifications(filters?: NotificationFilters): Promise<PaginatedNotificationResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<PaginatedNotificationResponse>(
      `${API_ENDPOINTS.notifications.base}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get unread notifications with count
   */
  async getUnreadNotifications(): Promise<{ notifications: Notification[]; count: number }> {
    const response = await apiClient.get<Notification[] | { notifications: Notification[]; count: number }>(
      API_ENDPOINTS.notifications.unread
    );

    const payload = response.data;
    if (Array.isArray(payload)) {
      return { notifications: payload, count: payload.length };
    }
    if (payload && 'notifications' in payload) {
      return { notifications: payload.notifications, count: payload.count };
    }
    return { notifications: [], count: 0 };
  }

  /**
   * Get unread notification count
   */
  async getUnreadCount(): Promise<number> {
    const { count } = await this.getUnreadNotifications();
    return count;
  }

  /**
   * Mark notification as read
   */
  async markAsRead(id: number): Promise<void> {
    await apiClient.put(API_ENDPOINTS.notifications.markRead(id));
  }

  /**
   * Mark all notifications as read
   */
  async markAllAsRead(): Promise<void> {
    await apiClient.put(API_ENDPOINTS.notifications.markAllRead);
  }

  /**
   * Send notification to user(s) (Admin only)
   */
  async sendNotification(data: SendNotificationRequest): Promise<ApiResponse<Notification>> {
    const response = await apiClient.post<Notification>(
      API_ENDPOINTS.notifications.send,
      data
    );
    return response;
  }
}

export const notificationService = new NotificationService();
