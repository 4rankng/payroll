import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FlexiblePayrateEditor } from "./FlexiblePayrateEditor";

describe("FlexiblePayrateEditor", () => {
  it("clearly edits both the shift time and its hourly pay", () => {
    const onChange = vi.fn();

    render(
      <FlexiblePayrateEditor
        rates={{
          "Công nhân": {
            "ngày thường": {
              "08:00-17:00": 30000,
            },
          },
        }}
        onChange={onChange}
      />,
    );

    expect(screen.getByText("Mức lương (₫/giờ)")).toBeInTheDocument();

    fireEvent.change(
      screen.getByRole("textbox", {
        name: "Mức lương theo giờ cho Công nhân, ca 08:00-17:00",
      }),
      { target: { value: "35000" } },
    );

    expect(onChange).toHaveBeenLastCalledWith({
      "Công nhân": {
        "ngày thường": {
          "08:00-17:00": 35000,
        },
      },
    });

    fireEvent.click(
      screen.getByRole("button", { name: "Chỉnh khung giờ 08:00-17:00" }),
    );
    const shiftInput = screen.getByDisplayValue("08:00-17:00");
    fireEvent.change(shiftInput, { target: { value: "09:00-18:00" } });
    fireEvent.keyDown(shiftInput, { key: "Enter" });

    expect(onChange).toHaveBeenLastCalledWith({
      "Công nhân": {
        "ngày thường": {
          "09:00-18:00": 35000,
        },
      },
    });
  });

  it("does not expose edit actions when the configuration is read-only", () => {
    const onChange = vi.fn();

    render(
      <FlexiblePayrateEditor
        rates={{
          "Công nhân": {
            "ngày thường": {
              "08:00-17:00": 30000,
            },
          },
        }}
        onChange={onChange}
        readOnly
      />,
    );

    expect(
      screen.queryByRole("button", { name: "Chỉnh khung giờ 08:00-17:00" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("textbox", {
        name: "Mức lương theo giờ cho Công nhân, ca 08:00-17:00",
      }),
    ).toHaveAttribute("readonly");
    expect(onChange).not.toHaveBeenCalled();
  });
});
