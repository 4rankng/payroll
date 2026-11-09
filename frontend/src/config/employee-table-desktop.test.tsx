import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CellContext } from "@tanstack/react-table";

import { createEmployeeColumns } from "./employee-table-desktop";
import type { Employee } from "@/types/api/employee.types";

// Agribank — tên ngân hàng dài nhất trong dữ liệu thật, bị cắt
// "Nông nghiệp và Phát tri…" trong evidence PAYR-34.
const LONG_BANK = "Ngân hàng Nông nghiệp và Phát triển Nông thôn";

const employee = {
  id: 1,
  fullname: "Lê Viết Anh",
  bank: { branch_name: LONG_BANK },
} as Employee;

describe("Employee table cột Ngân hàng (PAYR-34)", () => {
  const renderBankCell = (employee: Employee) => {
    const columns = createEmployeeColumns({ onRowClick: vi.fn() });
    const bankColumn = columns.find((column) => column.id === "bank")!;
    const cell = bankColumn.cell as (ctx: CellContext<Employee, unknown>) => ReactNode;
    return render(<>{cell({ row: { original: employee } } as unknown as CellContext<Employee, unknown>)}</>);
  };

  it("tên ngân hàng dài hiển thị TRỌN vẹn — wrap xuống dòng, không ellipsis", () => {
    const { container } = renderBankCell(employee);

    expect(screen.getByText(LONG_BANK)).toBeTruthy();
    expect(container.textContent).not.toContain("…");

    const label = screen.getByText(LONG_BANK);
    // Wrap full theo shipmate-uiux doctrine (PAYR-27): break-words thay vì
    // co ngang — cấm truncate/line-clamp/max-w cứng gây cắt giữa từ.
    expect(label).toHaveClass("break-words");
    expect(label).not.toHaveClass("truncate");
    expect(label).not.toHaveClass("line-clamp");
    expect(label.parentElement).not.toHaveClass("max-w-[180px]");
  });

  it("không có ngân hàng vẫn hiển thị gạch nối thay vì nội dung rỗng", () => {
    const { container } = renderBankCell({ ...employee, bank: undefined } as Employee);
    expect(container.textContent).toBe("—");
  });
});
