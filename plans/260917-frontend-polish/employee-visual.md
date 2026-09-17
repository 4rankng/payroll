# Employee, AdvancePartner and Accountant visual QA — 2026-09-17

## Environment and scope

Local Vite5173/API8080 only, synthetic QA identities, main checkout left uncommitted. No production traffic or payment-provider interaction. Browser viewports1280,768,390,320px at900px height; a final1023/1024/1440 route pass checks the responsive component boundary and wide desktop.

This report distinguishes screenshots captured by automation from screenshots opened and visually inspected. It does not claim exhaustive coverage of every possible data, hardware or external-provider state.

## Routes and actual interactions

- Regular Employee `/employee`: full-name header, all wage figures/privacy toggle, account/password sheet and validation, month/year chooser and past periods, payroll/attendance rows, receiving-bank card, notification tabs and notification details.
- Flexible Employee `/employee`: current/past periods, allowance/use totals, amount presets, fee-loading/failure/retry, current-amount fee confirmation, request history expansion, receiving-bank card, location permission explanation and attendance empty state, password sheet/validation, notifications.
- AdvancePartner: `/adv-partner/advance-payments` request list, desktop request/employee tabs and employee detail sheet, mobile options/import sheet; `/advance-payments/employees` list and per-employee detail sheet; `/advance-payments/check-in-settings` project selector, status filters and empty search; `/users` desktop rows/mobile edit controls, employee editor and existing-bank picker/search/selection.
- Accountant `/accountant`: all four tabs selected by actual clicks; approval selection and date-filter selection reset; horizontal table scrolling; weekly/monthly/custom export modes, date calendars, advanced project/employee filters; result upload dialog; statement calendar dialog. No approve/export/import/payment submission was performed through the visual browser.
- Shared/subcomponents: MobileSearchInput, notification list/detail/read control, BankSelector desktop dialog/mobile drawer, employee confirmation sheet, password reveal controls, check-in employee cards and both AdvancePartner employee-detail variants.

Read-only browser runs allow GET/OPTIONS and block API mutations. Synthetic fee/error responses and notification-read responses are explicitly intercepted where needed to exercise state changes without sending funds. Initial file-history exploration used synthetic rows, then source/API inspection showed AdvancePartner has no global file permission; that unsupported desktop entry point was removed instead of widening the global API.

## Capture and visual evidence

Artifacts are in `/tmp/payroll-ui-qa/visual-owned/`:

- `inventory.json`:40 baseline route/tab captures at1280/768/390/320, all40 opened with `view_image`. All had no document overflow or uncaught page errors.
- `interactions.json`:140 recorded stages,139 screenshot captures and1 screenshot timeout (AdvancePartner320 employee detail). Actual tab/filter/dialog behavior was exercised, not inferred from URL query parameters. The failed detail capture was replaced successfully in `deep.json`.
- `deep.json`:65 recorded stages,57 screenshots and8 failed exploratory stages. Failures were the unsupported file-history action on3 mobile/tablet widths, bank option locator/API403 on4 widths, and1 notification opener during local runtime reload. These are retained rather than counted as passes. Bank permission and unsupported entry points were subsequently fixed and rerun below; notification details succeeded at390/320/768.
- `last-check.json`:24 final stages across all4 widths. Existing-bank search/selection and absent unsupported file action are asserted; Accountant export/upload/statement dialogs measured with zero internal overflow. At320/390/768 each dialog starts at0 and ends exactly at viewport width. Desktop dialogs remain centered.
- `confirm-axe-settled.log`:5 confirmed contrast nodes in the open flexible Employee confirmation state; `confirm-axe-fixed.log`:zero violations after existing-token corrections. The confirmation was opened and settled before analysis.
- `bank-stable.json` and `boundaries/inventory.json`:final narrow recapture and additional responsive route evidence, completed results recorded below.

