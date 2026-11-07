# ADR-013: Staff Day Console — toto.sg Mission-Control Port

Date: 2026-11-07
Status: Accepted (implementation on `feat/staff-day-console`; deploy gated on owner screenshot review)

## Context

The payroll staff surfaces (Admin, Partner, Accountant — desktop and mobile) carried per-page chrome: frosted header cards, radial washes, icon chips, pill sidebars, wrapped filter rows. The owner directed porting toto.sg's dense mission-control layout language — same data, same permissions, same Vietnamese copy, same behaviour — as a light day-console. Visual source of truth: toto.sg's dashboard console idiom (hairline panels on a measurement grid, mono kickers, tabular figures, 36px tables, one-row filters), applied with the deviations below.

## Decision

**One instrument language, applied additively — no engine swaps, no palette change:**

1. **Palette locked.** The literal toto palette (`#EEF6F8` canvas / `#0F9E86` signal) is explicitly NOT adopted. The identity comes from geometry, density, and typography, not hue: payroll keeps the `#e2e8e4` canvas, white panels, emerald `--primary` signal, amber/red status. A literal-palette swap, if ever wanted, is a token-only follow-up in `variables.css`.
2. **Light only.** No dark mode and no theme toggle for staff surfaces.
3. **Console tokens are derived, not absolute.** `--console-grid`/`--console-glow-*`/`--console-panel-shadow` are declared once at `:root` referencing live role tokens (`--foreground`, `--primary`, `--info`), so admin/partner/accountant scopes resolve their own values through normal inheritance.
4. **One backdrop per shell.** The 48px measurement grid + corner glows (`.console-shell`) lives on `AdminLayout`'s `main`, `PartnerLayout`'s `#main-content`, and the AccountantPage root. All per-page radial washes (`AdminPageCanvas`, `PartnerLayout`, `.admin-shell`/`.partner-shell` CSS) are removed. Below 768px the grid is dropped entirely (fixed attachment janks on iOS; hairlines are noise under thumb scrolling).
5. **Console styling lands at call sites, not in shared base components.** GUARDRAIL: `Button`, `Input`, `Select`, `Badge`, `Tabs` base classes are untouched (shared with login/employee). Console styling reaches pages through (a) the `console-*` classes, (b) explicit new variants on `Table` (`variant="console"`) and `Card` (`variant="console"`), and (c) shared page primitives (`PageHeader`, `AdminPageFrame`, mobile primitives) that only staff surfaces consume. Employee portal and login are excluded and must not regress.
6. **`.console-table` rules are declared outside any cascade layer.** The `Table` primitive bakes utility classes (`text-xs`, `text-fg-tertiary`) onto `th`; a `@layer components` rule would lose to them everywhere. Unlayered author styles beat all layered styles, so the 36px rows and mono headers win without `!important` and without editing the primitive's defaults (default variant unchanged — NotFound and employee surfaces keep the old look).
7. **Form comfort is out of scope for density.** 44px mobile targets, 16px mobile inputs, and 36px desktop controls stay (the `tap-target-44` / `input-font-size-16` / `control-density` test floors are preserved). Density applies to data display only. Desktop (≥1200px) filter inputs/selects settle at 36px; tablets keep the existing 40px tablet floor.
8. **Accountant joins the admin token scope.** The admin override block in `variables.css` now also matches `[data-accountant-ui], html.accountant-route-active`, and `AccountantPage` mirrors `AdminLayout`'s route-active effect, so portalled dialogs inherit tokens outside the page wrapper.
9. **Charts consume semantic tokens.** Chart series colors move from hardcoded HSL literals to `--chart-1…6`/`--chart-other` (Tailwind `chart-*` bridge), with a fixed per-series mapping that keeps every series distinct within each dashboard group. New chart primitives (`ConsoleDonut`, `ConsoleGauge`) are plain SVG — no chart-library rewrite; recharts stays where it already renders.
10. **Sidebar navigation stays.** The 4-item mobile top bar and auto-hide chrome from toto do NOT transfer; the existing sidebar + 64px mobile bottom nav remain, restyled to the instrument language (mono kickers, 2px signal bars, 8% emerald active tint, 36px items, 44px nav targets).

