import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Login from "./Login";

// Login contract tests (W10 UU PRO migration): the visual-baseline harness and
// the e2e LoginPage page object both key on these exact hooks, so the tests
// pin them: input ids, placeholders, the type=submit "Đăng nhập" button, the
// role="alert" error surface, and the mutateAsync wiring.
const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  mutateAsync: vi.fn().mockResolvedValue({ data: {} }),
  apiGet: vi.fn().mockResolvedValue({ data: { required: false } }),
}));

// The component reads mutation flags on every render; the test mutates this
// object between renders to drive the error state.
const loginState = vi.hoisted(() => ({
  current: {
    isPending: false,
    error: null as { message?: string; http_status?: number } | null,
    isError: false,
    isSuccess: false,
  },
}));

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
    hasSession: () => false,
    clearSession: vi.fn(),
    getUserRole: vi.fn(),
  },
}));

vi.mock("@/services/api/client", () => ({
  apiClient: { get: mocks.apiGet },
}));

describe("Login page contract (W10)", () => {
  beforeEach(() => {
    mocks.mutateAsync.mockClear();
    mocks.apiGet.mockClear();
    mocks.apiGet.mockResolvedValue({ data: { required: false } });
    loginState.current = { isPending: false, error: null, isError: false, isSuccess: false };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders the harness-login contract: ids, placeholders, submit button, forgot link", () => {
    render(<MemoryRouter><Login /></MemoryRouter>);

    // Visual harness fills by id; placeholders are the documented contract.
    expect(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập")).toHaveAttribute("id", "emailOrUsername");
    expect(screen.getByPlaceholderText("Nhập mật khẩu của bạn")).toHaveAttribute("id", "password");
    expect(screen.getByPlaceholderText("Nhập mật khẩu của bạn")).toHaveAttribute("type", "password");

    // e2e page object: button[type="submit"] + text "Đăng nhập".
    const submit = document.querySelector('button[type="submit"]');
    expect(submit).not.toBeNull();
    expect(submit).toHaveAccessibleName("Đăng nhập");

    expect(screen.getByRole("link", { name: "Quên mật khẩu?" })).toHaveAttribute("href", "/forgot-password");
  });

  it("wires submit to useLogin.mutateAsync with the entered credentials", async () => {
    render(<MemoryRouter><Login /></MemoryRouter>);

    fireEvent.change(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"), {
      target: { value: "frankng" },
    });
    fireEvent.change(screen.getByPlaceholderText("Nhập mật khẩu của bạn"), {
      target: { value: "Admin123" },
    });
    fireEvent.submit(document.querySelector("form")!);

    await waitFor(() => {
      expect(mocks.mutateAsync).toHaveBeenCalledTimes(1);
    });
    expect(mocks.mutateAsync).toHaveBeenCalledWith({ username: "frankng", password: "Admin123" });
  });

  it("renders the error alert when the mutation failed", () => {
    loginState.current = {
      isPending: false,
      error: { message: "Sai thông tin" },
      isError: true,
      isSuccess: false,
    };
    render(<MemoryRouter><Login /></MemoryRouter>);

    const alert = screen.getByRole("alert");
    expect(alert).toHaveAttribute("data-testid", "error-message");
    expect(alert).toHaveTextContent("Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.");
  });

  // Captcha rejections retry inline (stale-proof fix): the widget row stays
  // mounted with a fresh challenge and a role="status" hint appears — no
  // window.location.reload().
  it("captcha rejection shows the inline notice and keeps the widget mounted for retry", async () => {
    loginState.current = {
      isPending: false,
      error: { message: "Xác thực captcha không hợp lệ" },
      isError: true,
      isSuccess: false,
    };
    mocks.apiGet.mockResolvedValue({ data: { required: true } });
    render(<MemoryRouter><Login /></MemoryRouter>);

    // Server-required preflight mounts the captcha row.
    fireEvent.change(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"), {
      target: { value: "frankng" },
    });
    fireEvent.blur(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"));

    const notice = await screen.findByTestId("captcha-notice");
    expect(notice).toHaveAttribute("role", "status");
    expect(notice).toHaveTextContent("Xác thực bảo mật không hợp lệ hoặc đã hết hạn. Vui lòng thử lại.");
    // Inline retry: the widget is still there for a fresh challenge.
    expect(document.querySelectorAll("altcha-widget")).toHaveLength(1);
  });

  // Regression: the server can demand a captcha while the preflight answered
  // required=false (counter crosses the threshold between preflight and POST,
  // or the identifier resolves differently server-side). The rejection must
  // mount the widget anyway and name captcha — not the generic credentials
  // line — or the user is locked out with nothing to solve.
  it("captcha rejection mounts the widget even when the preflight said required=false", async () => {
    loginState.current = {
      isPending: false,
      error: { message: "Tài khoản cần xác thực captcha. Vui lòng hoàn tất captcha rồi đăng nhập lại.", http_status: 400 },
      isError: true,
      isSuccess: false,
    };
    mocks.apiGet.mockResolvedValue({ data: { required: false } });
    render(<MemoryRouter><Login /></MemoryRouter>);

    const captchaRow = await screen.findByTestId("altcha-captcha");
    expect(captchaRow.querySelector("altcha-widget")).not.toBeNull();

    // The alert carries the server's captcha copy, never the vague one.
    // (The altcha widget renders its own role="alert"; key on the test id.)
    const alert = screen.getByTestId("error-message");
    expect(alert).toHaveTextContent("cần xác thực captcha");
    expect(alert).not.toHaveTextContent("Thông tin đăng nhập không hợp lệ");

    // Inline retry hint under the freshly mounted widget.
    expect(await screen.findByTestId("captcha-notice")).toHaveTextContent("Xác thực bảo mật không hợp lệ hoặc đã hết hạn");
  });

  // required=false with no captcha error keeps the form captcha-free.
  it("does not mount the captcha widget when captcha is not required", async () => {
    mocks.apiGet.mockResolvedValue({ data: { required: false } });
    render(<MemoryRouter><Login /></MemoryRouter>);

    fireEvent.change(screen.getByPlaceholderText("CCCD, số điện thoại hoặc tên đăng nhập"), {
      target: { value: "frankng" },
    });
    fireEvent.change(screen.getByPlaceholderText("Nhập mật khẩu của bạn"), { target: { value: "Admin123" } });
    fireEvent.submit(document.querySelector("form")!);

    await waitFor(() => {
      expect(mocks.mutateAsync).toHaveBeenCalledTimes(1);
    });
    expect(mocks.mutateAsync).toHaveBeenCalledWith({ username: "frankng", password: "Admin123" });
    expect(screen.queryByTestId("altcha-captcha")).toBeNull();
    expect(document.querySelectorAll("altcha-widget")).toHaveLength(0);
  });
});
