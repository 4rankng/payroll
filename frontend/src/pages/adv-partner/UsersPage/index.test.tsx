import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AdvPartnerUsersPage from "./index";
const mocks = vi.hoisted(() => ({ query: {} as Record<string, unknown>, refetch: vi.fn(), fetchNextPage: vi.fn() }));
vi.mock("@/hooks/api/useAdvancePayments", () => ({ useFlexPayEmployeesInfinite: () => mocks.query }));
vi.mock("./EditAdvPartnerUserSheet", () => ({ default: ({ fullname }: { fullname: string }) => <div role="dialog">{fullname}</div> }));
beforeEach(() => {
  vi.clearAllMocks();
  mocks.query = { data: undefined, isLoading: false, isError: false, isFetchingNextPage: false,
    isFetchNextPageError: false, hasNextPage: false, refetch: mocks.refetch, fetchNextPage: mocks.fetchNextPage };
});
describe("Advance partner responsive employee directory", () => {
  it("offers retry on initial failure without claiming no employees exist", () => {
    mocks.query.isError = true;
    render(<AdvPartnerUsersPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Không thể tải danh sách nhân viên");
    expect(screen.queryByText("Không tìm thấy nhân viên nào")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
  it("offers the same employee detail action and real load-more controls in both layouts", () => {
    mocks.query = { ...mocks.query, hasNextPage: true, data: { pages: [{ data: [{ employeeId: 1, fullname: "Nguyễn Văn An" }] }] } };
    render(<AdvPartnerUsersPage />);
    expect(screen.getByRole("searchbox", { name: "Tìm nhân viên theo tên hoặc CCCD" })).toBeInTheDocument();
    const actions = screen.getAllByRole("button", { name: "Chỉnh sửa nhân viên Nguyễn Văn An" });
    expect(actions).toHaveLength(2);
    fireEvent.click(actions[1]);
    expect(screen.getByRole("dialog")).toHaveTextContent("Nguyễn Văn An");
    const loadMore = screen.getAllByRole("button", { name: "Tải thêm nhân viên" });
    expect(loadMore).toHaveLength(2);
    fireEvent.click(loadMore[1]);
    expect(mocks.fetchNextPage).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "1" })).not.toBeInTheDocument();
  });
  it("keeps employees visible and retries only a failed next page", () => {
    mocks.query = { ...mocks.query, isError: true, isFetchNextPageError: true, hasNextPage: true,
      data: { pages: [{ data: [{ employeeId: 1, fullname: "Nguyễn Văn An" }] }] } };
    render(<AdvPartnerUsersPage />);
    expect(screen.getAllByRole("button", { name: "Chỉnh sửa nhân viên Nguyễn Văn An" })).toHaveLength(2);
    expect(screen.queryByRole("button", { name: "Tải thêm nhân viên" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(mocks.fetchNextPage).toHaveBeenCalledOnce();
    expect(mocks.refetch).not.toHaveBeenCalled();
  });
});
