import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProjectUserAccessTab } from "./ProjectUserAccessTab";
import type { Project, ProjectUser } from "@/types/api/project.types";

let currentRole: "admin" | "partner" = "partner";
let currentUserId = 200;
let projectUsers: ProjectUser[] = [];

const grantMock = vi.fn();
const revokeMock = vi.fn();

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
}));

vi.mock("@/components/ui/user-selector", () => ({
  UserSelector: ({ onValueChange }: { onValueChange: (value: string) => void }) => (
    <button type="button" onClick={() => onValueChange("999")}>
      chọn người dùng
    </button>
  ),
}));

vi.mock("@/components/ui/confirm-dialog", () => ({
  ConfirmDialog: ({ open, title }: { open: boolean; title: string }) =>
    open ? <div>{title}</div> : null,
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

  it("partner không quản lý dự án chỉ thấy thông báo không có quyền, không có nút cấp/xóa", () => {
    currentUserId = 300;
    projectUsers = [creatorRow, managerRow];

    render(<ProjectUserAccessTab project={project} />);

    expect(screen.getByText("Không có quyền truy cập")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /cấp quyền/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /thu hồi quyền của/i })).not.toBeInTheDocument();
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
});
