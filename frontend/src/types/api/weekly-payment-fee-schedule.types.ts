// Types for the admin "Phí trả lương tuần" (weekly-payment fee schedule) API.
// Mirrors internal/app/dto/weekly_payment_fee_schedule.go on the backend.

export interface WeeklyPaymentFeeScheduleEntry {
  id: string;
  effectiveDate: string;        // YYYY-MM-DD
  percentage: number;           // e.g. 2 for 2%
  notes?: string;
  createdAt: string;
  createdByUserId?: number;
  isCurrentlyActive: boolean;
  isPending: boolean;
  summary: string;              // Vietnamese human-readable summary
}

export interface WeeklyPaymentFeeScheduleListResponse {
  entries: WeeklyPaymentFeeScheduleEntry[];
}

export interface CreateWeeklyPaymentFeeScheduleRequest {
  effectiveDate: string;
  percentage: number;
  notes?: string;
}

export type UpdateWeeklyPaymentFeeScheduleRequest =
  CreateWeeklyPaymentFeeScheduleRequest;
