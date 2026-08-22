import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

function readSource(path: string) {
  return readFileSync(resolve(process.cwd(), path), "utf8");
}

describe("check-in settings route parity", () => {
  it("registers the dedicated page for Admin and Advance Partner", () => {
    const appSource = readSource("src/App.tsx");
    const route = '<Route path="advance-payments/check-in-settings" element={<CheckInSettingsPage />} />';

    expect(appSource.split(route)).toHaveLength(3);
  });

  it("keeps the page entry point in every active desktop and mobile surface", () => {
    const adminDesktop = readSource("src/pages/admin/AdvancePaymentsPage/index.tsx");
    const advancePartnerDesktop = readSource(
      "src/pages/admin/AdvancePaymentsPage/AdvPartnerView.tsx",
    );
    const sharedMobile = readSource(
      "src/pages/mobile/admin/AdvancePaymentsPage/index.tsx",
    );

    expect(adminDesktop).toContain("/admin/advance-payments/check-in-settings");
    expect(advancePartnerDesktop).toContain(
      "/adv-partner/advance-payments/check-in-settings",
    );
    expect(sharedMobile).toContain("/admin/advance-payments/check-in-settings");
    expect(sharedMobile).toContain(
      "/adv-partner/advance-payments/check-in-settings",
    );
  });
});
