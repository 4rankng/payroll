# Admin and Partner visual QA — 2026-09-17

## Scope and environment

Local Vite5173/API8080, synthetic QA identities and data only. No production reads or writes. Main checkout remains uncommitted. Viewports:320,390,768,1023,1024,1280,1440 pixels,900px height. The1023/1024 pair exercises the actual mobile/desktop component switch.

Coverage is separated below: a screenshot capture or DOM overflow check is not an individual visual inspection. No assertion that every application state or real device was exhaustively tested.

## Browser evidence

- Initial baseline:30 route/viewport captures (Admin users/employees/projects/timesheet/loans/audit/cron; Partner employees/projects/timesheet at320/390/1280).
- Extended route matrix:68 captures, adding768/1023/1024/1440 and Admin+Partner dynamic payrate edit and payment history.0 document overflow. Inventory:`/tmp/payroll-ui-qa/visual-role/extended-inventory.json`.
- Initial control interactions:87 captures,0 stage failures,0 attempted API writes. Opened filters/select options, employee/user/project detail and edit panels, project employees/payrate/permissions tabs, timesheet actions or expanded row, loan create/detail, audit filter/detail. This pass waited only200ms after opening some sheets: those images can be mid-animation and are not final overflow proof.
- Final role capture:103 distinct settled-animation surfaces (Admin62,Partner41),0 page errors,0 document/dialog overflow,0 attempted API writes.14 Admin+Partner width combinations passed month/year chooser, next, previous, and All. Evidence:`/tmp/payroll-ui-qa/visual-role/final/summary.json` and`inventory.json`.
- Cron toggle fixture:320/390/1280, click and keyboard Space each;6 PUT responses intercepted locally,0 real toggle writes. GET data copied from current local response. Evidence:`cron-toggle-regression.json`.
- Payrate behavior:Admin+Partner320/390/1280, changed a matrix value locally, verified JSON value, switched back preserving the edit, cancelled to project detail.6/6 passed,0 server writes. Evidence:`payrate-actions.json`.
- Final ledger/lenders density pass:14/14 route-width checks passed across7 widths, native details click/Enter expand/collapse below1024, lender create panel open/Escape below1024, no submission. Evidence:`density-final.json`.0 attempted writes/overflow.

Initial screenshots used actual remote fonts. Google Fonts subsequently stalled navigation/screenshot completion; final QA contexts explicitly blocked Google Fonts and used system fallback. The employee agent separately fixed blocking font loading. These screenshots are local Chromium emulation, not physical iOS/Android testing.

## Grounded changes and comparisons

- Mobile user cards now expose role text, distinguish accountant/advance-partner roles, wrap names across2lines and usernames rather than showing indistinguishable prefixes. Tablet uses3columns.
- Employee cards now reveal more of long employee names; phone summary counts use one compact4-column strip. Partner employee header keeps export as a labeled44px icon with visible create action.
- Admin payroll metrics pair counts on phones while full currency metrics retain fullwidth. Shared metric height reduced76→64px. Cash readiness remains visible.
- Partner desktop used separate previous/next-only month text; now uses shared month/year selector like both mobile roles and Admin desktop, preserving yyyy-MM/all state.
- Mobile filters have padding and bounded scrolling. Audit detail now has padding, a named44px close control, and retry instead of a blank failed request.
- Cron mobile exposes a recognizable switch with clear on/off state and44px target. Disabled rows/cards use neutral backgrounds without making metadata unreadable. The final disabled-state screenshot was regenerated after removing desktop opacity.
- Loan/lender phone titles fit beside labeled44px create controls. Loan detail's initially apparent clipping was a mid-animation capture artifact; settled1280 panel has no overflow, so no unnecessary sheet-width rewrite was made.
- Ledger320 originally placed ten stacked metric blocks before the list. Now3main balances occupy compact rows, secondary7account balances are one accessible expandable disclosure. Final320 closed screenshot shows records from~595px; all account values remain available on expansion.
- Bank-warning title vertical wrapping and payrate mobile heading truncation were reported to root, who fixed those shared surfaces. Final Partner employee/payrate screenshots show those corrections.

## Tests and independent review

- New HealthDrilldownSheet error tests cover failed-attempts, quota anomalies, attendance list, and successful checkouts on desktop/mobile, retry and close. Audit detail tests cover retry/close on both views.12/12 passed.
- Targeted ESLint passed for17 visual/error files and3final density files;git diff --check passed. Root owns final full lint/types/unit/build/API validation.
- Reviewed root MobileOperationsPanel and MonthPicker source changes: no callback, permission, or period-contract regression found.
- Financial independent review caught two defects in the proposed worker change: legacy regular-bank codes store gross amount, and mixed already-paid rows may differ from a recalculated allocation. Runtime corrected these with OnePay-only optional transfer snapshot and mismatch rejection before writes. Re-review found no further blocker; runtime package race tests passed. Legacy codes retain prior behavior; no historical correction is claimed.

