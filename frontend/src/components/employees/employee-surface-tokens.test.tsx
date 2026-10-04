import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Per-surface retirement guard for the employee portal (Untitled UI PRO
 * migration, W11). The daisyUI `ct-` component classes and the legacy
 * slate/emerald/amber/red/orange/sky palette classes were replaced by the
 * `--employee-*` role tokens and the UU bridge tokens on this surface. Any
 * regression re-introducing them must fail here, not in review.
 *
 * EmployeeLocationMap is exempt from the palette scan only: its marker dots,
 * gate labels and route chips deliberately share the maplibre canvas paint
 * colors (#047857 / #0284c7) so DOM overlays match the map layers beneath.
 */
const PORTAL_SURFACE_FILES = [
  "../../pages/employee/EmployeeRouter/index.tsx",
  "../../pages/employee/EmployeePage/index.tsx",
  "../../pages/employee/FlexiblePayEmployeePage/index.tsx",
  "./EmployeeMobileShell.tsx",
  "./EmployeePortalHeader.tsx",
  "./EmployeeCanopy.tsx",
  "./EmployeeTimesheetPanel.tsx",
  "./EmployeeMonthNavigator.tsx",
  "./EmployeeDataError.tsx",
  "./EmployeeBankInfoCard.tsx",
  "./EmployeeCheckInCard.tsx",
  "./EmployeeAttendanceActionDock.tsx",
  "./EmployeeAttendanceHistoryCard.tsx",
  "./ChangePasswordSheet.tsx",
  "./EmployeeAdBanner.tsx",
  "./EmployeeAdSheet.tsx",
  "../advance-payment/AdvancePaymentRequestForm.tsx",
  "../advance-payment/AdvancePaymentHistoryCard.tsx",
  "../advance-payment/AdvancePaymentConfirmSheet.tsx",
] as const;

/** Map-canvas overlay file exempt from the legacy-palette scan (see above). */
const MAP_CANVAS_EXEMPT = new Set(["./EmployeeLocationMap.tsx"]);

const daisyUiClassPattern = /(^|[^\w-])ct-[a-z]/;
const legacyPalettePattern =
  /(^|[^\w-])(text|bg|border|from|to|via|ring|divide|fill|stroke)-(slate|emerald|amber|red|orange|sky|gray|zinc|neutral|stone)-[0-9]/;

describe("employee portal UU token retirement", () => {
  it("keeps every portal file free of daisyUI ct- classes", () => {
    const offenders = PORTAL_SURFACE_FILES.filter((relativePath) =>
      daisyUiClassPattern.test(readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), "utf8")),
    );
    expect(offenders).toEqual([]);
  });

  it("keeps portal chrome on employee/UU tokens instead of legacy palette classes", () => {
    const offenders = PORTAL_SURFACE_FILES.flatMap((relativePath) => {
      if (MAP_CANVAS_EXEMPT.has(relativePath)) return [];
      const content = readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), "utf8");
      return legacyPalettePattern.test(content) ? [relativePath] : [];
    });
    expect(offenders).toEqual([]);
  });

  it("keeps portal icons on @untitledui/icons", () => {
    const offenders = PORTAL_SURFACE_FILES.filter((relativePath) =>
      readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), "utf8").includes("lucide-react"),
    );
    expect(offenders).toEqual([]);
  });
});
