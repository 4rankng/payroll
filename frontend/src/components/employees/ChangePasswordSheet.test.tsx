import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useIsMobile } from "@/hooks/useBreakpoint";
import { ChangePasswordSheet } from "./ChangePasswordSheet";

vi.mock("@/hooks/useBreakpoint", () => ({ useIsMobile: vi.fn() }));

function fillPasswords() {
  fireEvent.change(screen.getByLabelText("Mật khẩu hiện tại", { exact: true }), { target: { value: "Current123!" } });
  fireEvent.change(screen.getByLabelText("Mật khẩu mới", { exact: true }), { target: { value: "Updated123!" } });
  fireEvent.change(screen.getByLabelText("Xác nhận mật khẩu mới", { exact: true }), { target: { value: "Updated123!" } });
}

describe.each([false, true])("ChangePasswordSheet (mobile: %s)", (isMobile) => {
  beforeEach(() => { vi.mocked(useIsMobile).mockReturnValue(isMobile); });

  it("explains invalid current and confirmation fields without submitting", () => {
    const onSubmit = vi.fn();
    render(<ChangePasswordSheet open onOpenChange={vi.fn()} onSubmit={onSubmit} isPending={false} />);
    fireEvent.change(screen.getByLabelText("Mật khẩu mới", { exact: true }), { target: { value: "Updated123!" } });
    fireEvent.click(screen.getByRole("button", { name: "Đổi mật khẩu" }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Mật khẩu hiện tại", { exact: true })).toHaveAccessibleDescription("Nhập mật khẩu hiện tại.");
    expect(screen.getByLabelText("Mật khẩu hiện tại", { exact: true })).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Xác nhận mật khẩu mới", { exact: true })).toHaveAccessibleDescription("Xác nhận mật khẩu chưa khớp.");
  });

  it("keeps failed submissions recoverable and catches the rejection", async () => {
    const onSubmit = vi.fn().mockRejectedValueOnce(new Error("Mật khẩu hiện tại không đúng.")).mockResolvedValueOnce(undefined);
    render(<ChangePasswordSheet open onOpenChange={vi.fn()} onSubmit={onSubmit} isPending={false} />);
    fillPasswords();
    fireEvent.click(screen.getByRole("button", { name: "Đổi mật khẩu" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Mật khẩu hiện tại không đúng.");
    expect(screen.getByLabelText("Mật khẩu mới", { exact: true })).toHaveValue("Updated123!");

    fireEvent.change(screen.getByLabelText("Mật khẩu hiện tại", { exact: true }), { target: { value: "Correct123!" } });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Đổi mật khẩu" }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.getByLabelText("Mật khẩu mới", { exact: true })).toHaveValue(""));
  });

  it("clears sensitive fields and visibility when closed and reopened", () => {
    const props = { onOpenChange: vi.fn(), onSubmit: vi.fn(), isPending: false };
    const { rerender } = render(<ChangePasswordSheet {...props} open />);
    fillPasswords();
    fireEvent.click(screen.getByRole("button", { name: "Hiện mật khẩu hiện tại" }));
    expect(screen.getByLabelText("Mật khẩu hiện tại", { exact: true })).toHaveAttribute("type", "text");
    rerender(<ChangePasswordSheet {...props} open={false} />);
    rerender(<ChangePasswordSheet {...props} open />);

    for (const label of ["Mật khẩu hiện tại", "Mật khẩu mới", "Xác nhận mật khẩu mới"]) {
      expect(screen.getByLabelText(label, { exact: true })).toHaveValue("");
      expect(screen.getByLabelText(label, { exact: true })).toHaveAttribute("type", "password");
    }
  });

  it("prevents a second submission while the first request is unresolved", async () => {
    let resolveSubmission: () => void = () => {};
    const onSubmit = vi.fn(() => new Promise<void>((resolve) => { resolveSubmission = resolve; }));
    render(<ChangePasswordSheet open onOpenChange={vi.fn()} onSubmit={onSubmit} isPending={false} />);
    fillPasswords();
    const submit = screen.getByRole("button", { name: "Đổi mật khẩu" });
    fireEvent.click(submit);
    fireEvent.click(submit);
    expect(onSubmit).toHaveBeenCalledTimes(1);
    resolveSubmission();
    await waitFor(() => expect(screen.getByLabelText("Mật khẩu mới", { exact: true })).toHaveValue(""));
  });
});
