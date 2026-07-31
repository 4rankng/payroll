import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PayrateRateGrid } from "./PayrateRateGrid";

describe("PayrateRateGrid", () => {
  it("labels displayed rates as hourly earnings", () => {
    render(
      <PayrateRateGrid
        rates={{
          "Công nhân": {
            "ngày thường": {
              "08:00-17:00": 30000,
            },
          },
        }}
      />,
    );

    expect(screen.getByText("30.000")).toBeInTheDocument();
    expect(screen.getByText("đ/giờ")).toBeInTheDocument();
    expect(screen.getByLabelText("Bảng mức lương theo giờ")).toHaveClass("overflow-x-auto");
  });
});
