import { describe, expect, it } from "vitest";
import { getAttendanceIssueSummary, getCheckoutWindowSummary } from "./attendanceHelpers";

describe("attendanceHelpers", () => {
  it("extracts checkout window times from backend messages", () => {
    expect(
      getCheckoutWindowSummary("Đã hết hạn tan ca (Vào làm: 08:35; Tan ca: 17:00 (hạn chót 18:00))")
    ).toEqual({
      checkInTime: "08:35",
      validStartTime: "17:00",
      validEndTime: "18:00",
    });
  });

  it("summarizes expired checkout reasons without parenthetical noise", () => {
    expect(
      getAttendanceIssueSummary("Đã hết hạn tan ca (Vào làm: 08:35; Tan ca: 17:00 (hạn chót 18:00))")
    ).toEqual({
      title: "Quá hạn tan ca",
      description: "Ca không được ghi nhận vì đã quá hạn tan ca.",
      details: [
        { label: "Vào làm", value: "08:35" },
        { label: "Tan ca", value: "17:00" },
        { label: "Hạn chót", value: "18:00" },
      ],
    });
  });
});