## Visual images actually inspected

The following 47 images were opened with view_image, in addition to older root QA reference images. Paths are relative to`/tmp/payroll-ui-qa/visual-role/`. Initial action images noted above are comparisons only; final images wait for animation completion.

- `admin-320-timesheet.png`
- `admin-390-loans.png`
- `admin-1280-projects.png`
- `partner-1280-timesheet.png`
- `admin-390-audit-log.png`
- `admin-320-cron-health.png`
- `partner-390-employees.png`
- `admin-1280-loans.png`
- `admin-320-employees.png`
- `admin-1280-audit-log.png`
- `admin-1280-cron-health.png`
- `admin-768-users.png`
- `admin-768-projects_999971_payrates_1_edit.png`
- `admin-768-payment-history.png`
- `admin-1024-employees.png`
- `admin-1024-timesheet.png`
- `admin-1024-projects_999971_payrates_1_edit.png`
- `actions/admin-1280-user-edit.png`
- `actions/admin-1280-loan-details.png`
- `actions/admin-390-users-filters.png`
- `actions/admin-320-loan-details.png`
- `actions/admin-320-audit-details.png`
- `partner-1024-projects_999971_payrates_1_edit.png`
- `partner-320-projects_999971_payrates_1_edit.png`
- `final/admin-320-users.png`
- `final/admin-320-employees.png`
- `final/admin-320-timesheet.png`
- `final/admin-320-cron-health.png`
- `final/admin-320-audit-details.png`
- `final/admin-320-loans.png`
- `final/admin-1280-loan-details.png`
- `final/admin-390-employee-edit.png`
- `final/admin-768-users.png`
- `final/admin-1023-timesheet.png`
- `final/admin-1024-timesheet.png`
- `final/partner-320-employees.png`
- `final/partner-320-projects.png`
- `final/partner-1024-timesheet.png`
- `final/partner-1280-month-chooser.png`
- `final/admin-1280-cron-fixture-off.png`
- `final/admin-320-cron-fixture-off.png`
- `final/admin-320-ledger-compact.png`
- `final/admin-320-ledger-expanded.png`
- `final/admin-1280-payrate-json.png`
- `final/partner-320-payrate-json.png`

- `final/admin-320-loans_lenders-compact.png`
- `final/admin-390-ledger-compact.png`

## Limits

- Physical devices, Safari/WebKit/Firefox, offline/PWA installation, and every possible localization/financial magnitude were not tested in this role pass.
- Submit/cancel-payment/approve/disburse/delete/reverse operations were not performed against the local API by this visual pass. It used open/edit/cancel behavior and explicit fixture interception for cron writes. Other agents own synthetic API integration scenarios.
- Route loading, opened-component checks, screenshot inspection, automated semantics, and backend integration are distinct evidence; use the root verification packet for combined results.

## Follow-up: tablet sheet edge gap caught after initial QA

The earlier103-surface check reported **overflow**, not edge-to-edge coverage. It did not open audit details at768/1023. A left-anchored512px sheet does not overflow, so the previous geometry check could not detect the empty right side. The broad width coverage must not be interpreted as every modal at every width being individually inspected.

After the user caught a transfer-dialog width gap, independent review confirmed `DialogContent`'s mobile width/max-width/edge override is compatible with existing callers, including fullscreen maps that explicitly reset transforms. It also found the analogous `SheetContent` issue: `useIsMobile` remains true below1024, while numerous bottom-sheet callers apply desktop `sm:` widths at640.

- Before: Audit detail at768 had left0,width512,right512 (256px right gap); at1023 it had left0,width512,right512 (511px right gap). Captured in `/tmp/payroll-ui-qa/sheet-width-review/results.json`. The768 screenshot was visually inspected.
- Root fixed the shared primitive only for mobile top/bottom sheets; side drawers and desktop dimensions remain caller-controlled. Root added regression cases for those preserved contracts.
- After: Admin audit details and Partner project details at both768/1023 each have left0,width=viewport,right=viewport. All4 screenshots were opened and visually inspected after animations settled. Evidence:`/tmp/payroll-ui-qa/sheet-width-review-after/results.json`.
- This adds5 actual image inspections to the47-file inventory above (52 total in this packet): before768audit plus after768/1023audit and after768/1023Partnerproject. No application source files were edited in this independent-review pass.

Follow-up images:

