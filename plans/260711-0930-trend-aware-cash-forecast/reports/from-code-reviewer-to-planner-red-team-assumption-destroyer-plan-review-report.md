# Red Team Review: Assumption Destroyer (Scope Auditor)

**Plan:** Trend-aware cash-readiness forecast (detrend gamma + growth-adjusted P50)
**Reviewer role:** SCOPE AUDITOR — verify every "reuse X" / "already exists" / config / signature claim against the codebase
**Verdict:** Plan is built on a false premise about what the math engine operates on, cites unverifiable baseline numbers, and references fields/files/wiring points that do not exist. Multiple Critical findings.

---

## Finding 1: The plan's foundational premise is false — the gamma fit does NOT operate on "grand totals"

- **Severity:** Critical
- **Location:** plan.md "The problem" (lines 19-34); plan.md "Data flow change" (lines 126-140); phase-02 "Overview" (line 13) and Requirements (line 21)
- **Flaw:** The plan repeatedly asserts the deployed forecast "fits a gamma distribution to the **grand totals** of historical per-Ky timesheet cohorts" (plan.md:21) and frames the entire fix as "remove the trend from the historical grand totals" (plan.md:69, plan.md:83, plan.md:128, plan.md:131; phase-02:13, phase-02:21). This is factually wrong. `forecastDemandDistributionBetween` builds `rem[]` where each entry is `endAmount - cumulativeAt(fromCycleDay)` — i.e. the **remaining demand between two cycle-day positions**, not the grand total.
- **Failure scenario:** An implementer following the plan will either (a) refactor the engine to fit grand totals (breaking the existing remaining-demand semantics that the wallet forecast also depends on), or (b) follow phase-02 step 4's contradictory instruction to "fit OLS on `rem`, not grand totals" and discover the overview/data-flow/architecture diagrams all describe a different algorithm. Either path produces a forecast of a different quantity than the one currently shipped, invalidating every baseline number in the plan.
- **Evidence:**
  - `backend/internal/app/services/wallet_demand_forecast_math.go:426-443` — `rem` is built from `cumulativeAt(throughCycleDay) - cumulativeAt(fromCycleDay)`, gated on `throughCycleDay > fromCycleDay`.
  - `backend/internal/app/services/cash_readiness_forecast.go:165` — the provider is called with `fromCycleDay = pc.CycleDayToday, throughCycleDay = pc.MaxCycleDay`, i.e. remaining demand from today to pay date.
  - The plan even contradicts itself: phase-02 step 4 (line 85) says "Fit OLS on the per-cycle remaining values (`rem`, not grand totals — the remaining-demand curve is what we're forecasting)", directly negating phase-02 line 13/21 and plan.md line 21/69.
- **Suggested fix:** Rewrite the plan around the actual semantics. The decision of whether to detrend `rem` (remaining-demand series) vs. `grandTotal` (full-cycle totals) is load-bearing and currently unresolved inside the plan itself. State which one, and recompute every baseline number against that quantity.

---

## Finding 2: The "41.4% baseline error" is not reproducible — no test or fixture produces it

- **Severity:** Critical
- **Location:** plan.md "The backtest evidence" (lines 36-49); phase-01 "Success Criteria" (line 97)
- **Flaw:** The plan's central justification ("flat gamma = 41.4% error, trend methods = 28-31%") is presented as a measured backtest result, and phase-01 commits to codifying "Ky-2 baseline = 41.4% (from the 2026-07-11 backtest)" as a constant. The only backtest test in the repo uses synthetic data (amounts 100–800) and reports 7.3% and 0.0% error — nothing like 41.4%. There is no fixture, no committed number, and no way to reproduce the claim from the codebase. It is an assertion by the author.
- **Failure scenario:** Phase 1's success criterion ("Baseline mean abs % error is captured as a committed constant: Ky-2 baseline = 41.4%") cannot be met because the fixtures that allegedly produce 41.4% do not exist and must be hand-extracted from a production DB the reviewer cannot access. Phase 2's acceptance criterion ("reduces error by ≥10pp vs baseline") is then measured against a fabricated anchor, so "improvement" is unfalsifiable.
- **Evidence:**
  - `backend/internal/app/services/cash_readiness_backtest_test.go` — entire file uses fabricated cycles (e.g. `{Amount: 100}, {Amount: 200}, {Amount: 700}` totaling 1000).
  - Running it: `cycle 2026-05: projected=1020 actual=1100 abs%err=7.3%` / `cycle 2026-06: ... abs%err=0.0%` — no 41.4% anywhere.
  - phase-01 lines 64-68 list 4 fixture JSON files to "Create" — none exist (`find` for `*backtest*` returns only the test file).
