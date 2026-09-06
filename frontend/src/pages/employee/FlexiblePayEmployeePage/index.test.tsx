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
  useCheckInAdvanceInfo: () => ({ data: undefined, isLoading: false, isError: false, refetch: vi.fn() }),
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
vi.mock("@/components/employees/EmployeeAdBanner", () => ({ EmployeeAdBanner: () => null }));
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

  it("does not override the calendar month for a check-in employee", async () => {
    profileQuery.data = { fullname: "Đỗ Văn Hùng", check_in_enabled: true };

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
});
