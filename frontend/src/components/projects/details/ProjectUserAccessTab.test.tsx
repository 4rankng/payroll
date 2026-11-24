import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProjectUserAccessTab } from "./ProjectUserAccessTab";
import type { Project, ProjectUser } from "@/types/api/project.types";

let currentRole: "admin" | "partner" = "partner";
let currentUserId = 200;
let projectUsers: ProjectUser[] = [];

const grantMock = vi.fn();
const revokeMock = vi.fn();
const requestAccessMock = vi.fn();
const releaseAccessMock = vi.fn();

vi.mock("@/lib/auth", () => ({
  authManager: {
    getUserRole: () => currentRole,
    getUserId: () => currentUserId,
  },
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useProjectUsers: () => ({ data: projectUsers, isLoading: false }),
  useGrantProjectAccess: () => ({ mutateAsync: grantMock, isPending: false }),
  useRevokeProjectAccess: () => ({ mutateAsync: revokeMock, isPending: false }),
  useRequestProjectAccess: () => ({ mutateAsync: requestAccessMock, isPending: false }),
  useReleaseProjectAccess: () => ({ mutateAsync: releaseAccessMock, isPending: false }),
}));

vi.mock("@/components/ui/user-selector", () => ({
  UserSelector: ({ onValueChange }: { onValueChange: (value: string) => void }) => (
    <button type="button" onClick={() => onValueChange("999")}>
      chọn người dùng
    </button>
  ),
}));

vi.mock("@/components/ui/confirm-dialog", () => ({
  ConfirmDialog: ({
    open,
    title,
    onConfirm,
    confirmText,
  }: {
    open: boolean;
    title: string;
    onConfirm: () => void;
    confirmText?: string;
  }) =>
    open ? (
      <div>
        {title}
        <button type="button" onClick={onConfirm}>
          {confirmText ?? "confirm"}
        </button>
      </div>
    ) : null,
}));

const project = {
  id: 58,
  name: "Dự án thử nghiệm",
  code: "DA58",
  client_name: "Khách hàng",
  start_date: "2026-01-01",
  end_date: null,
  employee_count: 0,
  weekly_salary_employee_count: 0,
  monthly_salary_employee_count: 0,
  status: "active",
  is_flexible: true,
  shift_names: [{ range: "08:00-17:00", name: "Ca ngày" }],
  created_by: 100,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
} satisfies Project;

const creatorRow: ProjectUser = {
  id: 1,
  user_id: 100,
  user_fullname: "Người Tạo Dự Án",
  user_email: "creator@test.local",
  granted_by: 100,
  granted_at: "2026-01-01T00:00:00Z",
};

const managerRow: ProjectUser = {
  id: 2,
  user_id: 200,
  user_fullname: "Quản Lý Đối Tác",
  user_email: "manager@test.local",
  granted_by: 100,
  granted_at: "2026-01-02T00:00:00Z",
};

describe("ProjectUserAccessTab (PAYR-30)", () => {
  beforeEach(() => {
    grantMock.mockReset();
    revokeMock.mockReset();
    requestAccessMock.mockReset();
    releaseAccessMock.mockReset();
    currentRole = "partner";
    currentUserId = 200;
    projectUsers = [creatorRow, managerRow];
  });

  it("partner-admin được cấp quyền (không phải người tạo) thấy ô cấp quyền và nút thu hồi", () => {
    render(<ProjectUserAccessTab project={project} />);

    expect(screen.getByText("Cấp quyền quản lý")).toBeInTheDocument();
    expect(screen.queryByText("Không có quyền truy cập")).not.toBeInTheDocument();

    // Grant button only waits for a target selection, not for role/creator-ship.
    expect(screen.getByRole("button", { name: /cấp quyền/i })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: /chọn người dùng/i }));
    expect(screen.getByRole("button", { name: /cấp quyền/i })).toBeEnabled();

    expect(screen.getByRole("button", { name: /thu hồi quyền của quản lý đối tác/i })).toBeInTheDocument();
  });

  it("partner không quản lý dự án thấy thông báo không có quyền + nút yêu cầu quản lý (tự cấp ngay), không có nút cấp/xóa", () => {
    currentUserId = 300;
    projectUsers = [creatorRow, managerRow];

    render(<ProjectUserAccessTab project={project} />);

    expect(screen.getByText("Không có quyền truy cập")).toBeInTheDocument();
    // Self-service claim is instant (owner ruling 2026-11-24) — the button
    // is the only action offered, grant/revoke UI stays hidden.
    expect(screen.getByRole("button", { name: "Yêu cầu quản lý dự án" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /cấp quyền/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /thu hồi quyền của/i })).not.toBeInTheDocument();
  });

  it("partner không quản lý thấy nút yêu cầu quản lý gọi request-access ngay lập tức", async () => {
    currentUserId = 300;
    projectUsers = [creatorRow, managerRow];

    render(<ProjectUserAccessTab project={project} />);

    fireEvent.click(screen.getByRole("button", { name: "Yêu cầu quản lý dự án" }));
    expect(requestAccessMock).toHaveBeenCalledWith(58);
  });

  it("admin thấy nút xóa trên cả row Người tạo", () => {
    currentRole = "admin";
    currentUserId = 1;
    projectUsers = [creatorRow];

    render(<ProjectUserAccessTab project={project} />);

    expect(screen.getByRole("button", { name: /thu hồi quyền của người tạo dự án/i })).toBeInTheDocument();
  });

  it("partner-admin không thấy nút xóa trên row Người tạo", () => {
    render(<ProjectUserAccessTab project={project} />);

    expect(screen.queryByRole("button", { name: /thu hồi quyền của người tạo dự án/i })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /thu hồi quyền của quản lý đối tác/i }));
    expect(screen.getByText("Thu hồi quyền truy cập")).toBeInTheDocument();
  });

  it("partner quản lý qua cấp quyền (không phải người tạo) thấy nút Yêu cầu loại khỏi dự án", () => {
    render(<ProjectUserAccessTab project={project} />);

    expect(screen.getByRole("button", { name: "Yêu cầu loại khỏi dự án" })).toBeInTheDocument();
  });

  it("yêu cầu loại khỏi dự án gọi release-access và báo cho sheet đóng", async () => {
    const onSelfReleased = vi.fn();
    releaseAccessMock.mockResolvedValue({ message: "Đã loại bạn khỏi quản lý dự án này" });

    render(<ProjectUserAccessTab project={project} onSelfReleased={onSelfReleased} />);

    fireEvent.click(screen.getByRole("button", { name: "Yêu cầu loại khỏi dự án" }));
    // ConfirmDialog mock wires onConfirm to a button labelled by confirmText.
    fireEvent.click(screen.getByRole("button", { name: "Loại khỏi dự án" }));
    await waitFor(() => expect(releaseAccessMock).toHaveBeenCalledWith(58));
    await waitFor(() => expect(onSelfReleased).toHaveBeenCalled());
  });

  it("người tạo dự án không thấy nút tự loại (chỉ admin thu hồi được quyền người tạo)", () => {
    currentUserId = 100; // creator

    render(<ProjectUserAccessTab project={project} />);

    expect(screen.queryByRole("button", { name: "Yêu cầu loại khỏi dự án" })).not.toBeInTheDocument();
  });

  it("admin không thấy nút tự loại", () => {
    currentRole = "admin";

    render(<ProjectUserAccessTab project={project} />);

    expect(screen.queryByRole("button", { name: "Yêu cầu loại khỏi dự án" })).not.toBeInTheDocument();
  });
});
