import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import UsersPage from "./UsersPage";
import UsersPageMobile from "@/pages/mobile/admin/UsersPage";
import LoansPage from "./LoansPage";
import LoansPageMobile from "@/pages/mobile/admin/LoansPage";
import AuditLogPage from "./AuditLogPage";
import AuditLogPageMobile from "@/pages/mobile/admin/AuditLogPage";
import CronHealthPage from "./CronHealthPage";
import CronHealthPageMobile from "@/pages/mobile/admin/CronHealthPage";

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
  refetch: vi.fn(), fetchNextPage: vi.fn(), setPageTitle: vi.fn(),
  useInfiniteScroll: vi.fn(() => ({ observerRef: { current: null } })),
}));
vi.mock("@/contexts", () => ({
  useAppState: () => ({ setPageTitle: mocks.setPageTitle }),
  useAuth: () => ({ user: { role: "admin" } }),
}));
vi.mock("@/hooks/api/useLoans", () => ({
  useLoans: () => mocks.query, useLenders: () => ({ data: undefined }),
}));
vi.mock("@/hooks/api/useAuditLogs", () => ({ useInfiniteAuditLogs: () => mocks.query }));
vi.mock("@/hooks/api/useCronHealth", () => ({
  useCronJobs: () => mocks.query, useToggleCronJob: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@/hooks/users/useUserDataInfinite", () => ({ useUserDataInfinite: () => mocks.query }));
vi.mock("@/hooks/api/useUsers", () => ({
  useUsersSummary: () => ({ data: undefined, isLoading: false }),
  useResetPassword: () => ({ mutate: vi.fn() }),
}));
vi.mock("@/hooks/useModalNavigation", () => ({
  useUserModals: () => ({ openUserDetails: vi.fn(), openAddUser: vi.fn() }),
}));
vi.mock("@/hooks/use-infinite-scroll", () => ({ useInfiniteScroll: mocks.useInfiniteScroll }));
vi.mock("@/hooks/use-infinite-scroll.tsx", () => ({ useInfiniteScroll: mocks.useInfiniteScroll }));
vi.mock("@/components/shared/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/users/UserPageHeader", () => ({ UserPageHeader: () => <h1>Người dùng</h1> }));
vi.mock("@/components/users/UserStatsCards", () => ({ UserStatsCards: () => null }));
vi.mock("@/components/users/UserFiltersBar", () => ({ UserFiltersBar: () => null }));
vi.mock("@/components/users/UserMobileList", () => ({
  UserMobileList: ({ users, emptyState }: { users: { id: number; username: string }[]; emptyState: React.ReactNode }) => (
    <div>{users.length ? users.map((u) => <p key={u.id}>{u.username}</p>) : emptyState}</div>
  ),
}));
vi.mock("@/components/ui/responsive-table", () => ({
  ResponsiveTable: ({ data, emptyState }: { data: { id: number; username?: string }[]; emptyState: React.ReactNode }) => (
    <div>{data.length ? data.map((u) => <p key={u.id}>{u.username}</p>) : emptyState}</div>
  ),
}));
vi.mock("@/config/user-table-columns", () => ({ createUserColumns: () => [] }));
vi.mock("@/config/user-table-mobile", () => ({ createUserMobileConfig: () => ({}) }));
vi.mock("@/components/sheets/AddLoanSheet", () => ({ AddLoanSheet: () => null }));
vi.mock("@/components/sheets/LoanDetailsSheet", () => ({ LoanDetailsSheet: () => null }));
vi.mock("@/components/sheets/LenderManagementSheet", () => ({ LenderManagementSheet: () => null }));
vi.mock("./AuditLogPage/AuditLogDetailSheet", () => ({ AuditLogDetailSheet: () => null }));
vi.mock("./AuditLogPage/AuditLogFilters", () => ({ AuditLogFilters: () => null }));
vi.mock("./AuditLogPage/AuditLogCard", () => ({
  AuditLogCard: ({ log }: { log: { id: number } }) => <p>Bản ghi {log.id}</p>,
  AuditLogCardSkeleton: () => <div>Đang tải nhật ký</div>,
}));
vi.mock("@/components/cron-health", () => ({
  CronJobTable: () => <p>Danh sách tác vụ</p>, CronJobCard: () => <p>Tác vụ</p>,
  computeCronSummary: () => ({ total: 0, enabled: 0, failed: 0 }),
}));
const views = [
  ["users desktop", UsersPage], ["users mobile", UsersPageMobile],
  ["loans desktop", LoansPage], ["loans mobile", LoansPageMobile],
  ["audit desktop", AuditLogPage], ["audit mobile", AuditLogPageMobile],
  ["cron desktop", CronHealthPage], ["cron mobile", CronHealthPageMobile],
] as const;
beforeEach(() => {
  vi.clearAllMocks();
  mocks.query = { data: undefined, users: [], error: new Error("Network failure"), isError: true,
    isLoading: false, isFetchingNextPage: false, isFetchNextPageError: false,
    hasMore: false, hasNextPage: false, refetch: mocks.refetch, fetchNextPage: mocks.fetchNextPage };
});
describe.each(views)("%s failure state", (_label, Component) => {
  it("announces request failure without inventing an empty result and retries", () => {
    render(<MemoryRouter><Component /></MemoryRouter>);
    expect(screen.getByRole("alert")).toHaveTextContent("Không thể tải");
    expect(screen.queryByText(/^(Không có dữ liệu|Không có bản ghi nào|Không tìm thấy người dùng nào|Chưa có khoản vay nào)$/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
    expect(mocks.fetchNextPage).not.toHaveBeenCalled();
  });
});
describe.each(views.filter(([name]) => name.startsWith("users") || name.startsWith("audit")))("%s next-page failure", (label, Component) => {
  it("preserves loaded records and retries the failed next page", () => {
    mocks.query = { ...mocks.query, isFetchNextPageError: true, hasMore: true, hasNextPage: true,
      users: [{ id: 1, username: "loaded-user", role: "employee" }],
      data: { pages: [{ data: [{ id: 1 }], pagination: { totalRecords: 2 } }] } };
    render(<MemoryRouter><Component /></MemoryRouter>);
    expect(screen.getByText(label.startsWith("users") ? "loaded-user" : "Bản ghi 1")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("đã tải vẫn được giữ lại");
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(mocks.fetchNextPage).toHaveBeenCalledOnce();
    expect(mocks.refetch).not.toHaveBeenCalled();
    if (label !== "users mobile") {
      expect(mocks.useInfiniteScroll).toHaveBeenLastCalledWith(expect.objectContaining({ hasMore: false }));
    }
  });
});
