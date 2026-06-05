# TypeScript Error Fix Strategy

**Date**: 2026-06-05
**Total Errors**: 681 across 192 files
**Goal**: Reach zero `tsc -b` errors to unblock CI pipeline

## Error Landscape Summary

| Category | Root Cause | Errors | Files | Impact |
|----------|-----------|--------|-------|--------|
| A | ApiResponse unwrap mismatch | ~80 | ~30 | Hooks + services |
| B | Missing/wrong properties on types | ~60 | ~25 | Pages + components |
| C | BackendAuditLogDetailResponse incomplete | ~28 | 2 | Audit log |
| D | DayType enum / Vietnamese strings | ~17 | ~5 | Timesheet |
| E | Filters not assignable to Record<string, unknown> | ~30 | ~15 | Services + hooks |
| F | Duplicate identifiers / redeclarations | ~13 | 3 | Modal registry |
| G | Missing modules / exports | ~25 | ~15 | Various |
| H | Recharts element typing | ~5 | 2 | Dashboard charts |
| I | Service Worker typing | ~21 | 1 | sw.ts |
| J | Responsive table generic constraint | ~11 | 1 | UI component |

## Strategy: 8 Focused Sessions

Each session targets one root cause category. Sessions are ordered by **cascade impact** — fixing earlier sessions may auto-resolve errors in later sessions.

### Key Principle: Fix Types, Not Consumers

Most errors are caused by incorrect/missing type definitions. Fixing the type definition in `src/types/` resolves all downstream consumer errors at once. We should almost never add `as any` or `@ts-ignore`.

---

### Session 1: ApiResponse Unwrap (Category A)
**~80 errors → ~30 files**

**Problem**: Services return `ApiResponse<T>` but callers expect `T` directly.

**Approach**:
1. Read `src/types/api.types.ts` to understand `ApiResponse<T>` shape
2. Read 3-4 service files and their corresponding hooks to see the pattern
3. Decide: either (a) add a generic unwrap utility, (b) change hooks to extract `.data`, or (c) update service return types
4. Apply the fix consistently across all affected hooks

**Files to touch**:
- `src/types/api.types.ts` (type definition)
- `src/hooks/api/*.ts` (all hooks — ~15 files)
- Possibly `src/services/api/*.ts` (if changing service layer)

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~80

---

### Session 2: Missing Properties on Core Types (Categories B + C)
**~88 errors → ~27 files**

**Problem**: Type definitions don't match what the backend actually returns. Properties like `budget`, `priority`, `manager`, `total_payout_vnd`, `spent`, `progress` are missing from `Project`. Audit log types are incomplete.

**Approach**:
1. Read each type definition file in `src/types/api/`
2. Read the consumers to identify which properties they expect
3. Add missing properties to type definitions (with correct types and optional markers)
4. For AuditLogDetailSheet: extend `BackendAuditLogDetailResponse` with all accessed fields

**Priority types to fix**:
- `src/types/api/project.types.ts` — add `budget`, `priority`, `manager`, `total_payout_vnd`, `spent`, `progress`, `clients`
- `src/types/api/audit.types.ts` — add all fields accessed in AuditLogDetailSheet
- `src/types/api/timesheet.types.ts` — add missing fields
- `src/types/api/employee.types.ts` — add missing fields
- `src/types/api/loan.types.ts` — add missing fields
- `src/types/api/dashboard.types.ts` — add missing fields
- `src/types/api/import-export.types.ts` — add missing fields
- `src/types/api/project-employee.types.ts` — add missing fields
- `src/types/api/advance-payment.types.ts` — add missing fields

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~88

---

### Session 3: Filter Index Signatures (Category E)
**~30 errors → ~15 files**

**Problem**: Filter interfaces like `LedgerFilters`, `BankFilters`, `EmployeeFilters`, `AuditLogsParams` fail when passed to functions expecting `Record<string, unknown>`. TypeScript doesn't allow implicit index signatures on interfaces.

**Approach**:
1. Read all `*Filters` and `*Params` interfaces
2. Add `[key: string]: unknown` index signature to each, OR
3. Change the functions that receive them to use a generic constraint instead of `Record<string, unknown>`

**Files to touch**:
- `src/types/api/*.ts` (filter/params interfaces)
- `src/services/api/*.ts` (service functions with `Record<string, unknown>` params)
- `src/components/ui/responsive-table.tsx` (Category J — same root cause)

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~30 (+11 from Category J)

---

