# Test Plan — Treasury Band Propagation (W1–W4)

**RESULT: ALL SURFACES PASS — 2026-09-05 23:40 (local dev, agent-browser + vision verification)**

| Surface | Result | Key evidence |
|---|---|---|
| A dashboard | ✅ | 3 tiles (no Chờ giải ngân); LEDs green/green/gray = primary/success/neutral; computed `gap:0` + seam `::before` fading gradient (flush single surface — vision "gaps" read disproven by computed styles); priority detail has no paid figure; tile click → `/admin/ledger?period=salary` |
| B timesheet | ✅ | treasury-grid + panel + signal LED (`bg-muted-foreground/40` = uncalibrated, honest) + rail; KPI tile click → `aria-pressed=true` + selected border, second click → false; 5 tiles |
| C wallet desktop | ✅ | as-of "Cập nhật 05/09/2026 23:34"; Đang chi trả rail (no fee rail); aurora ×1; sync btn h-11; old hero + header Đồng bộ gone; sync click → card's mismatch dialog opens (sandbox NCC 0 vs local 53.5M — correct); bulk section present below fold |
| D wallet mobile | ✅ | card + as-of + pending rail; old ct-stat tile absent; 3 action buttons ≥44px; no clipping |
| E advance regression | ✅ | desktop: grid ×1, fee rail INTACT on band wallet card, aurora ×1, mesh ×3, signal ×3, demand panel present (shortfall>0), LED ×1; mobile 390px: compact card + fee rail, no as-of (by design — compact disables balance query) |
| F drilldown | ✅ | opens via tile (programmatic click; earlier coordinate misses were automation artifacts — count-up re-render offset, zero console errors); `slateTokensRemaining: 0` in live DOM; empty state renders |

Known non-issues: "NCC: 0 ₫" amber chip = pre-existing sandbox provider divergence data. Mesh/tick sub-visibility in screenshots = intentional faintness + compression; confirmed via computed styles.

---


## Unit / type gates (already run — recorded evidence)

| Gate | Result |
|---|---|
| `vitest run src/components/timesheet` | 34/34 pass |
| `vitest run src/components/disbursement` (WalletBalanceCard 4 tests) | 4/4 pass |
| `vitest run src/components/admin-dashboard` | 6/6 pass |
| `tsc -p tsconfig.app.json --noEmit` non-test errors | 34 before → 34 after (zero regression; known backlog incl. pre-existing PayrollControlCenter summary-type errors + test-globals class) |
| eslint touched files | clean |

## Browser matrix (agent-browser, localhost:3000)

### A. Admin dashboard — desktop 1440×900
1. Strip shows **3** tiles: Đã trả kỳ này / Lợi nhuận kỳ này / Nhân sự đang làm — NO "Chờ giải ngân" tile.
2. Tiles carry: pulsing signal LED (tone-colored), mesh dot grid, fading seams between cells, `font-financial` tabular values, count-in animation.
3. No duplicate metrics: pending salary appears ONLY in priority list; paid salary ONLY in strip; priority "Lương chờ giải ngân" detail contains NO paid figure.
4. Click "Đã trả kỳ này" → navigates to salary ledger. (logic)
5. Priority list + activity panel unchanged (min-h rows, refresh button).

### B. Timesheet `/admin/timesheet` — desktop
1. PayrollControlCenter: fading seam between forecast zone (left) and KPI grid (right); no hard border-r line.
2. Forecast panel: emerald wash + mesh + pulsing signal LED (tone = calibration state) + "Nên chuẩn bị" figure animates in.
3. Footer "Khoảng dự báo trung tâm" has accent-tick rail.
4. KPI tiles: click "Chờ duyệt" → table filters to pending (filter state syncs); click again → clears. (logic)
5. `paid` tile still spans 3 cols; primary tile tint intact.

### C. Wallet `/admin/wallet` — desktop
1. Dark treasury hero: aurora + HUD grid + LED label "Ví tiền", sync icon-button (44px), balance figure.
2. NEW: "Cập nhật {timestamp}" line under balance; rail shows "Đang chi trả" + amount (NOT the fee rail).
3. Old HeroBalance light card + StatTile + header "Đồng bộ" button are GONE.
4. Transactions list + bulk-transfer list render below.
5. Click sync icon → toast (khớp or mismatch dialog). (logic — safe, read-only op)

### D. Wallet — mobile viewport 390×844
1. Dark treasury hero (compact non-fee rail + as-of), action buttons (Tra cứu / Tải file / Chuyển tiền), transactions panel.
2. No daisy ct-stat "Đang chi trả" tile (absorbed into card rail).

### E. Advance payments `/admin/advance-payments` — REGRESSION (desktop + mobile)
1. Hero band unchanged: 3↔4 col grid, seams, mesh, aurora on wallet panel, fee rail (Tổng phí trả / Phí tháng này) on wallet card — my card edit must not have altered band usages.
2. Pipeline band: LED rail + status cells + processing-time cells unchanged.
3. Mobile full-bleed wallet hero renders (may now also show as-of line — accepted info gain).
4. Demand panel appears only when shortfall > 0 (presence = signal).

### F. Health drilldown (dashboard → Chấm công strip tile)
1. Sheet opens; desktop table + mobile cards use theme tokens (no visible change expected — regression smoke only).
2. Monogram initials still render.

## Deploy (after browser pass)

1. `git push origin main`.
2. `make deploy` from repo ROOT (amd64; never arm64). Watch build → push → SSH deploy to completion (piped output hides failures — check exit + prod state).
3. Verify prod container tag == `HEAD` after deploy. This deploy also carries wave #48 (`59e2c177`…`abc23dd1`, previously deploy-unverified).
