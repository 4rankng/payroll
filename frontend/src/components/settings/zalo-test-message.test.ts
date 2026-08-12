import { describe, expect, it } from "vitest";

import { buildSalaryZnsTestPayload } from "./zalo-test-message";

describe("buildSalaryZnsTestPayload", () => {
  it("keeps the salary sample date ahead of the send date", () => {
    expect(
      buildSalaryZnsTestPayload(new Date(2026, 7, 12, 12, 0, 0)),
    ).toEqual({
      template_id: "619686",
      template_data: {
        customer_name: "Nhân viên kiểm thử",
        max_amount: "1000000",
        expiry_date: "11/09/2026",
      },
    });
  });
});