- `/tmp/payroll-ui-qa/sheet-width-review/admin-768-audit.png`
- `/tmp/payroll-ui-qa/sheet-width-review-after/admin-768-audit.png`
- `/tmp/payroll-ui-qa/sheet-width-review-after/admin-1023-audit.png`
- `/tmp/payroll-ui-qa/sheet-width-review-after/partner-768-project.png`
- `/tmp/payroll-ui-qa/sheet-width-review-after/partner-1023-project.png`


## Final URL-modal edge inventory and cold-load fixes

The adapted, local-only harness is `/tmp/payroll-ui-qa/modal-edges-final.cjs` (derived from the ignored `qa/modal-audit.cjs`; the original harness was not changed). Results and viewport screenshots are in `/tmp/payroll-ui-qa/modal-edges-final/`. This pass did not rerun axe. External Google Fonts and service workers were blocked, so screenshots use the system fallback font. Every capture waited at least500ms and for finite animations to settle, with a bounded500ms fallback. API writes were blocked.

Final automated results: **150 authorized route-width combinations** at320/390/768/1023/1024/1280: Admin96, Partner48, AdvPartner6. The25 variants comprise Admin16 (project detail tabs4, employee/user details2, common/create modals7, project assignment, add employee to project, timesheet entry), Partner8 (project tabs4, employee details, add employee to project, profile, password), and AdvPartner password1. The150 combinations contain78 mobile top/bottom sheet edge assertions,68 intentional side drawers or desktop dialogs, and4 mobile timesheet entry pages. Mobile timesheet entry is intentionally a full page below1024, not a missing dialog. Intentional right/left drawers retain their caller-defined widths.

Final `summary.json`: **zero horizontal edge gaps, dialog/document overflow, browser exceptions, failing API responses, or attempted API writes**. The last30 captures also explicitly recorded denied-permission toasts: all zero. They cover both roles' assignment sheets, Admin notifications, AdvPartner password, and Partner profile at all six widths. This geometry inventory is automated evidence; it does not mean150 screenshots were individually visually inspected.

Defects found during the inventory and resolved:

- Cold-opening `add_employee_to_project` passed a null project before the query completed and crashed on `project.id`. The shared Admin/Partner container now displays loading, retryable query error, or invalid-link state before mounting the assignment UI.14 regression cases passed.
- That allowed assignment modal was absent from the manual permission registry and domain registry, producing a false denial toast while rendering. Added its existing Admin/Partner update-Project contract and schema without widening roles.7 regression cases passed; all12 live role-width recaptures have no warning.
- Notifications at768/1023 were still420px wide because caller `!important` classes overrode the shared primitive. Root removed the conflicting mobile override and moved fixed width to `lg:`; the final six-width recapture passes. The two tablet screenshots were visually inspected before and after. Before geometry remains in `/tmp/payroll-ui-qa/modal-edges-final-before-followups.json`; the final screenshot files contain the corrected versions.
- AdvPartner profile URL was an invalid inherited test case: the actual profile control uses local sheet state. It was excluded from the authorized inventory rather than expanding profile URL permissions. Its denial screenshot was inspected to establish the cause.
- AdvPartner's real password control did use the URL modal but was blocked by role metadata. The employee agent confirmed the own-account backend contract and introduced a narrowly scoped `SelfAccount` permission, also removing the false denial for Partner's existing profile URL. Final AdvPartner password and Partner profile each pass six widths; no generic User-management permission was added.

Own-source validation: the cold-load and assignment-permission suites pass together (**2 files,21 tests**), targeted ESLint passes, and `git diff --check` is clean. Root owns the combined type/lint/build/E2E/API gates; the employee agent owns the additional self-account regressions. Source changes are uncommitted.

This pass adds **15 actual screenshot inspections** to the52 previously recorded, for **67 image inspections** in this packet. This counts before/after versions separately, not unique filenames:

- Assignment320: crash before fix, loading-guard fix with old permission toast, final Admin clean version (3).
- Notifications768/1023: before and after width correction (4).
- AdvPartner320 invalid profile URL denial, then password320/768 after role correction (3).
- Partner profile390 before and390/768 after false-warning correction (3).
- Partner assignment390/1023 after both fixes (2).

Final inspected captures include `qa_admin-320-_admin_projects_modal_add_employee_to_project_projectId_999971.png`, `qa_admin-{768,1023}-_admin_modal_notification_sheet.png`, `qa_adv_partner-{320,768}-_adv_partner_advance_payments_modal_change_password.png`, `qa_partner-{390,1023}-_partner_projects_modal_add_employee_to_project_projectId_999971.png`, and `qa_partner-{390,768}-_partner_dashboard_modal_user_profile.png` in the final evidence directory. The drawer at1023 and profile at768 intentionally remain right-side drawers; they are not bottom-sheet edge failures.