- **Suggested fix:** Either (a) commit the real production fixtures and the script that produces 41.4% before this plan is approved, so the baseline is reproducible; or (b) mark the 41.4% as a hypothesis to be validated by Phase 1, and make Phase 1's first deliverable "extract fixtures + reproduce or refute the 41.4% figure" with a go/no-go gate.

---

## Finding 3: The claimed "foundation" (trendRatio, trendRatioCutoff) is uncommitted working-tree state, not a landed baseline

- **Severity:** Critical
- **Location:** plan.md "Constraints" (line 161); plan.md "Dependencies" (line 182); phase-02 Requirements (line 20)
- **Flaw:** The plan states "`trendRatio` field + `trendRatioCutoff` constant added in this session are the foundation — Phase 2 builds on them, does not re-add them" (plan.md:161) and "Builds on the `trendRatio` field and `trendRatioCutoff` constant added in the 2026-07-11 diagnosis session" (plan.md:182). These are **uncommitted modifications** in the working tree, not part of any commit. `git log -S trendRatio` returns nothing. The plan treats a dirty working tree as a landed dependency.
- **Failure scenario:** If Phase 1 or 2 is picked up on a fresh checkout (or the working-tree changes are stashed/discarded, which is routine), `trendRatio` and `trendRatioCutoff` do not exist and every Phase 2 instruction that says "the constant added in the 2026-07-11 session" silently fails to compile. The implementer re-adds them, duplicating work and risking divergence from the values the backtest numbers were computed against.
- **Evidence:**
  - `git status --short` shows ` M backend/internal/app/services/wallet_demand_forecast_math.go` (modified, not staged/committed).
  - `git log --oneline --all -S "trendRatio" -- backend/internal/app/services/wallet_demand_forecast_math.go` → empty.
  - `git diff HEAD` confirms `const trendRatioCutoff = 3.0` and the `trendRatio float64` field are in the uncommitted diff.
- **Suggested fix:** Land the trendRatio/trendRatioCutoff changes as a prerequisite commit (or a Phase 0) before this plan starts. Do not predicate four phases of work on uncommitted edits.

---

## Finding 4: Phase 2 adds fields to `demandDistribution`, but those fields can never reach the provider or the API

- **Severity:** High
- **Location:** phase-02 "Implementation Steps" step 3 (lines 77-82); phase-02 Architecture (line 55); phase-03 Requirements (line 27)
- **Flaw:** Phase 2 adds `trendSlope`, `trendIntercept`, `detrended` to `demandDistribution` (the internal math struct). But `TimesheetAccrualProvider.ProjectAccrual` maps `demandDistribution` → `AccrualProjection` by hand-copying only six fields (`P50, Expected, P95, Method, Confidence, BasisCycles`); everything else is dropped. Phase 3 then claims the `TrendAwareAccrualProvider` "computes `trendRatio` (already available from the math engine)" and that `AccrualProjection` gains `GrowthRate`/`BlendWeights` — but `trendRatio`, `trendSlope`, `trendIntercept`, `detrended` are never plumbed into `AccrualProjection`, so the wrapper provider has no way to read them to decide whether to blend.
- **Failure scenario:** `TrendAwareAccrualProvider` calls `inner.ProjectAccrual(...)`, receives an `AccrualProjection` that lacks `trendRatio`/`detrended`, and cannot execute its own core branch ("If `trendRatio < trendRatioCutoff` → return inner unchanged; else blend"). The auto-selection acceptance criterion ("selected automatically when `trendRatio ≥ 3.0`") is unimplementable as specified.
- **Evidence:**
  - `backend/internal/app/services/cash_readiness_forecast.go:40-47` — `AccrualProjection` has exactly 6 fields, none of them trend-related.
  - `backend/internal/app/services/cash_readiness_forecast.go:69-77` — the mapping discards `dist.trendRatio`, `dist.gammaShape`, etc.
  - phase-03:93 — "Compute `trendRatio` (already available from the math engine)" is false at the provider layer.
