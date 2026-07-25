import { describe, expect, it } from "vitest";
import { getBankAccountNameMismatch } from "./bank-account-warning";

describe("getBankAccountNameMismatch", () => {
  it("extracts the entered and bank-reported account holder names", () => {
    expect(
      getBankAccountNameMismatch(
        "Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: PHAM THI THUY HANG)",
        "Phạm Thị Thúy Hằng",
      ),
    ).toEqual({
      enteredName: "Phạm Thị Thúy Hằng",
      bankName: "PHAM THI THUY HANG",
    });
  });

  it("does not turn other bank validation errors into a name comparison", () => {
    expect(
      getBankAccountNameMismatch(
        "Số tài khoản không hợp lệ",
        "Nguyễn Văn An",
      ),
    ).toBeNull();
  });

  it("falls back when the bank did not return the correct holder name", () => {
    expect(
      getBankAccountNameMismatch(
        "Tên chủ tài khoản không khớp với ngân hàng",
        "Nguyễn Văn An",
      ),
    ).toBeNull();
  });
});
