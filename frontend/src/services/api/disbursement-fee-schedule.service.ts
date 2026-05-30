// Admin disbursement-fee schedule API client.
// All endpoints require admin role; the backend enforces that via Casbin.

import { apiClient, type ApiResponse } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  DisbursementFeeScheduleListResponse,
  DisbursementFeeScheduleEntry,
  CreateDisbursementFeeScheduleRequest,
  UpdateDisbursementFeeScheduleRequest,
} from "@/types/api/disbursement-fee-schedule.types";

class DisbursementFeeScheduleService {
  list(): Promise<ApiResponse<DisbursementFeeScheduleListResponse>> {
    return apiClient.get<DisbursementFeeScheduleListResponse>(
      API_ENDPOINTS.disbursementFees.base,
    );
  }

  create(
    body: CreateDisbursementFeeScheduleRequest,
  ): Promise<ApiResponse<DisbursementFeeScheduleEntry>> {
    return apiClient.post<DisbursementFeeScheduleEntry>(
      API_ENDPOINTS.disbursementFees.base,
      body,
    );
  }

  update(
    id: string,
    body: UpdateDisbursementFeeScheduleRequest,
  ): Promise<ApiResponse<DisbursementFeeScheduleEntry>> {
    return apiClient.patch<DisbursementFeeScheduleEntry>(
      API_ENDPOINTS.disbursementFees.byId(id),
      body,
    );
  }

  delete(id: string): Promise<ApiResponse<{ id: string }>> {
    return apiClient.delete<{ id: string }>(
      API_ENDPOINTS.disbursementFees.byId(id),
    );
  }
}

export const disbursementFeeScheduleService =
  new DisbursementFeeScheduleService();