Actual nested screenshots inspected include: Employee320/390/768 password/validation,320 month/year and notifications,1280 password/notification,320 fee failure/confirmation,768 confirmation,320 notification detail; AdvancePartner1280 request/employee tab,desktop employee detail,file-history exploration/user editor,320 user editor/bank/import,390 employee detail,768 check-in card; Accountant320 weekly/monthly/export filters/project picker/upload/statement,1280 monthly/custom/statement,390 upload/employee picker,768 custom/date calendar. Final bank options at1280/390/320 and final Accountant320/768 export and1280 statement were inspected again after fixes. Captured states not included here were used for interaction/DOM evidence and are not individually claimed as visually reviewed.

The approval table scroll was exercised directly:390px viewport had client356/scroll407 and scrolled51px;320 had client286/scroll407 and scrolled121px. The surrounding page did not overflow. Selection reset after changing date was verified, without approving payroll.

## Demonstrated changes

- Search icon/clear padding persists at tablet/desktop breakpoints; the shared input now has an accessible name and a non-submit clear control.
- Check-in cards retain full employee identity, CCCD/code, status and action in compact wrapping rows. Both Admin and AdvancePartner views were inspected. AdvancePartner request/list cards wrap names, dates and amounts without hiding identities.
- Remote font stylesheets no longer block React startup. Existing fonts load asynchronously with system fallback; there is no added font asset dependency or inline script handler. Login and authenticated Accountant plus dialog remain usable while font requests are deliberately held pending in Chromium and WebKit.
- Notification rows use readable metadata colors, native independent detail/read buttons, keyboard activation and44px mark-read targets. The previous nested interactive structure and pale read opacity were removed.
- Fee previews clear immediately when amount changes, ignore stale responses and expose a retry after calculation failure. A request cannot proceed using a stale or failed quote. Current-amount confirmation is covered without executing payment.
- Confirmation amount/account/fee/button colors now pass contrast checks using existing employee700/red600 tokens; the request preview uses emerald700.
- AdvancePartner bank reference GET is now permitted for its existing scoped employee editor. Tests explicitly keep bank writes, individual bank administration, employee/accountant reference access and global file history denied. Actual bank GET and all denied cases were separately checked by the runtime agent.
- The unsupported AdvancePartner global file-history action was removed from the desktop view, matching mobile permissions. Admin file history is unchanged.
- Backend SaoKe export notification now records the authenticated asset uploader as SenderID; previously SenderID0 violated the sender foreign key. Unit coverage checks actor and metadata; the existing live export integration flow now checks newly created history when a file is generated.

Root-owned fixes reported during this review include clipped compact-dialog close buttons and bottom-sheet right gaps at768px. The final Accountant action checks confirm close/header fit and edge alignment after those shared changes.

## Focused verification

- Employee portal:26/26 Chromium/WebKit tests passed (`employee-fee-final.log`), including stale quote/error/retry and no financial submission.
- Font startup:4/4 Chromium/WebKit tests passed (`font-loading-final.log`), anonymous and authenticated while font requests are pending.
- Focused visual-change unit suite:38/38 across4 files (`visual-unit-final.log`). Final color-only form check:17/17 (`confirmation-color-unit.log`).
- Bank role-policy package `go test ./internal/app/services/auth -race`:passed (`bank-permissions-unit.log`). Sender notification handler package race test passed; integration harness compiled (`notification-handler-unit.log`, `notification-integration-compile.log`). Runtime agent owns full API/race/vet evidence.
- Source lint/types and final broad frontend suites are maintained by root; scoped final source check results recorded below. Earlier root full frontend result was143 files/614 tests passed before the final color-only correction.

## Limits

Fonts were blocked in most deterministic screenshot runs after remote Google Fonts stalled browser navigation. Initial screenshots used loaded remote fonts; dedicated startup tests exercise pending and eventual loaded stylesheets. Screenshots show Chromium emulation and system fallback where blocked, not physical iOS/Android rendering.

Real GPS/camera/push, physical keyboard safe areas, external email/Zalo delivery, real payment rails and production data were not exercised. The visual browser deliberately did not send/cancel money, change passwords, save employee edits, toggle check-in permission or submit files. Separate local integration suites cover authorized sandbox writes. Unrelated legacy auth/employees/projects/timesheet browser fixtures remain stale and are not reported as passing.

## Final corrections and verification