**Dead-code note:** the spec-referenced components `PremiumStatStrip`, `PendingApprovalCard`, `RecentActivitiesCard`, and `FinancialChart` had no rendering consumers on `main` — the live dashboard implements the same patterns inline in `DashboardOverview.tsx`. Restyling targeted the live components; the unused files were left in place (pre-existing dead code, not made dead by this change).

### Addendum — packed gallery (latest toto.sg insights pass, 2026-11-07)

The later toto.sg commits (`32843bf` insights gallery, `29e4dbb` pack + drop prose, `e15bcbc`/`f58465b` row tightening) moved the console from a stack of cards to a **packed gallery**. Payroll adopts the parts that transfer:

11. **Panels own their headers.** `ConsolePanel` (`components/shared/console-panel.tsx`) renders the mono kicker, 13px title, optional short meta, actions and footer as part of the panel. `DashboardSectionHeader` and `DashboardAreaHeader` are retired; page-level `ConsoleSectionHeading` (16px, meta line, no description) separates galleries.
12. **Masonry packing.** `MasonryGrid` packs uneven panels on a 12-column track by spanning 1px rows from measured heights (8px gap); below `lg` panels stack normally. Admin finance/workforce galleries use `lg:col-span-7/5/5` so a short panel's column is filled by the next panel instead of leaving page background. The two-panel partner gallery uses a stretch grid, not masonry, so an empty leaderboard fills the row height.
13. **Observed vs previous period.** `ConsoleDelta` renders a signed change beside the figure. The monthly financial table carries a Δ column (visible at `xl`) and three stat chips (accumulated, latest month, latest revenue); partner KPI tiles carry MoM deltas, suppressed when the current value is zero so a meaningless `+46%` on `0` is never shown.
14. **Stat chips under charts.** `ConsoleStat`/`ConsoleStatRow` use the bare variant (top hairline) for 2–3 figures under charts: bank donut (headcount, leader, bank count), salary distribution (mean/min/max), profitability (leader/total/count), monthly table. No icon chrome.
15. **Non-zero row law.** Queues render only real rows: a completed approval queue, a fully disbursed salary queue, a zero activity channel, or a zero mobile task row is omitted. An empty priority panel shows one all-clear line; an empty task list renders nothing.
16. **No filler prose.** Descriptions that restated a title were removed; remaining descriptions are period/count/source meta. The fair-cage explanatory line from toto is not ported.
17. **Compact empty states.** `EmptyState` gains `showIllustration`; dashboard empties pass `false` and render text plus a next action only.
18. **Slow-load fallback.** The bank-distribution panel switches from its skeleton to a compact "tải chậm" row with a retry after 8s (the endpoint can stall); no giant empty skeleton.
19. **Mobile parity.** Mobile dashboards use the same `ConsolePanel` primitives via `MobileOperationsPanel`/`MobileTaskList`; the month bar, header kicker, stat strip (20px value / 9px mono label), sidebar active bar and 13px labels match the desktop contract. Zero rows and navigation-only "tasks" are removed; quick actions/nav keep the destinations.

## Consequences

- Per-page chrome (glass header cards, radial washes, icon chips) is retired on staff surfaces; `AdminPageHeaderCard` becomes a transparent layout wrapper.
- KPI figures on dashboards render compact (`formatCompactVND` → `1,03 tỷ`) with the exact figure carried in `title`/`aria-label`; tables, exports, and PDFs keep exact values.
- Visual baselines require re-baselining for every staff surface after the Phase 1/2 review; new baselines added for accountant, ledger, and partner payment history. Employee-portal and login baselines must stay byte-identical.

## Verification

Phase gates: per-phase `pnpm type-check`, targeted unit tests (`formatCompactVND`, `getConsoleKicker`, donut/gauge math), `pnpm build`; Phase 1 screenshot review at 1280/768/390/320; Phase 2 screenshots at 1440/1280/1024/768/390/320 with data numbers matching pre-change baselines; overflow assertion `scrollWidth === clientWidth` on every swept route; keyboard/reduced-motion/AA contrast pass; Playwright visual suite after re-baseline.
