import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PayrateRateGrid } from "./PayrateRateGrid";

describe("PayrateRateGrid", () => {
  const rates = {
    "Công nhân": {
      "ngày thường": {
        "08:00-17:00": 30000,
      },
    },
  };

  it("labels standard-project rates as hourly earnings", () => {
    render(
      <PayrateRateGrid
        rates={rates}
      />,
    );

    expect(screen.getByText("30.000")).toBeInTheDocument();
    expect(screen.getByText("₫/giờ")).toBeInTheDocument();
    expect(screen.getByLabelText("Bảng mức lương theo giờ")).toHaveClass("overflow-x-auto");
  });

  it("labels flexible-project rates as full-shift earnings", () => {
    render(<PayrateRateGrid rates={rates} isFlexible />);

    expect(screen.getByText("30.000")).toBeInTheDocument();
    expect(screen.getByText("₫/ca")).toBeInTheDocument();
    expect(screen.queryByText("₫/giờ")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Bảng lương trọn ca")).toHaveClass(
      "overflow-x-auto",
    );
  });
});
