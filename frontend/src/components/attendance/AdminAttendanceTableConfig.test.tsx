import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";
import { getAdminAttendanceColumns } from "./AdminAttendanceTableConfig";

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuTrigger: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, onClick }: { children: ReactNode; onClick?: () => void }) => (
    <button type="button" role="menuitem" onClick={onClick}>{children}</button>
  ),
}));

describe("getAdminAttendanceColumns", () => {
  it("surfaces an admin-approved attendance missing checkout as needing repair", () => {
    const attendance = {
      id: 8,
      status: "checked_in",
      review_action: "approved",
    } as AdminAttendanceResponse;
    const columns = getAdminAttendanceColumns();
    const statusColumn = columns.find((column) => "accessorKey" in column && column.accessorKey === "status");
    const statusCell = statusColumn?.cell;
    if (typeof statusCell !== "function") {
      throw new Error("Expected desktop attendance status cell");
    }

    render(<>{statusCell({ row: { original: attendance } } as never)}</>);

    expect(screen.getByText("Cần duyệt lại")).toBeInTheDocument();
    expect(screen.queryByText("Hoàn thành")).not.toBeInTheDocument();
  });

  it("does not allow rejecting an admin-approved attendance that needs checkout repair", () => {
    const attendance = {
      id: 8,
      status: "checked_in",
      review_action: "approved",
    } as AdminAttendanceResponse;
    const columns = getAdminAttendanceColumns({
      onViewMap: vi.fn(),
      onApprove: vi.fn(),
      onReject: vi.fn(),
    });
    const actionsColumn = columns.find((column) => column.id === "actions");
    const cell = actionsColumn?.cell;
    if (typeof cell !== "function") {
      throw new Error("Expected desktop attendance actions cell");
    }

    render(<>{cell({ row: { original: attendance } } as never)}</>);

    expect(screen.queryByRole("menuitem", { name: "Từ chối" })).not.toBeInTheDocument();
  });

  it("shows a physically completed admin approval as completed on desktop", () => {
    const attendance = {
      id: 11,
      status: "completed",
      review_action: "approved",
      check_out_time: "2026-08-05T10:00:00+07:00",
    } as AdminAttendanceResponse;
    const columns = getAdminAttendanceColumns();
    const statusColumn = columns.find((column) => "accessorKey" in column && column.accessorKey === "status");
    const statusCell = statusColumn?.cell;
    if (typeof statusCell !== "function") {
      throw new Error("Expected desktop attendance status cell");
    }

    render(<>{statusCell({ row: { original: attendance } } as never)}</>);

    expect(screen.getByText("Hoàn thành")).toBeInTheDocument();
    expect(screen.getByText("Đã duyệt")).toBeInTheDocument();
    expect(screen.queryByText("Cần duyệt lại")).not.toBeInTheDocument();
  });

  it("does not allow rejecting a physically completed attendance after quota credit", () => {
    const attendance = {
      id: 10,
      status: "completed",
      review_action: null,
      quota_credited_at: "2026-08-05T01:00:00+07:00",
    } as AdminAttendanceResponse;
    const columns = getAdminAttendanceColumns({
      onViewMap: vi.fn(),
      onApprove: vi.fn(),
      onReject: vi.fn(),
    });
    const actionsColumn = columns.find((column) => column.id === "actions");
    const cell = actionsColumn?.cell;
    if (typeof cell !== "function") {
      throw new Error("Expected desktop attendance actions cell");
    }

    render(<>{cell({ row: { original: attendance } } as never)}</>);

    expect(screen.queryByRole("menuitem", { name: "Từ chối" })).not.toBeInTheDocument();
  });

  it("offers desktop repair when an approval has no persisted checkout", () => {
    const onApprove = vi.fn();
    const attendance = {
      id: 9,
      status: "completed",
      review_action: "approved",
    } as AdminAttendanceResponse;
    const columns = getAdminAttendanceColumns({
      onViewMap: vi.fn(),
      onApprove,
      onReject: vi.fn(),
    });
    const actionsColumn = columns.find((column) => column.id === "actions");

    const statusColumn = columns.find((column) => "accessorKey" in column && column.accessorKey === "status");
    const statusCell = statusColumn?.cell;
    if (typeof statusCell !== "function") {
      throw new Error("Expected desktop attendance status cell");
    }
    const statusView = render(<>{statusCell({ row: { original: attendance } } as never)}</>);
    expect(screen.getByText("Cần duyệt lại")).toBeInTheDocument();
    expect(screen.queryByText("Đã từ chối")).not.toBeInTheDocument();
    statusView.unmount();

    expect(actionsColumn?.cell).toBeTypeOf("function");
    const cell = actionsColumn?.cell;
    if (typeof cell !== "function") {
      throw new Error("Expected desktop attendance actions cell");
    }

    render(<>{cell({ row: { original: attendance } } as never)}</>);
    const repairAction = screen.getByRole("menuitem", { name: "Duyệt lại" });
    fireEvent.click(repairAction);

    expect(onApprove).toHaveBeenCalledWith(attendance);
  });
});
