// Types for the admin "Cấu hình phí ứng lương" (advance-payment fee schedule) API.
// Mirrors internal/app/dto/advance_payment_fee_schedule.go on the backend.

export interface FeeScheduleTier {
  minAmount: number;   // VND, integer
  percentage: number;  // 0-100, e.g. 2 for 2%
}

export interface FeeScheduleEntry {
  id: string;
  effectiveDate: string;          // YYYY-MM-DD
  tiers: FeeScheduleTier[];
  minFeeVnd: number;
  notes?: string;
  createdAt: string;
  createdByUserId?: number;
  isCurrentlyActive: boolean;
  isPending: boolean;
  summary: string;                // Vietnamese human-readable summary
}

export interface FeeScheduleListResponse {
  entries: FeeScheduleEntry[];
}

export interface CreateFeeScheduleRequest {
  effectiveDate: string;
  tiers: FeeScheduleTier[];
  minFeeVnd: number;
  notes?: string;
}

export type UpdateFeeScheduleRequest = CreateFeeScheduleRequest;
