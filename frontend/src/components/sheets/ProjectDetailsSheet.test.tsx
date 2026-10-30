import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import ProjectDetailsSheet from "./ProjectDetailsSheet";
import type { Project, ProjectUser } from "@/types/api/project.types";

const navigateMock = vi.fn();
const closeModalMock = vi.fn();
const invalidateQueriesMock = vi.fn();

let currentRole: "admin" | "partner" = "admin";

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
  created_by: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
} satisfies Project;

const managerRow: ProjectUser = {
  id: 7,
  user_id: 999,
  user_fullname: "Quản Lý Đối Tác",
  user_email: "manager@test.local",
  granted_by: 1,
  granted_at: "2026-01-02T00:00:00Z",
};

// accessManage.list feeds the REAL useProjectAccessManagement hook through the
// mocked useProjectUsers, so gate tests exercise the real membership logic.
const { mockUseProject, accessManage } = vi.hoisted(() => ({
  mockUseProject: vi.fn(),
  accessManage: { list: [] as unknown[] },
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: invalidateQueriesMock,
  }),
}));

vi.mock("@/lib/auth", () => ({
  authManager: {
    getUserRole: () => currentRole,
    getUserId: () => 999,
  },
}));

vi.mock("@/hooks/useModalNavigation", () => ({
  useModalNavigation: () => ({
    closeModal: closeModalMock,
  }),
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useProject: mockUseProject,
  useProjectUsers: () => ({ data: accessManage.list, isLoading: false }),
  usePauseProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useResumeProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useStartProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCompleteProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCancelProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteProject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useProjectApprovedTimesheets: () => ({ data: { data: [] }, isLoading: false }),
}));

vi.mock("./templates/SlideSheetTemplate", () => ({
  SlideSheetTemplate: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: ReactNode }) => <button type="button">{children}</button>,
  TabsContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/projects/details/ProjectHeader", () => ({
  ProjectHeader: () => <div>Header</div>,
}));

vi.mock("@/components/projects/details/ProjectInfoTab", () => ({
  ProjectInfoTab: () => <div>Project info</div>,
}));

vi.mock("@/components/projects/details/ProjectUserAccessTab", () => ({
  ProjectUserAccessTab: () => <div>Project access</div>,
}));

vi.mock("@/components/projects/details/GeofenceSection", () => ({
  GeofenceSection: () => <div>Geofence</div>,
}));

vi.mock("@/components/shared/GroupedStatCard", () => ({
  GroupedStatCard: () => <div>Stats</div>,
}));

vi.mock("@/components/project-employees/AddEmployeesToProject", () => ({
  AddEmployeesToProject: () => null,
}));

vi.mock("@/components/project-employees/ProjectEmployeesList", () => ({
  ProjectEmployeesList: () => null,
}));

vi.mock("./ProjectEditSheet", () => ({
  default: () => null,
}));

vi.mock("@/components/ui/confirm-dialog", () => ({
  ConfirmDialog: () => null,
}));

vi.mock("@/components/ui/skeleton", () => ({
  Skeleton: () => <div />,
}));

vi.mock("@/components/payrates/PayrateConfigTab", () => ({
  PayrateConfigTab: () => <div>Payrates</div>,
}));

vi.mock("@/components/projects/details/ShiftNamesSection", () => ({
  ShiftNamesSection: ({
    canEdit,
    onEditPayrate,
  }: {
    canEdit: boolean;
    onEditPayrate: (payrateId: number) => void;
  }) => (
    <div>
      <span>{canEdit ? "editable" : "read only"}</span>
      <button type="button" onClick={() => onEditPayrate(91)}>
        Trigger payrate edit
      </button>
    </div>
  ),
}));

describe("ProjectDetailsSheet", () => {
  beforeEach(() => {
    navigateMock.mockReset();
    closeModalMock.mockReset();
    invalidateQueriesMock.mockReset();
    accessManage.list = [];
    mockUseProject.mockReturnValue({
      data: null,
      isLoading: false,
    });
  });

  it("navigates admins to the admin payrate editor", () => {
    currentRole = "admin";

    render(
      <ProjectDetailsSheet
        project={project}
        isOpen
        onClose={vi.fn()}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Trigger payrate edit" }));

    expect(navigateMock).toHaveBeenCalledWith("/admin/projects/58/payrates/91/edit");
  });

  it("navigates partners to the partner payrate editor", () => {
    currentRole = "partner";

    render(
      <ProjectDetailsSheet
        project={project}
        isOpen
        onClose={vi.fn()}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Trigger payrate edit" }));

    expect(navigateMock).toHaveBeenCalledWith("/partner/projects/58/payrates/91/edit");
  });

  it("shows the Phân quyền tab to a partner who holds a project grant (PAYR-30)", () => {
    currentRole = "partner";
    accessManage.list = [managerRow];

    render(
      <ProjectDetailsSheet
        project={project}
        isOpen
        onClose={vi.fn()}
      />
    );

    expect(screen.getAllByText("Phân quyền").length).toBeGreaterThan(0);
  });

  it("hides the Phân quyền tab from a partner without a project grant", () => {
    currentRole = "partner";
    accessManage.list = [];

    render(
      <ProjectDetailsSheet
        project={project}
        isOpen
        onClose={vi.fn()}
      />
    );

    expect(screen.queryByText("Phân quyền")).not.toBeInTheDocument();
  });

  it("survives the deep-link loading → loaded transition (hook order, PAYR-30)", () => {
    currentRole = "partner";
    accessManage.list = [managerRow];
    mockUseProject.mockReturnValue({ data: undefined, isLoading: true });

    const { rerender } = render(<ProjectDetailsSheet id="58" isOpen onClose={vi.fn()} />);
    expect(screen.queryByText("Project info")).not.toBeInTheDocument();

    mockUseProject.mockReturnValue({ data: project, isLoading: false });
    rerender(<ProjectDetailsSheet id="58" isOpen onClose={vi.fn()} />);

    expect(screen.getByText("Project info")).toBeInTheDocument();
    expect(screen.getAllByText("Phân quyền").length).toBeGreaterThan(0);
  });
});
