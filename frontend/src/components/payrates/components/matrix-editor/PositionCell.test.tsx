import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PositionCell } from "./PositionCell";

const rates = {
  "Lương 500": { "ngày thường": { HC: 62500 } },
  "Lương 520": { "ngày thường": { HC: 65000 } },
};

describe("PositionCell", () => {
  it("keeps the full position name separate from its row actions", () => {
    const onEditPosition = vi.fn();
    const onRemovePosition = vi.fn();
    const onCopyRates = vi.fn();

    render(
      <PositionCell
        position="Lương 520"
        positionIndex={1}
        firstPosition="Lương 500"
        positions={["Lương 500", "Lương 520"]}
        rates={rates}
        editingPosition={null}
        editingPositionValue=""
        onEditPosition={onEditPosition}
        onSavePosition={vi.fn()}
        onRemovePosition={onRemovePosition}
        onCopyRates={onCopyRates}
        setEditingPositionValue={vi.fn()}
        setEditingPosition={vi.fn()}
      />,
    );

    expect(screen.getByText("Lương 520")).toHaveClass("text-center");

    fireEvent.pointerDown(
      screen.getByRole("button", { name: "Tùy chọn cho vị trí Lương 520" }),
      { button: 0, ctrlKey: false },
    );
    fireEvent.click(screen.getByRole("menuitem", { name: "Đổi tên" }));

    expect(onEditPosition).toHaveBeenCalledWith("Lương 520");
    expect(onRemovePosition).not.toHaveBeenCalled();
    expect(onCopyRates).not.toHaveBeenCalled();
  });
});
