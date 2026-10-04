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
    isTokenValid: () => false,
    removeToken: vi.fn(),
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
});
