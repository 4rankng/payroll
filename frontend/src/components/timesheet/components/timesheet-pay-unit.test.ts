import { describe, expect, it } from "vitest";
import {
  calculateTimesheetPreviewAmount,
  findTimesheetRate,
  getPayrateUnit,
} from "./timesheet-pay-unit";

describe("timesheet pay unit", () => {
  it("uses a flexible project's configured amount once per shift", () => {
    expect(calculateTimesheetPreviewAmount(252000, 10, true)).toBe(252000);
    expect(getPayrateUnit(true)).toBe("đ/ca");
  });

  it("preserves hourly calculation for standard projects", () => {
    expect(calculateTimesheetPreviewAmount(252000, 10, false)).toBe(2520000);
    expect(getPayrateUnit(false)).toBe("đ/giờ");
  });

  it("selects the assigned position instead of the first configured position", () => {
    const rates = {
      "Công nhân": {
        "Ngày thường": { "08:00-17:00": 252000 },
      },
      "Thợ hàn": {
        "Ngày thường": { "08:00-17:00": 315000 },
      },
    };

    expect(
      findTimesheetRate(rates, "Tho han", "Ngày thường", "08:00-17:00"),
    ).toBe(315000);
  });
});
