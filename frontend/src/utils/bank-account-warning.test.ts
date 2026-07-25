import { describe, expect, it } from "vitest";
import {
  getBankAccountNameMismatch,
  getBankAccountWarningTitle,
  getRejectedBankAccountWarning,
} from "./bank-account-warning";

describe("getRejectedBankAccountWarning", () => {
  it("maps the typed save rejection to a dialog warning", () => {
    expect(
      getRejectedBankAccountWarning(
        {
          code: "BANK_ACCOUNT_INVALID",
          message:
            "Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: NGUYEN VAN AN)",
          details: {
            original_account_number: "0123456789",
            attempted_account_number: "0987654321",
          },
        },
        "Nguyễn Văn Ân",
      ),
    ).toEqual({
      accountName: "Nguyễn Văn Ân",
      originalAccountNumber: "0123456789",
      attemptedAccountNumber: "0987654321",
      reason:
        "Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: NGUYEN VAN AN)",
    });
  });

  it("leaves unrelated save errors to normal error handling", () => {
    expect(
      getRejectedBankAccountWarning({
        code: "VALIDATION_ERROR",
        message: "CCCD không hợp lệ",
      }),
    ).toBeNull();
  });
});

describe("getBankAccountWarningTitle", () => {
  it("removes provider diagnostics from an invalid account number warning", () => {
    expect(
      getBankAccountWarningTitle(
        "Số tài khoản không hợp lệ: Invalid account info",
      ),
    ).toBe("Số tài khoản không hợp lệ");
  });

  it("uses a safe Vietnamese title for an unknown provider reason", () => {
    expect(getBankAccountWarningTitle("Unexpected provider response")).toBe(
      "Tài khoản ngân hàng cần kiểm tra",
    );
  });
});

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
