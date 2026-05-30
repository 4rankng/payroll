// Types for the admin "Cấu hình phí giao dịch chi hộ" (disbursement fee
// schedule) API. Mirrors internal/app/dto/disbursement_fee_schedule.go on
// the backend.

export interface DisbursementFeeScheduleEntry {
  id: string;
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
  effectiveDate: string;
  feeVnd: number;
  notes?: string;
}

export type UpdateDisbursementFeeScheduleRequest =
  CreateDisbursementFeeScheduleRequest;