- **Suggested fix:** Phase 2 must also extend `AccrualProjection` (and the `ProjectAccrual` mapping) with `TrendRatio`, `TrendSlope`, `Detrended` — or Phase 3's `TrendAwareAccrualProvider` must bypass the interface and call `forecastDemandDistributionBetween` directly (which breaks the "reuse the interface" claim). Pick one and specify it.

---

## Finding 5: The DI wiring point is `services/init.go`, not `container.go`

- **Severity:** High
- **Location:** phase-03 "Related Code Files" (line 68); phase-03 step 6 (line 106)
- **Flaw:** Phase 3 instructs: "Modify `backend/internal/app/bootstrap/container.go` — swap `NewTimesheetAccrualProvider()` for `NewTrendAwareAccrualProvider(...)` in the DI wiring." The actual construction happens in `internal/app/bootstrap/services/init.go:514` (`services.NewCashReadinessForecastService(..., nil, clk, cfg.CashForecast)`), which passes `nil` as the provider (the service then defaults it to `NewTimesheetAccrualProvider()` internally at `cash_readiness_forecast.go:127`). `container.go` contains zero references to the forecast provider.
- **Failure scenario:** The implementer edits `container.go`, finds no `NewTimesheetAccrualProvider()` call there, and either (a) gives up, or (b) adds wiring in the wrong place that never executes because the service constructor's `nil`-default path still selects the statistical provider. The trend-aware provider silently never runs in production.
- **Evidence:**
  - `grep -rn "NewTimesheetAccrualProvider\|ForecastProvider" internal/app/bootstrap/` → only `routes_timesheet.go` (route registration) and `container.go:329` (handler construction, which takes the already-built `services.CashReadiness`).
  - `internal/app/bootstrap/services/init.go:514` is the true construction site, and it passes `nil`.
  - `internal/app/services/cash_readiness_forecast.go:126-128` — the `nil` provider is defaulted inside the service, so even fixing `init.go` requires passing the wrapper explicitly, not just editing a wiring line.
- **Suggested fix:** Rewrite phase-03 step 6 to target `services/init.go:514`, pass `NewTrendAwareAccrualProvider(NewTimesheetAccrualProvider())` explicitly (not `nil`), and delete the claim that the swap is a one-line edit in `container.go`.

---

## Finding 6: Phase 1 references a `prepareByCycleDay` field that does not exist

