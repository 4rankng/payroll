import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  BankAccountWarningProvider,
  useBankAccountWarning,
} from "./BankAccountWarningContext";

function NameMismatchTrigger() {
  const { showBankAccountWarning } = useBankAccountWarning();

  return (
    <button
      type="button"
      onClick={() =>
        showBankAccountWarning({
          accountName: "Phạm Thị Thúy Hằng",
          reason:
            "Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: PHAM THI THUY HANG)",
        })
      }
    >
      Mở cảnh báo
    </button>
  );
}

function InvalidAccountTrigger() {
  const { showBankAccountWarning } = useBankAccountWarning();

  return (
    <button
      type="button"
      onClick={() =>
        showBankAccountWarning({
          originalAccountNumber: "0123456789",
          attemptedAccountNumber: "0987654321",
          reason: "Số tài khoản không hợp lệ: Invalid account info",
        })
      }
    >
      Mở cảnh báo số tài khoản
    </button>
  );
}

describe("BankAccountWarningProvider", () => {
  it("shows a concise name comparison without naming the payment provider", () => {
    render(
      <BankAccountWarningProvider>
        <NameMismatchTrigger />
      </BankAccountWarningProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Mở cảnh báo" }));

    expect(
      screen.getByRole("alertdialog", {
        name: "Tên chủ tài khoản không khớp",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Tên đã nhập")).toBeInTheDocument();
    expect(screen.getByText("Phạm Thị Thúy Hằng")).toBeInTheDocument();
    expect(screen.getByText("Ngân hàng ghi")).toBeInTheDocument();
    expect(screen.getByText("PHAM THI THUY HANG")).toBeInTheDocument();
    expect(screen.queryByText(/OnePay/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Đã lưu thông tin nhân viên/i)).not.toBeInTheDocument();
  });

  it("shows one Vietnamese message without exposing provider diagnostics", () => {
    render(
      <BankAccountWarningProvider>
        <InvalidAccountTrigger />
      </BankAccountWarningProvider>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Mở cảnh báo số tài khoản" }),
    );

    expect(
      screen.getByRole("alertdialog", {
        name: "Số tài khoản không hợp lệ",
      }),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Số tài khoản không hợp lệ")).toHaveLength(1);
    expect(screen.getByText("Đang lưu")).toBeInTheDocument();
    expect(screen.getByText("0123456789")).toBeInTheDocument();
    expect(screen.getByText("Vừa nhập")).toBeInTheDocument();
    expect(screen.getByText("0987654321")).toBeInTheDocument();
    expect(screen.queryByText(/Invalid account info/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText("Tài khoản ngân hàng không hợp lệ"),
    ).not.toBeInTheDocument();
  });
});
