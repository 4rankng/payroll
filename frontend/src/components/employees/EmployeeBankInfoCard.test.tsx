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

    expect(screen.getByText("Tài khoản nhận tiền")).toBeInTheDocument();
    expect(screen.queryByText("Tiền ứng lương sẽ được chuyển vào tài khoản này")).not.toBeInTheDocument();
    expect(screen.getByText("Ngân hàng")).toBeInTheDocument();
    expect(screen.getByText("Số tài khoản")).toBeInTheDocument();
    expect(screen.getByText("Chủ tài khoản")).toBeInTheDocument();
    expect(screen.getByText(profile.bank_account_number)).toBeInTheDocument();
    expect(screen.getByText(profile.bank.branch_name)).toBeInTheDocument();
    expect(screen.getByText(profile.bank_account_name)).toBeInTheDocument();
  });

  it("always shows the account holder even when it matches the employee name", () => {
    render(
      <EmployeeBankInfoCard
        profile={{
          fullname: "Nguyễn Văn An",
          bank_account_name: "Nguyễn Văn An",
          bank_account_number: "11122111918888",
          bank: { branch_name: "Quân đội (MB)" },
        } as EmployeeProfile}
      />
    );

    expect(screen.getByText("Chủ tài khoản")).toBeInTheDocument();
    expect(screen.getByText("Nguyễn Văn An")).toBeInTheDocument();
  });

  it("shows a clear empty account state without inventing verification", () => {
    render(
      <EmployeeBankInfoCard
        profile={{ fullname: "Nguyễn Văn An" } as EmployeeProfile}
      />
    );

    expect(screen.getByText("Chưa có thông tin ngân hàng")).toBeInTheDocument();
    expect(screen.getByText("Liên hệ quản lý để cập nhật")).toBeInTheDocument();
    expect(screen.queryByText("Đã xác minh")).not.toBeInTheDocument();
  });

  it("keeps partial bank data in the incomplete state used by request eligibility", () => {
    render(
      <EmployeeBankInfoCard
        profile={{
          fullname: "Nguyễn Văn An",
          bank_account_number: "11122111918888",
          bank_account_name: "Nguyễn Văn An",
        } as EmployeeProfile}
      />
    );

    expect(screen.getByText("Chưa có thông tin ngân hàng")).toBeInTheDocument();
    expect(screen.queryByText("11122111918888")).not.toBeInTheDocument();
  });
});
