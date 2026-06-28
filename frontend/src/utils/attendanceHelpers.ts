import type { AttendanceRecord } from "@/services/attendance";

type SalaryFields = Pick<AttendanceRecord, "salary_status" | "earning_amount">;

export interface CheckoutWindowSummary {
  checkInTime: string | null;
  validStartTime: string | null;
  validEndTime: string | null;
}

/**
 * Whether salary was recorded for a shift. Treats an explicit backend
 * salary_status of "recorded" OR a positive earning_amount as recorded,
 * so the flag stays correct even when the status field is absent.
 * Shared by the check-in card and the attendance history card.
 */
export function isSalaryRecorded(att: SalaryFields): boolean {
  return att.salary_status === "recorded" || (att.earning_amount != null && att.earning_amount > 0);
}

/**
 * Whether a completed shift is missing its salary record (checked out but
 * no earnings captured) — drives the "Chưa ghi nhận lương" warning.
 */
export function isSalaryMissing(att: Pick<AttendanceRecord, "status" | "salary_status" | "earning_amount">): boolean {
  return att.status === "completed" && !isSalaryRecorded(att);
}

export function getCheckoutWindowSummary(message: string | null): CheckoutWindowSummary | null {
  if (!message) return null;

  const times = Array.from(message.matchAll(/\b\d{2}:\d{2}\b/g), (match) => match[0]);
  if (times.length >= 3) {
    return {
      checkInTime: times[0],
      validStartTime: times[1],
      validEndTime: times[2],
    };
  }
  if (times.length >= 2) {
    return {
      checkInTime: null,
      validStartTime: times[0],
      validEndTime: times[1],
    };
  }

  return null;
}
