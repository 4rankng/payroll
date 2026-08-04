import { describe, expect, it } from "vitest";
import {
  canApproveAttendance,
  getAttendanceReviewStatusLabel,
  needsAttendanceApprovalRepair,
} from "./attendanceReviewState";

describe("attendanceReviewState", () => {
  it("allows a corrupted approved/rejected attendance to be repaired", () => {
    const attendance = { status: "rejected", review_action: "approved" as const };

    expect(needsAttendanceApprovalRepair(attendance)).toBe(true);
    expect(canApproveAttendance(attendance)).toBe(true);
    expect(getAttendanceReviewStatusLabel(attendance)).toBe("Cần duyệt lại");
  });

  it("keeps a consistent approval terminal", () => {
    const attendance = { status: "checked_in", review_action: "approved" as const };

    expect(needsAttendanceApprovalRepair(attendance)).toBe(false);
    expect(canApproveAttendance(attendance)).toBe(false);
    expect(getAttendanceReviewStatusLabel(attendance)).toBe("Đã duyệt");
  });
});
