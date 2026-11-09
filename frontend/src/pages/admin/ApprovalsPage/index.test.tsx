import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import ApprovalsPage from "./index";

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
}));

vi.mock("@/hooks/api/useApprovals", () => ({
  useApprovalsList: (filters: Record<string, unknown>) => ({
    data: {
      items:
        (filters.statuses as string[] | undefined)?.length === 0
          ? [
              { id: 1, summary: "Yêu cầu A", type_label: "A", status: "pending", status_label: "Chờ duyệt", requester_name: "an", created_at: "2026-10-28T10:00:00Z", retry_count: 0 },
            ]
          : [],
      pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 0 },
    },
    isLoading: false,
    isError: false,
    refetch: mocks.refetch,
  }),
  useApproveApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useRejectApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useCancelApproval: () => ({ mutate: vi.fn(), isPending: false }),
  useFileApproval: () => ({ mutate: vi.fn(), isPending: false }),
}));

describe("ApprovalsPage tab strip (PAYR-34)", () => {
  it("TabsList không khóa chiều cao cứng — hết vertical scrollbar thừa bên phải tab 'Tất cả'", () => {
    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    const tabsList = screen.getByRole("tablist");
    // Base TabsList đã có min-h-11 (≥44px touch target); h-11 cứng + content
    // 44px vừa khít sinh vertical scrollbar — cấm quay lại.
    expect(tabsList).not.toHaveClass("h-11");
    expect(tabsList).toHaveClass("min-h-11");
    // Scroll ngang cho màn hẹp vẫn giữ nguyên.
    expect(tabsList).toHaveClass("overflow-x-auto", "w-full", "justify-start", "sm:w-auto");

    for (const label of ["Chờ duyệt", "Đã duyệt", "Từ chối", "Leo thang", "Tất cả"]) {
      expect(within(tabsList).getByRole("tab", { name: label })).toBeInTheDocument();
    }
  });
});
