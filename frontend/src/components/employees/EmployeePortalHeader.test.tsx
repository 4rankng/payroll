import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmployeePortalHeader } from "./EmployeePortalHeader";

describe("EmployeePortalHeader", () => {
  it("shows a capped notification count and exposes account actions", async () => {
    const onNotificationClick = vi.fn();
    const onChangePassword = vi.fn();
    const onLogout = vi.fn();

    render(
      <EmployeePortalHeader
        employeeName="Nguyễn An"
        unreadCount={12}
        onNotificationClick={onNotificationClick}
        onChangePassword={onChangePassword}
        onLogout={onLogout}
      />
    );

    expect(screen.getByText("Nguyễn An")).toBeInTheDocument();
    expect(screen.getByText("9+")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thông báo" }));
    expect(onNotificationClick).toHaveBeenCalledOnce();

    fireEvent.pointerDown(screen.getByRole("button", { name: "Menu tài khoản" }), { button: 0 });
    expect(await screen.findByText("Đổi mật khẩu")).toBeInTheDocument();
  });
});