- **Severity:** High
- **Location:** phase-01 "Implementation Steps" step 3 (line 82)
- **Flaw:** The snapshot-store instruction says: "after each `GetCashReadiness` call, if `cycleDayToday == prepareByCycleDay`, write the P50/expected to Redis." There is no `prepareByCycleDay` field anywhere in the codebase. The pay-cycle model exposes `CycleDayToday` and `MaxCycleDay`, and a `PrepareByDate(leadDays)` method (returns a `time.Time`, not a cycle-day int). The "prepare-by cycle day" concept is invented by the plan.
- **Failure scenario:** The snapshot trigger condition cannot be coded. An implementer must invent the mapping (prepare-by date → cycle day) themselves, with no spec for which cycle day counts as "prepare-by", leading to either snapshots taken too early (wrong forecast window) or never (no accuracy data ever logged, defeating Phase 1's purpose).
- **Evidence:**
  - `grep -rn "prepareByCycleDay\|PrepareByCycleDay"` across `internal/` → zero hits.
  - `internal/pkg/clock/pay_cycle.go:189-191` — `PrepareByDate(leadDays int) time.Time` is the only prepare-by API.
  - `internal/pkg/clock/pay_cycle.go:140-141` — struct has `CycleDayToday` and `MaxCycleDay`, no prepare-by cycle day.
- **Suggested fix:** Define the snapshot trigger in terms of existing fields (e.g. `CycleDayToday == MaxCycleDay - leadDays`, or compute the prepare-by date and compare calendar dates), and state the exact condition.

---

## Finding 7: Phase 2 internal contradiction on the damping factor φ (0.9 vs 0.98)

- **Severity:** Medium
- **Location:** phase-02 "Why OLS" (line 58) vs phase-02 step 2 (line 75) vs phase-02 Risk Assessment (line 116)
- **Flaw:** The same phase specifies three different damping factors for `nextFitted`: line 58 says "φ=0.9"; line 75 says "Phi defaults to 0.98"; line 116 says "φ=0.98 damping". With a 6-point series where the latest value is 457M, the difference between φ=0.9 and φ=0.98 on the extrapolated `next_fitted = intercept + slope*n*φ` changes the July forecast by a material margin. The plan also never makes φ configurable (unlike Phase 3's blend weights), so the chosen value is a silent hardcoded constant.
- **Failure scenario:** The implementer picks one of the two values; the July P50 lands somewhere unexpected; the "≥300M" acceptance criterion either passes or fails based on an undocumented coin-flip between 0.9 and 0.98, and no test pins which is correct.
- **Evidence:** phase-02:58 ("φ=0.9"), phase-02:75 ("Phi defaults to 0.98"), phase-02:116 ("φ=0.98 damping") — three statements, two distinct numbers.
- **Suggested fix:** Pick one φ value, justify it against the backtest (which doesn't exist yet — see Finding 2), and add it to `CashForecastConfig` with validation so it is tunable, consistent with how Phase 3 treats the blend weights.

---

## Finding 8: Phase 4's accuracy endpoint depends on a log-file path with no directory/rotation infrastructure

- **Severity:** Medium
- **Location:** phase-01 Architecture (line 56), phase-01 step 4 (line 88); phase-04 step 2 (line 64)
- **Flaw:** Phase 1 writes accuracy entries to `logs/cash-readiness-accuracy.jsonl` and Phase 4's accuracy endpoint "Reads the last N entries from `logs/cash-readiness-accuracy.jsonl`". The codebase's log handling centers on `logs/app.log` (config.go:351) via a `LogConfig.File` setting; there is no existing JSON-lines log writer, no rotation helper for arbitrary `.jsonl` files, and the asynq worker (where the accuracy logger runs) has no guarantee of write access to a relative `logs/` path in the containerized prod layout (config.go:325-328 shows prod uses `/app/uploads`, implying container paths). The plan calls rotation "monthly for cleanliness" but specifies no mechanism.
- **Failure scenario:** In production the asynq worker either fails to create `logs/cash-readiness-accuracy.jsonl` (permission/path error, swallowed because Phase 1 says "errors are logged, never surfaced"), so no accuracy data is ever recorded; or the file grows unbounded because no rotation is actually implemented; or the Phase 4 endpoint reads an empty/missing file and always returns "chưa đủ dữ liệu", making the entire accuracy feature a permanent placeholder.
- **Evidence:** `config.go:349-351` — only `app.log` is configured; no `.jsonl` infrastructure. phase-01:108 — "rotate monthly for cleanliness" with no mechanism. phase-01:88 — the JSON-lines append is specified with no library or rotation strategy.
- **Suggested fix:** Either reuse Redis (the plan already introduces a Redis snapshot key) to store the rolling accuracy window as a list, or specify the exact rotation library/path/permissions. Do not leave "log file" as an unspecified IO surface that two phases depend on.

---

## Finding 9: The re-trend math is applied to a remaining-demand distribution, but the headline combines it with `confirmed + p50` — the trend is on the wrong quantity

- **Severity:** High
- **Location:** phase-02 Architecture (lines 49-54); plan.md "Decision" (lines 64-67); `cash_readiness_forecast.go:171-174`
- **Flaw:** The deployed headline is `cashToPrepare = confirmed + proj.P50` where `proj.P50` is the p50 of the **remaining-demand** distribution (accrual still to come from today to pay date). Phase 2's re-trend step (`re_trended_p50 = detrended_p50 + nextFitted`) adds the OLS trend's next-step value to this remaining-demand quantile. But the OLS trend (per phase-02 step 4) is fit on `rem` — the remaining-demand series across cycles — so `nextFitted` is the trend's projection of *next cycle's remaining demand*, which is exactly the quantity being forecast. Adding `detrended_p50` (residual spread) to `nextFitted` (trend level) is correct *only if* the detrend/retrend is self-consistent on the same series. The plan's overview (plan.md:64-67) instead writes `P50 = detrended_gamma_p50 × growth_factor`, a **multiplicative** model, while phase-02 is **additive** (`detrended_p50 + nextFitted`). These are different algorithms.
- **Failure scenario:** The implementer follows plan.md's multiplicative blend (`detrended × growth_factor`) or phase-02's additive re-trend (`detrended + nextFitted`) depending on which section they read first; the two produce materially different July forecasts; the "≥300M" gate is satisfied by one and failed by the other, and the backtest numbers in plan.md (computed by the author on an unspecified model) match neither.
- **Evidence:** plan.md:64-67 (multiplicative: `P50 = detrended_gamma_p50 × growth_factor`); phase-02:50-53 (additive: `re_trended_p50 = detrended_p50 + next_fitted`); phase-03:54-56 (a third formulation: `blendP50 = 0.6*detrendedP50 + 0.4*growthP50`). Three different composition formulas across three documents.
- **Suggested fix:** Pick exactly one composition model (multiplicative growth-adjust, additive re-trend, or weighted blend) and use it consistently in plan.md, phase-02, and phase-03. Re-derive the backtest table against the chosen model.

---

## Finding 10: Phase 1's asynq scheduling for "days 11, 18, 25, 2" is underspecified and the existing scheduler has no per-Ky date-hook pattern

- **Severity:** Medium
- **Location:** phase-01 step 5 (line 90)
- **Flaw:** Phase 1 says "add an asynq task `cash_readiness:accuracy` scheduled for 1 day after each pay date (days 11, 18, 25, 2)." The existing asynq periodic tasks use either `@every <duration>` or a single crontab spec like `"1 0 * * *"` (daily). There is no existing pattern for "four specific day-of-month triggers," and crontab day-of-month cannot cleanly express "day 2 of next month" for Ky-4 (which pays on day 1 of the next month) alongside same-month days. The plan provides no crontab spec and no registration function signature.
- **Failure scenario:** The implementer registers one crontab per Ky, but Ky-4's "day 2" logic crosses month boundaries and depends on the *work* month not the calendar month, so the snapshot key `{ky}:{yyyy-mm}` and the trigger date go out of sync. The accuracy logger fires on the wrong day, reads a stale or missing snapshot, and logs "no snapshot" forever for Ky-4.
- **Evidence:** `internal/infra/asynq/mux.go:62-64` — `RegisterPeriodicTasks` is currently a no-op stub; all real registrations use `@every` or a single crontab string (e.g. `mux.go:113` `"1 0 * * *"`). No multi-date or per-Ky scheduling exists. phase-01:90 gives no crontab spec.
- **Suggested fix:** Specify the exact crontab spec(s) or, better, register a single daily `@every 24h` task that internally checks `clock.NextTimesheetPayCycle` to decide whether today is T+1 for any Ky, and idempotently logs. Reuse the existing pay-cycle model rather than hardcoding calendar days.

---

## Summary

The plan reads as a confident narrative, but its load-bearing factual claims about the codebase are wrong or unverifiable:

1. The math engine does not fit grand totals (Finding 1) — the entire premise is misdescribed.
2. The 41.4% baseline is fabricated/unreproducible (Finding 2).
3. The "foundation" fields are uncommitted working-tree edits (Finding 3).
4. The new fields cannot cross the provider boundary (Finding 4).
5. The DI wiring location is wrong (Finding 5).
6. A referenced field does not exist (Finding 6).
7. The damping factor contradicts itself (Finding 7).
8. The log infrastructure is hand-waved (Finding 8).
9. The composition formula appears in three incompatible forms (Finding 9).
10. The scheduler integration is underspecified (Finding 10).

Recommendation: do not approve this plan for implementation. The author must first (a) land the trendRatio working-tree changes, (b) commit reproducible fixtures + a script that actually yields 41.4%, (c) reconcile the rem-vs-grand-totals premise, and (d) pick one composition formula and one damping value.
