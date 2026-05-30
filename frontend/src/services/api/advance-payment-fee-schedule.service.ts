// Admin advance-payment fee schedule API client.
// All endpoints require admin role; the backend enforces that via Casbin.

import { apiClient, type ApiResponse } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  FeeScheduleListResponse,
  FeeScheduleEntry,
  CreateFeeScheduleRequest,
  UpdateFeeScheduleRequest,
} from "@/types/api/advance-payment-fee-schedule.types";

class AdvancePaymentFeeScheduleService {
  list(): Promise<ApiResponse<FeeScheduleListResponse>> {
    return apiClient.get<FeeScheduleListResponse>(
      API_ENDPOINTS.advancePaymentFees.base,
    );
  }

  create(
    body: CreateFeeScheduleRequest,
  ): Promise<ApiResponse<FeeScheduleEntry>> {
    return apiClient.post<FeeScheduleEntry>(
      API_ENDPOINTS.advancePaymentFees.base,
      body,
    );
  }

  update(
    id: string,
    body: UpdateFeeScheduleRequest,
  ): Promise<ApiResponse<FeeScheduleEntry>> {
    return apiClient.patch<FeeScheduleEntry>(
      API_ENDPOINTS.advancePaymentFees.byId(id),
      body,
    );
  }

  delete(id: string): Promise<ApiResponse<{ id: string }>> {
    return apiClient.delete<{ id: string }>(
      API_ENDPOINTS.advancePaymentFees.byId(id),
    );
  }
}

export const advancePaymentFeeScheduleService =
  new AdvancePaymentFeeScheduleService();
