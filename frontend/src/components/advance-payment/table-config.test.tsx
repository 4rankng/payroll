import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CellContext, ColumnDef } from "@tanstack/react-table";

import { getFlexPayColumns, type SortContext } from "./table-config";

const sortCtx: SortContext = { sortBy: "max_advance_amount", sortOrder: "ASC", onSort: vi.fn() };

const renderHeader = <T,>(column: ColumnDef<T>) => {
  const header = typeof column.header === "function" ? column.header({} as never) : column.header;
  if (typeof header !== "string" && header !== undefined && header !== null) {
    return render(<>{header}</>);
  }
  return null;
};

describe("FlexPay header số căn phải (PAYR-34)", () => {
  it("mọi cột số align right đều dùng SortHeader inline-level để th text-right có tác dụng", () => {
    const columns = getFlexPayColumns(sortCtx);
    const numericColumns = columns.filter(
      (column) => (column.meta as { align?: string } | undefined)?.align === "right",
    );
    // 4 cột SortHeader (Hạn mức, Đã dùng, Chờ xử lý, Còn lại) + Phí thu.
    expect(numericColumns).toHaveLength(5);

    for (const column of numericColumns) {
      if (typeof column.header === "string") {
        // Header string ("Phí thu") nhận text-right trực tiếp từ th.
        continue;
      }
      const { container, unmount } = renderHeader(column)!;
      const button = container.querySelector("button")!;
      // display:flex block-level full-width khiến text-right của th vô hiệu —
      // header số căn trái lệch với giá trị căn phải. inline-flex là bắt buộc.
      expect(button).toHaveClass("inline-flex");
      expect(button).not.toHaveClass("flex");
      unmount();
    }
  });

  it("SortHeader giữ nguyên hành vi sort khi bấm", () => {
    const onSort = vi.fn();
    const columns = getFlexPayColumns({ sortBy: "", sortOrder: "ASC", onSort });
    const { container, unmount } = renderHeader(columns[2])!;
    container.querySelector("button")!.click();
    expect(onSort).toHaveBeenCalledWith("max_advance_amount");
    unmount();
  });
});
