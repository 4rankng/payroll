import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";

describe("Tap targets ≥44px trên mobile (PAYR-24)", () => {
  it("Button gốc mở rộng hit area bằng .tap-target-44 (gỡ nút 36px diện rộng)", () => {
    render(<Button>Nhấn</Button>);
    expect(screen.getByRole("button", { name: "Nhấn" }).className).toContain("tap-target-44");
  });

  it("Switch giữ pseudo hit area 48×44 — rect 28px nhưng hit ≥44", () => {
    render(<Switch />);
    const sw = screen.getByRole("switch");
    expect(sw.className).toContain("before:-inset-y-2");
    expect(sw.className).toContain("before:inset-x-0");
  });

  it("CSS utility: chỉ <768px, grow max(100%, 44px) — không mở rộng chiều đã ≥44", () => {
    const css = readFileSync(resolve(process.cwd(), "src/styles/utilities.css"), "utf8");
    expect(css).toContain(".tap-target-44::after");
    expect(css).toContain("width: max(100%, 44px)");
    expect(css).toContain("height: max(100%, 44px)");
    expect(css).toContain("max-width: 767.98px");
  });

  it("control ngoài Button cũng được fix: link Quên mật khẩu, chips approvals, month selector", () => {
    const login = readFileSync(resolve(process.cwd(), "src/pages/Login.tsx"), "utf8");
    expect(login).toContain("tap-target-44 inline-flex min-h-6");

    const approvals = readFileSync(
      resolve(process.cwd(), "src/pages/mobile/admin/ApprovalsPage/index.tsx"),
      "utf8",
    );
    expect(approvals).toContain("tap-target-44 h-9 shrink-0 rounded-full");

    // Selector đóng overflow-hidden + nút dính nhau → phải grow thật, không pseudo.
    const monthSelector = readFileSync(
      resolve(process.cwd(), "src/components/timesheet/TimesheetMonthSelector.tsx"),
      "utf8",
    );
    expect(monthSelector).toContain("h-11 w-11 shrink-0");
    expect(monthSelector).toContain("md:h-9 md:w-9");
    expect(monthSelector).toContain("md:h-9");
  });
});
