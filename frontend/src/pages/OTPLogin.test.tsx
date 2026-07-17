import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import OTPLogin from "./OTPLogin";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  resendOtp: vi.fn().mockResolvedValue({ data: {} }),
}));

vi.mock("react-router-dom", async (importOriginal) => {
  const original = await importOriginal<typeof import("react-router-dom")>();
  return { ...original, useNavigate: () => mocks.navigate };
});

vi.mock("@/contexts", () => ({
  useAuth: () => ({ login: vi.fn() }),
}));

vi.mock("@/services/api/auth.service", () => ({
  authService: {
    resendOtp: mocks.resendOtp,
    verifyLoginOtp: vi.fn(),
  },
  getPendingOtpSessionId: () => "pending-session",
  clearPendingOtp: vi.fn(),
}));

vi.mock("@/utils/avatarHelpers", () => ({
  generateAvatarUrl: () => "avatar",
}));

vi.mock("@/utils/error-handler", () => ({
  showErrorNotification: vi.fn(),
}));

describe("OTPLogin resend protection", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.navigate.mockReset();
    mocks.resendOtp.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts with resend disabled and collapses same-tick resend clicks", async () => {
    render(<OTPLogin />);

    const resendButton = screen.getByRole("button", { name: "Gửi lại (30s)" });
    expect(resendButton).toBeDisabled();

    for (let second = 0; second < 30; second += 1) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1_000);
      });
    }

    const enabledButton = screen.getByRole("button", { name: "Gửi lại mã" });
    fireEvent.click(enabledButton);
    fireEvent.click(enabledButton);

    await act(async () => {
      await Promise.resolve();
    });
    expect(mocks.resendOtp).toHaveBeenCalledTimes(1);
    expect(mocks.resendOtp).toHaveBeenCalledWith({ otp_session_id: "pending-session" });
  });
});
