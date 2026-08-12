import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ZaloConnectionSection } from "./ZaloConnectionSection";

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  saveCredentials: vi.fn(),
  setEnabled: vi.fn(),
  setFlexPayZNSEnabled: vi.fn(),
  testSend: vi.fn(),
  testSendPending: false,
  refreshToken: vi.fn(),
  status: {
    enabled: true,
    configured: true,
    connected: true,
    app_id: "app-123",
    flexpay_zns_enabled: false,
    expires_at: "2099-01-01T00:00:00.000Z",
  },
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("@/hooks/api/useZaloConnection", () => ({
  useZaloStatus: () => ({
    data: { data: mocks.status },
    isLoading: false,
    isError: false,
    refetch: mocks.refetch,
  }),
  useSaveZaloCredentials: () => ({
    mutateAsync: mocks.saveCredentials,
    isPending: false,
  }),
  useSetZaloEnabled: () => ({
    mutateAsync: mocks.setEnabled,
    isPending: false,
  }),
  useSetFlexPayZNSEnabled: () => ({
    mutateAsync: mocks.setFlexPayZNSEnabled,
    isPending: false,
  }),
  useTestZaloSend: () => ({
    mutateAsync: mocks.testSend,
    isPending: mocks.testSendPending,
  }),
  useRefreshZaloToken: () => ({
    mutateAsync: mocks.refreshToken,
    isPending: false,
  }),
}));

