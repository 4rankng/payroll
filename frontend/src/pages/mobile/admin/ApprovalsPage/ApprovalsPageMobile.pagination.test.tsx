import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ApprovalsPageMobile from "./index";

const mocks = vi.hoisted(() => ({
  listCalls: [] as Record<string, unknown>[],
  confirm: vi.fn(),
}));

vi.mock("@/hooks/api/useApprovals", () => ({
  useApprovalsList: (filters: Record<string, unknown>) => {
    mocks.listCalls.push(filters);
    if (filters.pageSize === 1) {
      return {
        data: { items: [], pagination: { page: 1, pageSize: 1, totalPages: 1, totalRecords: 66 } },
        isLoading: false,
        isError: false,
        isFetching: false,
        refetch: vi.fn(),
      };
    }
    return {
      data: {
        items: [
          { id: 1, summary: "Yêu cầu A", type_label: "A", status: "pending", status_label: "Chờ duyệt", requester_name: "an", created_at: "2026-10-28T10:00:00Z", retry_count: 0 },
          { id: 2, summary: "Yêu cầu B", type_label: "B", status: "pending", status_label: "Chờ duyệt", requester_name: "binh", created_at: "2026-10-28T11:00:00Z", retry_count: 0 },
        ],
        pagination: { page: filters.page, pageSize: 15, totalPages: 5, totalRecords: 66 },
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    };
  },
  useApproveApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useRejectApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useCancelApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useFileApproval: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/components/approvals/ApprovalDecisionModal", () => ({
  ApprovalDecisionModal: (props: { open: boolean; onClose: () => void }) =>
    props.open ? (
      <div role="dialog" aria-label="Quyết định">
        <button type="button" onClick={props.onClose}>Đóng</button>
      </div>
    ) : null,
}));

describe("ApprovalsPageMobile pagination (PAYR-19)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listCalls.length = 0;
  });

  it("không còn cap 15: hiển thị phân trang đầy đủ, không copy 'mở bản máy tính'", () => {
    render(<ApprovalsPageMobile />);
    expect(screen.getAllByText("Yêu cầu A").length).toBeGreaterThan(0);
    expect(screen.getByText(/Trang 1 \/ 5/)).toBeTruthy();
    expect(screen.getAllByText(/66 yêu cầu/).length).toBeGreaterThan(0);
    expect(document.body.textContent).not.toContain("mở bản máy tính");
  });

  it("bấm Sau chuyển trang 2; đổi filter reset về trang 1", () => {
    render(<ApprovalsPageMobile />);
    fireEvent.click(screen.getByRole("button", { name: /Sau/ }));
    const listCalls = mocks.listCalls.filter((f) => f.pageSize !== 1);
    expect(listCalls[listCalls.length - 1].page).toBe(2);

    fireEvent.click(screen.getByRole("tab", { name: "Từ chối" }));
    const afterFilter = mocks.listCalls.filter((f) => f.pageSize !== 1);
    expect(afterFilter[afterFilter.length - 1].page).toBe(1);
  });

  it("Duyệt/Từ chối hiện trên mọi item hiển thị; filter chips giữ nguyên", () => {
    render(<ApprovalsPageMobile />);
    expect(screen.getAllByRole("button", { name: "Duyệt" })).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "Từ chối" })).toHaveLength(2);
    for (const label of ["Chờ duyệt", "Đã duyệt", "Từ chối", "Leo thang", "Tất cả"]) {
      expect(screen.getByRole("tab", { name: label })).toBeTruthy();
    }
  });
});
