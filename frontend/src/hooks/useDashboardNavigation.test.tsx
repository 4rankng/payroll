import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useDashboardNavigation } from "./useDashboardNavigation";

const navigate = vi.fn();

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigate,
}));

describe("useDashboardNavigation", () => {
  beforeEach(() => {
    navigate.mockReset();
  });

  it("routes salary drill-downs through the ledger period contract", () => {
    const { result } = renderHook(() => useDashboardNavigation());

    act(() => result.current.navigateToSalaryLedger());
    act(() => result.current.navigateToCurrentMonthSalary());

    expect(navigate).toHaveBeenNthCalledWith(1, "/admin/ledger?period=salary");
    expect(navigate).toHaveBeenNthCalledWith(
      2,
      "/admin/ledger?period=current-month",
    );
  });
});
