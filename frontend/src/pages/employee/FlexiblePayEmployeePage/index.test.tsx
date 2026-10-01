import { act, render, screen } from "@testing-library/react";
import { useLocation, MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import FlexiblePayEmployeePage from ".";

const activeJulyInfo = {
  forMonth: "2026-07",
  maxAdvanceAmount: 1_000_000,
  completedAmount: 0,
  pendingAmount: 0,
  remainingAmount: 1_000_000,
  canRequest: true,
  feePercentage: 2,
  minFee: 10_000,
  hasFlexible: true,
  quotas: [{
    forMonth: "2026-07",
    maxAdvanceAmount: 1_000_000,
    completedAmount: 0,
    pendingAmount: 0,
    remainingAmount: 1_000_000,
  }],
};

// The self check-in surface names the period it serves. During the days 1-8
// tail that is the PREVIOUS month, and the regular flow refuses that period, so
// which month the backend reports is what decides the routing.
let checkInInfoQueryState: { data?: { data: { forMonth: string } } | undefined } = { data: undefined };

const profileQuery = {
  data: { fullname: "Đỗ Văn Hùng", check_in_enabled: false } as { fullname: string; check_in_enabled: boolean } | undefined,
  isLoading: false,
};

vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({ logout: vi.fn() }),
}));

vi.mock("@/hooks/api/useEmployeePortal", () => ({
  useEmployeeProfile: () => profileQuery,
  useUpdateEmployeePassword: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/api/useAdvancePayments", () => ({
  useAdvancePaymentInfo: () => ({
    data: { data: activeJulyInfo },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCheckInAdvanceInfo: () => checkInInfoQueryState,
  useAdvancePaymentHistory: () => ({
    data: { data: [], pagination: { totalRecords: 0 } },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useRequestAdvancePayment: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useRequestCheckInAdvance: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useCalculateFee: () => ({ isPending: false, mutate: vi.fn() }),
  useCancelAdvancePaymentRequest: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/api/useNotifications", () => ({ useUnreadNotifications: () => ({ data: { count: 0 } }) }));
vi.mock("@/components/employees/EmployeeMobileShell", () => ({ EmployeeMobileShell: ({ canopy, children }: { canopy?: ReactNode; children: ReactNode }) => <>{canopy}{children}</> }));
vi.mock("@/components/employees/EmployeeMonthNavigator", () => ({ EmployeeMonthNavigator: ({ month }: { month: { value: string } }) => <output aria-label="Tháng đang xem">{month.value}</output> }));
vi.mock("@/components/advance-payment/AdvancePaymentRequestForm", () => ({ AdvancePaymentRequestForm: ({ viewMonth, isPastMonth }: { viewMonth: string; isPastMonth: boolean }) => <output aria-label="Hạn mức đang xem">{`${viewMonth}:${isPastMonth}`}</output> }));
vi.mock("@/components/employees/EmployeeBankInfoCard", () => ({ EmployeeBankInfoCard: () => null }));
vi.mock("@/components/employees/EmployeeAdBanner", () => ({ EmployeeAdBanner: () => <div data-testid="employee-ad-banner" /> }));
vi.mock("@/components/employees/EmployeeCheckInCard", () => ({ EmployeeCheckInCard: () => null }));
vi.mock("@/components/employees/EmployeeAttendanceHistoryCard", () => ({ EmployeeAttendanceHistoryCard: () => null }));
vi.mock("@/components/employees/ChangePasswordSheet", () => ({ ChangePasswordSheet: () => null }));
vi.mock("@/components/advance-payment/AdvancePaymentHistoryCard", () => ({ AdvancePaymentHistoryCard: () => null }));
vi.mock("@/components/advance-payment/AdvancePaymentConfirmSheet", () => ({ AdvancePaymentConfirmSheet: () => null }));
vi.mock("@/components/notifications/NotificationSheet", () => ({ NotificationSheet: () => null }));

function LocationProbe() {
  const location = useLocation();
  return <output aria-label="Đường dẫn tháng">{location.search}</output>;
}

describe("FlexiblePayEmployeePage", () => {
  beforeEach(() => {
    profileQuery.data = { fullname: "Đỗ Văn Hùng", check_in_enabled: false };
    checkInInfoQueryState = { data: undefined };
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-05T12:00:00+07:00"));
  });

  afterEach(() => vi.useRealTimers());

  it("opens the server-active July period through the August 8 cutoff", async () => {
    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
        <LocationProbe />
      </MemoryRouter>,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByLabelText("Tháng đang xem")).toHaveTextContent("2026-07");
    expect(screen.getByLabelText("Hạn mức đang xem")).toHaveTextContent("2026-07:false");
    expect(screen.getByLabelText("Đường dẫn tháng")).toHaveTextContent("?month=2026-07");
  });

  it("corrects a future URL month to the server-active period", async () => {
    render(
      <MemoryRouter initialEntries={["/employee?month=2026-09"]}>
        <FlexiblePayEmployeePage />
        <LocationProbe />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByLabelText("Tháng đang xem")).toHaveTextContent("2026-07");
    expect(screen.getByLabelText("Đường dẫn tháng")).toHaveTextContent("?month=2026-07");
  });

  it("opens on the month the check-in surface serves during the 1-8 tail", async () => {
    // 2026-08-05: the check-in flow serves JULY (the regular flow refuses a
    // check-in period), so July is the only month the app can actually request
    // against. The check-in response names the period; the UI must follow it
    // instead of recomputing the calendar and dead-ending on the regular flow.
    profileQuery.data = { fullname: "Đỗ Văn Hùng", check_in_enabled: true };
    checkInInfoQueryState = { data: { data: { forMonth: "2026-07" } } };

    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
        <LocationProbe />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByLabelText("Tháng đang xem")).toHaveTextContent("2026-07");
  });

  it("keeps the calendar month once the check-in surface serves it", async () => {
    // From day 10 the check-in flow serves the CURRENT month, which is the
    // month the app opens on — the response confirms rather than redirects.
    profileQuery.data = { fullname: "Đỗ Văn Hùng", check_in_enabled: true };
    checkInInfoQueryState = { data: { data: { forMonth: "2026-08" } } };

    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
        <LocationProbe />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByLabelText("Tháng đang xem")).toHaveTextContent("2026-08");
    expect(screen.getByLabelText("Đường dẫn tháng")).toHaveTextContent("");
  });

  it("does not infer a non-check-in flow before the profile is available", async () => {
    profileQuery.data = undefined;

    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
        <LocationProbe />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByLabelText("Tháng đang xem")).toHaveTextContent("2026-08");
    expect(screen.getByLabelText("Đường dẫn tháng")).toHaveTextContent("");
  });

  it("shows the ad banner to employees without self check-in", async () => {
    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.getByTestId("employee-ad-banner")).toBeInTheDocument();
  });

  it("hides the ad banner from self check-in employees", async () => {
    // Self check-in employees live in the check-in feed; ad campaigns are not
    // run against them.
    profileQuery.data = { fullname: "Đỗ Văn Hùng", check_in_enabled: true };
    checkInInfoQueryState = { data: { data: { forMonth: "2026-07" } } };

    render(
      <MemoryRouter initialEntries={["/employee"]}>
        <FlexiblePayEmployeePage />
      </MemoryRouter>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(screen.queryByTestId("employee-ad-banner")).not.toBeInTheDocument();
  });
});
