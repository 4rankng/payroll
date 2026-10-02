# Admin marks attendance (điểm danh / tan ca) for self check-in employees + Cấu hình điểm danh page overhaul

## Outcome

Admin can, from the **Cấu hình điểm danh** page (`CheckInSettingsPage`), mark a self check-in
employee as **đã điểm danh** (check-in) and **đã tan ca** (check-out) for today when the employee
could not use the app. The page is revamped with Untitled UI PRO patterns rendered in the repo's
shadcn/Tailwind dialect.

## Background (verified in source)

- `AdminCreateCheckIn` (POST `/admin/attendances`) creates a **check-in only**, today only,
  and only before `shift.end + 4h` — the employee must self-checkout.
- `Approve` (POST `/admin/attendances/:id/approve`) already completes an existing record
  (checkout = configured shift end K, full earning, immediate quota credit, review stamp).
- Gap: once a shift is over, an employee with **no record at all** can never get one —
  the missing feature is a complete check-in+check-out record.
- Casbin: admin has `/api/*` wildcard; `adv_partner` has no `/admin/attendances` → marking is admin-only.
- Page is mounted for admin and adv-partner (`advance-payments/check-in-settings`), no separate mobile variant.

## Constraints

- Repo UI laws: Vietnamese copy, 44px mobile targets, one-row controls, WCAG 4.5:1, light mode, shadcn dialect.
- Do not touch the dirty bulk-transfer files in the worktree.
- Existing page tests (`CheckInSettingsPage.test.tsx`, route parity tests) must keep passing.
- Quota/earning rules must reuse existing machinery — no new money logic.

- Marking surface must be mobile-admin friendly (responsive page; dialog mobile-first) and must
  target **only** self check-in enabled employees (`check_in_enabled`) — enforced in UI and
  re-enforced server-side by the shifts endpoint.

## Non-goals

- Marking for past days (today only, server-enforced).
- Arbitrary check-in/out times (configured shift anchors only).
- Changes to the standalone `AdminCreateCheckInDialog` on AdvancePaymentsPage.

## Phases

1. **Backend** — `AdminCreateCheckInRequest.WithCheckout`; `AdminCreateCheckIn(..., withCheckout)`:
   when true and `now ≥ shift.End`, create a completed record (in = shift start, out = shift end,
   gates "admin", earning via `calculateEarningAmount`, quota hold `QuotaCreditHoldDuration` +
   after-commit credit enqueue, no auto-reject task). Bypass the past-end+grace rejection.
   Zero/uncomputable earning → validation error, no record. Unit tests for all branches.
2. **Frontend marking** — `with_checkout` in types/hook; new `CheckInMarkAttendanceDialog`
   (resolve today's record via admin attendance list; create with shift select + "đã tan ca"
   checkbox when shift ended; else Approve to mark tan ca; completed → info). Card gets a
   "Điểm danh" action (admin + current month + enabled only); page wires the dialog.
3. **Untitled UI PRO overhaul** — restyle header/tab bar/toolbar/cards in Untitled UI dashboard
   patterns using repo tokens; keep all tested labels/aria intact.
4. **Verify** — `go test ./internal/app/services/attendance/...`, `go build ./...`,
   `pnpm lint && pnpm type-check`, targeted vitest; `make api-test` if backend env available.

## Acceptance criteria

- Admin marks a no-record employee after shift end → complete record with earning; appears in admin attendance list.
- Admin marks tan ca for an open record → completed, quota credited, review stamped.
- adv_partner sees no marking controls; API rejects them anyway (Casbin).
- All existing + new tests pass.
