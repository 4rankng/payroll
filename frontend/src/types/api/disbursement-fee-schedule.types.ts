// Types for the admin "Cấu hình phí giao dịch chi hộ" (disbursement fee
// schedule) API. Mirrors internal/app/dto/disbursement_fee_schedule.go on
// the backend.

export interface DisbursementFeeScheduleEntry {
  id: string;
  provider: string;               // "9pay" | "1pay"
  effectiveDate: string;          // YYYY-MM-DD
  feeVnd: number;                 // flat per-transfer fee in VND
  notes?: string;
  createdAt: string;
  createdByUserId?: number;
  isCurrentlyActive: boolean;
  isPending: boolean;
  summary: string;                // Vietnamese human-readable summary
}

export interface DisbursementFeeScheduleListResponse {
  entries: DisbursementFeeScheduleEntry[];
}

export interface CreateDisbursementFeeScheduleRequest {
  provider: string;               // "9pay" | "1pay"
  effectiveDate: string;
  feeVnd: number;
  notes?: string;
}

export const DISBURSEMENT_PROVIDERS = ["9pay", "1pay"] as const;
export type DisbursementProvider = (typeof DISBURSEMENT_PROVIDERS)[number];

export const PROVIDER_LABELS: Record<DisbursementProvider, string> = {
  "9pay": "9Pay",
  "1pay": "OnePay",
};

export type UpdateDisbursementFeeScheduleRequest =
  CreateDisbursementFeeScheduleRequest;
