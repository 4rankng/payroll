import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { EmployeeBankInfoCard } from "./EmployeeBankInfoCard";

const profile = {
  fullname: "Nguyễn Văn Nhân Viên Có Tên Rất Dài",
  bank_account_number: "123456789012345678901234567890",
  bank_account_name: "NGUYEN VAN NHAN VIEN CO TEN RAT DAI",
  bank: { branch_name: "Ngân hàng Thương mại Cổ phần Ngoại thương Việt Nam" },
} as EmployeeProfile;

describe("EmployeeBankInfoCard", () => {
  it("shows the receiving destination and keeps the full account number", () => {
    render(<EmployeeBankInfoCard profile={profile} />);

    expect(screen.getByText("Tài khoản sẽ nhận tiền ứng lương")).toBeInTheDocument();
    expect(screen.getByText(profile.bank_account_number)).toBeInTheDocument();
    expect(screen.getByText(profile.bank.branch_name)).toBeInTheDocument();
  });
});
