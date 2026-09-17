// Admin weekly-payment fee schedule API client (phí trả lương tuần).
// All endpoints require admin role; the backend enforces that via Casbin.

import { apiClient, type ApiResponse } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  WeeklyPaymentFeeScheduleListResponse,
  WeeklyPaymentFeeScheduleEntry,
  CreateWeeklyPaymentFeeScheduleRequest,
  UpdateWeeklyPaymentFeeScheduleRequest,
} from "@/types/api/weekly-payment-fee-schedule.types";

class WeeklyPaymentFeeScheduleService {
  list(): Promise<ApiResponse<WeeklyPaymentFeeScheduleListResponse>> {
    return apiClient.get<WeeklyPaymentFeeScheduleListResponse>(
      API_ENDPOINTS.weeklyPaymentFees.base,
    );
  }

  create(
    body: CreateWeeklyPaymentFeeScheduleRequest,
  ): Promise<ApiResponse<WeeklyPaymentFeeScheduleEntry>> {
    return apiClient.post<WeeklyPaymentFeeScheduleEntry>(
      API_ENDPOINTS.weeklyPaymentFees.base,
      body,
    );
  }

  update(
    id: string,
    body: UpdateWeeklyPaymentFeeScheduleRequest,
  ): Promise<ApiResponse<WeeklyPaymentFeeScheduleEntry>> {
    return apiClient.patch<WeeklyPaymentFeeScheduleEntry>(
      API_ENDPOINTS.weeklyPaymentFees.byId(id),
      body,
    );
  }

  delete(id: string): Promise<ApiResponse<{ id: string }>> {
    return apiClient.delete<{ id: string }>(
      API_ENDPOINTS.weeklyPaymentFees.byId(id),
    );
  }
}

export const weeklyPaymentFeeScheduleService =
  new WeeklyPaymentFeeScheduleService();
