import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { LedgerMobileList } from "./LedgerMobileList";
import type { LedgerEntry } from "@/types/api/financial.types";

vi.mock("@/hooks/api/useAssets", () => ({
  useDownloadAsset: () => ({ mutate: vi.fn() }),
}));

const LONG_REF = "itest2026102842426746400000000000000000_vendor"; // 46 ký tự

const entry: LedgerEntry = {
  id: 1,
  account: "cash",
  party: LONG_REF,
  description: "Chi lương tuần cho nhóm vendor thanh lý",
  debit: 1_000_000,
  credit: 1_000_000,
  balance: 0,
  net_amount: 1_000_000,
  date: "2026-10-28T10:00:00Z",
  created_by: 1,
  created_at: "2026-10-28T10:00:00Z",
  updated_at: "2026-10-28T10:00:00Z",
};

describe("LedgerMobileList card wrap (PAYR-27)", () => {
  it("ref ≥43 ký tự hiển thị TRỌN vẹn — không clip cứng, không ellipsis", () => {
    const { container } = render(
      <LedgerMobileList entries={[entry]} onRowClick={() => undefined} />,
    );

    expect(screen.getByText(LONG_REF)).toBeTruthy();
    expect(container.textContent).not.toContain("…");
    // Wrap full: title tự xuống dòng thay vì co ngang với amount
    const title = screen.getByText(LONG_REF);
    expect(title.className).toContain("break-all");
    expect(title.className).not.toContain("line-clamp");
    expect(title.className).not.toContain("truncate");
  });

  it("amount ở vùng riêng, không cùng hàng tiêu đề", () => {
    render(<LedgerMobileList entries={[entry]} onRowClick={() => undefined} />);
    const title = screen.getByText(LONG_REF);
    const amount = screen.getByText(/1\.000\.000|1000000/);
    // Amount là block riêng ngay dưới title — không còn flex row chung
    // (title flex-1 + amount shrink-0) giành bề ngang như trước.
    expect(title.nextElementSibling).toBe(amount);
    expect(title.className).not.toContain("flex-1");
    expect(amount.className).not.toContain("shrink-0");
  });

  it("tap card giữ nguyên đường tới giá trị đầy đủ (handleRowClick)", () => {
    const onRowClick = vi.fn();
    render(<LedgerMobileList entries={[entry]} onRowClick={onRowClick} />);
    fireEvent.click(screen.getByRole("button", { name: new RegExp(LONG_REF) }));
    expect(onRowClick).toHaveBeenCalledWith(entry);
  });
});
