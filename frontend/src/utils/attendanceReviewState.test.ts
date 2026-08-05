import { describe, expect, it } from "vitest";
import {
  canApproveAttendance,
  canRejectAttendance,
  getAttendanceOperationalStatus,
  getAttendanceReviewStatusLabel,
  needsAttendanceApprovalRepair,
} from "./attendanceReviewState";

describe("attendanceReviewState", () => {
  it("allows an approved attendance without a persisted checkout to be repaired", () => {
    const attendance = { status: "completed", review_action: "approved" as const, check_out_time: undefined };

    expect(needsAttendanceApprovalRepair(attendance)).toBe(true);
    expect(canApproveAttendance(attendance)).toBe(true);
    expect(getAttendanceReviewStatusLabel(attendance)).toBe("Cần duyệt lại");
  });

  it("keeps a consistent approval terminal", () => {
    const attendance = {
      status: "completed",
      review_action: "approved" as const,
      check_out_time: "2026-08-05T17:00:00+07:00",
    };

    expect(needsAttendanceApprovalRepair(attendance)).toBe(false);
    expect(canApproveAttendance(attendance)).toBe(false);
    expect(getAttendanceOperationalStatus(attendance)).toBe("completed");
    expect(getAttendanceReviewStatusLabel(attendance)).toBe("Đã duyệt");
  });

  it("does not allow rejecting an attendance after its earning was credited", () => {
    expect(canRejectAttendance({
      status: "completed",
      review_action: null,
      quota_credited_at: "2026-08-05T01:00:00+07:00",
    })).toBe(false);
  });
});
