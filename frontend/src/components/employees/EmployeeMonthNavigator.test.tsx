import { fireEvent, render, screen } from "@testing-library/react";
import { format, startOfMonth, subMonths } from "date-fns";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { useEmployeeMonth } from "@/hooks/useEmployeeMonth";
import { EmployeeMonthNavigator } from "./EmployeeMonthNavigator";

function MonthHarness() {
  const month = useEmployeeMonth();
  const location = useLocation();
  return (
    <>
      <EmployeeMonthNavigator month={month} />
      <output aria-label="Địa chỉ hiện tại">{location.search}</output>
    </>
  );
}

describe("EmployeeMonthNavigator", () => {
  it("keeps the current month as the URL default and disables future navigation", () => {
    render(<MemoryRouter initialEntries={["/employee"]}><MonthHarness /></MemoryRouter>);

    expect(screen.getByRole("button", { name: "Xem tháng sau" })).toBeDisabled();
    expect(screen.getByLabelText("Địa chỉ hiện tại")).toHaveTextContent("");
  });

  it("writes previous-month navigation to the URL", () => {
    render(<MemoryRouter initialEntries={["/employee"]}><MonthHarness /></MemoryRouter>);
    fireEvent.click(screen.getByRole("button", { name: "Xem tháng trước" }));

    const expected = format(subMonths(startOfMonth(new Date()), 1), "yyyy-MM");
    expect(screen.getByLabelText("Địa chỉ hiện tại")).toHaveTextContent(`?month=${expected}`);
    expect(screen.getByRole("button", { name: "Xem tháng sau" })).toBeEnabled();
  });

  it("disables backward navigation at the 24-month history floor", () => {
    const floor = format(subMonths(startOfMonth(new Date()), 24), "yyyy-MM");
    render(<MemoryRouter initialEntries={[`/employee?month=${floor}`]}><MonthHarness /></MemoryRouter>);

    expect(screen.getByRole("button", { name: "Xem tháng trước" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Xem tháng sau" })).toBeEnabled();
  });

  it("opens an accessible month and year picker", () => {
    render(<MemoryRouter initialEntries={["/employee"]}><MonthHarness /></MemoryRouter>);
    fireEvent.click(screen.getByRole("button", { name: /Kỳ lương tháng/i }));

    expect(screen.getByRole("combobox", { name: "Chọn năm" })).toBeInTheDocument();
    expect(screen.getByText("Xem lịch sử lương và yêu cầu")).toBeInTheDocument();
  });
});