### Session 4: DayType Enum + Missing Exports (Categories D + G)
**~42 errors → ~20 files**

**Problem**: 
- DayType enum doesn't include Vietnamese string literals used in code
- Missing type exports: `TimesheetStatus`, `PaymentStatus`, `PayrateStructureType`, etc.
- Missing module declarations: `zustand`, `zustand/middleware`, `xlsx`

**Approach**:
1. Read `src/types/api/timesheet.types.ts` and find DayType
2. Add Vietnamese string literals to the union: `"ngày thường"`, `"ngày nghỉ"`, `"ngày lễ"`
3. For missing exports: search for where they're defined and fix the export, or re-export from the correct file
4. For missing modules: check if packages are installed, add type declarations if needed

**Files to touch**:
- `src/types/api/timesheet.types.ts`
- `src/types/api/payrate.types.ts`
- `src/types/payrates.ts`
- Various import fixes across ~15 files

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~42

---

### Session 5: Duplicate Identifiers (Category F)
**~13 errors → 3 files**

**Problem**: Modal registry has duplicate exports. Types have duplicate interface declarations.

**Approach**:
1. Read `src/constants/modalRegistry/index.ts` — find duplicates
2. Remove duplicate re-exports, keep single source of truth
3. Fix any duplicate type declarations

**Files to touch**:
- `src/constants/modalRegistry/index.ts`
- Possibly `src/types/modal-config.types.ts`
- Possibly `src/types/modal.types.ts`

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~13

---

### Session 6: Service Worker Types (Category I)
**~21 errors → 1 file**

**Problem**: `src/sw.ts` uses Service Worker globals (`self`, `respondWith`, `waitUntil`, `notification`, etc.) without proper type declarations.

**Approach**:
1. Add `/// <reference lib="webworker" />` at the top of sw.ts
2. Add `declare let self: ServiceWorkerGlobalScope`
3. Or create a `src/sw.d.ts` declaration file with all needed types
4. Ensure tsconfig includes the SW file with proper lib settings

**Files to touch**:
- `src/sw.ts`
- Possibly `src/sw.d.ts` (new file)
- Possibly `tsconfig.app.json` (if lib adjustment needed)

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~21

---

### Session 7: Recharts + Misc UI Typing (Categories H + J)
**~16 errors → ~5 files**

**Problem**: Recharts elements lack proper generic types. Responsive table generic constraint too strict.

**Approach**:
1. For Recharts: use proper recharts types or minimal type assertions
2. For responsive-table: relax the `Record<string, unknown>` constraint (likely fixed in Session 3)

**Files to touch**:
- `src/components/admin-dashboard/BankDonutChart.tsx`
- `src/components/admin-dashboard/FinancialChart.tsx`
- `src/components/ui/responsive-table.tsx`

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should drop by ~16

---

### Session 8: Cleanup & Remaining
**~0-30 remaining errors**

**Problem**: After sessions 1-7, there will be a tail of misc errors. Some may have been introduced by fixes, others are edge cases.

**Approach**:
1. Run `npx tsc -b 2>&1 | grep "error TS"` to see what's left
2. Fix remaining errors one by one
3. Run final verification

**Verify**: `npx tsc -b 2>&1 | grep "error TS" | wc -l` — should be **0**

---

## Verification Protocol

After **each session**:
1. `cd /Users/dev/Documents/projects/payroll/frontend && npx tsc -b 2>&1 | grep "error TS" | wc -l`
2. Compare to previous count — verify expected reduction
3. `npx tsc -b 2>&1 | grep "error TS" | sed 's/(.*//' | sort | uniq -c | sort -rn | head -20` — check no new files appeared
4. Run `npm run build` to confirm Vite still builds (tsc errors don't block Vite, but we want both clean)

## Commit Strategy

- One commit per session, with message: `fix(ts): session N - [category description]`
- Do NOT squash — we want bisection capability if a fix introduces a regression

## Error Count Tracking

| Session | Target | Expected Reduction | Running Total |
|---------|--------|--------------------|---------------|
| Start | — | — | 681 |
| 1 | ApiResponse | -80 | ~601 |
| 2 | Missing properties | -88 | ~513 |
| 3 | Filter signatures | -41 | ~472 |
| 4 | DayType + exports | -42 | ~430 |
| 5 | Duplicates | -13 | ~417 |
| 6 | Service Worker | -21 | ~396 |
| 7 | Recharts + UI | -16 | ~380 |
| 8 | Cleanup | -remaining | **0** |
