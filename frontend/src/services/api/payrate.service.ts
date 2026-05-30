import { apiClient, buildQueryString } from "./client";
import { API_ENDPOINTS } from "@/config/api.config";
import type {
  PayRate,
  PayRateListResponse,
  PayRateListParams,
  PayRateResponse,
  CreatePayRateRequest,
  UpdatePayRateRequest,
  PayRateActionResponse,
  DeletePayRateResponse,
  RateCategory,
} from "@/types/api/payrate.types";

/**
 * New Flexible Pay Rates API Service (v1)
 * Manages pay rate configurations with flexible JSON structure
 */

export type FieldStatus = 'ok' | 'error' | 'warning';

export interface FieldResult {
  submitted: string;
  status: FieldStatus;
  locked: boolean;
  message?: string;
  hint?: string;
  min_value?: string;
  suggested_value?: string;
}

export interface RatesFieldResult extends FieldResult {
  timesheet_count?: number;
}

export interface DryRunValidateResponse {
  valid: boolean;
  can_save_with_warnings: boolean;
  summary: string;
  fields: {
    effective_from: FieldResult;
    effective_to: FieldResult;
    rates: RatesFieldResult;
  };
}

export class PayRateService {
  // ========== PROJECT PAY RATES ==========

  /**
   * Get current active or nearest upcoming payrate for a project
   */
  async getCurrentProjectPayRate(projectId: number): Promise<PayRateResponse> {
    const response = await apiClient.get<PayRate>(
      API_ENDPOINTS.projects.payrate(projectId),
    );
    return {
      status: response.status,
      message: response.message || "",
      data: response.data!,
    };
  }

  /**
   * Get all pay rate configurations for a specific project
   */
  async getProjectPayRates(
    projectId: number,
    params: PayRateListParams = {},
  ): Promise<PayRateListResponse> {
    const queryString = buildQueryString({ ...params, project_id: projectId });
    const response = await apiClient.get<PayRate[]>(`/payrates${queryString}`);
    return {
      status: response.status,
      message: response.message || "",
      data: response.data || [],
      pagination: response.pagination,
    };
  }

  /**
   * Create new pay rate configuration for a project
   */
  async createProjectPayRate(
    projectId: number,
    data: CreatePayRateRequest,
  ): Promise<PayRateResponse> {
    const backendData = {
      rates: data.rates,
      effective_from: data.effective_from,
      effective_to: data.effective_to,
    };
    const response = await apiClient.post<PayRate>(
      API_ENDPOINTS.projects.payrate(projectId),
      backendData,
    );
    return {
      status: response.status,
      message: response.message || "",
      data: response.data!,
    };
  }

  // ========== PAY RATE MANAGEMENT ==========

  /**
   * Create new pay rate configuration
   */
  async createPayRate(data: CreatePayRateRequest): Promise<PayRateResponse> {
    // Transform frontend field names to backend expectations
    const backendData = {
      rates: data.rates,
      effective_from: data.effective_from,
      effective_to: data.effective_to,
    };
    return (await apiClient.post<PayRateResponse>(
      "/payrates",
      backendData,
    )) as unknown as PayRateResponse;
  }

  /**
   * Get specific pay rate details
   */
  async getPayRate(payRateId: number): Promise<PayRateResponse> {
    return (await apiClient.get<PayRateResponse>(
      API_ENDPOINTS.payrates.byId(payRateId),
    )) as unknown as PayRateResponse;
  }

  /**
   * List pay rate configurations with optional filtering
   */
  async listPayRates(
    params: PayRateListParams = {},
  ): Promise<PayRateListResponse> {
    const queryString = buildQueryString(params);
    return (await apiClient.get<PayRateListResponse>(
      `/payrates${queryString}`,
    )) as unknown as PayRateListResponse;
  }

  /**
   * Update existing pay rate configuration
   */
  async updatePayRate(
    payRateId: number,
    data: UpdatePayRateRequest,
  ): Promise<PayRateResponse> {
    // Transform frontend field names to backend expectations
    const backendData = {
      rates: data.rates,
      effective_from: data.effective_from,
      effective_to: data.effective_to,
    };
    return (await apiClient.put<PayRateResponse>(
      `/payrates/${payRateId}`,
      backendData,
    )) as unknown as PayRateResponse;
  }