- Final `pnpm type-check` found a remaining reference to the removed fee setter in the successful-submit path. It now invokes the new preview hook with zero, clearing the quote/error and cancelling older work. Added an isolated synthetic-success browser regression: payload once, dialog closes, amount/quote clears and another request stays disabled.2/2 Chromium/WebKit passed (`employee-success-final.log`); the earlier26 Employee cases remain separately recorded, not presented as a single rerun.
- Final targeted lint passed; full app/node `pnpm type-check` passed (`employee-success-lint.log`, `owned-final-types-retry.log`). The initial failure log is retained (`owned-final-types.log`).
- Stable bank selection recapture passed6/6 stages at390/320; both selected-editor screenshots were opened with `view_image`. The editor remains visible with bank selection retained and save/password controls reachable. No save was submitted.
- Additional boundary screenshots visually inspected so far: regular Employee1023, flexible Employee1440, AdvancePartner request list1023. All show readable full identities and no document overflow. Final complete boundary matrix result is appended after the capture finishes.

## Completed boundary, ad and self-account follow-up

- Final boundary matrix completed30/30 captures at1023/1024/1440,0 document overflow and0 uncaught page errors (`boundaries/inventory.json`). The initial screenshot timeout is retained in `boundaries.log`; `boundaries-retry.log` completed the remaining captures without repeating successful states. Additional images opened: AdvancePartner1024 request list/users and1440 check-in settings.
- The1024 AdvancePartner users table visibly collapsed project names into one-word columns, producing tall rows. Scoped table/name/project minimum widths now preserve compact readable rows inside the existing horizontal scroller. Added a named focusable region. Final768/1024/1280 table and keyboard ArrowRight checks passed6/6 stages (`table-ad.json`); final768/1024 table screenshots were visually inspected. No phone card behavior changed.
- Final synthetic employee campaign check passed8/8 sheet/card states at320/768/1023/1024 (`ad-final.json`). Below1024 the sheet reaches both viewport edges;1024 uses its intended420px side panel. Every sheet/card control measured at least44px, long title stays clear of the close button, sheet dismiss→card→reopen→dismiss card works. The320 sheet/card and768/1023/1024 sheet screenshots were opened with `view_image`. No external CTA was followed.
- Ad touch-target corrections are in `EmployeeAdSheet.tsx` and `EmployeeAdBanner.tsx`; root supplied the width breakpoint correction. Header reserves44px close space and at least56px height; compact card actions remain44px. Targeted ad/table lint and14/14 unit tests passed (`ad-table-final-lint.log`, `ad-table-final-unit.log`).
- Confirmed backend native self-password policy and handler: every known role can POST `/auth/change-password`; target identity comes from authenticated context and the current password is required. Fixed modal metadata/permissions to allow this existing self action for all5roles. Dedicated `SelfAccount` read/update capability avoids granting AdvPartner/Accountant/Employee generic User read/update. Existing Admin/Partner profile modal now checks self-account read; profile URL role scope remains Admin/Partner. Both registries are aligned.18/18 permission tests (11new,7existing assignment), targeted lint and full app/node types passed (`password-permissions-unit.log`, `password-permissions-lint.log`, `password-permissions-types.log`).
- Actual AdvancePartner account menu→password→Escape passed at390/1280, no submission (`self-password.json`). Admin/Partner agent independently checked the legitimate AdvancePartner password URL at6widths and Partner profile parity. Password dialog footer controls now have44px minimum height at tablet widths; final768 action check is recorded below. There is no standalone ChangePasswordModal unit file; the attempted filename selection is retained in `password-modal-final-unit.log` and is not counted as a test pass. The existing Employee ChangePasswordSheet tests and permission tests are separate coverage.

Application source is frozen for root's final build and patch packaging. Root's combined44browser cases and635unit tests are separate broader evidence; the final permission18, ad/table14 and live action measurements above cover the last changes.
- Final768 actual account-menu password open/close passed; both footer buttons and header close measured44px. Screenshot `qa_adv_partner-768-self-password.png` was visually inspected, with readable labels and full-width bottom-sheet alignment. Evidence:`self-password-768.json`/`.log`. No password was changed.
