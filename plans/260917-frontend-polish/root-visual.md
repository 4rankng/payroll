# Shared UI and root visual evidence — 2026-09-17

## Scope and evidence quality

Only the locally running app and synthetic data were used. Automated captures, automated geometry/axe results, and images actually inspected are distinct evidence. The inventories are not proof of every possible production record, operating system or hardware state.

Initial37role/route combinations were captured at320/390/1280 (111renders); initial modal URL variants included duplicate employee query parameters and are not counted as distinct panels. Follow-up work expands the responsive matrix to320,390,768,1023,1024,1280,1440. Role-specific reports are visual-admin-partner.md and employee-visual.md.

## Direct browser interactions

The in-app browser was used for Admin monthly dashboard filters/year/month navigation, accordion detail, failed health drilldown, all six settings tabs, fee presets/tier fields, email HTML editing and live preview, notification recipient picker, disabled unconfigured Zalo settings, ad composition bullet/action/duration controls, wallet transaction details/copy affordances, bank/account lookup tabs, upload and manual transfer dialogs. These visual actions did not send email, publish ads, save settings or transfer money.

The mobile dashboard and wallet were checked after reducing decorative headers, metrics and action spacing. Partner uses the same month chooser and compact summary pattern. MobileSubPageHeader, lenders and ledger views retain44px actions while displaying more useful rows. Accountant tabs, selection/date-reset, keyboard focus return, statement calendars, upload/report dialog titles and close-button clearance have portable Chromium/WebKit coverage.

## Right-edge defect and correction

The earlier check measured overflow but missed under-width sheets. The user screenshot exposed a374px-wide manual transfer dialog in a390px viewport. Its local calc width overrode mobile w-full. At768/1023, sm max-width classes similarly constrained bottom sheets despite the application's1024px mobile breakpoint. Audit detail reproduced width512 at both tablet widths, leaving256/511px gaps.

DialogContent now pins mobile width/max-width/edges; SheetContent enforces the same rule only for top/bottom mobile sheets. NotificationSheet and EmployeeAdSheet had important420px overrides and now apply that desktop width only at lg. Side drawers and centered desktop modal dimensions are preserved. Shared unit tests verify both invariants. Browser bounding-box assertions wait for finite opening animations; comparing independently moving rectangles had produced misleading intermediate header failures.

Actual settled transfer screenshots and DOM geometry were inspected at320,390,768,1023; each starts at0 and ends at the viewport edge. At320 the form body scrolls and both verification/transfer actions remain reachable. At1024 the dialog is580px wide with222px on each side. Admin audit and Partner project sheets at768/1023 were independently recaptured and inspected after the fix. The final modal inventory and any limitations are recorded in visual-admin-partner.md.

## Follow-up accessibility results

- Final dense Admin dashboard/wallet/system-health sweep:21renders across all7widths, zero overflow, uncaught page errors or axe violations.
- Final dense Partner dashboard sweep:7renders across all7widths, zero overflow, uncaught page errors or axe violations. An earlier low-contrast9px active badge was replaced with a readable10px dark green badge.
- Admin monthly financial table now exposes a named focusable scrolling region; axe confirmed the earlier scroll-access failure is resolved.
- Earlier email route sweeps using Playwright serviceWorkers:block raised a sandboxed-iframe instrumentation error. Direct in-app browser HTML editing rendered the new heading/body in its sandboxed preview with an empty error log; no email was sent. The sandbox remains intact.
- Optional settings404 responses reflect absent local configuration. Live provider/hardware delivery is not inferred from these visual checks.

Raw screenshots and logs remain local in /tmp/payroll-ui-qa. They and credentials are excluded from the portable patch.
