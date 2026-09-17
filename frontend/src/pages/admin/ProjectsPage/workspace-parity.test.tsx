import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectFilters } from "@/types/api/project.types";
import AdminProjectsPage from "./index";
import AdminProjectsPageMobile from "@/pages/mobile/admin/ProjectsPage";
import PartnerProjectsPage from "@/pages/partner/ProjectsPage";
import PartnerProjectsPageMobile from "@/pages/mobile/partner/ProjectsPage";

const mocks = vi.hoisted(() => ({
  useProjects: vi.fn(), refetch: vi.fn(), openCreateProject: vi.fn(),
  openProjectDetails: vi.fn(), openPartnerProjectDetails: vi.fn(), navigate: vi.fn(),
}));
vi.mock("react-router-dom", async (importOriginal) => ({
  ...await importOriginal<typeof import("react-router-dom")>(), useNavigate: () => mocks.navigate,
}));
vi.mock("@/hooks/api/useProjects", () => ({ useProjects: mocks.useProjects }));
vi.mock("@/hooks/partner/usePartnerProjectSummary", () => ({
  usePartnerProjectSummary: () => ({ data: undefined, isLoading: false }),
}));
vi.mock("@/hooks/useModalNavigation", () => ({ useProjectModals: () => mocks }));
vi.mock("@/components/projects/ProjectPageHeader", () => ({ ProjectPageHeader: () => <h1>Dự án</h1> }));
vi.mock("@/components/shared/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <header><h1>{title}</h1>{actions}</header>,
}));
vi.mock("@/components/projects/ProjectFilters", () => ({ ProjectFilters: () => null }));
vi.mock("@/components/ui/responsive-table", () => ({
  ResponsiveTable: ({ data, emptyState }: { data: Project[]; emptyState: React.ReactNode }) => (
    <div>{data.length ? data.map((p) => <p key={p.id}>{p.name}</p>) : emptyState}</div>
  ),
}));
vi.mock("@/config/project-table-columns", () => ({ createProjectColumns: () => [] }));
vi.mock("@/config/project-table-mobile", () => ({ createProjectMobileConfig: () => ({}) }));
const project = { id: 101, name: "Dự án 101", code: "PR101", status: "active", employee_count: 10 } as Project;
function queryResult(overrides: Record<string, unknown> = {}) {
  return {
    data: { data: [project], pagination: { page: 1, pageSize: 20, totalPages: 6, totalRecords: 101 } },
    isLoading: false, isFetching: false, isError: false, refetch: mocks.refetch, ...overrides,
  };
}
function renderPage(Component: React.ComponentType, url = "/") {
  return render(<MemoryRouter initialEntries={[url]}><Component /></MemoryRouter>);
}
const views = [
  ["Admin desktop", AdminProjectsPage], ["Admin mobile", AdminProjectsPageMobile],
  ["Partner desktop", PartnerProjectsPage], ["Partner mobile", PartnerProjectsPageMobile],
] as const;
beforeEach(() => { vi.clearAllMocks(); mocks.useProjects.mockReturnValue(queryResult()); });

describe.each(views)("%s project workspace", (_label, Component) => {
  it("distinguishes a failed request from an empty project list and offers retry", () => {
    mocks.useProjects.mockReturnValue(queryResult({ data: undefined, isError: true }));
    renderPage(Component);
    expect(screen.getByRole("alert")).toHaveTextContent("Không thể tải danh sách dự án");
    expect(screen.queryByText(/Không (có|tìm thấy) dự án nào/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
});
describe("Admin mobile pagination", () => {
  it("requests subsequent server pages so projects beyond the first 100 remain reachable", () => {
    mocks.useProjects.mockImplementation((filters: ProjectFilters) => queryResult({
      data: { data: [{ ...project, name: `Dự án trang ${filters.page}` }],
        pagination: { page: filters.page, pageSize: filters.pageSize, totalPages: 6, totalRecords: 101 } },
    }));
    renderPage(AdminProjectsPageMobile);
    expect(mocks.useProjects).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, pageSize: 20 }));
    expect(screen.getByRole("button", { name: "Trang dự án trước" })).toBeDisabled();
    for (let page = 2; page <= 6; page++) {
      fireEvent.click(screen.getByRole("button", { name: "Trang dự án sau" }));
      expect(mocks.useProjects).toHaveBeenLastCalledWith(expect.objectContaining({ page }));
      expect(screen.getByText(`Dự án trang ${page}`)).toBeInTheDocument();
    }
    expect(screen.getByRole("button", { name: "Trang dự án sau" })).toBeDisabled();
    expect(screen.getByText("101 dự án")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Trang dự án trước" }));
    expect(mocks.useProjects).toHaveBeenLastCalledWith(expect.objectContaining({ page: 5 }));
  });
  it("disables page changes while the next page is fetching", () => {
    mocks.useProjects.mockReturnValue(queryResult({ isFetching: true }));
    renderPage(AdminProjectsPageMobile);
    expect(screen.getByRole("button", { name: "Trang dự án sau" })).toBeDisabled();
  });
});
describe.each([["desktop", AdminProjectsPage], ["mobile", AdminProjectsPageMobile]] as const)("Admin %s dashboard filter", (_label, Component) => {
  it("applies the URL status once without a render loop and allows clearing it", () => {
    mocks.useProjects.mockReturnValue(queryResult({ data: { data: [], pagination: { page: 1, pageSize: 20, totalPages: 0, totalRecords: 0 } } }));
    renderPage(Component, "/admin/projects?status=active");
    expect(mocks.useProjects).toHaveBeenLastCalledWith(expect.objectContaining({ status: ["active"] }));
    expect(mocks.useProjects.mock.calls.length).toBeLessThan(10);
    fireEvent.click(screen.getByRole("button", { name: "Xóa bộ lọc" }));
    expect(mocks.useProjects).toHaveBeenLastCalledWith(expect.not.objectContaining({ status: expect.anything() }));
  });
});
describe.each([["desktop", PartnerProjectsPage], ["mobile", PartnerProjectsPageMobile]] as const)("Partner %s project actions", (_label, Component) => {
  it("keeps the timesheet keyboard action separate from project details", () => {
    renderPage(Component);
    const timesheet = screen.getByRole("button", { name: "Bảng công" });
    fireEvent.keyDown(timesheet, { key: "Enter" });
    fireEvent.click(timesheet);
    expect(mocks.openPartnerProjectDetails).not.toHaveBeenCalled();
    expect(mocks.navigate).toHaveBeenCalledWith("/partner/timesheet?project=101");
    fireEvent.click(screen.getByRole("button", { name: "Xem chi tiết dự án Dự án 101" }));
    expect(mocks.openPartnerProjectDetails).toHaveBeenCalledWith("101");
  });
});