  /**
   * Delete pay rate configuration (Admin only)
   */
  async deletePayRate(payRateId: number): Promise<DeletePayRateResponse> {
    return (await apiClient.delete<DeletePayRateResponse>(
      `/payrates/${payRateId}`,
    )) as unknown as DeletePayRateResponse;
  }

  // ========== APPROVAL WORKFLOW REMOVED ==========
  // No approval process needed - payrates are immediately active when created

  // ========== CONVENIENCE METHODS ==========

  /**
   * Dry-run validate a payrate create/update before saving.
   * The response mirrors back every submitted field with its validation state,
   * so the frontend can render inline feedback directly on each input.
   */
  async dryRunValidate(
    projectId: number,
    data: { rates: unknown; effective_from: string; effective_to?: string | null },
    payrateId?: number,
  ): Promise<DryRunValidateResponse> {
    const body = {
      project_id: projectId,
      rates: data.rates,
      effective_from: data.effective_from,
      effective_to: data.effective_to || undefined,
    };
    const url = payrateId
      ? `/payrates/${payrateId}/validate`
      : `/payrates/validate?project_id=${projectId}`;
    const response = await apiClient.post<DryRunValidateResponse>(url, body);
    // Backend returns the validate response directly (not wrapped in ApiResponse.data)
    // so the payload is at response level, not response.data
    const payload = (response.data ?? response) as unknown as DryRunValidateResponse;
    if (!payload?.fields) {
      throw new Error('Phản hồi từ máy chủ không hợp lệ');
    }
    return payload;
  }

  /**
   * Copy pay rate configuration to another project
   */
  async copyPayRate(
    sourceRateId: number,
    targetProjectId: number,
    options: {
      effective_from: string;
      effective_to: string;
      modify_rates?: {
        adjustment_type: "percentage" | "fixed";
        adjustment_value: number;
      };
    },
  ): Promise<PayRateResponse> {
    // First, get the source pay rate
    const sourceRate = await this.getPayRate(sourceRateId);
    let targetRates = sourceRate.data.rates;

    // Apply modifications if specified
    if (options.modify_rates) {
      targetRates = this.adjustRates(
        targetRates as Record<string, RateCategory>,
        options.modify_rates,
      );
    }

    return this.createProjectPayRate(targetProjectId, {
      rates: targetRates as Record<string, RateCategory>,
      effective_from: options.effective_from,
      effective_to: options.effective_to,
    });
  }

  /**
   * Helper function to adjust all rates in a configuration
   */
  private adjustRates(
    rates: Record<string, RateCategory>,
    adjustment: {
      adjustment_type: "percentage" | "fixed";
      adjustment_value: number;
    },
  ): Record<string, RateCategory> {
    const adjustRate = (value: number): number => {
      if (adjustment.adjustment_type === "percentage") {
        return Math.round(value * (1 + adjustment.adjustment_value / 100));
      } else {
        return Math.round(value + adjustment.adjustment_value);
      }
    };

    const adjustCategory = (category: RateCategory): RateCategory => {
      const adjusted: RateCategory = {};

      Object.entries(category).forEach(([key, value]) => {
        if (typeof value === "number") {
          adjusted[key] = adjustRate(value);
        } else {
          adjusted[key] = adjustCategory(value);
        }
      });

      return adjusted;
    };

    const adjustedRates: Record<string, RateCategory> = {};
    Object.entries(rates).forEach(([skillLevel, category]) => {
      adjustedRates[skillLevel] = adjustCategory(category);
    });

    return adjustedRates;
  }

  // ========== BULK OPERATIONS REMOVED ==========
  // No approval workflow means no bulk approval/rejection needed

  /**
   * Check if a payrate configuration has associated timesheets
   */
  async hasTimesheets(payRateId: number): Promise<boolean> {
    try {
      const response = await apiClient.get<{
        has_timesheets: boolean;
        timesheet_count: number;
      }>(`/payrates/${payRateId}/timesheets/count`);

      return response.data?.has_timesheets || false;
    } catch (error) {
      // If endpoint doesn't exist, fall back to checking if active status
      // This is a safe fallback - if it's active, it likely has timesheets
      console.warn("Unable to check timesheet count, using fallback logic");
      return false;
    }
  }
}

// Create singleton instance
export const payRateService = new PayRateService();