describe("ZaloConnectionSection", () => {
  const openStoredCredentials = () => {
    fireEvent.click(
      screen.getByRole("button", { name: "Thay đổi cấu hình" }),
    );
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.saveCredentials.mockResolvedValue({ data: undefined });
    mocks.setEnabled.mockResolvedValue({ data: undefined });
    mocks.setFlexPayZNSEnabled.mockResolvedValue({ data: undefined });
    mocks.refreshToken.mockResolvedValue({ data: undefined });
    mocks.testSend.mockResolvedValue({
      data: { error_code: 0, error_msg: "Thành công", msg_id: "msg-123" },
    });
    mocks.testSendPending = false;
    Object.assign(mocks.status, {
      enabled: true,
      configured: true,
      connected: true,
      app_id: "app-123",
      flexpay_zns_enabled: false,
    });
  });

  it("saves the complete credential payload", async () => {
    render(<ZaloConnectionSection />);
    openStoredCredentials();

    fireEvent.change(screen.getByLabelText("App ID"), {
      target: { value: "app-123" },
    });
    fireEvent.change(screen.getByLabelText("Khóa bí mật (Secret Key)"), {
      target: { value: "secret-123" },
    });
    fireEvent.change(screen.getByLabelText("Mã truy cập (Access Token)"), {
      target: { value: "access-123" },
    });
    fireEvent.change(screen.getByLabelText("Mã làm mới (Refresh Token)"), {
      target: { value: "refresh-123" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Lưu cấu hình" }));

    await waitFor(() => {
      expect(mocks.saveCredentials).toHaveBeenCalledWith({
        app_id: "app-123",
        secret_key: "secret-123",
        access_token: "access-123",
        refresh_token: "refresh-123",
      });
    });
  });

  it("checks the stored credentials without sending a message", async () => {
    render(<ZaloConnectionSection />);
    openStoredCredentials();

    fireEvent.click(
      screen.getByRole("button", { name: "Kiểm tra cấu hình đã lưu" }),
    );

    await waitFor(() => expect(mocks.refreshToken).toHaveBeenCalledOnce());
    expect(mocks.testSend).not.toHaveBeenCalled();
  });

  it("requires confirmation before disabling the employee OTP flow", async () => {
    render(<ZaloConnectionSection />);

    fireEvent.click(
      screen.getByRole("switch", { name: "Bật đặt lại mật khẩu qua Zalo" }),
    );
    expect(mocks.setEnabled).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Xác nhận tắt" }));

    await waitFor(() => expect(mocks.setEnabled).toHaveBeenCalledWith(false));
  });

  it("keeps salary notifications independently configurable from the OTP flow", () => {
    Object.assign(mocks.status, { enabled: false, connected: true });
    render(<ZaloConnectionSection />);

    expect(
      screen.getByRole("switch", { name: "Bật thông báo ZNS lương linh hoạt" }),
    ).toBeEnabled();
  });

  it("sends the default sample OTP with a trimmed test phone", async () => {
    render(<ZaloConnectionSection />);

    fireEvent.change(screen.getByLabelText("Số điện thoại nhận thử"), {
      target: { value: " 0357210887 " },
    });
    fireEvent.click(screen.getByRole("button", { name: "Gửi OTP mẫu" }));

    await waitFor(() =>
      expect(mocks.testSend).toHaveBeenCalledWith({
        phone: "0357210887",
        template_data: { otp: "000000" },
      }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "OTP mẫu đã gửi thành công",
    );
  });

  it("sends the salary notification sample separately from the OTP sample", async () => {
    render(<ZaloConnectionSection />);

    fireEvent.change(screen.getByLabelText("Số điện thoại nhận thử"), {
      target: { value: " 0357210887 " },
    });
    fireEvent.click(screen.getByRole("button", { name: "Gửi mẫu lương" }));

    await waitFor(() =>
      expect(mocks.testSend).toHaveBeenCalledWith({
        phone: "0357210887",
        template_id: "619686",
        template_data: {
          customer_name: "Nhân viên kiểm thử",
          max_amount: "1000000",
          expiry_date: expect.stringMatching(/^\d{2}\/\d{2}\/\d{4}$/),
        },
      }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Mẫu thông báo lương đã gửi thành công",
    );
    expect(screen.getByText(/0xxxxxxxxx hoặc 84xxxxxxxxx/)).toBeInTheDocument();
  });

  it("disables both sends while a test message is in progress", () => {
    mocks.testSendPending = true;
    render(<ZaloConnectionSection />);

    expect(screen.getByRole("button", { name: "Gửi OTP mẫu" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Gửi mẫu lương" })).toBeDisabled();
  });

  it("identifies a failed OTP sample in the result", async () => {
    mocks.testSend.mockResolvedValue({
      data: { error_code: -124, error_msg: "Access token không hợp lệ" },
    });
    render(<ZaloConnectionSection />);

    fireEvent.change(screen.getByLabelText("Số điện thoại nhận thử"), {
      target: { value: "0357210887" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Gửi OTP mẫu" }));

    expect(await screen.findByRole("status")).toHaveTextContent(
      "OTP mẫu lỗi -124",
    );
  });

  it("presents both test actions as equal choices beside the input", () => {
    render(<ZaloConnectionSection />);

    expect(screen.getByLabelText("Chọn loại tin nhắn gửi thử")).toHaveClass(
      "sm:grid-cols-2",
    );
    expect(screen.getByRole("button", { name: "Gửi OTP mẫu" })).toHaveClass(
      "h-11",
    );
    expect(screen.getByRole("button", { name: "Gửi mẫu lương" })).toHaveClass(
      "h-11",
    );
  });

  it("exposes the setup state without clipping labels", () => {
    render(<ZaloConnectionSection />);

    const activationStep = screen.getByText("Dịch vụ");
    expect(activationStep).not.toHaveClass("truncate");
    expect(activationStep).toHaveTextContent("đã hoàn tất");
  });

  it("groups both ZNS services into one independently controlled stage", () => {
    render(<ZaloConnectionSection />);

    const servicesHeading = screen.getByRole("heading", {
      name: "Dịch vụ sử dụng kết nối",
    });
    const servicesSection = servicesHeading.closest("section");

    expect(servicesSection).toContainElement(
      screen.getByRole("switch", { name: "Bật đặt lại mật khẩu qua Zalo" }),
    );
    expect(servicesSection).toContainElement(
      screen.getByRole("switch", { name: "Bật thông báo ZNS lương linh hoạt" }),
    );
    expect(screen.queryByText("4", { selector: "span" })).not.toBeInTheDocument();
  });

  it("keeps stored credentials collapsed until an admin chooses to edit", () => {
    render(<ZaloConnectionSection />);

    expect(screen.queryByLabelText("App ID")).not.toBeInTheDocument();
    openStoredCredentials();
    expect(screen.getByLabelText("App ID")).toBeInTheDocument();
    expect(
      screen.getAllByPlaceholderText("Chỉ nhập khi cần thay đổi"),
    ).toHaveLength(3);
  });

  it("does not imply unsaved credentials are being checked", () => {
    render(<ZaloConnectionSection />);
    openStoredCredentials();

    fireEvent.change(screen.getByLabelText("App ID"), {
      target: { value: "app-changed" },
    });

    expect(
      screen.getByRole("button", { name: "Kiểm tra cấu hình đã lưu" }),
    ).toBeDisabled();
    expect(screen.getByText("Lưu thay đổi trước khi kiểm tra.")).toBeInTheDocument();
  });

  it("does not expose a configurable ZNS template", () => {
    render(<ZaloConnectionSection />);
    openStoredCredentials();

    expect(screen.queryByLabelText("Mã mẫu ZNS")).not.toBeInTheDocument();
    expect(screen.queryByText("619684")).not.toBeInTheDocument();
  });

  it("keeps first-time configuration open when a partial save is still incomplete", async () => {
    Object.assign(mocks.status, { configured: false, connected: false });
    render(<ZaloConnectionSection />);

    const appIdInput = await screen.findByLabelText("App ID");
    fireEvent.change(appIdInput, { target: { value: "app-only" } });
    fireEvent.click(screen.getByRole("button", { name: "Lưu cấu hình" }));

    await waitFor(() => expect(mocks.saveCredentials).toHaveBeenCalledOnce());
    expect(screen.getByLabelText("App ID")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Thay đổi cấu hình" }),
    ).not.toBeInTheDocument();
  });
});
