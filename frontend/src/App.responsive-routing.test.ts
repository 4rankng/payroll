import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const appSource = readFileSync(resolve(process.cwd(), "src/App.tsx"), "utf8");

describe("Responsive route wiring (PAYR-23)", () => {
  it("3 route always-mobile trước đây render variant theo width qua ResponsivePage", () => {
    expect(appSource).toContain(
      '<Route path="dashboard/activity" element={<ResponsivePage desktopComponent={ActivityUsersPage} mobileComponent={ActivityUsersPageMobile} />} />',
    );
    expect(appSource).toContain(
      '<Route path="loans/lenders" element={<ResponsivePage desktopComponent={LendersPage} mobileComponent={LendersPageMobile} />} />',
    );
    expect(appSource).toContain(
      '<Route path="advance-payments/employees" element={<ResponsivePage desktopComponent={AdvancePaymentsEmployeeListPage} mobileComponent={AdvancePaymentsEmployeeListPageMobile} />} />',
    );
    // Route này xuất hiện 2 lần (admin + adv-partner) — cả hai đều responsive.
    expect(
      appSource.split(
        '<Route path="advance-payments/employees" element={<ResponsivePage desktopComponent={AdvancePaymentsEmployeeListPage} mobileComponent={AdvancePaymentsEmployeeListPageMobile} />} />',
      ).length - 1,
    ).toBe(2);

    // Không còn route nào pin mobile component ở mọi width.
    expect(appSource).not.toContain("element={<ActivityUsersPageMobile />}");
    expect(appSource).not.toContain("element={<LendersPageMobile />}");
    expect(appSource).not.toContain("element={<AdvancePaymentsEmployeeListPageMobile />}");
    expect(appSource).not.toContain("element={<PartnerPaymentHistoryPage />}");
  });

  it("payment-history partner mở UI mobile trên mobile (trỏ lại PaymentHistoryPage mobile)", () => {
    expect(appSource).toContain(
      '<Route path="timesheet/payment-history" element={<ResponsivePage desktopComponent={PartnerPaymentHistoryPage} mobileComponent={PartnerPaymentHistoryPageMobile} />} />',
    );
    expect(appSource).toContain(
      'const PartnerPaymentHistoryPageMobile = lazy(() => import("./pages/mobile/partner/TimesheetsPage/PaymentHistoryPage"));',
    );
  });
});
