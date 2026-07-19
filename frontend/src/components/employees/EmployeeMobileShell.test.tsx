import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmployeeMobileShell } from "./EmployeeMobileShell";

describe("EmployeeMobileShell", () => {
  it("renders interactive account chrome only when profile actions are available", () => {
    render(
      <EmployeeMobileShell
        employeeName="Nguyễn An"
        onNotificationClick={vi.fn()}
        onChangePassword={vi.fn()}
        onLogout={vi.fn()}
      >
        <p>Nội dung</p>
      </EmployeeMobileShell>
    );

    expect(screen.getByRole("button", { name: "Thông báo" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Menu tài khoản" })).toBeInTheDocument();
  });

  it("does not render dead actions while profile data is loading", () => {
    render(<EmployeeMobileShell chrome="skeleton"><p>Nội dung</p></EmployeeMobileShell>);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Đang tải thông tin nhân viên")).toBeInTheDocument();
  });

  it("reserves safe content space for the attendance action toolbar", () => {
    const { container } = render(
      <EmployeeMobileShell chrome="error" hasActionToolbar><p>Nội dung</p></EmployeeMobileShell>
    );

    expect(container.firstElementChild).toHaveAttribute("data-has-action-toolbar", "true");
  });
});
