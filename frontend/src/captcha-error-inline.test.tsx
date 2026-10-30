import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Login from "./pages/Login";

// PAYR-29: tooltip lỗi captcha của widget ALTCHA render absolute CHỒNG ĐÈ
// checkbox captcha + nút Đăng nhập. Fix: ép popover lỗi vào normal flow ngay
// dưới hàng captcha bằng CSS override (logic captcha giữ nguyên).
//
// jsdom không chạy cascade CSS thật trên stylesheet bundle, nên hợp đồng
// no-overlap được pin ở hai mối ghép kiểm chứng được:
//  (1) rule override trong src/styles/admin-daisy.css — selector đúng phạm vi
//      login, position: static, bỏ mũi tên (đọc trực tiếp file CSS);
//  (2) DOM order — hàng captcha nằm TRƯỚC nút submit trong cùng luồng form,
//      nên lỗi inline chỉ có thể đẩy nút xuống, không thể phủ lên.
const css = readFileSync(resolve(process.cwd(), "src/styles/admin-daisy.css"), "utf8");

const mocks = vi.hoisted(() => ({
  mutateAsync: vi.fn().mockRejectedValue(new Error("Thông tin đăng nhập không hợp lệ")),
  apiGet: vi.fn().mockResolvedValue({ data: { required: true } }),
}));

// The login error surfaces drive needsCaptcha; a plain (non-captcha) failure
// increments failedAttempts, then the /auth/captcha/required preflight
// (mocked required:true) forces the widget row to mount.
const loginState = vi.hoisted(() => ({
  current: {
    isPending: false,
    error: null as { message?: string; http_status?: number } | null,
    isError: false,
    isSuccess: false,
  },
}));

vi.mock("altcha", () => ({}));
vi.mock("altcha/i18n/vi", () => ({}));

vi.mock("@/hooks/api/useAuth", () => ({
  useLogin: () => ({
    ...loginState.current,
    mutateAsync: mocks.mutateAsync,
    reset: vi.fn(),
  }),
}));

vi.mock("@/contexts", () => ({
  useAuth: () => ({ isAuthenticated: false, isLoading: false }),
}));

vi.mock("@/lib/auth", () => ({
  authManager: {
    isTokenValid: () => false,
    removeToken: vi.fn(),
    getUserRole: vi.fn(),
  },
}));

vi.mock("@/services/api/client", () => ({
  apiClient: { get: mocks.apiGet },
}));

describe("Lỗi captcha render inline dưới hàng captcha, không đè control (PAYR-29)", () => {
  beforeEach(() => {
    mocks.mutateAsync.mockClear();
    mocks.apiGet.mockClear();
    mocks.apiGet.mockResolvedValue({ data: { required: true } });
    loginState.current = {
      isPending: false,
      error: { message: "Thông tin đăng nhập không hợp lệ" },
      isError: true,
      isSuccess: false,
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("CSS override ép popover lỗi vào static flow dưới hàng captcha (không absolute)", () => {
    // Selector phải đúng phạm vi login + đúng variant lỗi + đúng display chuẩn,
    // và phải thắng specificity stylesheet widget (0-5-0 > 0-4-0 data-top).
    expect(css).toMatch(
      /\[data-login-ui\] \.altcha-popover\[data-popover\]\[data-variant="error"\]\[data-display="standard"\] \{\s*position: static;/,
    );
    // Mũi tên popover (chỉ có ý nghĩa khi nổi) phải bị tắt ở cùng phạm vi.
    expect(css).toMatch(
      /\[data-login-ui\] \.altcha-popover\[data-popover\]\[data-variant="error"\]\[data-display="standard"\] \.altcha-popover-arrow \{\s*display: none;/,
    );
  });

  it("không rule login nào đặt lại popover lỗi về absolute", () => {
    const popoverRules = css.match(/\[data-login-ui\][^{]*\.altcha-popover[^{]*\{[^}]*\}/g) ?? [];
    expect(popoverRules.length).toBeGreaterThan(0);
    for (const rule of popoverRules) {
      expect(rule).not.toMatch(/position:\s*absolute/);
      expect(rule).not.toMatch(/position:\s*fixed/);
    }
  });

  it("hàng captcha nằm trước nút Đăng nhập trong cùng luồng form — lỗi inline đẩy nút xuống thay vì phủ lên", async () => {
    render(<MemoryRouter><Login /></MemoryRouter>);

    // Preflight blur → server báo captcha required → hàng widget mount.
    fireEvent.change(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"), {
      target: { value: "frankng" },
    });
    fireEvent.blur(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"));

    const captchaRow = await waitFor(() => {
      const row = screen.getByTestId("altcha-captcha");
      expect(row).toBeTruthy();
      return row;
    });

    const submit = document.querySelector('button[type="submit"]');
    expect(submit).toHaveAccessibleName("Đăng nhập");

    // DOM order: submit ĐI SAU hàng captcha → cùng normal flow.
    expect(!!(captchaRow.compareDocumentPosition(submit!) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);

    // Cùng cây form (không portal/floating layer): height tăng của hàng captcha
    // chuyển thành khoảng trống phía trên nút, không thể tạo overlap.
    const form = document.querySelector("form");
    expect(form?.contains(captchaRow)).toBe(true);
    expect(form?.contains(submit!)).toBe(true);
  });
});
