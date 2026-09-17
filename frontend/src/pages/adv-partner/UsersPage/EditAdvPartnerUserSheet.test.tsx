import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EditAdvPartnerUserSheet from "./EditAdvPartnerUserSheet";
const mocks = vi.hoisted(() => ({ getEmployeeById: vi.fn(), updateAdvPartnerUser: vi.fn(), mobile: false }));
vi.mock("@/services/api/employee.service", () => ({ employeeService: mocks }));
vi.mock("@/hooks/useBreakpoint", () => ({ useIsMobile: () => mocks.mobile }));
vi.mock("@/components/ui/bank-selector", () => ({ BankSelector: () => <button>Chọn ngân hàng</button> }));
const employee = { id: 1, fullname: "Nguyễn Văn An", cccd: "000000000001", email: "qa@example.test", mobile: "0900000001", bank_account_number: "111111", bank_account_name: "NGUYEN VAN AN", current_projects: [] };
function renderSheet() {
  return render(<QueryClientProvider client={new QueryClient()}><EditAdvPartnerUserSheet employeeId={1} fullname={employee.fullname} onClose={vi.fn()} /></QueryClientProvider>);
}
beforeEach(() => { vi.clearAllMocks(); mocks.getEmployeeById.mockResolvedValue(employee); });
describe.each([false, true])("Advance partner employee sheet mobile=%s", (mobile) => {
  it("does not allow editing an unloaded employee and retries safely", async () => {
    mocks.mobile = mobile;
    mocks.getEmployeeById.mockRejectedValueOnce(new Error("Request failed"));
    renderSheet();
    expect(await screen.findByRole("alert")).toHaveTextContent("Không thể tải thông tin nhân viên");
    expect(screen.queryByRole("button", { name: "Lưu thông tin" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Đổi mật khẩu" })).not.toBeInTheDocument();
    expect(mocks.updateAdvPartnerUser).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    await waitFor(() => expect(screen.getByLabelText("Họ và tên *")).toHaveValue(employee.fullname));
    expect(screen.getByLabelText("CCCD *")).toHaveValue(employee.cccd);
    expect(screen.getByLabelText("Email")).toHaveValue(employee.email);
    expect(screen.getByRole("button", { name: "Lưu thông tin" })).toBeEnabled();
    expect(mocks.getEmployeeById).toHaveBeenCalledTimes(2);
  });
  it("provides labeled bank inputs and accessible password visibility controls", async () => {
    mocks.mobile = mobile;
    renderSheet();
    expect(await screen.findByLabelText("Số tài khoản")).toHaveValue(employee.bank_account_number);
    expect(screen.getByLabelText("Tên chủ tài khoản")).toHaveValue(employee.bank_account_name);
    const password = screen.getByLabelText("Mật khẩu mới");
    expect(password).toHaveAttribute("type", "password");
    fireEvent.click(screen.getByRole("button", { name: "Hiện mật khẩu" }));
    expect(password).toHaveAttribute("type", "text");
    expect(screen.getByRole("button", { name: "Ẩn mật khẩu" })).toBeInTheDocument();
  });
});
