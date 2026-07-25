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
});
